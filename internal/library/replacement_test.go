// SPDX-License-Identifier: Apache-2.0
package library

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeReplacementRetainedSubtitleRequiresPolicy(t *testing.T) {
	s, db := libraryFixture(t)
	path := filepath.Join(t.TempDir(), "retained.srt")
	os.WriteFile(path, []byte("1\n00:00:00,000 --> 00:00:00,005\nRetained text\n\n"), 0600)
	entry := seedLegacyAdmission(t, s, db, path)
	s.legacyFixture = false
	s.Tools = nativeTools(t)
	executable := os.Getenv("CUESON_EXECUTABLE")
	if executable == "" {
		t.Skip("pinned Cueson not selected")
	}
	digest, _, e := identity(context.Background(), executable)
	if e != nil {
		t.Fatal(e)
	}
	driver, e := subtitles.New(subtitles.Tool{Executable: executable, ExecutableSHA256: digest})
	if e != nil {
		t.Fatal(e)
	}
	s.NativeIngest = driver.Ingest
	yes := true
	missing := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: "missing.wav", Options: Options{ReplaceAudio: &yes}})
	if missing.Error != "invalid_request" {
		t.Fatalf("retained subtitle policy %+v", missing)
	}
	result := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: wavFixture(t), Options: Options{ReplaceAudio: &yes, ExistingTranscript: "keep", TranscriptApplies: &yes}})
	if result.State != "replaced" {
		t.Fatalf("retained subtitle keep %+v", result)
	}
	current, e := db.Recording(context.Background(), entry.ID)
	if e != nil || current.State != "ready" || !bytes.Contains(current.Document, []byte("Retained text")) {
		t.Fatal("retained document", e)
	}
	accepted, _ := db.Library(context.Background(), entry.ID)
	if accepted.SubtitlePublicationID != nil {
		t.Fatal("second transcript authority retained")
	}
	if _, e = os.Stat(path); e != nil {
		t.Fatal("owner input removed", e)
	}
}

func TestKeptSelectedSourceRejectsIncompatibleLayoutAndBounds(t *testing.T) {
	one, two := int64(1), int64(2)
	zero := int64(0)
	ticks := int64(1000)
	stream := Stream{Index: 0, Kind: "audio", Channels: &one, TimeBase: "1/1000", StartPTS: &zero, DurationTS: &ticks}
	old := catalog.LibraryEntry{Facts: marshal(Facts{Container: "flac", Streams: []Stream{stream}})}
	previous := &catalog.Recording{SourceMap: json.RawMessage(`{"stream_index":0,"channel":0}`), Document: json.RawMessage(`{"cues":[{"timing":{"start_milliseconds":0,"end_milliseconds":500}}]}`)}
	if e := validateKeptSelection(old, old, previous); e != nil {
		t.Fatal(e)
	}
	stream.Channels = &two
	candidate := catalog.LibraryEntry{Facts: marshal(Facts{Container: "flac", Streams: []Stream{stream}})}
	if e := validateKeptSelection(old, candidate, previous); e == nil {
		t.Fatal("changed known channels accepted")
	}
	stream.Channels = &one
	short := int64(100)
	stream.DurationTS = &short
	candidate.Facts = marshal(Facts{Container: "flac", Streams: []Stream{stream}})
	if e := validateKeptSelection(old, candidate, previous); e == nil {
		t.Fatal("cue beyond selected audio accepted")
	}
	stream.Index = 1
	candidate.Facts = marshal(Facts{Container: "flac", Streams: []Stream{stream}})
	if e := validateKeptSelection(old, candidate, previous); e == nil {
		t.Fatal("missing selected track accepted")
	}
}

type electedResponseLost struct {
	catalog.Catalog
	lost bool
}

func (f *electedResponseLost) CommitElectedAdmission(ctx context.Context, w catalog.Work, ordinal int, expected int64, e catalog.LibraryEntry, r *catalog.Recording, election catalog.AdmissionElection, result ...json.RawMessage) (catalog.LibraryEntry, *catalog.Recording, error) {
	entry, doc, err := f.Catalog.CommitElectedAdmission(ctx, w, ordinal, expected, e, r, election, result...)
	if err == nil && !f.lost {
		f.lost = true
		return entry, doc, contracts.Fail("unavailable")
	}
	return entry, doc, err
}
func TestNativeReplacementResponseLostReconcilesWithoutInput(t *testing.T) {
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = nativeTools(t)
	ctx := context.Background()
	yes := true
	first := executeTranscript(t, s, db, Item{Source: wavFixture(t)})
	if first.Error != "" {
		t.Fatal(first)
	}
	source := wavFixture(t)
	input := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Kind: "media", Record: first.MediaID, Source: source, Options: Options{ReplaceAudio: &yes, ExistingTranscript: "clear"}}}})
	work := claimLibraryWork(t, db, "media.import", input)
	s.Catalog = &electedResponseLost{Catalog: db}
	result, e := s.Execute(ctx, work)
	if e != nil || result.(ImportResult).Items[0].Error != "unavailable" {
		t.Fatalf("response loss %+v %v", result, e)
	}
	accepted, _ := db.Library(ctx, first.MediaID)
	os.Remove(source)
	result, e = s.Execute(ctx, work)
	if e != nil || result.(ImportResult).Items[0].State != "replaced" {
		t.Fatalf("accepted reconciliation %+v %v", result, e)
	}
	current, _ := db.Library(ctx, first.MediaID)
	if current.Revision != accepted.Revision {
		t.Fatal("replay advanced current")
	}
}

func TestNativeReplacementRosterElectionAndReplay(t *testing.T) {
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = nativeTools(t)
	ctx := context.Background()
	yes := true
	sp, e := db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Known voice", State: "active"}, Aliases: []catalog.SpeakerAlias{}})
	if e != nil {
		t.Fatal(e)
	}
	request := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Source: wavFixture(t), Options: Options{KnownSpeakers: []string{sp.Speaker.ID}}}}})
	w := claimLibraryWork(t, db, "media.import", request)
	v, e := s.Execute(ctx, w)
	if e != nil {
		t.Fatal(e)
	}
	first := v.(ImportResult).Items[0]
	if first.Error != "" {
		t.Fatalf("initial %+v", first)
	}
	entry, e := db.Library(ctx, first.MediaID)
	if e != nil {
		t.Fatal(e)
	}
	roster, e := db.Roster(ctx, entry.ID)
	if e != nil || len(roster.Members) != 1 {
		t.Fatalf("roster %+v %v", roster, e)
	}
	missing := filepath.Join(t.TempDir(), "missing.wav")
	skip := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: missing})
	if skip.State != "skipped" {
		t.Fatalf("skip before acquire %+v", skip)
	}
	reject := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: missing, Options: Options{ReplaceAudio: &yes, ExistingTranscript: "clear"}})
	if reject.Error != "invalid_request" {
		t.Fatalf("required roster election %+v", reject)
	}
	source := wavFixture(t)
	// A different waveform produces a different canonical artifact.
	data, _ := os.ReadFile(source)
	data[len(data)-1] ^= 1
	os.WriteFile(source, data, 0600)
	input := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Kind: "media", Record: entry.ID, Source: source, Options: Options{ReplaceAudio: &yes, ExistingTranscript: "clear", ExistingRoster: "retain"}}}})
	lease, e := s.Artifacts.Materialize(ctx, *entry.OriginalPublicationID)
	if e != nil {
		t.Fatal(e)
	}
	w = claimLibraryWork(t, db, "media.import", input)
	v, e = s.Execute(ctx, w)
	if e != nil {
		t.Fatal(e)
	}
	r := v.(ImportResult).Items[0]
	if r.State != "replaced" {
		t.Fatalf("replacement %+v", r)
	}
	current, _ := db.Library(ctx, entry.ID)
	doc, _ := db.Recording(ctx, entry.ID)
	if current.Digest == entry.Digest || doc.State != "untranscribed" || current.Revision != doc.Revision {
		t.Fatal("replacement authority")
	}
	got, _ := db.Roster(ctx, entry.ID)
	if got.Revision != roster.Revision || len(got.Members) != 1 {
		t.Fatal("retained roster")
	}
	oldPublication, e := db.Publication(ctx, *entry.OriginalPublicationID)
	if e != nil || oldPublication.State != "available" {
		t.Fatal("leased original retired", e)
	}
	if e = s.Artifacts.Release(ctx, *entry.OriginalPublicationID, lease.Lease.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.admissionCleanup(ctx, entry.ID); e != nil {
		t.Fatal(e)
	}
	oldPublication, e = db.Publication(ctx, *entry.OriginalPublicationID)
	if e != nil || oldPublication.State != "retired" {
		t.Fatal("unleased original retained", e)
	}
	os.Remove(source)
	v, e = s.Execute(ctx, w)
	if e != nil || v.(ImportResult).Items[0].State != "replaced" {
		t.Fatalf("replay %v %v", v, e)
	}
	snap, e := db.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(snap)
	if len(raw) == 0 {
		t.Fatal("snapshot")
	}
}

func TestNativeKeepAndReplaceDocumentPolicies(t *testing.T) {
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = nativeTools(t)
	ctx := context.Background()
	yes := true
	path, raw := transcriptFixture(t)
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	delete(doc, "media_timing")
	var cues []map[string]json.RawMessage
	json.Unmarshal(doc["cues"], &cues)
	cues = cues[:1]
	cues[0]["timing"] = json.RawMessage(`{"start_milliseconds":0,"end_milliseconds":5,"duration_milliseconds":5}`)
	delete(cues[0], "speaker_attributions")
	doc["cues"], _ = json.Marshal(cues)
	for _, key := range []string{"document", "stats"} {
		var metrics map[string]json.RawMessage
		json.Unmarshal(doc[key], &metrics)
		metrics["media_span_milliseconds"] = json.RawMessage(`5`)
		if key == "document" {
			metrics["media_end_milliseconds"] = json.RawMessage(`5`)
		}
		doc[key], _ = json.Marshal(metrics)
	}
	raw, _ = json.Marshal(doc)
	os.WriteFile(path, raw, 0600)
	first := executeTranscript(t, s, db, Item{Source: wavFixture(t), Transcript: path})
	if first.Error != "" {
		t.Fatalf("initial %+v", first)
	}
	entry, _ := db.Library(ctx, first.MediaID)
	old, _ := db.Recording(ctx, entry.ID)
	missing := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: "missing.wav", Options: Options{ReplaceAudio: &yes, ExistingTranscript: "keep"}})
	if missing.Error != "invalid_request" {
		t.Fatalf("assertion required %+v", missing)
	}
	req := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC", Attribution: "off"}, Items: []Item{{Kind: "media", Record: entry.ID, Source: wavFixture(t), Options: Options{ReplaceAudio: &yes, ExistingTranscript: "keep", TranscriptApplies: &yes}}}})
	work := claimLibraryWork(t, db, "media.import", req)
	v, e := s.Execute(ctx, work)
	if e != nil || v.(ImportResult).Items[0].State != "replaced" {
		t.Fatalf("keep %+v %v", v, e)
	}
	kept, _ := db.Recording(ctx, entry.ID)
	if kept.DocumentDigest != old.DocumentDigest || string(kept.Document) != string(old.Document) || kept.Revision <= old.Revision || kept.MappedAudioPublicationID != nil {
		t.Fatal("kept authority")
	}
	stale := claimLibraryWork(t, db, "media.import", req)
	v, e = s.Execute(ctx, stale)
	if e != nil || v.(ImportResult).Items[0].Error != "conflict" {
		t.Fatal("frozen stale source accepted", v, e)
	}
	absent := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: wavFixture(t), Options: Options{ReplaceAudio: &yes, ExistingTranscript: "replace"}})
	if absent.Error != "invalid_request" {
		t.Fatalf("replacement document required %+v", absent)
	}
	replacement := executeTranscript(t, s, db, Item{Kind: "media", Record: entry.ID, Source: wavFixture(t), Transcript: path, Options: Options{ReplaceAudio: &yes, ExistingTranscript: "replace"}})
	if replacement.State != "replaced" {
		t.Fatalf("replace %+v", replacement)
	}
}

func TestNativeFrozenRosterCannotRetargetAlias(t *testing.T) {
	s, db := libraryFixture(t)
	s.legacyFixture = false
	s.Tools = nativeTools(t)
	ctx := context.Background()
	id, alias := contracts.ID(), contracts.ID()
	first, e := db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: id, Name: "First", State: "active"}, Aliases: []catalog.SpeakerAlias{{ID: alias, SpeakerID: id, Text: "Frozen alias", State: "active", Provenance: json.RawMessage(`{}`)}}})
	if e != nil {
		t.Fatal(e)
	}
	frozen := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Source: wavFixture(t), Options: Options{KnownSpeakers: []string{"Frozen alias"}}}}})
	first.Aliases[0].Text = "Renamed alias"
	if _, e = db.PutSpeaker(ctx, contracts.ID(), first.Speaker.Revision, first); e != nil {
		t.Fatal(e)
	}
	otherID := contracts.ID()
	if _, e = db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: otherID, Name: "Other", State: "active"}, Aliases: []catalog.SpeakerAlias{{ID: contracts.ID(), SpeakerID: otherID, Text: "Frozen alias", State: "active", Provenance: json.RawMessage(`{}`)}}}); e != nil {
		t.Fatal(e)
	}
	work := claimLibraryWork(t, db, "media.import", frozen)
	v, e := s.Execute(ctx, work)
	if e != nil {
		t.Fatal(e)
	}
	item := v.(ImportResult).Items[0]
	if item.Error != "" {
		t.Fatal(item)
	}
	roster, e := db.Roster(ctx, item.MediaID)
	if e != nil || len(roster.Members) != 1 || roster.Members[0].ID != id {
		t.Fatalf("retargeted %+v %v", roster, e)
	}
	yes := true
	replacement := FreezeTargets(ctx, db, ImportRequest{Defaults: Options{Timezone: "UTC"}, Items: []Item{{Kind: "media", Record: item.MediaID, Source: "missing.wav", Options: Options{ReplaceAudio: &yes, ExistingTranscript: "clear", ExistingRoster: "retain"}}}})
	if _, e = db.MutateRoster(ctx, contracts.ID(), item.MediaID, roster.Revision, "clear", nil); e != nil {
		t.Fatal(e)
	}
	work = claimLibraryWork(t, db, "media.import", replacement)
	v, e = s.Execute(ctx, work)
	if e != nil || v.(ImportResult).Items[0].Error != "conflict" {
		t.Fatal("roster race accepted", v, e)
	}
}
