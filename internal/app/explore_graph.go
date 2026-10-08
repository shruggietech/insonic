// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/explore"
	"github.com/shruggietech/insonic/internal/graph"
	"path/filepath"
	"time"
)

func (a *App) openGraph(ctx context.Context) (graph.Adapter, error) {
	if a.Graph != nil {
		return a.Graph, nil
	}
	p := a.Workspace.Config.Profiles.Graph
	var g graph.Adapter
	var e error
	switch p.Adapter {
	case "ladybugdb":
		path, _ := p.Configuration["path"].(string)
		if path == "" {
			return nil, contracts.Fail("invalid_request")
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(a.Workspace.Root, path)
		}
		g, e = graph.OpenLadybug(path)
	case "arcadedb":
		raw, _ := json.Marshal(p.Configuration)
		var config graph.ArcadeConfig
		if strictPayload(raw, &config) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		g, e = graph.OpenArcade(config, a.secrets)
	default:
		return nil, contracts.Fail("unsupported_capability")
	}
	if e != nil {
		return nil, e
	}
	if p.ExpectedBackendVersion != "" && g.Capabilities()["backend_version"] != p.ExpectedBackendVersion {
		g.Close()
		return nil, contracts.Fail("unsupported_capability")
	}
	if e = g.EnsureSchema(ctx); e == nil {
		e = g.BindWorkspace(ctx, a.Workspace.Config.WorkspaceID)
	}
	if e != nil {
		g.Close()
		return nil, e
	}
	a.Graph = g
	return g, nil
}
func (a *App) drainGraph(ctx context.Context, g graph.Adapter) error {
	for {
		status, e := a.Catalog.GraphStatus(ctx)
		if e != nil {
			return e
		}
		if status["pending_events"] == 0 {
			return nil
		}
		claim, e := a.Catalog.ClaimOutbox(ctx, a.Catalog.GraphTarget(), a.Session, leaseTTL)
		if e != nil {
			return e
		}
		if e = a.applyOutbox(ctx, g, claim, false); e != nil {
			return e
		}
	}
}
func (a *App) applyOutbox(ctx context.Context, g graph.Adapter, claim catalog.OutboxClaim, rebuild bool) error {
	if !rebuild {
		if e := g.FenceGeneration(ctx, claim.WorkspaceID, claim.Target, claim.Generation); e != nil {
			return e
		}
	}
	// Database work can outlive one catalog lease. Renewal runs independently,
	// cancels stale publication and joins before acknowledgement.
	call, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	stop := make(chan struct{})
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				done <- nil
				return
			case <-call.Done():
				done <- call.Err()
				return
			case <-tick.C:
				if _, e := a.Catalog.RenewOutbox(call, claim, leaseTTL); e != nil {
					cancel()
					done <- e
					return
				}
			}
		}
	}()
	var e error
	if rebuild {
		e = g.Rebuild(call, claim)
	} else {
		e = g.ApplyRevision(call, claim)
	}
	close(stop)
	renewal := <-done
	cancel()
	if e != nil {
		return e
	}
	if renewal != nil {
		return renewal
	}
	return a.Catalog.AcknowledgeOutbox(ctx, claim)
}
func (a *App) publishGraph(ctx context.Context) (graph.Adapter, explore.Corpus, error) {
	g, e := a.openGraph(ctx)
	if e != nil {
		return nil, explore.Corpus{}, e
	}
	if e = a.drainGraph(ctx, g); e != nil {
		return g, explore.Corpus{}, e
	}
	for attempt := 0; attempt < 3; attempt++ {
		snap, e := a.Catalog.Export(ctx)
		if e != nil {
			return g, explore.Corpus{}, e
		}
		c, e := explore.Build(snap)
		if e != nil {
			return g, c, e
		}
		existing, e := g.ReadRefs(ctx, snap.WorkspaceID)
		if e != nil {
			return g, c, e
		}
		old, _ := json.Marshal(existing)
		current, _ := json.Marshal(c.Refs)
		if string(old) == string(current) {
			return g, c, nil
		}
		event, _ := json.Marshal(graph.Change{Kind: "refresh", CatalogRevision: snap.Revision, Nodes: c.Refs.Nodes, Edges: c.Refs.Edges})
		_, e = a.Catalog.EnqueueGraphRefresh(ctx, contracts.ID(), snap.Revision, event)
		if typed, ok := e.(*contracts.Error); ok && typed.Code == "conflict" {
			continue
		}
		if e != nil {
			return g, c, e
		}
		if e = a.drainGraph(ctx, g); e != nil {
			return g, c, e
		}
		// Re-read facts after publication, because another writer can invalidate it.
		status, e := a.Catalog.GraphStatus(ctx)
		if e != nil {
			return g, c, e
		}
		if status["pending_events"] != 0 {
			continue
		}
		return g, c, nil
	}
	return g, explore.Corpus{}, contracts.Fail("conflict")
}
func (a *App) graphState(ctx context.Context) (any, error) {
	status, e := a.Catalog.GraphStatus(ctx)
	if e != nil {
		return nil, e
	}
	out := map[string]any{"adapter_id": a.Workspace.Config.Profiles.Graph.Adapter, "outbox": status, "state": "configured"}
	if a.Graph != nil {
		cp, e := a.Graph.ReadCheckpoint(ctx, a.Workspace.Config.WorkspaceID, a.Catalog.GraphTarget())
		if e != nil {
			return nil, e
		}
		out["checkpoint"] = cp
		out["state"] = "available"
		out["fresh"] = status["pending_events"] == 0
	}
	return out, nil
}
func (a *App) maintainGraph(ctx context.Context) {
	defer a.wg.Done()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.graphMu.Lock()
			if a.Graph != nil {
				status, e := a.Catalog.GraphStatus(ctx)
				if e == nil && status["pending_events"] > 0 {
					_, _, _ = a.publishGraph(ctx)
				}
			}
			a.graphMu.Unlock()
		}
	}
}

var _ catalog.OutboxClaim
