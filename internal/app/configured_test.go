// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func configuredApp(t *testing.T) *App {
	t.Helper()
	w, e := workspace.Init(t.TempDir(), "configured pipelines")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(a.Close)
	return a
}

func TestElectedWorkFreezesContextModelsAndReplay(t *testing.T) {
	a := configuredApp(t)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	if os.WriteFile(mapped, []byte("deterministic mapped fixture"), 0600) != nil {
		t.Fatal("mapped fixture")
	}
	entered, release := make(chan processing.RecognitionOptions, 1), make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{"clock": "fixture"}, Provenance: map[string]any{"adapter": "fixture"}, Diagnostics: []any{}, Close: func() error { return nil },
				Recognize: func(ctx context.Context, _ string, o processing.RecognitionOptions) (processing.RecognitionResult, error) {
					entered <- o
					select {
					case <-release:
					case <-ctx.Done():
						return processing.RecognitionResult{}, contracts.Fail("cancelled")
					}
					return processing.RecognitionResult{SRT: []byte("1\n00:00:00,000 --> 00:00:01,000\nGenerated fixture\n\n"), Provenance: map[string]any{"adapter": "fixture"}}, nil
				},
				Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
					return processing.DiarizationResult{Turns: []processing.Turn{{Label: "voice", StartUS: 0, EndUS: 500000}}, Provenance: map[string]any{"adapter": "fixture"}}, nil
				}}, nil
		}}, nil
	}
	recording := importRecordingFixture(t, a)
	pipelineID, termID := contracts.ID(), contracts.ID()
	definition := localDefinition()
	definition["recognition"].(map[string]any)["recognition"] = map[string]any{"device": "cpu"}
	p := map[string]any{"id": pipelineID, "name": "Frozen", "preset": "local", "configuration": definition}
	saved := configuredResult(t, realRequest(a, "pipelines.set", pipelineID, map[string]any{"expected_revision": 0, "pipeline": p}))
	term := map[string]any{"id": termID, "canonical": "originalterm", "variants": []string{}, "state": "active"}
	savedTerm := configuredResult(t, realRequest(a, "terms.set", termID, map[string]any{"expected_revision": 0, "term": term}))
	raw, _ := json.Marshal(map[string]any{"pipeline_id": pipelineID, "transcription": "generate", "diarization": "run"})
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "recordings.process", ItemID: recording, Data: raw}
	queued := configuredResult(t, a.Dispatch(request))
	workID := queued["work_id"].(string)
	var used processing.RecognitionOptions
	select {
	case used = <-entered:
	case <-time.After(8 * time.Second):
		t.Fatal("elected stage not entered")
	}
	definition["recognition"].(map[string]any)["recognition"] = map[string]any{"device": "cuda"}
	definition["recognition"].(map[string]any)["model_id"] = contracts.ID()
	configuredResult(t, realRequest(a, "pipelines.set", pipelineID, map[string]any{"expected_revision": saved["revision"], "pipeline": p}))
	term["canonical"] = "newterm"
	configuredResult(t, realRequest(a, "terms.set", termID, map[string]any{"expected_revision": savedTerm["revision"], "term": term}))
	if used.Device != "cpu" || len(used.Hints) != 1 || used.Hints[0] != "originalterm" {
		t.Fatal("elected settings/context changed")
	}
	close(release)
	done := awaitWork(t, a, workID)
	if done.State != "succeeded" {
		t.Fatalf("frozen processing failed: %s", done.Result)
	}
	replay := configuredResult(t, a.Dispatch(request))
	if replay["work_id"] != workID {
		t.Fatal("replay replaced original election")
	}
	record, e := a.Catalog.Recording(context.Background(), recording)
	if e != nil {
		t.Fatal(e)
	}
	var provenance map[string]json.RawMessage
	if json.Unmarshal(record.Provenance, &provenance) != nil || len(provenance["pipeline_election"]) == 0 {
		t.Fatal("election provenance missing")
	}
}

func TestProcessingToolsElectionDoesNotFollowLaterControlChanges(t *testing.T) {
	a := configuredApp(t)
	executable := filepath.Join(t.TempDir(), "pinned-tool")
	bytes := []byte("synthetic pinned executable, never run")
	if os.WriteFile(executable, bytes, 0600) != nil {
		t.Fatal("tool fixture")
	}
	first := ProcessingTools{Kind: "processing-tools", Version: contracts.Version, Cueson: subtitles.Tool{Executable: executable, ExecutableSHA256: documentHash(bytes)}}
	first.Processing.Threads = 1
	control := filepath.Join(a.Workspace.Control, "processing-tools.json")
	if os.WriteFile(control, configurationBytes(map[string]any{"kind": first.Kind, "schema_version": first.Version, "cueson": first.Cueson, "processing": map[string]int{"threads": 1}}), 0600) != nil {
		t.Fatal("config fixture")
	}
	elected, e := a.electedProcessingTools()
	if e != nil {
		t.Fatal(e)
	}
	second := first
	second.Processing.Threads = 8
	if os.WriteFile(control, configurationBytes(map[string]any{"kind": second.Kind, "schema_version": second.Version, "cueson": second.Cueson, "processing": map[string]int{"threads": 8}}), 0600) != nil {
		t.Fatal("config replacement")
	}
	if elected.Processing.Threads != 1 {
		t.Fatal("elected tool settings mutated")
	}
	later, e := a.electedProcessingTools()
	if e != nil || later.Processing.Threads != 8 {
		t.Fatal("future election not updated", e)
	}
	if _, e = a.recordingExecutorWithTools(elected); e != nil {
		t.Fatal("frozen execution selection rejected", e)
	}
}
func configuredResult(t *testing.T, r contracts.Response) map[string]any {
	t.Helper()
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	raw, e := json.Marshal(r.Result)
	if e != nil {
		t.Fatal(e)
	}
	var value map[string]any
	if json.Unmarshal(raw, &value) != nil {
		t.Fatal("invalid response shape")
	}
	return value
}

func configuredContractResult(t *testing.T, response contracts.Response) map[string]any {
	t.Helper()
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if err = schemas.ValidateResponse(raw); err != nil {
		t.Fatalf("runtime response violates its published schema: %v; %s", err, raw)
	}
	return configuredResult(t, response)
}

func TestPipelineInspectUnresolvedModelDiagnostics(t *testing.T) {
	for _, reference := range []string{contracts.ID(), "missing-alias", "source:missing-source/default"} {
		t.Run(reference, func(t *testing.T) {
			a := configuredApp(t)
			id := contracts.ID()
			definition := localDefinition()
			definition["recognition"].(map[string]any)["model_id"] = reference
			p := map[string]any{"id": id, "name": "Missing selection", "preset": "local", "configuration": definition}
			configuredResult(t, realRequest(a, "pipelines.set", id, map[string]any{"expected_revision": 0, "pipeline": p}))
			result := configuredContractResult(t, realRequest(a, "pipelines.inspect", id, nil))
			selections := result["model_selections"].([]any)
			selected := selections[0].(map[string]any)
			target := selected["target"].(map[string]any)
			if selected["reference"] != reference || selected["state"] != "missing" || selected["compatible"] != false || selected["manifest_digest"] != "" || selected["upstream_revision"] != "" || target["kind"] != "unresolved" || target["operation"] != "transcription" || len(target) != 2 {
				t.Fatalf("missing reference invented a resolution: %#v", selected)
			}
			if len(selected["diagnostics"].([]any)) == 0 {
				t.Fatal("missing reference lacks recovery diagnostics")
			}
			work, err := a.Catalog.Works(context.Background())
			if err != nil || len(work) != 0 {
				t.Fatal("inspection queued acquisition", err)
			}
		})
	}
}
func localDefinition() map[string]any {
	return map[string]any{
		"recognition": map[string]any{"adapter": "faster-whisper", "contract_version": "1", "mode": "local", "model_id": contracts.ID()},
		"diarization": map[string]any{"adapter": "pyannote", "contract_version": "1", "mode": "local", "model_id": contracts.ID()},
	}
}
func TestConfiguredPipelineCRUDInspectCASAndStrictRequests(t *testing.T) {
	a := configuredApp(t)
	id := contracts.ID()
	input := map[string]any{"expected_revision": 0, "pipeline": map[string]any{"id": id, "name": "Local fixture", "preset": "local", "configuration": localDefinition()}}
	first := configuredResult(t, realRequest(a, "pipelines.set", id, input))
	revision, ok := first["revision"].(float64)
	if !ok || revision < 1 {
		t.Fatal("missing pipeline revision")
	}
	got := configuredResult(t, realRequest(a, "pipelines.show", id, nil))
	if got["id"] != id {
		t.Fatal("different pipeline identity")
	}
	configuredResult(t, realRequest(a, "pipelines.inspect", id, nil))
	configuredResult(t, realRequest(a, "pipelines.list", "", nil))
	if r := realRequest(a, "pipelines.set", id, input); r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("stale create accepted")
	}
	invalid := map[string]any{"expected_revision": revision, "pipeline": input["pipeline"], "unexpected": true}
	if r := realRequest(a, "pipelines.set", id, invalid); r.Error == nil {
		t.Fatal("unknown mutation field accepted")
	}
	invalid = map[string]any{"expected_revision": revision, "pipeline": map[string]any{"id": id, "name": "bad", "preset": "local", "configuration": map[string]any{"api_key": "synthetic-secret"}}}
	if r := realRequest(a, "pipelines.set", id, invalid); r.Error == nil {
		t.Fatal("sensitive pipeline field accepted")
	}
}

func TestConfiguredSpeakerTermsCompileAndAliasCAS(t *testing.T) {
	a := configuredApp(t)
	speakerID, aliasID, termID := contracts.ID(), contracts.ID(), contracts.ID()
	identity := map[string]any{"expected_revision": 0, "speaker": map[string]any{"id": speakerID, "name": "Jane", "state": "active"}, "aliases": []any{map[string]any{"id": aliasID, "speaker_id": speakerID, "text": "Janie", "language": "en", "state": "active"}}}
	configuredResult(t, realRequest(a, "speakers.set", speakerID, identity))
	show := configuredResult(t, realRequest(a, "speakers.show", speakerID, nil))
	if _, ok := show["speaker"]; !ok {
		t.Fatal("speaker identity missing")
	}
	configuredResult(t, realRequest(a, "speakers.aliases", speakerID, nil))
	term := map[string]any{"expected_revision": 0, "term": map[string]any{"id": termID, "canonical": "spectrogram", "variants": []string{"spectrogram", "spectrograms"}, "language": "en", "state": "active", "speaker_id": speakerID}}
	configuredResult(t, realRequest(a, "terms.set", termID, term))
	configuredResult(t, realRequest(a, "terms.show", termID, nil))
	compiled := configuredResult(t, realRequest(a, "terms.compile", "", map[string]any{"language": "en", "speaker_ids": []string{speakerID}, "max_hint_bytes": 200}))
	again := configuredResult(t, realRequest(a, "terms.compile", "", map[string]any{"language": "en", "speaker_ids": []string{speakerID}, "max_hint_bytes": 200}))
	if compiled["digest"] == "" || compiled["digest"] != again["digest"] {
		t.Fatal("context digest is not deterministic")
	}
	if r := realRequest(a, "terms.set", termID, term); r.Error == nil || r.Error.Code != "conflict" {
		t.Fatal("stale term accepted")
	}
	if r := realRequest(a, "speakers.aliases", speakerID, map[string]any{"expected_revision": 0, "aliases": []any{}}); r.Error == nil {
		t.Fatal("unfenced aliases accepted")
	}
}
