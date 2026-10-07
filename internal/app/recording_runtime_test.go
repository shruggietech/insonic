// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

type fixtureSubtitles struct{ doc json.RawMessage }

func (f fixtureSubtitles) Ingest(context.Context, []byte, string) (json.RawMessage, error) {
	return f.doc, nil
}
func (f fixtureSubtitles) Assemble(_ context.Context, b []byte, d *int64, t []subtitles.Turn, p []subtitles.Participation) (subtitles.Assembly, error) {
	return subtitles.AssembleDocument(b, d, t, p)
}
func (f fixtureSubtitles) Export(_ context.Context, b []byte, format string, strict bool) (subtitles.Export, error) {
	if format == "cueson" {
		return subtitles.Export{Bytes: b}, nil
	}
	ds := []subtitles.Diagnostic{{Code: "consumer_speaker_attribution_omitted", Message: "Native output omits consumer attributions."}}
	if strict {
		return subtitles.Export{Diagnostics: ds}, contracts.Fail("invalid_request")
	}
	return subtitles.Export{Bytes: []byte("fixture native export"), Diagnostics: ds}, nil
}
func importRecordingFixture(t *testing.T, a *App) string {
	t.Helper()
	source, _ := filepath.Abs("../../tests/fixtures/media/speech.flac")
	subtitle, _ := filepath.Abs("../../tests/fixtures/media/speech.srt")
	r := realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: source, Subtitle: subtitle, NewEntry: true}}})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	done := awaitWork(t, a, r.Result.(map[string]any)["work_id"].(string))
	var imported library.ImportResult
	if json.Unmarshal(done.Result, &imported) != nil || len(imported.Items) != 1 || imported.Items[0].MediaID == "" {
		t.Fatalf("admission %s", done.Result)
	}
	return imported.Items[0].MediaID
}
func TestRecordingRuntimeReplacementReplayAndRestart(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "recording")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	os.WriteFile(mapped, []byte("private decoded fixture"), 0600)
	var recognitions, diarizations atomic.Int32
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{"clock": "fixture"}, Provenance: map[string]any{"adapter": "fixture"}, Diagnostics: []any{},
				Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
					recognitions.Add(1)
					return processing.RecognitionResult{}, contracts.Fail("operation_failed")
				},
				Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
					diarizations.Add(1)
					return processing.DiarizationResult{Provenance: map[string]any{"adapter": "fixture"}, Turns: []processing.Turn{{Label: "voice", StartUS: 0, EndUS: 500000}}}, nil
				},
				Close: func() error { return nil }}, nil
		}}, nil
	}
	id := importRecordingFixture(t, a)
	options := RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID()}
	r := realRequest(a, "recordings.process", id, options)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	done := awaitWork(t, a, r.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatalf("processing %s", done.Result)
	}
	first, e := a.Catalog.Recording(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	if recognitions.Load() != 0 || diarizations.Load() != 1 || first.State != "ready" {
		t.Fatal("supplied policy", recognitions.Load(), diarizations.Load(), first.State)
	}
	show := realRequest(a, "media.show", id, nil)
	if show.Error != nil || show.Result.(library.EntryView).CurrentRecording == nil {
		t.Fatal("missing current embedded recording", show.Error)
	}
	source, _ := os.ReadFile("../../tests/fixtures/media/speech.flac")
	if documentHash(source) != "62a3fcd36560793a065d53b5ea749dcf710d85028c837802063247ef787e38c3" {
		t.Fatal("source modified")
	}
	options.Transcription = "reuse"
	r = realRequest(a, "recordings.process", id, options)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	done = awaitWork(t, a, r.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatalf("rerun %s", done.Result)
	}
	second, e := a.Catalog.Recording(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	if second.DocumentDigest == first.DocumentDigest || recognitions.Load() != 0 || diarizations.Load() != 2 {
		t.Fatal("independent rerun did not replace local identities")
	}
	p, e := a.Catalog.Publication(context.Background(), *first.MappedAudioPublicationID)
	if e != nil || p.State != "retired" {
		t.Fatal("old mapped audio retained", p.State, e)
	}
	denied := filepath.Join(t.TempDir(), "strict.srt")
	export := realRequest(a, "recordings.export", id, map[string]any{"format": "srt", "strict": true, "destination": denied})
	if export.Error == nil || len(export.Error.Diagnostics) != 1 || export.Error.Diagnostics[0].Code != "consumer_speaker_attribution_omitted" || export.Error.Diagnostics[0].Count != 1 {
		t.Fatal("strict loss accepted")
	}
	if _, e = os.Stat(denied); !os.IsNotExist(e) {
		t.Fatal("strict export published")
	}
	input := AssemblyInput{Turns: []processing.Turn{{Label: "manual", StartUS: 0, EndUS: 500000}}}
	raw, _ := json.Marshal(input)
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "recordings.assemble", ItemID: id, Data: raw}
	assembled := a.Dispatch(request)
	if assembled.Error != nil {
		t.Fatal(assembled.Error)
	}
	accepted, e := a.Catalog.Recording(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	replay := a.Dispatch(request)
	if replay.Error != nil {
		t.Fatal("replay", replay.Error)
	}
	current, _ := a.Catalog.Recording(context.Background(), id)
	if current.Revision != accepted.Revision {
		t.Fatal("replay replaced current")
	}
	input.Turns[0].Label = "different"
	request.Data, _ = json.Marshal(input)
	if conflict := a.Dispatch(request); conflict.Error == nil || conflict.Error.Code != "conflict" {
		t.Fatal("changed ephemeral intent accepted")
	}
	works, _ := a.Catalog.Works(context.Background())
	for _, work := range works {
		if work.Kind == "recordings.assemble" && containsAssignmentInput(work.Payload) {
			t.Fatal("assignments journaled")
		}
	}
	a.Close()
	a, e = New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	restored, e := a.Catalog.Recording(context.Background(), id)
	if e != nil || restored.DocumentDigest != accepted.DocumentDigest {
		t.Fatal("restart authority", e)
	}
}

func TestRecordingCancellationAndExplicitNoSpeech(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "cancellation")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	os.WriteFile(mapped, []byte("fixture"), 0600)
	entered := make(chan struct{})
	var started atomic.Bool
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{}, Provenance: map[string]any{}, Diagnostics: []any{},
				Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
					return processing.RecognitionResult{NoSpeech: true, Provenance: map[string]any{}}, nil
				},
				Diarize: func(ctx context.Context, _ string, _ processing.DiarizationOptions) (processing.DiarizationResult, error) {
					if !started.Swap(true) {
						close(entered)
						<-ctx.Done()
						return processing.DiarizationResult{}, contracts.Fail("cancelled")
					}
					return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{}}, nil
				},
				Close: func() error { return nil }}, nil
		}}, nil
	}
	id := importRecordingFixture(t, a)
	options := RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID()}
	r := realRequest(a, "recordings.process", id, options)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	workID := r.Result.(map[string]any)["work_id"].(string)
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("adapter not started")
	}
	if r = realRequest(a, "work.cancel", workID, nil); r.Error != nil {
		t.Fatal(r.Error)
	}
	if _, e = a.Catalog.Recording(context.Background(), id); e == nil {
		t.Fatal("cancelled candidate accepted")
	}
	options.Transcription = "generate"
	options.RecognitionModelID = contracts.ID()
	r = realRequest(a, "recordings.process", id, options)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	done := awaitWork(t, a, r.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatalf("silence %s", done.Result)
	}
	current, e := a.Catalog.Recording(context.Background(), id)
	if e != nil || current.State != "no-speech" || string(current.Document) != "null" || current.DocumentDigest != "" {
		t.Fatal("silence fabricated document", current.State, e)
	}
}

func TestDerivedCleanupProgressesPastRetainedCandidate(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "cleanup")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	id := importRecordingFixture(t, a)
	service, e := a.artifactService()
	if e != nil {
		t.Fatal(e)
	}
	source := filepath.Join(t.TempDir(), "candidate.wav")
	os.WriteFile(source, []byte("disposable candidate"), 0600)
	ids := []string{contracts.ID(), contracts.ID()}
	sort.Strings(ids)
	first, e := service.Publish(context.Background(), ids[0], source, "mapped-audio")
	if e != nil {
		t.Fatal(e)
	}
	second, e := service.Publish(context.Background(), ids[1], source, "mapped-audio")
	if e != nil {
		t.Fatal(e)
	}
	ref := contracts.ID()
	if e = a.Catalog.ArtifactReference(context.Background(), first.ID, ref, false); e != nil {
		t.Fatal(e)
	}
	for _, pub := range []catalog.Publication{first, second} {
		if e = a.Catalog.QueueDerivedCleanup(context.Background(), contracts.ID(), id, pub.ID); e != nil {
			t.Fatal(e)
		}
	}
	lib, e := a.libraryService()
	if e != nil {
		t.Fatal(e)
	}
	if e = lib.Cleanup(context.Background(), id); e == nil {
		t.Fatal("retention barrier ignored")
	}
	retired, e := a.Catalog.Publication(context.Background(), second.ID)
	if e != nil || retired.State != "retired" {
		t.Fatal("independent retirement starved", retired.State, e)
	}
	if _, e = service.Store.Stat(context.Background(), second); e == nil {
		t.Fatal("retired candidate bytes remain")
	}
	if e = a.Catalog.ArtifactReference(context.Background(), first.ID, ref, true); e != nil {
		t.Fatal(e)
	}
	if e = lib.Cleanup(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	if _, e = a.Catalog.Export(context.Background()); e != nil {
		t.Fatal("cleanup snapshot invalid", e)
	}
}
func TestLocalVoiceIDsAndDiagnosticAggregation(t *testing.T) {
	turn := []processing.Turn{{Label: "same voice", StartUS: 0, EndUS: 1000}}
	work := contracts.ID()
	a, _ := localTurns(contracts.ID(), work, turn)
	b, _ := localTurns(contracts.ID(), work, turn)
	if a[0].SpeakerID == b[0].SpeakerID {
		t.Fatal("voice identity leaked across recordings")
	}
	groups := []any{[]subtitles.Diagnostic{{Code: "timing-rounded", Pointer: "/cues/0", Message: "rounded"}, {Code: "timing-rounded", Pointer: "/cues/1", Message: "rounded"}}}
	raw := flattenDiagnostics(groups)
	var report []struct {
		Code    string
		Count   int64
		Pointer string
	}
	if json.Unmarshal(raw, &report) != nil || len(report) != 1 || report[0].Count != 2 || report[0].Pointer != "/cues/0" {
		t.Fatal("diagnostic occurrences lost", string(raw))
	}
}
