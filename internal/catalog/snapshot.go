// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
	"strings"
)

type stateTable struct{ name, columns string }

var stateTables = []stateTable{
	{"operation_receipt", "id,digest,revision,result"},
	{"setting_revision", "name,revision,value"},
	{"job", "id,operation_id,duration_ms,generation,state"},
	{"job_attempt", "id,job_id,owner_id,generation,state,lease_until,started_ns,ended_ns,reason"},
	{"workspace_lease", "role,owner_id,generation,lease_until"},
	{"graph_target", "id,next_sequence,checkpoint,owner_id,generation,lease_until"},
	{"graph_event", "target_id,sequence,predecessor,revision,operation_id,document,digest"},
	{"artifact_publication", "id,artifact_id,data"},
}

type TableData struct {
	Name string              `json:"name"`
	Rows [][]json.RawMessage `json:"rows"`
}
type Snapshot struct {
	Kind                      string      `json:"kind"`
	Version                   string      `json:"schema_version"`
	CatalogSchema             int         `json:"catalog_schema"`
	WorkspaceID               string      `json:"workspace_id"`
	Revision                  int64       `json:"revision"`
	Records                   Records     `json:"records"`
	State                     []TableData `json:"state"`
	Digest                    string      `json:"digest"`
	legacyDerivedArtifacts    []string
	legacyInvalidatedDatasets []string
}

func (s Snapshot) digest() (string, error) { s.Digest = ""; return intent(s) }
func (s *Store) Export(ctx context.Context) (Snapshot, error) {
	out := Snapshot{Kind: "catalog-snapshot", Version: contracts.Version, CatalogSchema: SchemaVersion, WorkspaceID: s.workspace}
	options := &sql.TxOptions{ReadOnly: true}
	if s.backend == "postgresql" {
		options.Isolation = sql.LevelRepeatableRead
	}
	tx, e := s.db.BeginTx(ctx, options)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	if e = s.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", s.workspace).Scan(&out.Revision); e != nil {
		return out, sanitize(e)
	}
	out.Records, e = s.readRecords(ctx, tx)
	if e != nil {
		return out, sanitize(e)
	}
	for _, table := range stateTables {
		rows, e := tx.QueryContext(ctx, s.query("SELECT "+table.columns+" FROM "+table.name+" WHERE workspace_id=? ORDER BY "+table.columns), s.workspace)
		if e != nil {
			return out, sanitize(e)
		}
		data := TableData{Name: table.name, Rows: [][]json.RawMessage{}}
		width := len(strings.Split(table.columns, ","))
		for rows.Next() {
			vals := make([]any, width)
			ptr := make([]any, width)
			for i := range vals {
				ptr[i] = &vals[i]
			}
			if e = rows.Scan(ptr...); e != nil {
				rows.Close()
				return out, sanitize(e)
			}
			r := make([]json.RawMessage, width)
			for i, v := range vals {
				if b, ok := v.([]byte); ok {
					v = string(b)
				}
				r[i], e = json.Marshal(v)
				if e != nil {
					rows.Close()
					return out, contracts.Fail("operation_failed")
				}
			}
			data.Rows = append(data.Rows, r)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return out, sanitize(e)
		}
		out.State = append(out.State, data)
	}
	if e = tx.Commit(); e != nil {
		return out, sanitize(e)
	}
	out.Digest, e = out.digest()
	return out, e
}
func (s *Store) Restore(ctx context.Context, snap Snapshot) error {
	if snap.Version != contracts.Version || (snap.CatalogSchema != SchemaVersion && snap.CatalogSchema != 1 && snap.CatalogSchema != 2) {
		return contracts.Fail("incompatible_version")
	}
	if snap.Kind != "catalog-snapshot" {
		return contracts.Fail("invalid_request")
	}
	if snap.WorkspaceID != s.workspace {
		return contracts.Fail("workspace_mismatch")
	}
	if snap.Revision < 0 || !digestPattern.MatchString(snap.Digest) {
		return contracts.Fail("invalid_request")
	}
	digest, e := snap.digest()
	if e != nil || digest != snap.Digest {
		return contracts.Fail("invalid_request")
	}
	if snap.CatalogSchema == 1 {
		if len(snap.State) != len(stateTables)-1 {
			return contracts.Fail("invalid_request")
		}
		snap.State = append(snap.State, TableData{Name: "artifact_publication", Rows: [][]json.RawMessage{}})
	}
	if len(snap.State) != len(stateTables) {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if rev != 0 {
			return contracts.Fail("conflict")
		}
		for _, table := range stateTables {
			var n int
			if e := s.row(ctx, tx, "SELECT count(*) FROM "+table.name+" WHERE workspace_id=?", s.workspace).Scan(&n); e != nil {
				return e
			}
			if n != 0 {
				return contracts.Fail("conflict")
			}
		}
		for _, d := range domains {
			var n int
			if e := s.row(ctx, tx, "SELECT count(*) FROM "+d.name+" WHERE workspace_id=?", s.workspace).Scan(&n); e != nil {
				return e
			}
			if n != 0 {
				return contracts.Fail("conflict")
			}
		}
		for i, table := range stateTables {
			data := snap.State[i]
			if data.Name != table.name {
				return contracts.Fail("invalid_request")
			}
			cols := strings.Split(table.columns, ",")
			marks := strings.TrimSuffix(strings.Repeat("?,", len(cols)+1), ",")
			q := "INSERT INTO " + table.name + "(workspace_id," + table.columns + ") VALUES(" + marks + ")"
			for _, r := range data.Rows {
				if len(r) != len(cols) {
					return contracts.Fail("invalid_request")
				}
				args := []any{s.workspace}
				for j, raw := range r {
					v, e := restoreCell(table.name, cols[j], raw)
					if e != nil {
						return e
					}
					args = append(args, v)
				}
				if _, e := s.exec(ctx, tx, q, args...); e != nil {
					return e
				}
			}
		}
		if e := s.insertRecords(ctx, tx, snap.Records); e != nil {
			return e
		}
		if e := s.validateRestoredState(ctx, tx, snap.Revision); e != nil {
			return e
		}
		if e := s.validateArtifactState(ctx, tx, true); e != nil {
			return e
		}
		// Fence imported authority immediately. Recovery still preserves the old attempts.
		for _, q := range []string{"UPDATE job_attempt SET lease_until=0 WHERE workspace_id=? AND state='running'", "UPDATE workspace_lease SET lease_until=0 WHERE workspace_id=?", "UPDATE graph_target SET lease_until=0 WHERE workspace_id=?"} {
			if _, e := s.exec(ctx, tx, q, s.workspace); e != nil {
				return e
			}
		}
		if _, e := s.exec(ctx, tx, "UPDATE workspace SET revision=? WHERE id=?", snap.Revision, s.workspace); e != nil {
			return e
		}
		result := map[string]any{"restored_revision": snap.Revision, "authority": "expired"}
		if len(snap.legacyInvalidatedDatasets) > 0 {
			result["invalidated_dataset_ids"] = snap.legacyInvalidatedDatasets
		}
		if len(snap.legacyDerivedArtifacts) > 0 {
			if e = s.queueEvidenceArtifacts(ctx, tx, s.workspace, snap.legacyDerivedArtifacts, snap.Revision+1); e != nil {
				return e
			}
			ids, err := s.evidenceArtifacts(ctx, tx, "SELECT id FROM library_cleanup WHERE workspace_id=? AND entry_id=? AND revision=? ORDER BY id", s.workspace, s.workspace, snap.Revision+1)
			if err != nil {
				return err
			}
			result["cleanup_entry_id"], result["cleanup_ids"] = s.workspace, ids
		}
		_, e := s.accept(ctx, tx, contracts.ID(), hash([]byte("restore:"+snap.Digest)), snap.Revision, result)
		if e != nil {
			return e
		}
		if e = s.validateCorpusState(ctx, tx, true); e != nil {
			return e
		}
		return s.validateLibraryState(ctx, tx, true)
	})
}

// Validate the portable representation before either SQL engine can coerce it.
// Only the two declared nullable operational columns admit null.
func restoreCell(table, column string, raw json.RawMessage) (any, error) {
	var value any
	if strict(raw, &value) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if value == nil {
		if column == "ended_ns" || (table == "graph_target" && column == "owner_id") {
			return nil, nil
		}
		return nil, contracts.Fail("invalid_request")
	}
	switch column {
	case "revision", "duration_ms", "generation", "lease_until", "started_ns", "ended_ns", "next_sequence", "checkpoint", "sequence", "predecessor":
		number, ok := value.(json.Number)
		if !ok {
			return nil, contracts.Fail("invalid_request")
		}
		n, e := number.Int64()
		if e != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return n, nil
	default:
		text, ok := value.(string)
		if !ok {
			return nil, contracts.Fail("invalid_request")
		}
		if (column == "id" || strings.HasSuffix(column, "_id")) && !contracts.ValidID(text) {
			return nil, contracts.Fail("invalid_request")
		}
		if (column == "role" && text == "") || (column == "name" && (text == "" || len(text) > 256)) {
			return nil, contracts.Fail("invalid_request")
		}
		return text, nil
	}
}

func (s *Store) validateRestoredState(ctx context.Context, tx *sql.Tx, revision int64) error {
	var count, max int64
	if e := s.row(ctx, tx, "SELECT count(*),coalesce(max(revision),0) FROM operation_receipt WHERE workspace_id=?", s.workspace).Scan(&count, &max); e != nil {
		return e
	}
	if count != revision || max != revision {
		return contracts.Fail("invalid_request")
	}
	checks := []string{
		"SELECT count(*) FROM graph_event e JOIN operation_receipt r ON r.workspace_id=e.workspace_id AND r.id=e.operation_id WHERE e.workspace_id=? AND e.revision<>r.revision",
		"SELECT count(*) FROM job j WHERE workspace_id=? AND generation<>(SELECT count(*) FROM job_attempt a WHERE a.workspace_id=j.workspace_id AND a.job_id=j.id)",
		"SELECT count(*) FROM job_attempt a JOIN job j ON j.workspace_id=a.workspace_id AND j.id=a.job_id WHERE a.workspace_id=? AND (a.generation>j.generation OR (a.generation<j.generation AND a.state='running'))",
		"SELECT count(*) FROM job j LEFT JOIN job_attempt a ON a.workspace_id=j.workspace_id AND a.job_id=j.id AND a.generation=j.generation WHERE j.workspace_id=? AND (a.id IS NULL OR a.state<>j.state)",
		"SELECT count(*) FROM job j LEFT JOIN operation_receipt r ON r.workspace_id=j.workspace_id AND r.id=j.operation_id WHERE j.workspace_id=? AND r.id IS NULL",
		"SELECT count(*) FROM setting_revision s LEFT JOIN operation_receipt r ON r.workspace_id=s.workspace_id AND r.revision=s.revision WHERE s.workspace_id=? AND r.id IS NULL",
		"SELECT count(*) FROM job_attempt WHERE workspace_id=? AND ((state='running' AND ended_ns IS NOT NULL) OR (state<>'running' AND ended_ns IS NULL))",
		"SELECT count(*) FROM graph_target t WHERE workspace_id=? AND (checkpoint>next_sequence OR next_sequence<>(SELECT count(*) FROM graph_event e WHERE e.workspace_id=t.workspace_id AND e.target_id=t.id))",
		"SELECT count(*) FROM graph_event e WHERE workspace_id=? AND (sequence> (SELECT next_sequence FROM graph_target t WHERE t.workspace_id=e.workspace_id AND t.id=e.target_id) OR predecessor<>sequence-1)",
	}
	for _, q := range checks {
		var invalid int64
		if e := s.row(ctx, tx, q, s.workspace).Scan(&invalid); e != nil {
			return e
		}
		if invalid != 0 {
			return contracts.Fail("invalid_request")
		}
	}
	rows, e := tx.QueryContext(ctx, s.query("SELECT document,digest FROM graph_event WHERE workspace_id=?"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var doc, digest string
		if e = rows.Scan(&doc, &digest); e != nil {
			rows.Close()
			return e
		}
		if !validJSON(json.RawMessage(doc)) || hash([]byte(doc)) != digest {
			rows.Close()
			return contracts.Fail("invalid_request")
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	type acknowledgement struct {
		digest   string
		revision int64
		result   string
	}
	receipts := map[string]acknowledgement{}
	rows, e = tx.QueryContext(ctx, s.query("SELECT id,digest,revision,result FROM operation_receipt WHERE workspace_id=?"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var id, digest, result string
		var revision int64
		if e = rows.Scan(&id, &digest, &revision, &result); e != nil {
			rows.Close()
			return e
		}
		if !digestPattern.MatchString(digest) || !validJSON(json.RawMessage(result)) {
			rows.Close()
			return contracts.Fail("invalid_request")
		}
		receipts[id] = acknowledgement{digest, revision, result}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	rows, e = tx.QueryContext(ctx, s.query("SELECT e.target_id,e.sequence,e.operation_id,e.digest,e.revision,t.checkpoint FROM graph_event e JOIN graph_target t ON t.workspace_id=e.workspace_id AND t.id=e.target_id WHERE e.workspace_id=? ORDER BY e.target_id,e.sequence"), s.workspace)
	if e != nil {
		return e
	}
	previous := map[string]int64{}
	for rows.Next() {
		var target, operation, digest string
		var sequence, eventRevision, checkpoint int64
		if e = rows.Scan(&target, &sequence, &operation, &digest, &eventRevision, &checkpoint); e != nil {
			rows.Close()
			return e
		}
		receipt, found := receipts[operationID("ack", target, operation, strconv.FormatInt(sequence, 10))]
		if sequence > checkpoint {
			if found {
				rows.Close()
				return contracts.Fail("invalid_request")
			}
			continue
		}
		var result struct {
			Target   string `json:"target"`
			Sequence int64  `json:"sequence"`
		}
		if !found || receipt.digest != hash([]byte("ack:"+digest)) || receipt.revision <= eventRevision || receipt.revision <= previous[target] || strict([]byte(receipt.result), &result) != nil || result.Target != target || result.Sequence != sequence {
			rows.Close()
			return contracts.Fail("invalid_request")
		}
		previous[target] = receipt.revision
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	rows, e = tx.QueryContext(ctx, s.query("SELECT value FROM setting_revision WHERE workspace_id=?"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var value string
		if e = rows.Scan(&value); e != nil {
			rows.Close()
			return e
		}
		if !nonsecret(json.RawMessage(value)) {
			rows.Close()
			return contracts.Fail("invalid_request")
		}
	}
	e = rows.Err()
	rows.Close()
	return e
}
func ReadSnapshot(data []byte) (Snapshot, error) {
	var envelope snapshotEnvelope
	if e := strict(data, &envelope); e != nil {
		return Snapshot{}, e
	}
	return decodedSnapshot(envelope)
}
func (s *Store) Status(ctx context.Context) (map[string]any, error) {
	rev, e := s.Revision(ctx)
	return map[string]any{"adapter_id": s.backend, "catalog_schema": SchemaVersion, "revision": rev, "workspace_id": s.workspace, "persistence": "durable"}, e
}
