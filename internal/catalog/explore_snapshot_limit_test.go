// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func TestGraphSnapshotIndependentOfWorkBudget(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	nodes := make([]map[string]any, 1400)
	for i := range nodes {
		nodes[i] = map[string]any{"id": contracts.ID(), "kind": "media", "reference": map[string]any{"source_id": strings.Repeat("x", 6000)}}
	}
	raw, _ := json.Marshal(map[string]any{"kind": "refresh", "catalog_revision": snap.Revision, "nodes": nodes})
	if len(raw) <= contracts.MaxWorkPayload {
		t.Fatal("fixture does not cross work budget")
	}
	op := contracts.ID()
	r, e := s.EnqueueGraphRefresh(ctx, op, snap.Revision, raw)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := s.EnqueueGraphRefresh(ctx, op, snap.Revision, raw)
	if e != nil || replay.Revision != r.Revision {
		t.Fatal(replay, e)
	}
	claim, e := s.ClaimOutbox(ctx, s.GraphTarget(), contracts.ID(), 5*time.Second)
	if e != nil || len(claim.Document) <= contracts.MaxWorkPayload {
		t.Fatal(len(claim.Document), e)
	}
	if e = s.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal(e)
	}
	exported, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restored := localStore(t, s.workspace)
	if e = restored.Restore(ctx, exported); e != nil {
		t.Fatal(e)
	}
	status, e := restored.GraphStatus(ctx)
	if e != nil || status["pending_events"] != 1 {
		t.Fatal(status, e)
	}
}
