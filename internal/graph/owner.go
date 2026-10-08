// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

// Native reads are authorized against one workspace-owned database, never a
// shared database with an assumed caller-supplied workspace predicate.
func (a *adapter) BindWorkspace(ctx context.Context, workspace string) error {
	if !contracts.ValidID(workspace) {
		return contracts.Fail("invalid_request")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, e := a.engine.Begin(ctx, false)
	if e != nil {
		return e
	}
	defer s.Rollback(ctx)
	rows, e := a.run(ctx, s, false, "SELECT workspace FROM GraphOwner WHERE id=:id", "MATCH (o:GraphOwner) WHERE o.id=$id RETURN o.workspace AS workspace", map[string]any{"id": "owner"})
	if e != nil {
		return e
	}
	if len(rows) > 0 {
		if len(rows) != 1 || rows[0]["workspace"] != workspace {
			return contracts.Fail("workspace_mismatch")
		}
		return nil
	}
	all, e := a.run(ctx, s, false, "SELECT workspace FROM Entity", "MATCH (n:Entity) RETURN n.workspace AS workspace", nil)
	if e != nil {
		return e
	}
	for _, row := range all {
		if row["workspace"] != workspace {
			return contracts.Fail("workspace_mismatch")
		}
	}
	for _, table := range []string{"GraphCheckpoint", "GraphReceipt"} {
		rows, e := s.Query(ctx, false, a.statement("SELECT id FROM "+table, "MATCH (n:"+table+") RETURN n.id AS id"), nil)
		if e != nil {
			return e
		}
		for _, row := range rows {
			id, ok := row["id"].(string)
			if !ok || !strings.HasPrefix(id, workspace+"|") {
				return contracts.Fail("workspace_mismatch")
			}
		}
	}
	_, e = a.run(ctx, s, true, "INSERT INTO GraphOwner SET id=:id,workspace=:workspace", "CREATE (:GraphOwner {id:$id,workspace:$workspace})", map[string]any{"id": "owner", "workspace": workspace})
	if e != nil {
		return e
	}
	return s.Commit(ctx)
}
