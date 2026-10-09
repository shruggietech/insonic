// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/explore"
	"github.com/shruggietech/insonic/internal/graph"
)

func restoredGraphSuite(t *testing.T, g graph.Adapter) {
	t.Helper()
	defer g.Close()
	ctx := context.Background()
	w, source, _ := fixture(t)
	sp, e := source.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Rebuilt speaker"}})
	if e != nil {
		t.Fatal(e)
	}
	// Prove restoration resets an acknowledged source checkpoint, rather than
	// merely working when the source graph happened to have no acknowledgement.
	owner := contracts.ID()
	claim, e := source.ClaimOutbox(ctx, source.GraphTarget(), owner, 500*time.Millisecond)
	if e != nil {
		t.Fatal(e)
	}
	if e = source.AcknowledgeOutbox(ctx, claim); e != nil {
		t.Fatal(e)
	}
	time.Sleep(510 * time.Millisecond)
	dir := filepath.Join(t.TempDir(), "graph-backup")
	if _, e = Create(ctx, w, source, nil, CreateOptions{ID: contracts.ID(), Directory: dir}); e != nil {
		t.Fatal(e)
	}
	target := destination(t)
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = g.EnsureSchema(ctx); e != nil {
		t.Fatal(e)
	}
	if e = g.BindWorkspace(ctx, target.Config.WorkspaceID); e != nil {
		t.Fatal(e)
	}
	drain := func() {
		for {
			status, e := db.GraphStatus(ctx)
			if e != nil {
				t.Fatal(e)
			}
			if status["pending_events"] == 0 {
				return
			}
			claim, e := db.ClaimOutbox(ctx, db.GraphTarget(), owner, time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			if e = g.FenceGeneration(ctx, claim.WorkspaceID, claim.Target, claim.Generation); e != nil {
				t.Fatal(e)
			}
			if e = g.ApplyRevision(ctx, claim); e != nil {
				t.Fatal("accepted evidence replay", e)
			}
			if e = db.AcknowledgeOutbox(ctx, claim); e != nil {
				t.Fatal(e)
			}
		}
	}
	drain()
	snap, e := db.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	corpus, e := explore.Build(snap)
	if e != nil {
		t.Fatal(e)
	}
	event, _ := json.Marshal(graph.Change{Kind: "refresh", CatalogRevision: snap.Revision, Nodes: corpus.Refs.Nodes, Edges: corpus.Refs.Edges})
	if _, e = db.EnqueueGraphRefresh(ctx, contracts.ID(), snap.Revision, event); e != nil {
		t.Fatal(e)
	}
	drain()
	refs, e := g.ReadRefs(ctx, target.Config.WorkspaceID)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, node := range refs.Nodes {
		if node.ID == "speaker:"+sp.Speaker.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("accepted speaker evidence not rebuilt", refs)
	}
	snapshot, e := db.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = catalog.ValidateSnapshot(ctx, snapshot); e != nil {
		t.Fatal("graph history invalid after rebuild", e)
	}
}
