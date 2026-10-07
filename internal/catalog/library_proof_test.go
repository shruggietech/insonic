// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func latestCurrentProofSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	cancelledOld, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CancelWork(ctx, contracts.ID(), cancelledOld.ID); e != nil {
		t.Fatal(e)
	}
	succeededOld, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if e != nil {
		t.Fatal(e)
	}
	succeeded, e := s.ClaimWork(ctx, succeededOld.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CheckpointWork(ctx, succeeded, "complete", "succeeded", json.RawMessage(`{"accepted":true}`), time.Minute); e != nil {
		t.Fatal(e)
	}
	interruptedOld, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[]}`))
	if e != nil {
		t.Fatal(e)
	}
	interrupted, e := s.ClaimWork(ctx, interruptedOld.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CheckpointWork(ctx, interrupted, "partial-import", "running", json.RawMessage(`{"accepted_count":1}`), time.Minute); e != nil {
		t.Fatal(e)
	}
	if e = s.InterruptOwner(ctx, interrupted.Owner); e != nil {
		t.Fatal(e)
	}
	claim, entry := entryFixture(t, s)
	oldReport, currentReport := availableArtifact(t, s), availableArtifact(t, s)
	entry.Metadata = json.RawMessage(`{"capture":"old-derived-content"}`)
	entry.ReportPublicationIDs, _ = json.Marshal([]string{oldReport.ID})
	oldEntry, e := s.CommitLibrary(ctx, claim, entry)
	if e != nil {
		t.Fatal(e)
	}
	currentEntry := oldEntry
	currentEntry.Metadata = json.RawMessage(`{"capture":"current-content"}`)
	currentEntry.ReportPublicationIDs, _ = json.Marshal([]string{currentReport.ID})
	if _, e = s.UpdateLibrary(ctx, contracts.ID(), oldEntry.Revision, currentEntry); e != nil {
		t.Fatal(e)
	}
	modelClaim, _ := entryFixture(t, s)
	p := availableArtifact(t, s)
	manifest, _ := json.Marshal(map[string]any{"kind": "base-model-manifest", "schema_version": contracts.Version, "name": "fixture", "model_version": "1", "upstream_revision": "immutable-fixture", "capabilities": []string{"transcription"}, "license": "unknown", "files": []any{map[string]any{"role": "weights", "url": "https://example.org/model", "sha256": p.Digest, "size": p.Size}}})
	model := BaseModelInstall{ID: contracts.ID(), Name: "fixture", Version: "1", Digest: hash(manifest), Manifest: manifest, PublicationIDs: json.RawMessage(`[]`), State: "registered"}
	oldModel, e := s.CommitBaseModel(ctx, modelClaim, model)
	if e != nil {
		t.Fatal(e)
	}
	modelClaim, _ = entryFixture(t, s)
	model = oldModel
	model.State = "available"
	model.PublicationIDs, _ = json.Marshal([]string{p.ID})
	if _, e = s.CommitBaseModel(ctx, modelClaim, model); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	valid := localStore(t, s.workspace)
	if e = valid.Restore(ctx, snap); e != nil {
		t.Fatal("current snapshot rejected", e)
	}
	recovered, e := valid.ClaimWork(ctx, interrupted.ID, contracts.ID(), time.Minute)
	if e != nil || recovered.Generation != interrupted.Generation+1 {
		t.Fatal("genuine interrupted import could not recover", e)
	}
	for _, old := range []Work{cancelledOld, succeededOld, interruptedOld} {
		t.Run("rollback-work-"+old.ID, func(t *testing.T) {
			forged := snap
			forged.Records.Works = append([]Work(nil), snap.Records.Works...)
			for i, w := range forged.Records.Works {
				if w.ID == old.ID {
					forged.Records.Works[i] = old
				}
			}
			forged.Digest, _ = forged.digest()
			if e := localStore(t, s.workspace).Restore(ctx, forged); e == nil {
				t.Fatal("historical work journal replaced newer accepted state")
			}
		})
	}
	t.Run("rollback-library", func(t *testing.T) {
		forged := snap
		forged.Records.Library = append([]LibraryEntry(nil), snap.Records.Library...)
		for i, v := range forged.Records.Library {
			if v.ID == oldEntry.ID {
				forged.Records.Library[i] = oldEntry
			}
		}
		forged.Digest, _ = forged.digest()
		if e := localStore(t, s.workspace).Restore(ctx, forged); e == nil {
			t.Fatal("superseded metadata resurrected using historical receipt")
		}
	})
	t.Run("rollback-model", func(t *testing.T) {
		forged := snap
		forged.Records.BaseModels = append([]BaseModelInstall(nil), snap.Records.BaseModels...)
		for i, v := range forged.Records.BaseModels {
			if v.ID == oldModel.ID {
				forged.Records.BaseModels[i] = oldModel
			}
		}
		forged.Digest, _ = forged.digest()
		if e := localStore(t, s.workspace).Restore(ctx, forged); e == nil {
			t.Fatal("historical model registration replaced available installation")
		}
	})
	t.Run("omitted-current-work", func(t *testing.T) {
		forged := snap
		forged.Records.Works = nil
		forged.Digest, _ = forged.digest()
		if e := localStore(t, s.workspace).Restore(ctx, forged); e == nil {
			t.Fatal("current work disappeared despite retained journals")
		}
	})
	t.Run("omitted-current-library", func(t *testing.T) {
		forged := snap
		forged.Records.Library = nil
		forged.Digest, _ = forged.digest()
		if e := localStore(t, s.workspace).Restore(ctx, forged); e == nil {
			t.Fatal("current library disappeared despite retained receipt")
		}
	})
	t.Run("omitted-current-model", func(t *testing.T) {
		forged := snap
		forged.Records.BaseModels = nil
		forged.Digest, _ = forged.digest()
		if e := localStore(t, s.workspace).Restore(ctx, forged); e == nil {
			t.Fatal("current model disappeared despite retained receipt")
		}
	})
}

func TestSQLiteLatestCurrentProof(t *testing.T) {
	latestCurrentProofSuite(t, localStore(t, contracts.ID()))
}

func cleanupSnapshotProofSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, entry := entryFixture(t, s)
	oldReport, currentReport := availableArtifact(t, s), availableArtifact(t, s)
	entry.ReportPublicationIDs, _ = json.Marshal([]string{oldReport.ID})
	entry, e := s.CommitLibrary(ctx, claim, entry)
	if e != nil {
		t.Fatal(e)
	}
	entry.ReportPublicationIDs, _ = json.Marshal([]string{currentReport.ID})
	if _, e = s.UpdateLibrary(ctx, contracts.ID(), entry.Revision, entry); e != nil {
		t.Fatal(e)
	}
	pending, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	forged := pending
	forged.Records.Cleanups = nil
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Error("pending cleanup obligation omitted despite schedule receipt")
	}
	retirement, e := s.ClaimRetirement(ctx, oldReport.ID, contracts.ID(), 0, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.FinishRetirement(ctx, retirement); e != nil {
		t.Fatal(e)
	}
	retired, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	forged = retired
	forged.Records.Cleanups = append([]Cleanup(nil), retired.Records.Cleanups...)
	forged.Records.Cleanups[0].State = "done"
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Error("cleanup completion forged without completion receipt")
	}
	if e = s.FinishCleanup(ctx, oldReport.ID); e != nil {
		t.Fatal(e)
	}
	completed, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = localStore(t, s.workspace).Restore(ctx, completed); e != nil {
		t.Fatal("genuine completed cleanup rejected", e)
	}
	forged = completed
	forged.Records.Cleanups = append([]Cleanup(nil), completed.Records.Cleanups...)
	forged.Records.Cleanups[0].State = "pending"
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Error("completed cleanup reverted to unfinishable pending state")
	}
	forged = completed
	forged.Records.Cleanups = nil
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Error("completed cleanup omitted despite retained receipts")
	}
}

func TestCleanupSnapshotProofRequiresScheduledAndCompletedState(t *testing.T) {
	cleanupSnapshotProofSuite(t, localStore(t, contracts.ID()))
}
