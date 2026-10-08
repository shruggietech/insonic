// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/explore"
	"github.com/shruggietech/insonic/internal/graph"
	"time"
)

func (a *App) lockGraph(ctx context.Context) error {
	for !a.graphMu.TryLock() {
		select {
		case <-ctx.Done():
			return contracts.Fail("cancelled")
		case <-time.After(5 * time.Millisecond):
		}
	}
	if ctx.Err() != nil {
		a.graphMu.Unlock()
		return contracts.Fail("cancelled")
	}
	return nil
}
func (a *App) runQuery(ctx context.Context, q contracts.QueryInput, operation string) (any, error) {
	if e := q.Validate(); e != nil {
		return nil, e
	}
	if e := a.lockGraph(ctx); e != nil {
		return nil, e
	}
	defer a.graphMu.Unlock()
	if q.Definition.Mode == "native" {
		if e := graph.ValidateNative(q.Definition.Dialect, q.Definition.Text, q.Parameters); e != nil {
			return nil, e
		}
	}
	g, c, e := a.publishGraph(ctx)
	if e != nil {
		if q.Definition.Mode == "normalized" && operation == "query.run" {
			return a.catalogQueryContext(ctx, q, e)
		}
		return nil, e
	}
	if q.Definition.Mode == "native" {
		var rows graph.Rows
		if operation == "query.explain" {
			rows, e = g.Explain(ctx, q.Definition.Dialect, q.Definition.Text, q.Parameters)
		} else {
			rows, e = g.Query(ctx, q.Definition.Dialect, q.Definition.Text, q.Parameters)
		}
		if e != nil {
			return nil, e
		}
		status, e := a.Catalog.GraphStatus(ctx)
		if e != nil {
			return nil, e
		}
		if status["pending_events"] > 0 {
			return nil, contracts.Fail("conflict")
		}
		return map[string]any{"items": rows, "dialect": q.Definition.Dialect, "catalog_revision": c.Revision, "freshness": status}, nil
	}
	if operation == "query.explain" {
		return map[string]any{"operation": q.Definition.Operation, "adapter": g.Capabilities(), "plan": "typed current-reference selection, catalog hydration, filtering, deterministic ordering and bounded pagination"}, nil
	}
	refs, e := g.ReadRefs(ctx, a.Workspace.Config.WorkspaceID)
	if e != nil {
		return nil, e
	}
	status, e := a.Catalog.GraphStatus(ctx)
	if e != nil {
		return nil, e
	}
	if status["pending_events"] > 0 {
		return nil, contracts.Fail("conflict")
	}
	out, e := explore.Query(c, refs, q)
	if ctx.Err() != nil {
		return nil, contracts.Fail("cancelled")
	}
	return out, e
}
