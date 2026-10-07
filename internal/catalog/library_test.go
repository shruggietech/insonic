package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
	"time"
)

func entryFixture(t *testing.T, s *Store) (Work, LibraryEntry) {
	t.Helper()
	ctx := context.Background()
	w, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"source":"fixture"}`))
	if e != nil {
		t.Fatal(e)
	}
	w, e = s.ClaimWork(ctx, w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	return w, LibraryEntry{ID: contracts.ID(), AssetID: contracts.ID(), Title: "Fixture", Class: "audio", Mode: "reference", SourceLocator: "fixture.wav", Digest: strings.Repeat("c", 64), Size: 16, Facts: json.RawMessage(`{}`), Metadata: json.RawMessage(`{"capture_state":"failed"}`), Dates: json.RawMessage(`[]`), ReportPublicationIDs: json.RawMessage(`[]`)}
}
func librarySuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, entry := entryFixture(t, s)
	report := availableArtifact(t, s)
	entry.ReportPublicationIDs, _ = json.Marshal([]string{report.ID})
	// Reference membership follows decoded UUIDs, including valid JSON escapes.
	entry.ReportPublicationIDs = json.RawMessage(strings.ReplaceAll(string(entry.ReportPublicationIDs), "-", `\u002d`))
	entry.Metadata = json.RawMessage(`{"capture_state":"captured","value":"old-derived-value"}`)
	current, e := s.CommitLibrary(ctx, claim, entry)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := s.CommitLibrary(ctx, claim, entry)
	if e != nil || replay.Revision != current.Revision {
		t.Fatal("library replay", e)
	}
	if _, e = s.ClaimRetirement(ctx, report.ID, contracts.ID(), 0, time.Minute); e == nil {
		t.Fatal("retired current report")
	}
	next := availableArtifact(t, s)
	updated := current
	updated.Metadata = json.RawMessage(`{"capture_state":"captured","value":"new-value"}`)
	updated.ReportPublicationIDs, _ = json.Marshal([]string{next.ID})
	zero := int64(0)
	updated.DurationUS = &zero
	updated, e = s.UpdateLibrary(ctx, contracts.ID(), current.Revision, updated)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.UpdateLibrary(ctx, contracts.ID(), current.Revision, updated); e == nil {
		t.Fatal("stale entry accepted")
	}
	cleanup, e := s.Cleanups(ctx)
	if e != nil || len(cleanup) != 1 || cleanup[0].ID != report.ID || cleanup[0].State != "pending" {
		t.Fatal("missing durable cleanup", cleanup, e)
	}
	if e = s.FinishCleanup(ctx, report.ID); e == nil {
		t.Fatal("claimed cleanup before retirement")
	}
	retiring, e := s.ClaimRetirement(ctx, report.ID, contracts.ID(), 0, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.FinishRetirement(ctx, retiring); e != nil {
		t.Fatal(e)
	}
	if e = s.FinishCleanup(ctx, report.ID); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(snap)
	if strings.Contains(string(raw), "old-derived-value") {
		t.Fatal("superseded metadata archived")
	}
	if len(snap.Records.Library) != 1 || snap.Records.Library[0].DurationUS == nil || *snap.Records.Library[0].DurationUS != 0 {
		t.Fatal("measured zero lost")
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	restored, e := dest.Work(ctx, claim.ID)
	if e != nil || restored.LeaseUntil != 0 {
		t.Fatal("work restore retained authority", e)
	}
	if _, e = dest.RenewWork(ctx, claim, time.Minute); e == nil {
		t.Fatal("restored claim renewed")
	}
	forged := snap
	forged.Records.Cleanups = append([]Cleanup{}, snap.Records.Cleanups...)
	unrelated := availableArtifact(t, s)
	forged, e = s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	forged.Records.Cleanups = append(forged.Records.Cleanups, Cleanup{ID: unrelated.ID, EntryID: entry.ID, State: "pending", Revision: updated.Revision})
	forged.Digest, _ = forged.digest()
	if e = localStore(t, s.workspace).Restore(ctx, forged); e == nil {
		t.Fatal("unrelated publication forged into pending cleanup")
	}
	snap.Records.Library[0].Metadata = json.RawMessage(`{"forged":true}`)
	snap.Digest, _ = snap.digest()
	if e = localStore(t, s.workspace).Restore(ctx, snap); e == nil {
		t.Fatal("forged current library proof accepted")
	}
}
func TestSQLiteCurrentLibrary(t *testing.T) { librarySuite(t, localStore(t, contracts.ID())) }
func TestCurrentDurationBounds(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	_, recording, _ := evidenceRecording(t, s)
	entry, e := s.Library(ctx, recording.ID)
	if e != nil {
		t.Fatal(e)
	}
	zero := int64(0)
	entry.DurationUS = &zero
	if _, e = s.UpdateLibrary(ctx, contracts.ID(), entry.Revision, entry); e == nil {
		t.Fatal("known zero accepted existing out-of-range current cues")
	}
}

func TestSQLiteBaseModelInstall(t *testing.T) { modelSuite(t, localStore(t, contracts.ID())) }
func modelSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	claim, _ := entryFixture(t, s)
	p := availableArtifact(t, s)
	raw, _ := json.Marshal(map[string]any{"kind": "base-model-manifest", "schema_version": contracts.Version, "name": "fixture", "model_version": "1", "files": []any{map[string]any{"sha256": p.Digest, "size": p.Size}}})
	install := BaseModelInstall{ID: contracts.ID(), Name: "fixture", Version: "1", Digest: hash(raw), Manifest: raw, PublicationIDs: json.RawMessage(`[]`), State: "available"}
	if _, e := s.CommitBaseModel(ctx, claim, install); e == nil {
		t.Fatal("available model omitted required bytes")
	}
	install.PublicationIDs, _ = json.Marshal([]string{p.ID})
	install.PublicationIDs = json.RawMessage(strings.ReplaceAll(string(install.PublicationIDs), "-", `\u002d`))
	got, e := s.CommitBaseModel(ctx, claim, install)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Minute); e == nil {
		t.Fatal("retired current model")
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest := localStore(t, s.workspace)
	if e = dest.Restore(ctx, snap); e != nil {
		t.Fatal(e)
	}
	restored, e := dest.BaseModel(ctx, got.ID)
	if e != nil || restored.Digest != got.Digest {
		t.Fatal("model manifest roundtrip", e)
	}
	install.Manifest = json.RawMessage(`{}`)
	if _, e = s.CommitBaseModel(ctx, claim, install); e == nil {
		t.Fatal("model manifest digest mismatch")
	}
}
func TestLibraryRetirementAdmissionRace(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	for i := 0; i < 8; i++ {
		claim, entry := entryFixture(t, s)
		p := availableArtifact(t, s)
		entry.ReportPublicationIDs, _ = json.Marshal([]string{p.ID})
		start := make(chan struct{})
		results := make(chan error, 2)
		go func() { <-start; _, e := s.CommitLibrary(ctx, claim, entry); results <- e }()
		go func() { <-start; _, e := s.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Minute); results <- e }()
		close(start)
		a, b := <-results, <-results
		if (a == nil) == (b == nil) {
			t.Fatal("current admission/retirement did not have one winner", a, b)
		}
	}
}

func TestLegacyCaptureReplacementPreservesOwnerDates(t *testing.T) {
	s := localStore(t, contracts.ID())
	ctx := context.Background()
	claim, entry := entryFixture(t, s)
	old := availableArtifact(t, s)
	fresh := availableArtifact(t, s)
	snapshotID, observationID, ownerDate, embeddedDate := contracts.ID(), contracts.ID(), contracts.ID(), contracts.ID()
	now := time.Now().UTC()
	rev, _ := s.Revision(ctx)
	rows := Records{Artifacts: []Artifact{{ID: entry.AssetID, Digest: entry.Digest, Size: entry.Size, Kind: "source"}}, Assets: []Asset{{ID: entry.AssetID, ArtifactID: entry.AssetID, Role: "original"}}, Media: []Media{{ID: entry.ID, Title: entry.Title, Class: entry.Class}}, MediaAssets: []MediaAsset{{ID: entry.ID, MediaID: entry.ID, AssetID: entry.AssetID, Role: "original"}}, Metadata: []MetadataSnapshot{{ID: snapshotID, AssetID: entry.AssetID, ReportArtifactID: &old.ArtifactID, State: "captured", Extractor: "fixture", Version: "1", Captured: Instant{ISO: now.Format(time.RFC3339Nano), UnixNS: now.UnixNano()}}}, Observations: []Observation{{ID: observationID, SnapshotID: snapshotID, Family: "EXIF", Tag: "CreateDate", Raw: json.RawMessage(`"old-computed-value"`), Value: json.RawMessage(`"old-computed-value"`)}}, Dates: []DateObservation{{ID: ownerDate, MediaID: entry.ID, ObservationID: &observationID, Basis: "owner", Literal: "2026-10-07", Precision: "day"}, {ID: embeddedDate, MediaID: entry.ID, ObservationID: &observationID, Basis: "embedded", Literal: "2000-01-01", Precision: "day"}}, DateSelections: []DateSelection{{ID: contracts.ID(), MediaID: entry.ID, Revision: 1, DateID: &ownerDate, Policy: "owner-first"}, {ID: contracts.ID(), MediaID: entry.ID, Revision: 2, DateID: &embeddedDate, Policy: "embedded-first"}}}
	if _, e := s.Commit(ctx, Mutation{OperationID: contracts.ID(), Expected: rev, Records: rows}); e != nil {
		t.Fatal(e)
	}
	entry.ReportPublicationIDs, _ = json.Marshal([]string{fresh.ID})
	if _, e := s.CommitLibrary(ctx, claim, entry); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(snap.Records.Metadata) != 0 || len(snap.Records.Observations) != 0 || len(snap.Records.Dates) != 1 || snap.Records.Dates[0].ID != ownerDate || snap.Records.Dates[0].ObservationID != nil || len(snap.Records.DateSelections) != 1 {
		t.Fatal("legacy replacement retained derived capture or lost owner evidence")
	}
	if len(snap.Records.Cleanups) != 1 || snap.Records.Cleanups[0].ID != old.ID {
		t.Fatal("legacy report omitted cleanup")
	}
	if _, e = s.ClaimRetirement(ctx, old.ID, contracts.ID(), 0, time.Minute); e != nil {
		t.Fatal("legacy report still structurally referenced", e)
	}
}
