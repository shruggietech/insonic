// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"github.com/shruggietech/insonic/internal/workspace"
)

func portableFixtureApp(t *testing.T, w *workspace.Workspace, store catalog.Catalog) *App {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	a := &App{Workspace: w, Catalog: store, Session: contracts.ID(), ctx: ctx, cancel: cancel, workers: map[string]*realWorker{}, attempts: map[string]*worker{}}
	t.Cleanup(a.Close)
	return a
}
func portableWorkFixture(t *testing.T) *App {
	t.Helper()
	w, e := workspace.Init(t.TempDir(), "portable work")
	if e != nil {
		t.Fatal(e)
	}
	s, e := catalog.OpenSQLite(context.Background(), filepath.Join(w.Control, "source.sqlite"), w.Config.WorkspaceID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.RegisterWorkspace(context.Background(), w); e != nil {
		t.Fatal(e)
	}
	return portableFixtureApp(t, w, s)
}
func portableWorkTarget(t *testing.T, source *App) *App {
	t.Helper()
	snap, e := source.Catalog.Export(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	snap, e = catalog.PortableSnapshot(context.Background(), snap)
	if e != nil {
		t.Fatal(e)
	}
	s, e := catalog.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "target.sqlite"), snap.WorkspaceID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.Restore(context.Background(), snap); e != nil {
		t.Fatal(e)
	}
	return portableFixtureApp(t, source.Workspace, s)
}
func portableDestinationConfig(t *testing.T, a *App) ProcessingTools {
	t.Helper()
	path := filepath.Join(t.TempDir(), "destination-cueson")
	body := []byte("destination platform executable fixture")
	if e := os.WriteFile(path, body, 0600); e != nil {
		t.Fatal(e)
	}
	return ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: path, ExecutableSHA256: documentHash(body)}, Processing: processing.Config{Threads: 3}}
}
func savePortableConfig(t *testing.T, a *App, c ProcessingTools) {
	t.Helper()
	raw, _ := json.Marshal(c)
	if e := os.WriteFile(filepath.Join(a.Workspace.Control, "processing-tools.json"), raw, 0600); e != nil {
		t.Fatal(e)
	}
}

func TestPortableEnsureDoesNotRequireUnusedProcessingTools(t *testing.T) {
	source := portableWorkFixture(t)
	manifest := dependencyManifest(t, "portable", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("synthetic model bytes")) }))
	selection, e := models.ResolveManifest(manifest, "transcription")
	if e != nil {
		t.Fatal(e)
	}
	acquireRaw, _ := json.Marshal(models.Request{Manifest: manifest})
	acquire, e := source.Catalog.EnqueueWork(source.ctx, contracts.ID(), "models.acquire", acquireRaw)
	if e != nil {
		t.Fatal(e)
	}
	acquire, e = source.Catalog.ClaimWork(source.ctx, acquire.ID, source.Session, time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	service, e := source.modelService()
	if e != nil {
		t.Fatal(e)
	}
	result, e := service.Execute(source.ctx, acquire)
	if e != nil {
		t.Fatal(e)
	}
	resultRaw, _ := json.Marshal(result)
	if _, e = source.Catalog.CheckpointWork(source.ctx, acquire, "fixture", "succeeded", resultRaw, time.Minute); e != nil {
		t.Fatal(e)
	}
	old := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: `C:\source-private\cueson.exe`, ExecutableSHA256: strings.Repeat("1", 64)}, Processing: processing.Config{Threads: 7}}
	raw, _ := json.Marshal(recordingPayload{Tools: &old, ModelSelections: []models.Resolution{selection}})
	w, e := source.Catalog.EnqueueWork(source.ctx, contracts.ID(), "models.ensure", raw)
	if e != nil {
		t.Fatal(e)
	}
	target := portableWorkTarget(t, source)
	stored, _ := target.Catalog.Work(target.ctx, w.ID)
	invocation, e := target.portableInvocation(stored)
	if e != nil {
		t.Fatal(e)
	}
	var bound recordingPayload
	if strictPayload(invocation, &bound) != nil || bound.Tools != nil || len(bound.ModelSelections) != 1 || bound.ModelSelections[0].Target.ID != selection.Target.ID {
		t.Fatal("ensure retained unused processing tools or changed selected model")
	}
	target.recoverWork()
	done := awaitWork(t, target, w.ID)
	if done.State != "succeeded" {
		t.Fatal("automatic configured recovery", done.Result)
	}
	if strings.Contains(string(done.Payload), "source-private") {
		t.Fatal("invocation paths persisted in portable catalog")
	}
	snap, e := target.Catalog.Export(target.ctx)
	if e != nil || catalog.ValidateSnapshot(target.ctx, snap) != nil {
		t.Fatal("recovered claim cannot be exported", e)
	}
	unchanged, _ := source.Catalog.Work(source.ctx, w.ID)
	if unchanged.State != "pending" || string(unchanged.Payload) != string(raw) {
		t.Fatal("source work changed")
	}
}

func TestPortableCustomTrainingPinsBindExactlyAndPreserveParameters(t *testing.T) {
	source := portableWorkFixture(t)
	executable := []byte("custom adapter bytes")
	support := []byte("custom support bytes")
	adapter := voicemodels.Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "fixture", OutputKinds: []string{"voice-embedding"}, Consumers: []string{}, Executable: library.PinnedFile{Path: `C:\source-private\adapter.exe`, SHA256: documentHash(executable)}, SupportFiles: []library.PinnedFile{{Path: `C:\source-private\config.json`, SHA256: documentHash(support)}}, Arguments: []string{"--config=" + `C:\source-private\config.json`, "--threads=2"}}
	old := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: `C:\source-private\cueson.exe`, ExecutableSHA256: strings.Repeat("2", 64)}}
	oldRaw, _ := json.Marshal(old)
	original := voicemodels.TrainPayload{ProcessingTools: oldRaw, Options: voicemodels.TrainOptions{DatasetID: contracts.ID(), Name: "elected", Kind: "voice-embedding", Adapter: adapter, Parameters: json.RawMessage(`{"epochs":9007199254740993,"seed":1781029324123456789}`)}}
	raw, _ := json.Marshal(original)
	w, e := source.Catalog.EnqueueWork(source.ctx, contracts.ID(), "models.train", raw)
	if e != nil {
		t.Fatal(e)
	}
	target := portableWorkTarget(t, source)
	config := portableDestinationConfig(t, target)
	savePortableConfig(t, target, config)
	stored, _ := target.Catalog.Work(target.ctx, w.ID)
	if _, e = target.portableInvocation(stored); e == nil {
		t.Fatal("unconfigured custom adapter was rebound")
	}
	for _, body := range [][]byte{executable, support} {
		path := filepath.Join(t.TempDir(), "custom-file")
		os.WriteFile(path, body, 0600)
		config.PortableFiles = append(config.PortableFiles, library.PinnedFile{Path: path, SHA256: documentHash(body)})
	}
	savePortableConfig(t, target, config)
	invocation, e := target.portableInvocation(stored)
	if e != nil {
		t.Fatal(e)
	}
	var bound voicemodels.TrainPayload
	if strictPayload(invocation, &bound) != nil {
		t.Fatal("bound training payload")
	}
	if bound.Options.Adapter.Executable != config.PortableFiles[0] || bound.Options.Adapter.SupportFiles[0] != config.PortableFiles[1] || bound.Options.Adapter.Arguments[0] != "--config="+config.PortableFiles[1].Path || bound.Options.Adapter.Arguments[1] != "--threads=2" || string(bound.Options.Parameters) != string(original.Options.Parameters) {
		t.Fatal("custom pin or elected parameter changed")
	}
	proven, origin, e := target.Catalog.PortableWorkInput(target.ctx, stored)
	adapterRaw, _ := json.Marshal(adapter)
	if e != nil || !proven || origin != documentHash(adapterRaw) {
		t.Fatal("checkpoint origin adapter proof", e, origin)
	}
	if e = os.WriteFile(config.PortableFiles[0].Path, []byte("changed bytes"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = target.portableInvocation(stored); e == nil {
		t.Fatal("mismatched custom executable admitted")
	}
}

func TestPortableSavedTrainingPipelineUsesDestinationPinsForFutureElection(t *testing.T) {
	source := portableWorkFixture(t)
	body := []byte("configured adapter")
	digest := documentHash(body)
	adapter := voicemodels.Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "fixture", OutputKinds: []string{"voice-embedding"}, Consumers: []string{}, Executable: library.PinnedFile{Path: `C:\source-private\adapter.exe`, SHA256: digest}}
	training := voicemodels.TrainingConfiguration{Adapter: adapter, Kind: "voice-embedding", Parameters: json.RawMessage(`{"steps":9007199254740993}`)}
	raw, _ := json.Marshal(map[string]any{"speaker_training": training})
	p, e := source.Catalog.PutPipeline(source.ctx, contracts.ID(), 0, catalog.Pipeline{ID: contracts.ID(), Name: "portable training", Preset: "custom", Configuration: raw})
	if e != nil {
		t.Fatal(e)
	}
	target := portableWorkTarget(t, source)
	stored, e := target.Catalog.Pipeline(target.ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if stored.Revision != p.Revision || strings.Contains(string(stored.Configuration), "source-private") {
		t.Fatal("saved pipeline revision or portable binding changed")
	}
	if _, e = target.configuredPipelineDefinition(stored); e == nil {
		t.Fatal("missing saved pipeline binding was executed")
	}
	config := portableDestinationConfig(t, target)
	path := filepath.Join(t.TempDir(), "configured-adapter")
	if e = os.WriteFile(path, body, 0600); e != nil {
		t.Fatal(e)
	}
	config.PortableFiles = []library.PinnedFile{{Path: path, SHA256: digest}}
	savePortableConfig(t, target, config)
	definition, e := target.configuredPipelineDefinition(stored)
	if e != nil {
		t.Fatal(e)
	}
	var bound voicemodels.TrainingConfiguration
	strictPayload(definition.SpeakerTraining, &bound)
	if bound.Adapter.Executable.Path != path || string(bound.Parameters) != string(training.Parameters) {
		t.Fatal("future training election changed pin or exact parameter")
	}
	service, e := target.speakerModelService()
	if e != nil {
		t.Fatal(e)
	}
	elected, e := service.ResolveTrainingPipeline(target.ctx, voicemodels.TrainOptions{PipelineID: p.ID, PipelineRevision: p.Revision})
	if e != nil || elected.Adapter.Executable.Path != path {
		t.Fatal("future training could not use restored configuration", e)
	}
	unchanged, _ := source.Catalog.Pipeline(source.ctx, p.ID)
	if string(unchanged.Configuration) != string(p.Configuration) || unchanged.Revision != p.Revision {
		t.Fatal("source pipeline mutated")
	}
}

func TestPortableFilesRejectAmbiguousMissingAndChangedPins(t *testing.T) {
	a := portableWorkFixture(t)
	c := portableDestinationConfig(t, a)
	body := []byte("custom pin")
	path := filepath.Join(t.TempDir(), "pin")
	os.WriteFile(path, body, 0600)
	good := library.PinnedFile{Path: path, SHA256: documentHash(body)}
	c.PortableFiles = []library.PinnedFile{good, good}
	if ValidateProcessingTools(c) == nil {
		t.Fatal("ambiguous binding admitted")
	}
	c.PortableFiles = []library.PinnedFile{{Path: t.TempDir(), SHA256: good.SHA256}}
	if ValidateProcessingTools(c) == nil {
		t.Fatal("directory binding admitted")
	}
	c.PortableFiles = []library.PinnedFile{{Path: path, SHA256: strings.Repeat("a", 64)}}
	if ValidateProcessingTools(c) == nil {
		t.Fatal("changed pin admitted")
	}
}

func TestPortableRecordingContextRebuildAndAutomaticProcessing(t *testing.T) {
	source := portableWorkFixture(t)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	if e = os.WriteFile(mapped, []byte("disposable mapped audio fixture"), 0600); e != nil {
		t.Fatal(e)
	}
	engine := func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{"clock": "fixture"}, Provenance: map[string]any{}, Diagnostics: []any{}, Close: func() error { return nil }, Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
				return processing.RecognitionResult{SRT: []byte("fixture"), Provenance: map[string]any{}}, nil
			}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				return processing.DiarizationResult{}, nil
			}}, nil
		}}, nil
	}
	source.recordingFactory = engine
	id := importRecordingFixture(t, source)
	entry, e := source.Catalog.Library(source.ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	options, election, e := source.electRecordingOptions(RecordingOptions{Transcription: "generate", Diarization: "run", DiarizationModelID: contracts.ID(), RecognitionModelID: contracts.ID(), Recognition: processing.RecognitionOptions{Hints: []string{"TransientSourcePrompt"}}})
	if e != nil {
		t.Fatal(e)
	}
	old := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: `C:\source-private\cueson.exe`, ExecutableSHA256: strings.Repeat("3", 64)}}
	p := recordingPayload{MediaID: id, SourceRevision: entry.Revision, Tools: &old, Options: options, Election: election}
	raw, _ := json.Marshal(p)
	w, e := source.Catalog.EnqueueWork(source.ctx, contracts.ID(), "recordings.process", raw)
	if e != nil {
		t.Fatal(e)
	}
	target := portableWorkTarget(t, source)
	target.recordingFactory = engine
	target.recoverWork()
	pending, e := target.Catalog.Work(target.ctx, w.ID)
	if e != nil || pending.State != "pending" || pending.Generation != 0 {
		t.Fatal("unconfigured processing consumed its automatic election", e)
	}
	config := portableDestinationConfig(t, target)
	savePortableConfig(t, target, config)
	stored, _ := target.Catalog.Work(target.ctx, w.ID)
	invocation, e := target.portableInvocation(stored)
	if e != nil {
		t.Fatal(e)
	}
	var bound recordingPayload
	strictPayload(invocation, &bound)
	if strings.Contains(string(invocation), "TransientSourcePrompt") || validateElection(bound) != nil || bound.Options.RecognitionModelID != options.RecognitionModelID {
		t.Fatal("context replayed transient hints or changed model election")
	}
	if bound.Tools.Cueson != config.Cueson {
		t.Fatal("standard role did not rebind destination platform identity")
	}
	target.recoverWork()
	done := awaitWork(t, target, w.ID)
	if done.State != "succeeded" {
		t.Fatal("portable recording did not recover automatically", done.Result)
	}
	accepted, e := target.Catalog.Recording(target.ctx, id)
	if e != nil || accepted.State != "ready" {
		t.Fatal("accepted recording missing", e)
	}
	// Matching reconstructs only the exact frozen accepted evidence.
	snapshot, e := target.Catalog.FreezeSpeakerMatching(target.ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	match := voicemodels.MatchPayload{ProcessingTools: json.RawMessage(`{"kind":"processing-tools","schema_version":"` + contracts.Version + `","cueson":{"executable":"C:\\source-private\\cueson.exe","executable_sha256":"` + strings.Repeat("3", 64) + `"},"processing":{}}`), Options: voicemodels.MatchOptions{RecordingID: id}, Snapshot: snapshot}
	matchRaw, _ := json.Marshal(match)
	mw, e := target.Catalog.EnqueueWork(target.ctx, contracts.ID(), "recordings.match", matchRaw)
	if e != nil {
		t.Fatal(e)
	}
	matching := portableWorkTarget(t, target)
	restored, _ := matching.Catalog.Work(matching.ctx, mw.ID)
	invoked, e := matching.portableInvocation(restored)
	if e != nil {
		t.Fatal(e)
	}
	var rebuilt voicemodels.MatchPayload
	strictPayload(invoked, &rebuilt)
	if rebuilt.Snapshot.Digest != snapshot.Digest || rebuilt.Snapshot.Recording.ID != id || len(rebuilt.Snapshot.Profiles) != len(snapshot.Profiles) {
		t.Fatal("matching evidence election changed")
	}
	if len(restored.Payload) >= len(invoked) {
		t.Fatal("matching snapshot was not excluded/rebuilt")
	}
}
