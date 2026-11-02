// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/shruggietech/insonic/internal/contracts"
)

type NeighborhoodRequest struct {
	Workspace, Target, Node, Direction, After string
	Kinds                                     []string
	Checkpoint                                Checkpoint
	Limit, MaxBytes                           int
}
type NeighborhoodPage struct {
	NodePresent       bool
	Rows              int
	Edges             []Edge
	Next, LimitReason string
	Bytes             int
}

// Neighborhood is a keyset page pinned to one published graph generation.
// Traversal callers share budgets across pages and resolve admitted cue IDs
// against the catalog. No whole-projection read or publication occurs here.
func (a *adapter) Neighborhood(ctx context.Context, r NeighborhoodRequest) (out NeighborhoodPage, err error) {
	out.Edges = []Edge{}
	if !validTarget(r.Workspace, r.Target) || !contracts.ValidTextLength(r.Node, 1, 1024) || !contracts.ValidTextLength(r.After, 0, 1024) || !slices.Contains([]string{"in", "out", "both"}, r.Direction) || len(r.Kinds) > 64 || r.Limit < 1 || r.Limit > 1000 || r.MaxBytes < 1 || r.MaxBytes > 8<<20 {
		return out, contracts.Fail("invalid_request")
	}
	for _, kind := range r.Kinds {
		if !contracts.ValidTextLength(kind, 1, 128) {
			return out, contracts.Fail("invalid_request")
		}
	}
	// Reserve the maximum JSON-escaped size of each bounded edge before the
	// driver fetch. Actual usage is returned, so later rounds reuse spare bytes.
	limit := min(r.Limit, 20, r.MaxBytes/(24<<10))
	if limit < 1 {
		out.LimitReason = "source_bytes"
		return out, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if ctx.Err() != nil {
		return out, contracts.Fail("cancelled")
	}
	s, err := a.engine.Begin(ctx, true)
	if err != nil {
		return out, err
	}
	defer s.Rollback(ctx)
	cp, exists, err := a.checkpoint(ctx, s, r.Workspace+"|"+r.Target)
	if err != nil {
		return out, err
	}
	if !exists || cp != r.Checkpoint {
		return out, contracts.Condition("conflict", "graph_checkpoint_changed")
	}
	params := map[string]any{"workspace": r.Workspace, "node": r.Node, "edge_cursor": r.After, "limit": limit}
	present, e := a.run(ctx, s, false, "SELECT 1 AS present FROM Entity WHERE id=:physical AND workspace=:workspace LIMIT 1", "MATCH (n:Entity) WHERE n.id=$physical AND n.workspace=$workspace RETURN true AS present LIMIT 1", map[string]any{"workspace": r.Workspace, "physical": r.Workspace + "|" + r.Node})
	if e != nil {
		return out, e
	}
	out.NodePresent = len(present) == 1
	out.Rows = 1
	out.Bytes = 2
	if !out.NodePresent {
		return out, nil
	}
	sqlWhere := "@out.workspace=:workspace AND @in.workspace=:workspace AND id>:edge_cursor"
	cypherWhere := "f.workspace=$workspace AND t.workspace=$workspace AND e.id>$edge_cursor"
	switch r.Direction {
	case "in":
		sqlWhere += " AND @in.entity_id=:node"
		cypherWhere += " AND t.entity_id=$node"
	case "out":
		sqlWhere += " AND @out.entity_id=:node"
		cypherWhere += " AND f.entity_id=$node"
	case "both":
		sqlWhere += " AND (@out.entity_id=:node OR @in.entity_id=:node)"
		cypherWhere += " AND (f.entity_id=$node OR t.entity_id=$node)"
	}
	if len(r.Kinds) > 0 {
		sqlKinds, cypherKinds := []string{}, []string{}
		for i, kind := range r.Kinds {
			key := "kind" + strconv.Itoa(i)
			params[key] = kind
			sqlKinds = append(sqlKinds, "kind=:"+key)
			cypherKinds = append(cypherKinds, "e.kind=$"+key)
		}
		sqlWhere += " AND (" + strings.Join(sqlKinds, " OR ") + ")"
		cypherWhere += " AND (" + strings.Join(cypherKinds, " OR ") + ")"
	}
	rows, err := a.run(ctx, s, false, "SELECT id,kind,@out.entity_id AS source,@in.entity_id AS destination FROM EvidenceLink WHERE "+sqlWhere+" ORDER BY id LIMIT :limit", "MATCH (f:Entity)-[e:EvidenceLink]->(t:Entity) WHERE "+cypherWhere+" RETURN e.id AS id,e.kind AS kind,f.entity_id AS source,t.entity_id AS destination ORDER BY id LIMIT $limit", params)
	if err != nil {
		return out, err
	}
	if len(rows) > limit {
		return out, contracts.Fail("output_limit")
	}
	for _, row := range rows {
		id, ok := row["id"].(string)
		kind, kok := row["kind"].(string)
		from, fok := row["source"].(string)
		to, tok := row["destination"].(string)
		if !ok || !kok || !fok || !tok || len(id) > 1024 || len(from) > 1024 || len(to) > 1024 || len(kind) > 128 {
			return out, contracts.Fail("operation_failed")
		}
		out.Edges = append(out.Edges, Edge{ID: id, From: from, To: to, Kind: kind})
	}
	raw, _ := json.Marshal(out.Edges)
	out.Bytes += len(raw)
	out.Rows += len(rows)
	if out.Bytes > r.MaxBytes {
		return out, contracts.Fail("output_limit")
	}
	if len(rows) == limit {
		out.Next = out.Edges[len(out.Edges)-1].ID
	}
	return out, nil
}
