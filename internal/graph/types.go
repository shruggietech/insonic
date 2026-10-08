// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
	"strings"
	"sync"
)

type Node struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	Reference json.RawMessage `json:"reference"`
}
type Edge struct {
	ID   string `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}
type Change struct {
	Kind            string `json:"kind"`
	CatalogRevision int64  `json:"catalog_revision"`
	Nodes           []Node `json:"nodes,omitempty"`
	Edges           []Edge `json:"edges,omitempty"`
}
type References struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}
type Checkpoint struct {
	Generation      int64 `json:"generation"`
	Sequence        int64 `json:"sequence"`
	CatalogRevision int64 `json:"catalog_revision"`
}
type Rows []map[string]any
type Adapter interface {
	Capabilities() map[string]any
	EnsureSchema(context.Context) error
	BindWorkspace(context.Context, string) error
	FenceGeneration(context.Context, string, string, int64) error
	ApplyRevision(context.Context, catalog.OutboxClaim) error
	ReadCheckpoint(context.Context, string, string) (Checkpoint, error)
	ReadRefs(context.Context, string) (References, error)
	Query(context.Context, string, string, map[string]contracts.QueryParameter) (Rows, error)
	Explain(context.Context, string, string, map[string]contracts.QueryParameter) (Rows, error)
	ExportSnapshot(context.Context, string) (References, error)
	Rebuild(context.Context, catalog.OutboxClaim) error
	Close()
}
type session interface {
	Query(context.Context, bool, string, map[string]any) (Rows, error)
	Commit(context.Context) error
	Rollback(context.Context)
}
type engine interface {
	Begin(context.Context, bool) (session, error)
	Close()
}
type adapter struct {
	engine  engine
	backend string
	version string
	mu      sync.Mutex
}

func (a *adapter) Close() { a.mu.Lock(); defer a.mu.Unlock(); a.engine.Close() }
func (a *adapter) Capabilities() map[string]any {
	dialects := []string{"ladybug-cypher"}
	if a.backend == "arcadedb" {
		dialects = []string{"arcade-opencypher", "arcade-sql"}
	}
	return map[string]any{"adapter_id": a.backend, "backend_version": a.version, "contract_version": contracts.Version, "state": "available", "dialects": dialects, "operations": []string{"media-list", "speaker-search", "model-list", "text-search", "time-range", "evidence-traverse", "graph-view"}, "schema_version": 1, "typed_parameters": []string{"string", "integer", "number", "boolean", "null"}, "read_only": true, "atomic_publication": true, "cancellation": true, "limits": map[string]any{"native_bytes": 65536, "result_rows": 500, "result_bytes": 512 << 10, "timeout_seconds": 20}}
}
func (a *adapter) sql() bool { return a.backend == "arcadedb" }
func (a *adapter) statement(sql, cypher string) string {
	if a.sql() {
		return sql
	}
	return cypher
}
func (a *adapter) run(ctx context.Context, s session, write bool, sql, cypher string, p map[string]any) (Rows, error) {
	return s.Query(ctx, write, a.statement(sql, cypher), p)
}
func (a *adapter) EnsureSchema(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, false)
	if e != nil {
		return e
	}
	defer s.Rollback(ctx)
	var statements []string
	if a.sql() {
		statements = []string{
			"CREATE VERTEX TYPE Entity IF NOT EXISTS", "CREATE PROPERTY Entity.id IF NOT EXISTS STRING", "CREATE INDEX IF NOT EXISTS ON Entity(id) UNIQUE",
			"CREATE EDGE TYPE EvidenceLink IF NOT EXISTS", "CREATE DOCUMENT TYPE GraphCheckpoint IF NOT EXISTS", "CREATE PROPERTY GraphCheckpoint.id IF NOT EXISTS STRING", "CREATE INDEX IF NOT EXISTS ON GraphCheckpoint(id) UNIQUE",
			"CREATE DOCUMENT TYPE GraphOwner IF NOT EXISTS", "CREATE PROPERTY GraphOwner.id IF NOT EXISTS STRING", "CREATE INDEX IF NOT EXISTS ON GraphOwner(id) UNIQUE",
			"CREATE DOCUMENT TYPE GraphReceipt IF NOT EXISTS", "CREATE PROPERTY GraphReceipt.id IF NOT EXISTS STRING", "CREATE INDEX IF NOT EXISTS ON GraphReceipt(id) UNIQUE",
		}
	} else {
		statements = []string{
			"CREATE NODE TABLE IF NOT EXISTS Entity(id STRING,workspace STRING,entity_id STRING,kind STRING,reference STRING,PRIMARY KEY(id))",
			"CREATE REL TABLE IF NOT EXISTS EvidenceLink(FROM Entity TO Entity,id STRING,kind STRING)",
			"CREATE NODE TABLE IF NOT EXISTS GraphCheckpoint(id STRING,generation INT64,sequence INT64,revision INT64,PRIMARY KEY(id))",
			"CREATE NODE TABLE IF NOT EXISTS GraphOwner(id STRING,workspace STRING,PRIMARY KEY(id))",
			"CREATE NODE TABLE IF NOT EXISTS GraphReceipt(id STRING,receipt STRING,PRIMARY KEY(id))",
		}
	}
	for _, q := range statements {
		if _, e = s.Query(ctx, true, q, nil); e != nil {
			return e
		}
	}
	return s.Commit(ctx)
}
func integer(v any) (int64, error) {
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	}
	return 0, contracts.Fail("operation_failed")
}
func (a *adapter) checkpoint(ctx context.Context, s session, key string) (Checkpoint, bool, error) {
	rows, e := a.run(ctx, s, false, "SELECT generation,sequence,revision FROM GraphCheckpoint WHERE id=:id", "MATCH (c:GraphCheckpoint) WHERE c.id=$id RETURN c.generation AS generation,c.sequence AS sequence,c.revision AS revision", map[string]any{"id": key})
	if e != nil {
		return Checkpoint{}, false, e
	}
	if len(rows) == 0 {
		return Checkpoint{}, false, nil
	}
	if len(rows) != 1 {
		return Checkpoint{}, false, contracts.Fail("operation_failed")
	}
	g, e := integer(rows[0]["generation"])
	if e != nil {
		return Checkpoint{}, false, e
	}
	seq, e := integer(rows[0]["sequence"])
	if e != nil {
		return Checkpoint{}, false, e
	}
	rev, e := integer(rows[0]["revision"])
	return Checkpoint{g, seq, rev}, true, e
}
func (a *adapter) setCheckpoint(ctx context.Context, s session, key string, c Checkpoint, exists bool) error {
	p := map[string]any{"id": key, "generation": c.Generation, "sequence": c.Sequence, "revision": c.CatalogRevision}
	var e error
	if exists {
		_, e = a.run(ctx, s, true, "UPDATE GraphCheckpoint SET generation=:generation,sequence=:sequence,revision=:revision WHERE id=:id", "MATCH (c:GraphCheckpoint) WHERE c.id=$id SET c.generation=$generation,c.sequence=$sequence,c.revision=$revision", p)
	} else {
		_, e = a.run(ctx, s, true, "INSERT INTO GraphCheckpoint SET id=:id,generation=:generation,sequence=:sequence,revision=:revision", "CREATE (:GraphCheckpoint {id:$id,generation:$generation,sequence:$sequence,revision:$revision})", p)
	}
	return e
}
func validTarget(w, t string) bool { return contracts.ValidID(w) && contracts.ValidID(t) }
func (a *adapter) FenceGeneration(ctx context.Context, w, t string, g int64) error {
	if !validTarget(w, t) || g < 1 {
		return contracts.Fail("invalid_request")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, false)
	if e != nil {
		return e
	}
	defer s.Rollback(ctx)
	key := w + "|" + t
	cp, exists, e := a.checkpoint(ctx, s, key)
	if e != nil {
		return e
	}
	if exists && g < cp.Generation {
		return contracts.Fail("conflict")
	}
	cp.Generation = g
	if e = a.setCheckpoint(ctx, s, key, cp, exists); e != nil {
		return e
	}
	return s.Commit(ctx)
}
func receipt(c catalog.OutboxClaim) string {
	raw, _ := json.Marshal(struct {
		Workspace   string
		Target      string
		Sequence    int64
		Predecessor int64
		Operation   string
		Digest      string
		Revision    int64
	}{c.WorkspaceID, c.Target, c.Sequence, c.Predecessor, c.OperationID, c.Digest, c.Revision})
	return string(raw)
}
func (a *adapter) readReceipt(ctx context.Context, s session, key string) (string, error) {
	rows, e := a.run(ctx, s, false, "SELECT receipt FROM GraphReceipt WHERE id=:id", "MATCH (r:GraphReceipt) WHERE r.id=$id RETURN r.receipt AS receipt", map[string]any{"id": key})
	if e != nil {
		return "", e
	}
	if len(rows) == 0 {
		return "", nil
	}
	if len(rows) != 1 {
		return "", contracts.Fail("operation_failed")
	}
	r, ok := rows[0]["receipt"].(string)
	if !ok {
		return "", contracts.Fail("operation_failed")
	}
	return r, nil
}
func (a *adapter) apply(ctx context.Context, s session, c catalog.OutboxClaim, change Change) error {
	key := c.WorkspaceID + "|" + c.Target
	rk := key + "|" + strconv.FormatInt(c.Sequence, 10)
	old, e := a.readReceipt(ctx, s, rk)
	if e != nil {
		return e
	}
	if old != "" {
		if old != receipt(c) {
			return contracts.Fail("conflict")
		}
		return nil
	}
	cp, exists, e := a.checkpoint(ctx, s, key)
	if e != nil {
		return e
	}
	if !exists || cp.Generation != c.Generation || cp.Sequence != c.Predecessor || c.Sequence != cp.Sequence+1 {
		return contracts.Fail("conflict")
	}
	// Touch the fence in the same write transaction; concurrent generations conflict.
	cp.Sequence = c.Sequence
	cp.CatalogRevision = change.CatalogRevision
	if e = a.setCheckpoint(ctx, s, key, cp, true); e != nil {
		return e
	}
	if _, e = a.run(ctx, s, true, "DELETE VERTEX FROM Entity WHERE workspace=:workspace", "MATCH (n:Entity) WHERE n.workspace=$workspace DETACH DELETE n", map[string]any{"workspace": c.WorkspaceID}); e != nil {
		return e
	}
	for _, n := range change.Nodes {
		p := map[string]any{"id": c.WorkspaceID + "|" + n.ID, "workspace": c.WorkspaceID, "entity_id": n.ID, "kind": n.Kind, "reference": string(n.Reference)}
		if _, e = a.run(ctx, s, true, "CREATE VERTEX Entity SET id=:id,workspace=:workspace,entity_id=:entity_id,kind=:kind,reference=:reference", "CREATE (:Entity {id:$id,workspace:$workspace,entity_id:$entity_id,kind:$kind,reference:$reference})", p); e != nil {
			return e
		}
	}
	for _, edge := range change.Edges {
		p := map[string]any{"from": c.WorkspaceID + "|" + edge.From, "to": c.WorkspaceID + "|" + edge.To, "id": edge.ID, "kind": edge.Kind}
		if _, e = a.run(ctx, s, true, "CREATE EDGE EvidenceLink FROM (SELECT FROM Entity WHERE id=:from) TO (SELECT FROM Entity WHERE id=:to) SET id=:id,kind=:kind", "MATCH (f:Entity),(t:Entity) WHERE f.id=$from AND t.id=$to CREATE (f)-[:EvidenceLink {id:$id,kind:$kind}]->(t)", p); e != nil {
			return e
		}
	}
	_, e = a.run(ctx, s, true, "INSERT INTO GraphReceipt SET id=:id,receipt=:receipt", "CREATE (:GraphReceipt {id:$id,receipt:$receipt})", map[string]any{"id": rk, "receipt": receipt(c)})
	return e
}
func validateChange(c catalog.OutboxClaim) (Change, error) {
	h := sha256.Sum256(c.Document)
	if !validTarget(c.WorkspaceID, c.Target) || c.Generation < 1 || c.Sequence < 1 || c.Revision < 1 || !contracts.ValidID(c.OperationID) || hex.EncodeToString(h[:]) != c.Digest {
		return Change{}, contracts.Fail("conflict")
	}
	var change Change
	if decodeChange(c.Document, &change) != nil || (change.Kind != "invalidate" && change.Kind != "refresh") || change.CatalogRevision < 0 || len(change.Nodes) > 100000 || len(change.Edges) > 200000 {
		return Change{}, contracts.Fail("invalid_request")
	}
	if change.Kind == "invalidate" && (len(change.Nodes) > 0 || len(change.Edges) > 0) {
		return Change{}, contracts.Fail("invalid_request")
	}
	ids := map[string]bool{}
	for _, n := range change.Nodes {
		if n.ID == "" || len(n.ID) > 1024 || n.Kind == "" || ids[n.ID] || len(n.Reference) > 8192 || !json.Valid(n.Reference) {
			return Change{}, contracts.Fail("invalid_request")
		}
		ids[n.ID] = true
	}
	edges := map[string]bool{}
	for _, e := range change.Edges {
		if e.ID == "" || edges[e.ID] || !ids[e.From] || !ids[e.To] || e.Kind == "" {
			return Change{}, contracts.Fail("invalid_request")
		}
		edges[e.ID] = true
	}
	return change, nil
}
func (a *adapter) ApplyRevision(ctx context.Context, c catalog.OutboxClaim) error {
	change, e := validateChange(c)
	if e != nil {
		return e
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, false)
	if e != nil {
		return e
	}
	if e = a.apply(ctx, s, c, change); e != nil {
		s.Rollback(ctx)
		return e
	}
	e = s.Commit(ctx)
	if e == nil {
		return nil
	}
	s.Rollback(context.Background())
	// Either engine can report COMMIT failure after commit. Verify exact receipt.
	probe, pe := a.engine.Begin(ctx, true)
	if pe != nil {
		return e
	}
	defer probe.Rollback(ctx)
	got, pe := a.readReceipt(ctx, probe, c.WorkspaceID+"|"+c.Target+"|"+strconv.FormatInt(c.Sequence, 10))
	if pe == nil && got == receipt(c) {
		return nil
	}
	return e
}
func (a *adapter) ReadCheckpoint(ctx context.Context, w, t string) (Checkpoint, error) {
	if !validTarget(w, t) {
		return Checkpoint{}, contracts.Fail("invalid_request")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, true)
	if e != nil {
		return Checkpoint{}, e
	}
	defer s.Rollback(ctx)
	cp, _, e := a.checkpoint(ctx, s, w+"|"+t)
	return cp, e
}
func (a *adapter) ReadRefs(ctx context.Context, w string) (References, error) {
	out := References{Nodes: []Node{}, Edges: []Edge{}}
	if !contracts.ValidID(w) {
		return out, contracts.Fail("invalid_request")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, true)
	if e != nil {
		return out, e
	}
	defer s.Rollback(ctx)
	rows, e := a.run(ctx, s, false, "SELECT entity_id AS id,kind,reference FROM Entity WHERE workspace=:workspace ORDER BY entity_id", "MATCH (n:Entity) WHERE n.workspace=$workspace RETURN n.entity_id AS id,n.kind AS kind,n.reference AS reference ORDER BY id", map[string]any{"workspace": w})
	if e != nil {
		return out, e
	}
	for _, r := range rows {
		id, ok := r["id"].(string)
		kind, kok := r["kind"].(string)
		ref, rok := r["reference"].(string)
		if !ok || !kok || !rok {
			return out, contracts.Fail("operation_failed")
		}
		out.Nodes = append(out.Nodes, Node{id, kind, json.RawMessage(ref)})
	}
	rows, e = a.run(ctx, s, false, "SELECT id,kind,@out.entity_id AS source,@in.entity_id AS destination FROM EvidenceLink WHERE @out.workspace=:workspace ORDER BY id", "MATCH (f:Entity)-[e:EvidenceLink]->(t:Entity) WHERE f.workspace=$workspace RETURN e.id AS id,e.kind AS kind,f.entity_id AS source,t.entity_id AS destination ORDER BY id", map[string]any{"workspace": w})
	if e != nil {
		return out, e
	}
	for _, r := range rows {
		id, ok := r["id"].(string)
		kind, kok := r["kind"].(string)
		from, fok := r["source"].(string)
		to, tok := r["destination"].(string)
		if !ok || !kok || !fok || !tok {
			return out, contracts.Fail("operation_failed")
		}
		out.Edges = append(out.Edges, Edge{id, from, to, kind})
	}
	return out, nil
}
func (a *adapter) ExportSnapshot(ctx context.Context, w string) (References, error) {
	return a.ReadRefs(ctx, w)
}
func (a *adapter) Query(ctx context.Context, dialect, text string, params map[string]contracts.QueryParameter) (Rows, error) {
	if e := ValidateNative(dialect, text, params); e != nil {
		return nil, e
	}
	if (a.sql() && dialect != "arcade-sql" && dialect != "arcade-opencypher") || (!a.sql() && dialect != "ladybug-cypher") {
		return nil, contracts.Fail("unsupported_capability")
	}
	values := map[string]any{}
	for k, p := range params {
		var v any
		d := json.NewDecoder(strings.NewReader(string(p.Value)))
		d.UseNumber()
		if d.Decode(&v) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		if n, ok := v.(json.Number); ok {
			if p.Type == "integer" {
				v, _ = n.Int64()
			} else {
				v, _ = n.Float64()
			}
		}
		values[k] = v
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, true)
	if e != nil {
		return nil, e
	}
	defer s.Rollback(ctx)
	if routed, ok := s.(interface {
		Native(context.Context, string, string, map[string]any) (Rows, error)
	}); ok {
		return routed.Native(ctx, dialect, text, values)
	}
	rows, err := s.Query(ctx, false, text, values)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(rows)
	if len(rows) > 500 || len(raw) > 512<<10 {
		return nil, contracts.Fail("output_limit")
	}
	return rows, nil
}
func (a *adapter) Explain(ctx context.Context, dialect, text string, p map[string]contracts.QueryParameter) (Rows, error) {
	if e := ValidateNative(dialect, text, p); e != nil {
		return nil, e
	}
	tokens, _ := nativeTokens(text)
	if len(tokens) > 0 && tokens[0] == "EXPLAIN" {
		return a.Query(ctx, dialect, text, p)
	}
	return a.Query(ctx, dialect, "EXPLAIN "+text, p)
}
