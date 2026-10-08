// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
)

func (a *adapter) Rebuild(ctx context.Context, c catalog.OutboxClaim) error {
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
	defer s.Rollback(ctx)
	key := c.WorkspaceID + "|" + c.Target
	cp, exists, e := a.checkpoint(ctx, s, key)
	if e != nil {
		return e
	}
	got, re := a.readReceipt(ctx, s, key+"|"+strconv.FormatInt(c.Sequence, 10))
	if re != nil {
		return re
	}
	if got == receipt(c) && cp.Generation == c.Generation {
		return nil
	}
	if exists && cp.Generation >= c.Generation {
		return contracts.Fail("conflict")
	}
	cp = Checkpoint{Generation: c.Generation, Sequence: c.Predecessor}
	if e = a.setCheckpoint(ctx, s, key, cp, exists); e != nil {
		return e
	}
	_, e = a.run(ctx, s, true, "DELETE FROM GraphReceipt WHERE id LIKE :prefix", "MATCH (r:GraphReceipt) WHERE starts_with(r.id,$prefix) DELETE r", map[string]any{"prefix": key + "|" + a.statement("%", "")})
	if e != nil {
		return e
	}
	// Replacement facts, the new event receipt and the reset checkpoint share
	// this transaction. Failed rebuilds leave the previous projection intact.
	if e = a.apply(ctx, s, c, change); e != nil {
		return e
	}
	e = s.Commit(ctx)
	if e == nil {
		return nil
	}
	s.Rollback(context.Background())
	probe, pe := a.engine.Begin(ctx, true)
	if pe != nil {
		return e
	}
	defer probe.Rollback(ctx)
	got, pe = a.readReceipt(ctx, probe, key+"|"+strconv.FormatInt(c.Sequence, 10))
	if pe == nil && got == receipt(c) {
		return nil
	}
	return e
}
