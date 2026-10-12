// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
)

func savedImportPreset(t *testing.T, a *App, options any) catalog.ImportPreset {
	t.Helper()
	p := catalog.ImportPreset{Kind: "import-preset", Version: contracts.Version, ID: contracts.ID(), Name: "Daily", Options: configurationBytes(options)}
	r := realRequest(a, "import-presets.create", "", map[string]any{"preset": p})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	raw, _ := json.Marshal(r.Result)
	if json.Unmarshal(raw, &p) != nil {
		t.Fatal("preset response")
	}
	return p
}
func TestImportPresetSharedPrecedenceAndReadiness(t *testing.T) {
	a := configuredApp(t)
	profile := workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "filesystem", Version: contracts.Version, Configuration: map[string]any{"root": filepath.Join(t.TempDir(), "private-artifacts")}}
	p := savedImportPreset(t, a, map[string]any{"timezone": "America/New_York", "date_precedence": "owner-first", "copy": true, "destination": profile})
	configuredResult(t, realRequest(a, "import-presets.show", "", map[string]any{"selection": "Daily"}))
	configuredResult(t, realRequest(a, "import-presets.list", "", nil))
	yes, no := true, false
	input := library.ImportRequest{Defaults: library.Options{Timezone: "UTC", Copy: &yes}, Overrides: library.Options{Copy: &no}, Items: []library.Item{{Source: "fixture", Options: library.Options{Preset: p.ID, Timezone: "Europe/London"}}, {Source: "other", Options: library.Options{Preset: "Daily"}}}}
	frozen, e := a.resolveImportChoices(input)
	if e != nil {
		t.Fatal(e)
	}
	for i, want := range []string{"Europe/London", "UTC"} {
		choice := frozen.Elections[frozen.Selections[i]]
		if choice.PresetID != p.ID || choice.PresetRevision != p.Revision || choice.Options.Timezone != want || choice.Options.Copy == nil || *choice.Options.Copy || choice.Options.Destination.ID != profile.ID {
			t.Fatalf("choice %d: %+v", i, choice)
		}
	}
	before, _ := a.Catalog.Works(a.ctx)
	r := realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: "fixture", Options: library.Options{Preset: "missing"}}}})
	if r.Error == nil || r.Error.Code != "not_found" {
		t.Fatal("missing selection accepted", r.Error)
	}
	pipeline := contracts.ID()
	p.Options = configurationBytes(map[string]any{"pipeline_id": pipeline})
	_, e = a.Catalog.MutateImportPreset(a.ctx, contracts.ID(), "update", p.Revision, p)
	if e != nil {
		t.Fatal(e)
	}
	r = realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: "fixture", Options: library.Options{Preset: p.ID}}}})
	if r.Error == nil {
		t.Fatal("missing pipeline admitted")
	}
	after, _ := a.Catalog.Works(a.ctx)
	if len(after) != len(before) {
		t.Fatal("failed readiness queued work")
	}
	newProfile := workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "filesystem", Version: contracts.Version, Configuration: map[string]any{"root": filepath.Join(t.TempDir(), "private-artifacts")}}
	if _, e = a.freezeImportSelection(library.Options{}, library.Options{Destination: &newProfile}, library.Options{}, false); e != nil {
		t.Fatal("explicit standalone destination", e)
	}
	empty := ""
	input = library.ImportRequest{Overrides: library.Options{PipelineID: &empty}, Items: []library.Item{{Source: "fixture", Options: library.Options{Preset: p.ID}}}}
	if _, e = a.resolveImportChoices(input); e != nil {
		t.Fatal("explicit admission-only override", e)
	}
}
func TestImportPresetQueuedReplayAfterEditDelete(t *testing.T) {
	a := configuredApp(t)
	p := savedImportPreset(t, a, map[string]any{"timezone": "UTC", "date_precedence": "owner-first"})
	input := library.ImportRequest{Items: []library.Item{{Source: filepath.Join(t.TempDir(), "missing.wav"), Options: library.Options{Preset: p.ID}}}}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "media.import", Data: configurationBytes(input)}
	first := configuredResult(t, a.Dispatch(req))
	id := first["work_id"].(string)
	original, e := a.Catalog.Work(a.ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	if first["import_selection_count"] != float64(1) {
		t.Fatal("no immediate selection feedback", first)
	}
	var saved library.ImportRequest
	if strictPayload(original.Payload, &saved) != nil {
		t.Fatal("payload")
	}
	choice := saved.Elections[saved.Selections[0]]
	if choice.Options.Timezone != "UTC" || choice.PresetRevision != p.Revision {
		t.Fatal("unfrozen election", choice)
	}
	p.Options = configurationBytes(map[string]any{"timezone": "Asia/Tokyo"})
	updated, e := a.Catalog.MutateImportPreset(a.ctx, contracts.ID(), "update", p.Revision, p)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.Catalog.MutateImportPreset(a.ctx, contracts.ID(), "delete", updated.Revision, catalog.ImportPreset{ID: p.ID}); e != nil {
		t.Fatal(e)
	}
	replay := configuredResult(t, a.Dispatch(req))
	current, e := a.Catalog.Work(a.ctx, id)
	if e != nil || replay["work_id"] != id || string(current.Payload) != string(original.Payload) {
		t.Fatal("replay reelected deleted preset", e)
	}
	req.RequestID = contracts.ID()
	if r := a.Dispatch(req); r.Error == nil || r.Error.Code != "not_found" {
		t.Fatal("deleted preset accepted for new work", r.Error)
	}
}
func TestImportPresetFrozenChildKeepsSuppliedTextAndDestination(t *testing.T) {
	a := configuredApp(t)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	os.WriteFile(mapped, []byte("fixture decoded audio"), 0600)
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{"clock": "fixture"}, Provenance: map[string]any{"adapter": "fixture"}, Diagnostics: []any{}, Close: func() error { return nil }, Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
				t.Error("supplied transcript unexpectedly recognized")
				return processing.RecognitionResult{}, contracts.Fail("operation_failed")
			}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				return processing.DiarizationResult{Turns: []processing.Turn{{Label: "voice", StartUS: 0, EndUS: 500000}}, Provenance: map[string]any{"adapter": "fixture"}}, nil
			}}, nil
		}}, nil
	}
	media := importRecordingFixture(t, a)
	pipeline := contracts.ID()
	configuredResult(t, realRequest(a, "pipelines.set", pipeline, map[string]any{"expected_revision": 0, "pipeline": map[string]any{"id": pipeline, "name": "Frozen", "preset": "local", "configuration": localDefinition()}}))
	profile := workspace.Profile{ID: contracts.ID(), Revision: 1, Adapter: "filesystem", Version: contracts.Version, Configuration: map[string]any{"root": filepath.Join(t.TempDir(), "private-artifacts")}}
	p := savedImportPreset(t, a, map[string]any{"pipeline_id": pipeline, "destination": profile})
	selected, e := a.freezeImportSelection(library.Options{}, library.Options{Preset: p.ID}, library.Options{}, true)
	if e != nil {
		t.Fatal(e)
	}
	parent, e := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.assemble", json.RawMessage(`{}`))
	if e != nil {
		t.Fatal(e)
	}
	claim, e := a.Catalog.ClaimWork(a.ctx, parent.ID, a.Session, leaseTTL)
	if e != nil {
		t.Fatal(e)
	}
	ids, e := a.queueImportProcessing(a.ctx, claim, 0, media, selected)
	if e != nil || len(ids) != 1 {
		t.Fatal(ids, e)
	}
	done := awaitWork(t, a, ids[0])
	if done.State != "succeeded" {
		t.Fatalf("child: %s", done.Result)
	}
	record, e := a.Catalog.Recording(a.ctx, media)
	if e != nil {
		t.Fatal(e)
	}
	publication, e := a.Catalog.Publication(a.ctx, *record.MappedAudioPublicationID)
	if e != nil || publication.ProfileID != profile.ID {
		t.Fatal("processing destination changed", publication, e)
	}
	if _, e = a.Catalog.MutateImportPreset(a.ctx, contracts.ID(), "delete", p.Revision, catalog.ImportPreset{ID: p.ID}); e != nil {
		t.Fatal(e)
	}
	again, e := a.queueImportProcessing(a.ctx, claim, 0, media, selected)
	if e != nil || len(again) != 1 || again[0] != ids[0] {
		t.Fatal("child duplicated after edit/delete", again, e)
	}
	a.Catalog.CancelWork(a.ctx, contracts.ID(), parent.ID)
	if _, e = a.queueImportProcessing(a.ctx, claim, 1, media, selected); e == nil {
		t.Fatal("cancelled parent created child")
	}
}

func TestImportPresetAmbiguityFailsBeforeQueue(t *testing.T) {
	a := configuredApp(t)
	p := savedImportPreset(t, a, map[string]any{"timezone": "UTC"})
	collision := catalog.ImportPreset{Kind: "import-preset", Version: contracts.Version, ID: contracts.ID(), Name: p.ID, Options: configurationBytes(map[string]any{"timezone": "UTC"})}
	if _, err := a.Catalog.MutateImportPreset(a.ctx, contracts.ID(), "create", 0, collision); err != nil {
		t.Fatal(err)
	}
	before, _ := a.Catalog.Works(a.ctx)
	response := realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: "fixture", Options: library.Options{Preset: p.ID}}}})
	after, _ := a.Catalog.Works(a.ctx)
	if response.Error == nil || response.Error.Code != "conflict" || len(before) != len(after) {
		t.Fatal("ambiguous election queued work", response.Error)
	}
}
func TestImportExistingTitlePreservesSuppliedTranscript(t *testing.T) {
	a := configuredApp(t)
	id := importRecordingFixture(t, a)
	entry, err := a.Catalog.Library(a.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !a.importHasTranscript(entry.Title) {
		t.Fatal("exact-title target lost supplied transcript")
	}
}

type presetReadCatalog struct {
	catalog.Catalog
	reads, registrations int
	importError          error
}

func (c *presetReadCatalog) ImportPresets(ctx context.Context) ([]catalog.ImportPreset, error) {
	c.reads++
	if c.importError != nil {
		return nil, c.importError
	}
	return c.Catalog.ImportPresets(ctx)
}
func (c *presetReadCatalog) RegisterWorkspace(ctx context.Context, w *workspace.Workspace) error {
	c.registrations++
	return c.Catalog.RegisterWorkspace(ctx, w)
}
func TestImportIdenticalElectionsResolveOnce(t *testing.T) {
	a := configuredApp(t)
	p := savedImportPreset(t, a, map[string]any{"timezone": "UTC"})
	counter := &presetReadCatalog{Catalog: a.Catalog}
	a.Catalog = counter
	input := library.ImportRequest{Defaults: library.Options{Preset: p.ID}}
	for i := 0; i < 50; i++ {
		input.Items = append(input.Items, library.Item{Source: "fixture"})
	}
	resolved, err := a.resolveImportChoices(input)
	if err != nil {
		t.Fatal(err)
	}
	if counter.reads != 1 || counter.registrations != 0 || len(resolved.Elections) != 1 || len(resolved.Selections) != 50 {
		t.Fatalf("repeated election resolution: reads=%d registrations=%d elections=%d", counter.reads, counter.registrations, len(resolved.Elections))
	}
	counter.importError = contracts.Fail("incompatible_version")
	before, _ := a.Catalog.Works(a.ctx)
	response := realRequest(a, "media.import", "", input)
	after, _ := a.Catalog.Works(a.ctx)
	if response.Error == nil || response.Error.Code != "incompatible_version" || response.Error.Message == contracts.Fail("incompatible_version").Message || len(before) != len(after) {
		t.Fatal("incompatible selection queued or lacks next action", response.Error)
	}
}
