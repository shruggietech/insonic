// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func processingPlanFixture(t *testing.T, s *Store) (Work, ProcessingPlan, LibraryEntry) {
	t.Helper()
	ctx := context.Background()
	importClaim, entry := entryFixture(t, s)
	original := availableArtifact(t, s)
	entry.Mode, entry.SourceLocator = "copy", ""
	entry.Digest, entry.Size = original.Digest, original.Size
	entry.OriginalPublicationID = &original.ID
	var err error
	entry, err = s.CommitLibrary(ctx, importClaim, entry)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"media_id": entry.ID, "source_revision": entry.Revision, "expected_revision": 0, "options": map[string]any{"core_us": 30000000, "context_us": 2000000, "concurrency": 1}})
	w, err := s.EnqueueWork(ctx, contracts.ID(), "recordings.process", payload)
	if err != nil {
		t.Fatal(err)
	}
	w, err = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	configuration := hash([]byte("configuration"))
	value := map[string]any{"kind": "processing-window-plan", "schema_version": contracts.Version, "plan_version": 1, "identity": map[string]any{"source_id": entry.ID, "source_sha256": entry.Digest, "source_revision": entry.Revision, "document_revision": 0, "configuration_sha256": configuration, "model_sha256": hash([]byte("models")), "adapter_id": "fixture", "adapter_version": "1", "device_policy": "auto"}, "options": map[string]any{"core_us": 30000000, "context_us": 2000000, "concurrency": 1}, "tracks": []any{map[string]any{"stream_index": 0, "source_sample_rate": 16000, "source_sample_count": 16000, "start_numerator": "0", "start_denominator": "1", "start_sample": 0, "end_sample": 16000}}, "windows": []any{map[string]any{"id": "10000000-0000-4000-8000-000000000063", "ordinal": 0, "track_ordinal": 0, "stream_index": 0, "source_sample_rate": 16000, "core_start_sample": 0, "core_end_sample": 16000, "context_start_sample": 0, "context_end_sample": 16000, "mapped_start_sample": 0, "mapped_end_sample": 16000, "mapping": map[string]any{"start_numerator": "0", "start_denominator": "1", "sample_count": 16000, "sample_rate": 16000, "stream_index": 0, "channel": nil, "policy": "canonical-source-samples;inward-window;continuous-resample;source-start-rational"}}}}
	raw, _ := json.Marshal(value)
	digest, _ := intent(json.RawMessage(raw))
	value["digest"] = digest
	raw, _ = json.Marshal(value)
	p := ProcessingPlan{ID: ProcessingPlanID(w.ID, digest), WorkID: w.ID, MediaID: entry.ID, SourceRevision: entry.Revision, SourceDigest: entry.Digest, ConfigurationDigest: configuration, PlanDigest: digest, Plan: raw, Attempt: w.Generation, State: "active"}
	p, err = s.CommitProcessingPlan(ctx, w, p)
	if err != nil {
		t.Fatal("plan", err)
	}
	return w, p, entry
}

func processingPublication(t *testing.T, s *Store, kind string, ids ...string) Publication {
	t.Helper()
	ctx := context.Background()
	base := availableArtifact(t, s)
	p := Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: base.ProfileID, ProfileRevision: base.ProfileRevision, Digest: hash([]byte("window-result")), Size: 13, Kind: kind, Key: "objects/" + contracts.ID(), Owner: contracts.ID()}
	if len(ids) == 1 {
		p.ID = ids[0]
	}
	var err error
	p, err = s.BeginPublication(ctx, p, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	p.Verification = "sha256-readback"
	p, err = s.AdmitPublication(ctx, p)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func processingCheckpointFixture(t *testing.T, s *Store, w Work, p ProcessingPlan) ProcessingWindowCheckpoint {
	t.Helper()
	publication := processingPublication(t, s, "processing-window")
	c := ProcessingWindowCheckpoint{ID: ProcessingCheckpointID(p.ID, "10000000-0000-4000-8000-000000000063", "combined"), PlanID: p.ID, WorkID: w.ID, WindowID: "10000000-0000-4000-8000-000000000063", Stage: "combined", Attempt: w.Generation, PublicationID: publication.ID, OutputDigest: publication.Digest, OutputSize: publication.Size, State: "ready"}
	got, err := s.CommitProcessingWindowCheckpoint(context.Background(), w, c)
	if err != nil {
		t.Fatal("checkpoint", err)
	}
	return got
}

func processingWindowsSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	w, p, _ := processingPlanFixture(t, s)
	c := processingCheckpointFixture(t, s, w, p)
	if _, err := s.ClaimRetirement(ctx, c.PublicationID, contracts.ID(), 0, time.Minute); err == nil {
		t.Fatal("ready checkpoint retired")
	}
	if _, err := s.CancelWork(ctx, contracts.ID(), w.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CommitProcessingWindowCheckpoint(ctx, w, c); err == nil {
		t.Fatal("cancelled owner accepted")
	}
	if _, err := s.RetryWork(ctx, contracts.ID(), w.ID); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	replayPlan := p
	replayPlan.Attempt = next.Generation
	if accepted, err := s.CommitProcessingPlan(ctx, next, replayPlan); err != nil || !reflect.DeepEqual(accepted, p) {
		t.Fatal("retry changed frozen plan producer", accepted, err)
	}
	checkpoints, err := s.ProcessingWindowCheckpoints(ctx, p.ID)
	if err != nil || len(checkpoints) != 1 || checkpoints[0] != c {
		t.Fatal("retry lost checkpoint", checkpoints, err)
	}
	if _, err = s.CommitProcessingWindowCheckpoint(ctx, w, c); err == nil {
		t.Fatal("superseded owner accepted")
	}
	// A later owner can read accepted old output but cannot relabel its producer.
	forged := c
	forged.Attempt = next.Generation
	if _, err = s.CommitProcessingWindowCheckpoint(ctx, next, forged); err == nil {
		t.Fatal("accepted checkpoint changed producer")
	}
	snap, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	validateSnapshotSchema(t, snap)
	if err = localStore(t, s.workspace).Restore(ctx, snap); err != nil {
		t.Fatal("checkpoint restore", err)
	}
	forgedSnap := snap
	forgedSnap.Records.ProcessingCheckpoints = append([]ProcessingWindowCheckpoint{}, snap.Records.ProcessingCheckpoints...)
	forgedSnap.Records.ProcessingCheckpoints[0].OutputDigest = hash([]byte("forged"))
	forgedSnap.Digest, _ = forgedSnap.digest()
	if err = localStore(t, s.workspace).Restore(ctx, forgedSnap); err == nil {
		t.Fatal("rehashed checkpoint forgery accepted")
	}
}

func TestSQLiteProcessingWindows(t *testing.T) {
	processingWindowsSuite(t, localStore(t, contracts.ID()))
}

func TestProcessingPlanRejectsAlteredWindowAuthority(t *testing.T) {
	_, accepted, _ := processingPlanFixture(t, localStore(t, contracts.ID()))
	for _, field := range []string{"source", "configuration", "context", "mapping", "content"} {
		t.Run(field, func(t *testing.T) {
			var value map[string]json.RawMessage
			json.Unmarshal(accepted.Plan, &value)
			switch field {
			case "source", "configuration":
				var identity map[string]json.RawMessage
				json.Unmarshal(value["identity"], &identity)
				key := "source_sha256"
				if field == "configuration" {
					key = "configuration_sha256"
				}
				identity[key], _ = json.Marshal(hash([]byte("changed")))
				value["identity"], _ = json.Marshal(identity)
			case "context", "mapping":
				var windows []map[string]json.RawMessage
				json.Unmarshal(value["windows"], &windows)
				if field == "context" {
					windows[0]["context_end_sample"] = json.RawMessage(`16001`)
				} else {
					var mapping map[string]json.RawMessage
					json.Unmarshal(windows[0]["mapping"], &mapping)
					mapping["start_numerator"] = json.RawMessage(`"1"`)
					windows[0]["mapping"], _ = json.Marshal(mapping)
				}
				value["windows"], _ = json.Marshal(windows)
			case "content":
				value["transcript_text"] = json.RawMessage(`"content cannot hide in metadata"`)
			}
			delete(value, "digest")
			digest, _ := intent(value)
			value["digest"], _ = json.Marshal(digest)
			candidate := accepted
			candidate.PlanDigest, candidate.ID = digest, ProcessingPlanID(candidate.WorkID, digest)
			candidate.Plan, _ = json.Marshal(value)
			if validProcessingPlan(candidate) {
				t.Fatal("rehashed malformed plan accepted")
			}
		})
	}
}

func TestProcessingCheckpointAdmissionAndRetirement(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	w, p, entry := processingPlanFixture(t, s)
	bad := ProcessingWindowCheckpoint{ID: ProcessingCheckpointID(p.ID, "10000000-0000-4000-8000-000000000063", "combined"), PlanID: p.ID, WorkID: w.ID, WindowID: "10000000-0000-4000-8000-000000000063", Stage: "combined", Attempt: w.Generation, State: "ready"}
	wrong := processingPublication(t, s, "other")
	bad.PublicationID = wrong.ID
	bad.OutputDigest = wrong.Digest
	bad.OutputSize = wrong.Size
	if _, err := s.CommitProcessingWindowCheckpoint(ctx, w, bad); err == nil {
		t.Fatal("wrong artifact kind accepted")
	}
	valid := processingPublication(t, s, "processing-window")
	bad.PublicationID, bad.OutputDigest, bad.OutputSize = valid.ID, valid.Digest, valid.Size
	for _, field := range []string{"digest", "size", "window", "plan", "publication"} {
		candidate := bad
		switch field {
		case "digest":
			candidate.OutputDigest = hash([]byte("wrong bytes"))
		case "size":
			candidate.OutputSize++
		case "window":
			candidate.WindowID = contracts.ID()
			candidate.ID = ProcessingCheckpointID(candidate.PlanID, candidate.WindowID, candidate.Stage)
		case "plan":
			candidate.PlanID = contracts.ID()
			candidate.ID = ProcessingCheckpointID(candidate.PlanID, candidate.WindowID, candidate.Stage)
		case "publication":
			candidate.PublicationID = contracts.ID()
		}
		if _, err := s.CommitProcessingWindowCheckpoint(ctx, w, candidate); err == nil {
			t.Fatal("invalid checkpoint authority accepted", field)
		}
	}
	partial := Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: valid.ProfileID, ProfileRevision: valid.ProfileRevision, Digest: valid.Digest, Size: valid.Size, Kind: valid.Kind, Key: "objects/" + contracts.ID(), Owner: contracts.ID()}
	partial, err := s.BeginPublication(ctx, partial, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	bad.PublicationID = partial.ID
	if _, err := s.CommitProcessingWindowCheckpoint(ctx, w, bad); err == nil {
		t.Fatal("partial output acquired checkpoint authority")
	}
	c := processingCheckpointFixture(t, s, w, p)
	r := Recording{ID: entry.ID, SourceRevision: entry.Revision, SourceDigest: entry.Digest, State: "no-speech", Document: json.RawMessage("null"), SourceMap: json.RawMessage("{}"), Provenance: json.RawMessage("{}"), Diagnostics: json.RawMessage("[]")}
	if _, err := s.CommitRecording(ctx, w, 0, r); err != nil {
		t.Fatal(err)
	}
	got, err := s.ProcessingPlan(ctx, p.ID)
	if err != nil || got.State != "completed" {
		t.Fatal("plan completion", got.State, err)
	}
	rows, err := s.ProcessingWindowCheckpoints(ctx, p.ID)
	if err != nil || len(rows) != 1 || rows[0].State != "retired" {
		t.Fatal("checkpoint retirement", rows, err)
	}
	if _, err = s.ClaimRetirement(ctx, c.PublicationID, contracts.ID(), 0, time.Minute); err != nil {
		t.Fatal("retired checkpoint pinned", err)
	}
	snap, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = localStore(t, s.workspace).Restore(ctx, snap); err != nil {
		t.Fatal("retirement proof restore", err)
	}
	snap.Records.ProcessingCheckpoints[0] = c
	snap.Digest, _ = snap.digest()
	if err = localStore(t, s.workspace).Restore(ctx, snap); err == nil {
		t.Fatal("old ready checkpoint resurrected")
	}
}

func TestProcessingInvalidation(t *testing.T) {
	processingInvalidationSuite(t, func(t *testing.T) *Store { return localStore(t, contracts.ID()) })
}

func processingInvalidationSuite(t *testing.T, open func(*testing.T) *Store) {
	t.Helper()
	for _, replacement := range []string{"source", "document"} {
		t.Run(replacement, func(t *testing.T) {
			ctx := context.Background()
			s := open(t)
			w, p, entry := processingPlanFixture(t, s)
			c := processingCheckpointFixture(t, s, w, p)
			if replacement == "source" {
				entry.Title = "Updated source revision"
				if _, err := s.UpdateLibrary(ctx, contracts.ID(), entry.Revision, entry); err != nil {
					t.Fatal(err)
				}
			} else {
				other := speakerWork(t, s, "recordings.process")
				r := Recording{ID: entry.ID, SourceRevision: entry.Revision, SourceDigest: entry.Digest, State: "no-speech", Document: json.RawMessage("null"), SourceMap: json.RawMessage("{}"), Provenance: json.RawMessage("{}"), Diagnostics: json.RawMessage("[]")}
				if _, err := s.CommitRecording(ctx, other, 0, r); err != nil {
					t.Fatal(err)
				}
			}
			got, err := s.ProcessingPlan(ctx, p.ID)
			if err != nil || got.State != "invalidated" {
				t.Fatal("replacement retained plan authority", got.State, err)
			}
			rows, err := s.ProcessingWindowCheckpoints(ctx, p.ID)
			if err != nil || len(rows) != 1 || rows[0].State != "retired" {
				t.Fatal("replacement retained checkpoint authority", rows, err)
			}
			if _, err = s.CommitProcessingWindowCheckpoint(ctx, w, c); err == nil {
				t.Fatal("obsolete source/document owner accepted output")
			}
			cleanup, err := s.Cleanups(ctx)
			if err != nil || len(cleanup) != 1 || cleanup[0].ID != c.PublicationID {
				t.Fatal("replacement did not queue physical cleanup", cleanup, err)
			}
			publication, err := s.ClaimRetirement(ctx, c.PublicationID, contracts.ID(), 0, time.Minute)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.FinishRetirement(ctx, publication); err != nil {
				t.Fatal(err)
			}
			if err = s.FinishCleanup(ctx, c.PublicationID); err != nil {
				t.Fatal(err)
			}
			snapshot, err := s.Export(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err = localStore(t, s.workspace).Restore(ctx, snapshot); err != nil {
				t.Fatal("retired checkpoint provenance restore", err)
			}
		})
	}
}

func TestProcessingCheckpointPortableSelection(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	w, p, entry := processingPlanFixture(t, s)
	c := processingCheckpointFixture(t, s, w, p)
	orphan := processingPublication(t, s, "processing-window")
	snapshot, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := BackupPublications(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, publication := range selected {
		seen[publication.ID] = true
	}
	if !seen[c.PublicationID] || seen[orphan.ID] {
		t.Fatal("backup confused accepted checkpoint and uncommitted artifact", seen)
	}
	destination, err := workspace.Init(t.TempDir(), "Checkpoint restore")
	if err != nil {
		t.Fatal(err)
	}
	destination.Config.WorkspaceID = s.workspace
	for n := range selected {
		selected[n].ProfileID = destination.Config.Profiles.Storage.ID
		selected[n].ProfileRevision = destination.Config.Profiles.Storage.Revision
		selected[n].Key = "objects/restore/" + selected[n].ID
		selected[n].Version = ""
	}
	restored := localStore(t, s.workspace)
	if err = restored.RestorePortable(ctx, snapshot, destination, selected, false); err != nil {
		t.Fatal("portable checkpoint restore", err)
	}
	checkpoints, err := restored.ProcessingWindowCheckpoints(ctx, p.ID)
	if err != nil || len(checkpoints) != 1 || checkpoints[0] != c {
		t.Fatal("portable restore changed producer identity", checkpoints, err)
	}
	if _, err = restored.CommitProcessingWindowCheckpoint(ctx, w, c); err == nil {
		t.Fatal("restored lease retained authority")
	}
	if _, err = restored.Export(ctx); err != nil {
		t.Fatal(err)
	}
	entry.Title = "Replacement"
	if _, err = s.UpdateLibrary(ctx, contracts.ID(), entry.Revision, entry); err != nil {
		t.Fatal(err)
	}
	snapshot, err = s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	selected, err = BackupPublications(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	for _, publication := range selected {
		if publication.ID == c.PublicationID || publication.ID == orphan.ID {
			t.Fatal("backup retained retired or uncommitted checkpoint bytes")
		}
	}
}

func TestPublishedV10ProcessingMigration(t *testing.T) {
	processingMigrationSuite(t, localStore(t, contracts.ID()))
}

func processingMigrationSuite(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	if historicalV10Digest() != historicalV10ExpectedDigest {
		t.Fatal("published schema-10 DDL changed")
	}
	w, entry := entryFixture(t, s)
	if _, err := s.CommitLibrary(ctx, w, entry); err != nil {
		t.Fatal(err)
	}
	before, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{"DROP TABLE processing_window_checkpoint", "DROP TABLE processing_plan"} {
		if _, err = s.db.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.db.ExecContext(ctx, s.query("UPDATE catalog_schema SET version=10,digest=?"), historicalV10Digest()); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.ExecContext(ctx, "UPDATE workspace SET schema_version=10"); err != nil {
		t.Fatal(err)
	}
	if err = s.migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = s.checkVersion(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("processing migration changed accepted authority")
	}
	before.CatalogSchema = 10
	before.Digest, _ = before.digest()
	if err = localStore(t, s.workspace).Restore(ctx, before); err != nil {
		t.Fatal("published schema-10 snapshot rejected", err)
	}
	processingPlanFixture(t, s)
}

func TestProcessingOrphanCleanup(t *testing.T) {
	processingOrphanCleanupSuite(t, localStore(t, contracts.ID()))
}

func processingOrphanCleanupSuite(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	w, p, _ := processingPlanFixture(t, s)
	accepted := processingCheckpointFixture(t, s, w, p)
	orphan := processingPublication(t, s, "processing-window", ProcessingWindowPublicationID(w.ID, p.PlanDigest, accepted.WindowID, w.Generation))
	if err := s.RecoverProcessingWindowCleanup(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	if cleanups, err := s.Cleanups(ctx); err != nil || len(cleanups) != 0 {
		t.Fatal("live producer publication was queued", cleanups, err)
	}
	if _, err := s.CancelWork(ctx, contracts.ID(), w.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RecoverProcessingWindowCleanup(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	if cleanups, err := s.Cleanups(ctx); err != nil || len(cleanups) != 1 || cleanups[0].ID != orphan.ID {
		t.Fatal("orphan not retired independently of ready checkpoint", cleanups, err)
	}
	if _, err := s.ClaimRetirement(ctx, accepted.PublicationID, contracts.ID(), 0, time.Minute); err == nil {
		t.Fatal("ready checkpoint lost its retirement fence")
	}
	before, _ := s.Revision(ctx)
	if err := s.RecoverProcessingWindowCleanup(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	after, _ := s.Revision(ctx)
	if before != after {
		t.Fatal("repeated orphan recovery changed accepted authority")
	}
	if _, err := s.RetryWork(ctx, contracts.ID(), w.ID); err != nil {
		t.Fatal(err)
	}
	next, err := s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	newPublication := processingPublication(t, s, "processing-window", ProcessingWindowPublicationID(w.ID, p.PlanDigest, accepted.WindowID, next.Generation))
	if err = s.RecoverProcessingWindowCleanup(ctx, w.ID); err != nil {
		t.Fatal(err)
	}
	if cleanups, err := s.Cleanups(ctx); err != nil || len(cleanups) != 1 {
		t.Fatal("successor publication queued while live", cleanups, newPublication.ID, err)
	}
	snapshot, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err = localStore(t, s.workspace).Restore(ctx, snapshot); err != nil {
		t.Fatal("orphan cleanup receipt rejected during restore", err)
	}
}
