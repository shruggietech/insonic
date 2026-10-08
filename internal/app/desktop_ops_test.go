// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func TestDesktopSettingsCASAndUnknownFields(t *testing.T) {
	a := configuredApp(t)
	r := realRequest(a, "settings.show", "", nil)
	if r.Error != nil {
		t.Fatal("shared settings unavailable", r.Error)
	}
	sections := r.Result.(map[string]any)["sections"].(map[string]any)
	appearance := sections["appearance"].(map[string]any)
	revision := appearance["revision"].(string)
	update := map[string]any{"section": "appearance", "revision": revision, "value": map[string]any{"theme": "dark", "reduced_motion": true}}
	saved := realRequest(a, "settings.set", "", update)
	if saved.Error != nil {
		t.Fatal(saved.Error)
	}
	update["value"] = map[string]any{"theme": "light", "reduced_motion": false}
	if realRequest(a, "settings.set", "", update).Error == nil {
		t.Fatal("stale settings mutation accepted")
	}
	if realRequest(a, "settings.set", "", map[string]any{"section": "appearance", "revision": saved.Result.(map[string]any)["revision"], "value": map[string]any{"theme": "dark", "password": "ignored"}}).Error == nil {
		t.Fatal("unknown setting accepted")
	}
	raw, e := os.ReadFile(filepath.Join(a.Workspace.Control, "appearance.json"))
	if e != nil {
		t.Fatal(e)
	}
	var state map[string]any
	if json.Unmarshal(raw, &state) != nil || state["theme"] != "dark" {
		t.Fatal("mutation did not persist")
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "settings.show", SourcePath: "unrelated"}
	if a.Dispatch(req).Error == nil {
		t.Fatal("settings accepted unrelated envelope")
	}
}

func TestDesktopPlaybackPreparationRenewsLeaseWithoutBlockingRegistry(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, true)
	service, e := a.artifactService()
	if e != nil {
		t.Fatal(e)
	}
	service.TTL = 150 * time.Millisecond
	entered, release := make(chan struct{}), make(chan struct{})
	a.playbackPreview = func(ctx context.Context, p *playbackEntry, e catalog.LibraryEntry) error {
		close(entered)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	done := make(chan contracts.Response, 1)
	go func() { done <- realRequest(a, "media.playback", entry.ID, map[string]any{"revision": entry.Revision}) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("preparation did not start")
	}
	active := make(chan int, 1)
	go func() { active <- a.Active() }()
	select {
	case n := <-active:
		if n < 1 {
			t.Fatal("preparation lost runtime activity")
		}
	case <-time.After(time.Second):
		t.Fatal("preparation blocked runtime admission")
	}
	// Exceed the lease duration while preparation keeps its original alive.
	timer := time.NewTimer(350 * time.Millisecond)
	<-timer.C
	close(release)
	select {
	case r := <-done:
		if r.Error != nil {
			t.Fatal("preparation lease expired", r.Error)
		}
	case <-time.After(time.Second):
		t.Fatal("preparation did not finish")
	}
}

func desktopMedia(t *testing.T, a *App, copyMode bool) catalog.LibraryEntry {
	t.Helper()
	ctx := context.Background()
	source := filepath.Join(t.TempDir(), "source.wav")
	if e := os.WriteFile(source, []byte("verified current playback fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	service, e := a.artifactService()
	if e != nil {
		t.Fatal(e)
	}
	stage, digest, size, e := service.Stage(ctx, source)
	if e != nil {
		t.Fatal(e)
	}
	stage.Close()
	os.Remove(stage.Name())
	claim, e := a.Catalog.EnqueueWork(ctx, contracts.ID(), "media.import", json.RawMessage(`{}`))
	if e != nil {
		t.Fatal(e)
	}
	claim, e = a.Catalog.ClaimWork(ctx, claim.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	entry := catalog.LibraryEntry{ID: contracts.ID(), AssetID: contracts.ID(), Title: "Desktop fixture", Class: "audio", Mode: "reference", SourceLocator: source, Digest: digest, Size: size, Facts: json.RawMessage(`{"extension":"wav"}`), Metadata: json.RawMessage(`{"capture_state":"failed"}`), Dates: json.RawMessage(`{}`), ReportPublicationIDs: json.RawMessage(`[]`)}
	if copyMode {
		publication, e := service.Publish(ctx, contracts.ID(), source, "original")
		if e != nil {
			t.Fatal(e)
		}
		entry.Mode = "copy"
		entry.OriginalPublicationID = &publication.ID
	}
	entry, e = a.Catalog.CommitLibrary(ctx, claim, entry)
	if e != nil {
		t.Fatal(e)
	}
	return entry
}

func desktopRecording(t *testing.T, a *App, entry catalog.LibraryEntry, expected int64, doc json.RawMessage) catalog.Recording {
	t.Helper()
	claim, e := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.assemble", json.RawMessage(`{}`))
	if e != nil {
		t.Fatal(e)
	}
	claim, e = a.Catalog.ClaimWork(a.ctx, claim.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	record := catalog.Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: entry.Revision, State: "ready", Document: doc, DocumentDigest: documentHash(doc), SourceMap: json.RawMessage(`{}`), Provenance: json.RawMessage(`{}`), Diagnostics: json.RawMessage(`[]`)}
	record, e = a.Catalog.CommitRecording(a.ctx, claim, expected, record)
	if e != nil {
		t.Fatal(e)
	}
	return record
}

func TestDesktopPlaybackLeasesIdentityAndReplacement(t *testing.T) {
	for _, copyMode := range []bool{false, true} {
		t.Run(map[bool]string{false: "reference", true: "copy"}[copyMode], func(t *testing.T) {
			a := configuredApp(t)
			entry := desktopMedia(t, a, copyMode)
			doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
			if e != nil {
				t.Fatal(e)
			}
			record := desktopRecording(t, a, entry, 0, doc)
			opened := realRequest(a, "media.playback", entry.ID, map[string]any{"revision": entry.Revision, "recording_revision": record.Revision, "document_digest": record.DocumentDigest})
			if opened.Error != nil {
				t.Fatal(opened.Error)
			}
			d := opened.Result.(PlaybackDescriptor)
			data, e := os.ReadFile(d.Path)
			if e != nil || documentHash(data) != entry.Digest || d.MIME != "audio/wav" || a.Active() < 1 {
				t.Fatal("playback source/lifetime", e, d)
			}
			checked := realRequest(a, "media.playback-check", entry.ID, map[string]any{"playback_id": d.PlaybackID})
			if checked.Error != nil {
				t.Fatal(checked.Error)
			}
			if realRequest(a, "media.playback-check", contracts.ID(), map[string]any{"playback_id": d.PlaybackID}).Error == nil {
				t.Fatal("wrong media accepted")
			}
			replaced := desktopRecording(t, a, entry, record.Revision, doc)
			if replaced.Revision == record.Revision {
				t.Fatal("replacement did not change authority")
			}
			if realRequest(a, "media.playback-check", entry.ID, map[string]any{"playback_id": d.PlaybackID}).Error == nil {
				t.Fatal("replaced recording retained playback")
			}
			if _, e = os.Stat(d.Path); !os.IsNotExist(e) {
				t.Fatal("obsolete playback bytes not released", e)
			}
			fresh := realRequest(a, "media.playback", entry.ID, map[string]any{"revision": entry.Revision})
			if fresh.Error != nil {
				t.Fatal(fresh.Error)
			}
			freshID := fresh.Result.(PlaybackDescriptor).PlaybackID
			if realRequest(a, "media.playback-close", entry.ID, map[string]any{"playback_id": freshID}).Error != nil {
				t.Fatal("close")
			}
			if realRequest(a, "media.playback-check", entry.ID, map[string]any{"playback_id": freshID}).Error == nil {
				t.Fatal("closed handle retained")
			}
			if !copyMode {
				if e = os.WriteFile(entry.SourceLocator, []byte("changed reference"), 0600); e != nil {
					t.Fatal(e)
				}
				if realRequest(a, "media.playback", entry.ID, map[string]any{"revision": entry.Revision}).Error == nil {
					t.Fatal("changed reference bytes played")
				}
			}
		})
	}
}

func TestDesktopCuePagesPreserveIntegerTimingAndRejectStaleAuthority(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	var cues []map[string]json.RawMessage
	json.Unmarshal(fields["cues"], &cues)
	voice := contracts.ID()
	cues[0]["speaker_attributions"], _ = json.Marshal([]map[string]any{{"speaker_id": voice}})
	// Preserve every source unix_ns RawMessage when projecting current cues.
	cues[0]["timing"] = json.RawMessage(`{"start_milliseconds":9007199254740993,"end_milliseconds":9007199254741993,"duration_milliseconds":1000}`)
	fields["cues"], _ = json.Marshal(cues)
	fields["document"] = json.RawMessage(`{"cue_count":1,"media_start_milliseconds":9007199254740993,"media_end_milliseconds":9007199254741993,"media_span_milliseconds":1000,"has_word_level_timing":false}`)
	fields["stats"] = json.RawMessage(`{"cue_count":1,"diagnostic_count":0,"warning_count":0,"error_count":0,"has_word_level_timing":false,"media_span_milliseconds":1000}`)
	doc, _ := json.Marshal(fields)
	record := desktopRecording(t, a, entry, 0, doc)
	r := realRequest(a, "recordings.cues", entry.ID, map[string]any{"revision": record.Revision, "document_digest": record.DocumentDigest, "limit": 1})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	page := r.Result.(map[string]any)
	projected := page["cues"].([]cueProjection)
	if len(projected) != 1 || projected[0].Timing.(map[string]string)["start_milliseconds"] != "9007199254740993" || projected[0].SeekSeconds != nil {
		t.Fatal("integer timing rounded or unsafe seek invented", projected)
	}
	if len(projected[0].Attributions) != 1 || projected[0].Attributions[0].(map[string]any)["seek_seconds"] != nil {
		t.Fatal("untimed participation acquired a seek interval")
	}
	if _, e = a.Catalog.Recording(a.ctx, entry.ID); e != nil {
		t.Fatal(e)
	}
	desktopRecording(t, a, entry, record.Revision, doc)
	if realRequest(a, "recordings.cues", entry.ID, map[string]any{"revision": record.Revision, "document_digest": record.DocumentDigest}).Error == nil {
		t.Fatal("obsolete cue selection accepted")
	}
}

func TestDesktopSettingsRepairsMalformedExplicitConfiguration(t *testing.T) {
	a := configuredApp(t)
	path := filepath.Join(a.Workspace.Control, "appearance.json")
	if e := os.WriteFile(path, []byte(`{"theme":`), 0600); e != nil {
		t.Fatal(e)
	}
	shown := realRequest(a, "settings.show", "", nil)
	if shown.Error != nil {
		t.Fatal(shown.Error)
	}
	section := shown.Result.(map[string]any)["sections"].(map[string]any)["appearance"].(map[string]any)
	if section["error"] == nil || section["value"] != nil {
		t.Fatal("malformed section hidden or echoed")
	}
	if realRequest(a, "settings.set", "", map[string]any{"section": "appearance", "revision": section["revision"], "value": map[string]any{"theme": "system", "reduced_motion": false}}).Error != nil {
		t.Fatal("explicit configuration could not be repaired")
	}
}

func TestDesktopPlaybackExpiryAndInterruptedScratchCleanup(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	opened := realRequest(a, "media.playback", entry.ID, map[string]any{"revision": entry.Revision})
	if opened.Error != nil {
		t.Fatal(opened.Error)
	}
	d := opened.Result.(PlaybackDescriptor)
	a.playbackMu.Lock()
	a.playbacks[d.PlaybackID].touched = time.Now().Add(-playbackIdle)
	a.playbackMu.Unlock()
	if realRequest(a, "media.playback-check", entry.ID, map[string]any{"playback_id": d.PlaybackID}).Error == nil {
		t.Fatal("inactive handle retained authority")
	}
	if _, e := os.Stat(d.Path); !os.IsNotExist(e) {
		t.Fatal("expired snapshot retained", e)
	}
	directory := filepath.Join(a.Workspace.Control, "artifact-scratch")
	for _, name := range []string{"playback-interrupted", "other-publication.source"} {
		if e := os.WriteFile(filepath.Join(directory, name), []byte("disposable"), 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e := RecoverPlaybackScratch(a.Workspace); e != nil {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(directory, "playback-interrupted")); !os.IsNotExist(e) {
		t.Fatal("interrupted snapshot retained", e)
	}
	if _, e := os.Stat(filepath.Join(directory, "other-publication.source")); e != nil {
		t.Fatal("publication bytes removed", e)
	}
}

func TestDesktopCapturePagesKeepNanosecondsExact(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	raw := json.RawMessage(`{"capture_state":"captured","observations":[{"raw_value":{"iso":"2026-10-07T20:46:13.3479284Z","unix_ns":1791405973347928400}}]}`)
	entry.Metadata = raw
	entry, e := a.Catalog.UpdateLibrary(a.ctx, contracts.ID(), entry.Revision, entry)
	if e != nil {
		t.Fatal(e)
	}
	var restored bytes.Buffer
	offset := 0
	digest := ""
	for {
		r := realRequest(a, "media.capture", entry.ID, map[string]any{"section": "metadata", "revision": entry.Revision, "offset": offset, "limit": 7, "sha256": digest})
		if r.Error != nil {
			t.Fatal(r.Error)
		}
		page := r.Result.(map[string]any)
		data, e := base64.StdEncoding.DecodeString(page["data"].(string))
		if e != nil {
			t.Fatal(e)
		}
		restored.Write(data)
		offset = page["next_offset"].(int)
		digest = page["sha256"].(string)
		if page["complete"].(bool) {
			break
		}
	}
	if !bytes.Equal(restored.Bytes(), raw) || digest != documentHash(raw) {
		t.Fatal("exact metadata bytes changed", restored.String())
	}
	entry.Metadata = json.RawMessage(`{"capture_state":"captured","value":"replacement"}`)
	if _, e = a.Catalog.UpdateLibrary(a.ctx, contracts.ID(), entry.Revision, entry); e != nil {
		t.Fatal(e)
	}
	if realRequest(a, "media.capture", entry.ID, map[string]any{"section": "metadata", "revision": entry.Revision, "sha256": digest}).Error == nil {
		t.Fatal("capture replacement retained old page authority")
	}
}
