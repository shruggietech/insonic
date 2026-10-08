// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
)

func (a *App) rebuildGraph(ctx context.Context, requestID string) error {
	g, e := a.openGraph(ctx)
	if e != nil {
		return e
	}
	cp, e := g.ReadCheckpoint(ctx, a.Workspace.Config.WorkspaceID, a.Catalog.GraphTarget())
	if e != nil {
		return e
	}
	if cp.Generation == math.MaxInt64 {
		return contracts.Fail("conflict")
	}
	if _, e = a.Catalog.EnqueueGraphRebuild(ctx, requestID); e != nil {
		return e
	}
	status, e := a.Catalog.GraphStatus(ctx)
	if e != nil {
		return e
	}
	if status["pending_events"] == 0 {
		_, _, e = a.publishGraph(ctx)
		return e
	}
	if e = a.Catalog.AdvanceOutboxGeneration(ctx, a.Session, cp.Generation+1, leaseTTL); e != nil {
		return e
	}
	claim, e := a.Catalog.ClaimOutbox(ctx, a.Catalog.GraphTarget(), a.Session, leaseTTL)
	if e != nil {
		return e
	}
	if e = a.applyOutbox(ctx, g, claim, true); e != nil {
		return e
	}
	_, _, e = a.publishGraph(ctx)
	return e
}
