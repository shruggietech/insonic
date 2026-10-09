// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func dependencyManifest(t *testing.T, version string, handler http.Handler) models.Manifest {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	body := []byte("synthetic model bytes")
	m := models.Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: "fixture-recognizer", ModelVersion: version, Revision: "immutable-fixture-" + version, License: "unknown", Capabilities: []string{"transcription"}}
	for _, role := range []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"} {
		m.Files = append(m.Files, models.File{Role: role, SHA256: documentHash(body), Size: int64(len(body)), URL: server.URL + "/" + role, LocalHTTP: true})
	}
	return m
}

func TestDependencyGateLeavesFourConsumersOutsideWorkerCapacity(t *testing.T) {
	a := configuredApp(t)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var requests atomic.Int32
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
			w.Write([]byte("synthetic model bytes"))
		case <-r.Context().Done():
		}
	}))
	defer close(release)
	selection, err := models.ResolveManifest(m, "transcription")
	if err != nil {
		t.Fatal(err)
	}
	acquisition, err := a.enqueueModelAcquisition(a.ctx, selection, false)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		raw, _ := json.Marshal(recordingPayload{ModelSelections: []models.Resolution{selection}, ModelDependencies: []string{acquisition.ID}, MediaID: contracts.ID(), Options: RecordingOptions{Transcription: "generate", RecognitionModelID: selection.Target.ID, Diarization: "reuse"}})
		if _, err = a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.process", raw); err != nil {
			t.Fatal(err)
		}
	}
	a.recoverWork()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("waiting consumers starved acquisition")
	}
	a.workMu.Lock()
	for _, worker := range a.workers {
		if worker.claim.Kind != "models.acquire" {
			t.Error("waiting consumer occupied worker capacity")
		}
	}
	a.workMu.Unlock()
	if requests.Load() != 1 {
		t.Fatal("unexpected concurrent download", requests.Load())
	}
}

func TestSharedAcquisitionAndParentRetryRetainFrozenManifest(t *testing.T) {
	a := configuredApp(t)
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("synthetic model bytes")) }))
	selection, err := models.ResolveManifest(m, "transcription")
	if err != nil {
		t.Fatal(err)
	}
	first, err := a.enqueueModelAcquisition(a.ctx, selection, false)
	if err != nil {
		t.Fatal(err)
	}
	second, err := a.enqueueModelAcquisition(a.ctx, selection, false)
	if err != nil || first.ID != second.ID {
		t.Fatal("same immutable bundle was not shared", err)
	}
	if _, err = a.Catalog.CancelWork(a.ctx, contracts.ID(), first.ID); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(recordingPayload{ModelSelections: []models.Resolution{selection}, ModelDependencies: []string{first.ID}, MediaID: contracts.ID()})
	parent, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "recordings.process", payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Catalog.CancelWork(a.ctx, contracts.ID(), parent.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Catalog.RetryWork(a.ctx, contracts.ID(), parent.ID); err != nil {
		t.Fatal(err)
	}
	recovered, err := a.Catalog.Work(context.Background(), first.ID)
	if err != nil || recovered.State != "pending" {
		t.Fatal("frozen dependency not repaired", err, recovered.State)
	}
	var request models.Request
	if json.Unmarshal(recovered.Payload, &request) != nil || request.Manifest.Digest() != m.Digest() {
		t.Fatal("retry changed selected manifest")
	}
}

func TestExplicitAcquisitionRequestsShareBytesAndCancelIndependently(t *testing.T) {
	a := configuredApp(t)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
			w.Write([]byte("synthetic model bytes"))
		case <-r.Context().Done():
		}
	}))
	input, _ := json.Marshal(models.Request{Manifest: m})
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "models.acquire", Data: input}
	first := configuredResult(t, a.Dispatch(request))
	originalRequest := request
	request.RequestID = contracts.ID()
	second := configuredResult(t, a.Dispatch(request))
	firstIDs := first["acquisition_ids"].([]any)
	secondIDs := second["acquisition_ids"].([]any)
	if first["work_id"] == second["work_id"] || len(firstIDs) != 1 || firstIDs[0] != secondIDs[0] {
		t.Fatal("independent requests did not share acquisition")
	}
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("download never started")
	}
	changed := m
	changed.ModelVersion = "changed"
	originalRequest.Data, _ = json.Marshal(models.Request{Manifest: changed})
	if response := a.Dispatch(originalRequest); response.Error == nil || response.Error.Code != "conflict" {
		t.Fatal("request identity lost immutable intent", response.Error)
	}
	response := realRequest(a, "work.cancel", first["work_id"].(string), nil)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	dependency, err := a.Catalog.Work(a.ctx, firstIDs[0].(string))
	if err != nil || dependency.State == "cancelled" {
		t.Fatal("one consumer cancelled shared acquisition", err)
	}
	close(release)
	released = true
	done := awaitWork(t, a, second["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatal("second consumer did not finish", string(done.Result))
	}
	cancelled, err := a.Catalog.Work(a.ctx, first["work_id"].(string))
	if err != nil || cancelled.State != "cancelled" {
		t.Fatal("cancelled parent revived", err)
	}
}

func registerDependencyManifest(t *testing.T, a *App, m models.Manifest) {
	t.Helper()
	out := configuredResult(t, realRequest(a, "models.register", "", models.Request{Manifest: m}))
	if work := awaitWork(t, a, out["work_id"].(string)); work.State != "succeeded" {
		t.Fatal("registration failed", string(work.Result))
	}
}

func TestProcessingAliasRetargetRestartAndRetryPreserveElection(t *testing.T) {
	a := configuredApp(t)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	released := false
	defer func() {
		if !released {
			close(release)
		}
	}()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
			w.Write([]byte("synthetic model bytes"))
		case <-r.Context().Done():
		}
	})
	first := dependencyManifest(t, "1", handler)
	first.Capabilities = append(first.Capabilities, "diarization")
	for _, role := range []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"} {
		f := first.Files[0]
		f.Role = role
		f.URL = first.Files[0].URL
		first.Files = append(first.Files, f)
	}
	second := first
	second.ModelVersion = "2"
	second.Revision = "immutable-fixture-2"
	registerDependencyManifest(t, a, first)
	registerDependencyManifest(t, a, second)
	target := catalog.ModelTarget{Kind: "base", ID: models.InstallationID(first), Operation: "transcription"}
	alias := catalog.ModelAlias{ID: contracts.ID(), Name: "speech", State: "active", Target: encodedModelTarget(target)}
	saved, err := a.Catalog.PutModelAlias(a.ctx, contracts.ID(), 0, alias)
	if err != nil {
		t.Fatal(err)
	}
	media := importRecordingFixture(t, a)
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	if err = os.WriteFile(mapped, []byte("synthetic mapped audio"), 0600); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	fixture := func() (*recordingExecution, error) {
		return &recordingExecution{Prepare: func(ctx context.Context, entry catalog.LibraryEntry, o processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{"clock": "fixture"}, Provenance: map[string]any{"adapter": "deterministic"}, Diagnostics: []any{}, Close: func() error { return nil }, Recognize: func(ctx context.Context, id string, o processing.RecognitionOptions) (processing.RecognitionResult, error) {
				if id != models.InstallationID(first) || o.ExpectedModelDigest != first.Digest() {
					t.Error("recognizer received changed model election")
				}
				calls.Add(1)
				return processing.RecognitionResult{NoSpeech: true, Provenance: map[string]any{"adapter": "deterministic"}}, nil
			}, Diarize: func(ctx context.Context, id string, o processing.DiarizationOptions) (processing.DiarizationResult, error) {
				if id != models.InstallationID(first) || o.ExpectedModelDigest != first.Digest() {
					t.Error("diarizer received changed model election")
				}
				calls.Add(1)
				return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{"adapter": "deterministic"}}, nil
			}}, nil
		}}, nil
	}
	a.recordingFactory = fixture
	a.enforceModelElection = true
	options := RecordingOptions{Transcription: "generate", RecognitionModelID: "speech", Diarization: "run", DiarizationModelID: models.InstallationID(first)}
	input, _ := json.Marshal(options)
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "recordings.process", ItemID: media, Data: input}
	queued := configuredResult(t, a.Dispatch(request))
	id := queued["work_id"].(string)
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("automatic acquisition did not begin")
	}
	if calls.Load() != 0 {
		t.Fatal("engine ran before acquisition")
	}
	target.ID = models.InstallationID(second)
	alias.Target = encodedModelTarget(target)
	if _, err = a.Catalog.PutModelAlias(a.ctx, contracts.ID(), saved.Revision, alias); err != nil {
		t.Fatal(err)
	}
	if response := realRequest(a, "work.cancel", id, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	a.Close()
	next, err := New(a.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(next.Close)
	next.recordingFactory = fixture
	next.enforceModelElection = true
	if response := realRequest(next, "work.retry", id, nil); response.Error != nil {
		t.Fatal(response.Error)
	}
	replay := configuredResult(t, next.Dispatch(request))
	if replay["work_id"] != id {
		t.Fatal("replay chose a new election")
	}
	work, err := next.Catalog.Work(next.ctx, id)
	var payload recordingPayload
	if err != nil || json.Unmarshal(work.Payload, &payload) != nil || payload.Options.RecognitionModelID != models.InstallationID(first) || len(payload.ModelDependencies) != 1 {
		t.Fatal("retarget altered durable selection", err)
	}
	close(release)
	released = true
	done := awaitWork(t, next, id)
	if done.State != "succeeded" || calls.Load() != 2 {
		t.Fatal("frozen processing failed", string(done.Result), calls.Load())
	}
	other, err := next.Catalog.BaseModel(next.ctx, models.InstallationID(second))
	if err != nil || other.State != "registered" {
		t.Fatal("retry acquired retargeted version", err)
	}
}

func TestImportFreezesModelReferencesAndRejectsClientAuthority(t *testing.T) {
	a := configuredApp(t)
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("synthetic model bytes")) }))
	url := m.Files[0].URL
	m.Capabilities = []string{"diarization"}
	m.Files = nil
	for _, role := range []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"} {
		m.Files = append(m.Files, models.File{Role: role, URL: url, SHA256: documentHash([]byte("synthetic model bytes")), Size: int64(len("synthetic model bytes")), LocalHTTP: true})
	}
	registerDependencyManifest(t, a, m)
	alias := catalog.ModelAlias{ID: contracts.ID(), Name: "voices", State: "active", Target: encodedModelTarget(catalog.ModelTarget{Kind: "base", ID: models.InstallationID(m), Operation: "diarization"})}
	if _, err := a.Catalog.PutModelAlias(a.ctx, contracts.ID(), 0, alias); err != nil {
		t.Fatal(err)
	}
	prepared, err := a.freezeImportModels(library.ImportRequest{Defaults: library.Options{Attribution: "diarize", DiarizationModelID: "voices"}, Items: []library.Item{{Source: "audio.wav"}, {Source: "other.wav"}}})
	if err != nil || len(prepared.ModelSelections) != 1 || len(prepared.ModelDependencies) != 1 {
		t.Fatal("import selection failed", err)
	}
	for _, item := range prepared.Items {
		if item.DiarizationModelID != models.InstallationID(m) || item.DiarizationModelDigest != m.Digest() {
			t.Fatal("import did not freeze exact model")
		}
	}
	response := realRequest(a, "media.import", "", prepared)
	if response.Error == nil || response.Error.Code != "invalid_request" {
		t.Fatal("client supplied internal model election", response.Error)
	}
}

func TestImportPinnedDiarizationVerifiesDigestBeforeEngine(t *testing.T) {
	if os.Getenv("INSONIC_LIBRARY_TOOLS_FILE") == "" {
		t.Skip("native media tool configuration not selected")
	}
	a := configuredApp(t)
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("synthetic model bytes")) }))
	url := m.Files[0].URL
	m.Capabilities = []string{"diarization"}
	m.Files = nil
	for _, role := range []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"} {
		m.Files = append(m.Files, models.File{Role: role, URL: url, SHA256: documentHash([]byte("synthetic model bytes")), Size: int64(len("synthetic model bytes")), LocalHTTP: true})
	}
	out := configuredResult(t, realRequest(a, "models.acquire", "", models.Request{Manifest: m}))
	if done := awaitWork(t, a, out["work_id"].(string)); done.State != "succeeded" {
		t.Fatal("synthetic model acquisition failed", string(done.Result))
	}
	media := importRecordingFixture(t, a)
	entry, err := a.Catalog.Library(a.ctx, media)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(ctx context.Context, entry catalog.LibraryEntry, o processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{SourceMap: processing.SourceMap{}, Provenance: map[string]any{"adapter": "deterministic"}, Diagnostics: []any{}, Close: func() error { return nil }, Diarize: func(ctx context.Context, id string, o processing.DiarizationOptions) (processing.DiarizationResult, error) {
				calls.Add(1)
				if id != models.InstallationID(m) || o.ExpectedModelDigest != m.Digest() {
					t.Error("import consumer lost elected identity")
				}
				return processing.DiarizationResult{NoSpeech: true, Turns: []processing.Turn{}, Provenance: map[string]any{"adapter": "deterministic"}}, nil
			}}, nil
		}}, nil
	}
	service, err := a.libraryService()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = service.DiarizePinned(a.ctx, entry, doc, models.InstallationID(m), documentHash([]byte("other"))); err == nil || calls.Load() != 0 {
		t.Fatal("changed model reached import engine", err)
	}
	admission, _, err := service.DiarizePinned(a.ctx, entry, doc, models.InstallationID(m), m.Digest())
	if err != nil || calls.Load() != 1 || subtitles.ValidateDocument(admission.Document) != nil {
		t.Fatal("frozen import consumer failed", err)
	}
}

func TestFailedDependencyReportsAcquisitionAndRetryRepairsExactBundle(t *testing.T) {
	a := configuredApp(t)
	var fixed atomic.Bool
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fixed.Load() {
			w.Write([]byte("synthetic model bytes"))
		} else {
			w.Write([]byte("incorrect model bytes"))
		}
	}))
	first := configuredResult(t, realRequest(a, "models.acquire", "", models.Request{Manifest: m}))
	id := first["work_id"].(string)
	done := awaitWork(t, a, id)
	var result map[string]any
	if done.State != "failed" || json.Unmarshal(done.Result, &result) != nil || result["error"] != "model_acquisition_failed" {
		t.Fatal("failed acquisition misreported as processing", string(done.Result))
	}
	var frozen recordingPayload
	if json.Unmarshal(done.Payload, &frozen) != nil || len(frozen.ModelDependencies) != 1 {
		t.Fatal("missing dependency election")
	}
	fixed.Store(true)
	retryRequest := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "work.retry", ItemID: id}
	if response := a.Dispatch(retryRequest); response.Error != nil {
		t.Fatal(response.Error)
	}
	done = awaitWork(t, a, id)
	if done.State != "succeeded" {
		t.Fatal("dependency retry failed", string(done.Result))
	}
	dependency, err := a.Catalog.Work(a.ctx, frozen.ModelDependencies[0])
	var request models.Request
	if err != nil || dependency.Generation < 2 || json.Unmarshal(dependency.Payload, &request) != nil || request.Manifest.Digest() != m.Digest() {
		t.Fatal("dependency retry changed immutable bundle", err)
	}
	if response := a.Dispatch(retryRequest); response.Error != nil {
		t.Fatal("completed retry request did not replay", response.Error)
	}
	replayed, err := a.Catalog.Work(a.ctx, dependency.ID)
	if err != nil || replayed.Generation != dependency.Generation || replayed.State != "succeeded" {
		t.Fatal("retry replay restarted acquisition", err)
	}
	snapshot, err := a.Catalog.Export(a.ctx)
	if err != nil {
		t.Fatal("atomic dependency retry lost portable proofs", err)
	}
	destination, err := catalog.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "restored.sqlite"), a.Workspace.Config.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	defer destination.Close()
	if err = destination.Restore(context.Background(), snapshot); err != nil {
		t.Fatal("atomic retry proofs did not restore", err)
	}
}

func TestAvailableModelWithMissingBytesIsReacquiredBeforeSuccess(t *testing.T) {
	a := configuredApp(t)
	var downloads atomic.Int32
	m := dependencyManifest(t, "1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		w.Write([]byte("synthetic model bytes"))
	}))
	first := configuredResult(t, realRequest(a, "models.acquire", "", models.Request{Manifest: m}))
	if work := awaitWork(t, a, first["work_id"].(string)); work.State != "succeeded" {
		t.Fatal(string(work.Result))
	}
	install, err := a.Catalog.BaseModel(a.ctx, models.InstallationID(m))
	if err != nil {
		t.Fatal(err)
	}
	ids, err := catalog.PublicationIDs(install.PublicationIDs)
	if err != nil {
		t.Fatal(err)
	}
	publication, err := a.Catalog.Publication(a.ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	storage := a.Workspace.Config.Profiles.Storage.Configuration["root"].(string)
	if !filepath.IsAbs(storage) {
		storage = filepath.Join(a.Workspace.Control, storage)
	}
	if err = os.Remove(filepath.Join(storage, filepath.FromSlash(publication.Key))); err != nil {
		t.Fatal(err)
	}
	before := downloads.Load()
	second := configuredResult(t, realRequest(a, "models.acquire", "", models.Request{Manifest: m}))
	if first["acquisition_ids"].([]any)[0] != second["acquisition_ids"].([]any)[0] {
		t.Fatal("repair changed acquisition identity")
	}
	if work := awaitWork(t, a, second["work_id"].(string)); work.State != "succeeded" {
		t.Fatal("missing bytes were not repaired", string(work.Result))
	}
	service, err := a.modelService()
	if err != nil {
		t.Fatal(err)
	}
	if err = service.Verify(a.ctx, models.InstallationID(m)); err != nil || downloads.Load() <= before {
		t.Fatal("repair did not verify newly acquired bytes", err)
	}
}
