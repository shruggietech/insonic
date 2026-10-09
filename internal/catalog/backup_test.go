// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestTransferFencesWritesAndStaleGeneration(t *testing.T) {
	transferSuite(t, localStore(t, contracts.ID()))
}

func TestPortablePendingInputSanitizationLeavesSourceUntouched(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	path := `A:\private original\input.wav`
	payload, _ := json.Marshal(map[string]any{"items": []any{map[string]any{"source": path}}, "defaults": map[string]any{}})
	w, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", payload)
	if e != nil {
		t.Fatal(e)
	}
	remote, e := s.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{"items":[{"source":"https://example.invalid/current.wav"}]}`))
	if e != nil {
		t.Fatal(e)
	}
	source, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	portable, e := PortableSnapshot(ctx, source)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(portable)
	needle, _ := json.Marshal(path)
	if strings.Contains(string(raw), string(needle)) || strings.Contains(string(raw), "private original") {
		t.Fatal("original input persisted anywhere in portable snapshot")
	}
	original, e := s.Work(ctx, w.ID)
	if e != nil || string(original.Payload) != string(w.Payload) || original.State != "pending" {
		t.Fatal("source admission mutated", e)
	}
	target := localStore(t, s.workspace)
	if e = target.Restore(ctx, portable); e != nil {
		t.Fatal("portable input proof", e)
	}
	got, e := target.Work(ctx, w.ID)
	if e != nil || got.State != "failed" || got.Phase != "portable-source-required" {
		t.Fatal("local admission revived", got, e)
	}
	if _, e = target.RetryWork(ctx, contracts.ID(), w.ID); e == nil {
		t.Fatal("missing original input retried")
	}
	retained, e := target.Work(ctx, remote.ID)
	if e != nil || retained.State != "pending" || string(retained.Payload) != string(remote.Payload) {
		t.Fatal("selected remote input reduced", e)
	}
}
func transferSuite(t *testing.T, s *Store) {
	ctx := context.Background()
	lease, e := s.BeginTransfer(ctx, contracts.ID(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutNamedSetting(ctx, contracts.ID(), "blocked", 0, json.RawMessage(`{}`)); e == nil {
		t.Fatal("ordinary write bypassed transfer")
	}
	owned := WithTransfer(ctx, lease)
	if _, e = s.PutNamedSetting(owned, contracts.ID(), "allowed", 0, json.RawMessage(`{}`)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.BeginTransfer(ctx, lease.OwnerID, time.Second); e == nil {
		t.Fatal("live transfer generation replaced")
	}
	if e = s.EndTransfer(ctx, lease); e != nil {
		t.Fatal(e)
	}
	replacement, e := s.BeginTransfer(ctx, contracts.ID(), time.Second)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutNamedSetting(owned, contracts.ID(), "stale", 0, json.RawMessage(`{}`)); e == nil {
		t.Fatal("stale transfer wrote")
	}
	if e = s.RenewTransfer(ctx, lease, time.Second); e == nil {
		t.Fatal("stale generation renewed")
	}
	if e = s.EndTransfer(ctx, replacement); e != nil {
		t.Fatal(e)
	}
	if _, e = s.PutNamedSetting(ctx, contracts.ID(), "ordinary", 0, json.RawMessage(`{}`)); e != nil {
		t.Fatal(e)
	}
}

func portableAuthorityFixture(t *testing.T, s *Store) (Snapshot, Recording, SpeakerOutput, Publication) {
	t.Helper()
	ctx := context.Background()
	p := availableArtifact(t, s)
	claim, entry := entryFixture(t, s)
	entry.Mode = "copy"
	entry.SourceLocator = ""
	entry.Digest = p.Digest
	entry.Size = p.Size
	entry.OriginalPublicationID = &p.ID
	entry.Metadata = json.RawMessage(`{"raw":"immutable","unix_ns":1781029324123456789}`)
	entry, e := s.CommitLibrary(ctx, claim, entry)
	if e != nil {
		t.Fatal(e)
	}
	claim = speakerWork(t, s, "recordings.process")
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	delete(doc, "media_timing")
	var cues []map[string]json.RawMessage
	json.Unmarshal(doc["cues"], &cues)
	voice := contracts.ID()
	cues[0]["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": voice, "start_milliseconds": 0, "end_milliseconds": 100}})
	doc["cues"], _ = json.Marshal(cues)
	raw, _ = json.Marshal(doc)
	r := Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: entry.Revision, State: "ready", Document: raw, DocumentDigest: hash(raw), SourceMap: json.RawMessage(`{"clock":"source","channels":2}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	r, e = s.CommitRecording(ctx, claim, 0, r)
	if e != nil {
		t.Fatal(e)
	}
	sp, e := s.PutSpeaker(ctx, contracts.ID(), 0, SpeakerIdentity{Speaker: Speaker{ID: contracts.ID(), Name: "Portable"}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: voice, SpeakerID: sp.Speaker.ID, DocumentDigest: r.DocumentDigest}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.MutateRoster(ctx, contracts.ID(), r.ID, 0, "replace", []string{sp.Speaker.ID}); e != nil {
		t.Fatal(e)
	}
	page, e := s.CurrentSpeakerReferences(ctx, SpeakerSelection{SpeakerID: sp.Speaker.ID, ConfirmedOnly: true})
	if e != nil {
		t.Fatal(e)
	}
	manifest := availableArtifact(t, s)
	dataset, e := s.CreateSpeakerDataset(ctx, contracts.ID(), sp.Speaker.ID, page.References, page.Epoch, json.RawMessage(`{}`), json.RawMessage(`{}`), manifest.ID)
	if e != nil {
		t.Fatal(e)
	}
	output := speakerOutputFixture(t, s, dataset)
	if _, e = s.SetSpeakerProfile(ctx, contracts.ID(), sp.Speaker.ID, 0, output.ID); e != nil {
		t.Fatal(e)
	}
	obsolete := availableArtifact(t, s)
	updated := entry
	updated.ReportPublicationIDs, _ = json.Marshal([]string{obsolete.ID})
	updated, e = s.UpdateLibrary(ctx, contracts.ID(), updated.Revision, updated)
	if e != nil {
		t.Fatal(e)
	}
	updated.ReportPublicationIDs = json.RawMessage(`[]`)
	if _, e = s.UpdateLibrary(ctx, contracts.ID(), updated.Revision, updated); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	return snap, r, output, obsolete
}
func TestPortableCurrentAuthorityAndDurableLineage(t *testing.T) {
	portableAuthoritySuite(t, localStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}
func portableAuthoritySuite(t *testing.T, s *Store, newStore func(string) *Store) {
	ctx := context.Background()
	snap, recording, model, obsolete := portableAuthorityFixture(t, s)
	pubs, e := BackupPublications(snap)
	if e != nil {
		t.Fatal(e)
	}
	seen := map[string]bool{}
	for _, p := range pubs {
		seen[p.ID] = true
	}
	if seen[obsolete.ID] {
		t.Fatal("superseded derivative retained")
	}
	var outputs []string
	json.Unmarshal(model.PublicationIDs, &outputs)
	for _, id := range outputs {
		if !seen[id] {
			t.Fatal("durable trained artifact omitted", id)
		}
	}
	w, e := workspace.Init(t.TempDir(), "Restore")
	if e != nil {
		t.Fatal(e)
	}
	w.Config.WorkspaceID = s.workspace
	for i := range pubs {
		pubs[i].ProfileID = w.Config.Profiles.Storage.ID
		pubs[i].ProfileRevision = w.Config.Profiles.Storage.Revision
		pubs[i].Key = "objects/restore/" + pubs[i].ID
		pubs[i].Version = ""
	}
	dest := newStore(s.workspace)
	if e = dest.RestorePortable(ctx, snap, w, pubs, false); e != nil {
		t.Fatal("portable restore", e)
	}
	restored, e := dest.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if len(restored.Records.Recordings) != 1 || string(restored.Records.Recordings[0].Document) != string(recording.Document) || len(restored.Records.RosterMembers) != 1 || len(restored.Records.SpeakerMappings) != 1 || len(restored.Records.SpeakerOutputs) != 1 || restored.Records.SpeakerOutputs[0].ID != model.ID {
		t.Fatal("current authoritative lineage lost")
	}
	if e = ValidateSnapshot(ctx, restored); e != nil {
		t.Fatal("restored authority cannot export again", e)
	}
}

func TestBackupPinAtomicityAndRelease(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	a, b := availableArtifact(t, s), availableArtifact(t, s)
	id := contracts.ID()
	forged := b
	forged.Digest = hash([]byte("forged"))
	before, _ := s.Revision(ctx)
	if e := s.PinBackup(ctx, id, []Publication{a, forged}); e == nil {
		t.Fatal("forged pin accepted")
	}
	after, _ := s.Revision(ctx)
	p, _ := s.Publication(ctx, a.ID)
	if before != after || p.References[id] {
		t.Fatal("partial pins committed")
	}
	if e := s.PinBackup(ctx, id, []Publication{a, b}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ClaimRetirement(ctx, a.ID, contracts.ID(), 0, time.Minute); e == nil {
		t.Fatal("pinned object retired")
	}
	if e := s.ReleaseBackup(ctx, id, []Publication{a, b}); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ClaimRetirement(ctx, a.ID, contracts.ID(), 0, time.Minute); e != nil {
		t.Fatal(e)
	}
}
