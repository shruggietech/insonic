package graph

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func testClaim(workspace, target string, seq, generation int64, change Change) catalog.OutboxClaim {
	raw, _ := json.Marshal(change)
	h := sha256.Sum256(raw)
	return catalog.OutboxClaim{WorkspaceID: workspace, Target: target, OwnerID: contracts.ID(), Generation: generation, Sequence: seq, Predecessor: seq - 1, Revision: seq + 100, OperationID: contracts.ID(), Document: raw, Digest: hex.EncodeToString(h[:])}
}
func qualifyAdapter(t *testing.T, a Adapter) {
	t.Helper()
	defer a.Close()
	ctx := context.Background()
	w, target := contracts.ID(), contracts.ID()
	if e := a.EnsureSchema(ctx); e != nil {
		t.Fatal(e)
	}
	if e := a.BindWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	if e := a.BindWorkspace(ctx, contracts.ID()); e == nil {
		t.Fatal("shared native database admitted")
	}
	if e := a.FenceGeneration(ctx, w, target, 1); e != nil {
		t.Fatal(e)
	}
	c := testClaim(w, target, 1, 1, Change{Kind: "refresh", CatalogRevision: 40, Nodes: []Node{{ID: "media:one", Kind: "media", Reference: json.RawMessage(`{"media_id":"11111111-1111-4111-8111-111111111111"}`)}, {ID: "cue:one", Kind: "cue", Reference: json.RawMessage(`{"cue_id":"source"}`)}}, Edges: []Edge{{ID: "link", From: "media:one", To: "cue:one", Kind: "has-cue"}}})
	if e := a.ApplyRevision(ctx, c); e != nil {
		t.Fatal(e)
	}
	if e := a.ApplyRevision(ctx, c); e != nil {
		t.Fatal("exact replay", e)
	}
	bad := c
	bad.Digest = "wrong"
	if e := a.ApplyRevision(ctx, bad); e == nil {
		t.Fatal("mismatched replay accepted")
	}
	cp, e := a.ReadCheckpoint(ctx, w, target)
	if e != nil || cp.Sequence != 1 || cp.CatalogRevision != 40 {
		t.Fatal(cp, e)
	}
	refs, e := a.ReadRefs(ctx, w)
	if e != nil || len(refs.Nodes) != 2 || len(refs.Edges) != 1 {
		t.Fatal(refs, e)
	}
	if e = a.FenceGeneration(ctx, w, target, 2); e != nil {
		t.Fatal(e)
	}
	stale := testClaim(w, target, 2, 1, Change{Kind: "invalidate", CatalogRevision: 41})
	if e = a.ApplyRevision(ctx, stale); e == nil {
		t.Fatal("stale writer accepted")
	}
	cp, e = a.ReadCheckpoint(ctx, w, target)
	if e != nil || cp.Sequence != 1 {
		t.Fatal("failed write advanced", cp, e)
	}
	stale.Generation = 2
	if e = a.ApplyRevision(ctx, stale); e != nil {
		t.Fatal(e)
	}
	refs, e = a.ReadRefs(ctx, w)
	if e != nil || len(refs.Nodes) != 0 {
		t.Fatal("replacement retained refs", refs, e)
	}
	gap := testClaim(w, target, 4, 2, Change{Kind: "invalidate", CatalogRevision: 42})
	if e = a.ApplyRevision(ctx, gap); e == nil {
		t.Fatal("gap accepted")
	}
	native := a.(*adapter)
	original := native.engine
	failure := &failureEngine{engine: original}
	native.engine = failure
	uncertain := testClaim(w, target, 3, 2, Change{Kind: "refresh", CatalogRevision: 43, Nodes: cNodes(c)})
	failure.uncertain = true
	if e = a.ApplyRevision(ctx, uncertain); e != nil {
		t.Fatal("uncertain committed outcome", e)
	}
	cp, e = a.ReadCheckpoint(ctx, w, target)
	if e != nil || cp.Sequence != 3 {
		t.Fatal(cp, e)
	}
	rebuild := testClaim(w, target, 1, 3, Change{Kind: "refresh", CatalogRevision: 40, Nodes: cNodes(c)})
	failure.failFacts = true
	if e = a.Rebuild(ctx, rebuild); e == nil {
		t.Fatal("failed fact write accepted")
	}
	cp, e = a.ReadCheckpoint(ctx, w, target)
	if e != nil || cp.Sequence != 3 || cp.Generation != 2 {
		t.Fatal("failed rebuild changed checkpoint", cp, e)
	}
	failure.failFacts = false
	failure.uncertain = true
	if e = a.Rebuild(ctx, rebuild); e != nil {
		t.Fatal("rebuild uncertain outcome", e)
	}
	if e = a.Rebuild(ctx, rebuild); e != nil {
		t.Fatal("rebuild replay", e)
	}
	cp, e = a.ReadCheckpoint(ctx, w, target)
	if e != nil || cp.Sequence != 1 || cp.Generation != 3 {
		t.Fatal("reset did not install restored lineage", cp, e)
	}
	native.engine = original

}

func cNodes(c catalog.OutboxClaim) []Node {
	var change Change
	_ = json.Unmarshal(c.Document, &change)
	return change.Nodes
}
