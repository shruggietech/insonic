// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
)

func suppliedWindowDocumentFixture(t *testing.T, declaredTiming bool) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	var cues []map[string]json.RawMessage
	json.Unmarshal(fields["cues"], &cues)
	second := map[string]json.RawMessage{}
	for key, value := range cues[0] {
		second[key] = value
	}
	second["id"] = json.RawMessage(`"cue-000001"`)
	second["ordinal"], second["source_order"] = json.RawMessage("1"), json.RawMessage("1")
	second["source_identifier"] = json.RawMessage(`"2"`)
	second["timing"] = json.RawMessage(`{"start_milliseconds":70000,"end_milliseconds":71000,"duration_milliseconds":1000}`)
	second["payload"] = json.RawMessage(`{"raw_text":"Later supplied text.","plain_text":"Later supplied text.","lines":["Later supplied text."]}`)
	second["format_data"] = json.RawMessage(`{"subrip":{"sequence_line_raw":"2","timing_line_raw":"00:01:10,000 --> 00:01:11,000"}}`)
	cues = append(cues, second)
	fields["cues"], _ = json.Marshal(cues)
	var summary map[string]json.RawMessage
	json.Unmarshal(fields["document"], &summary)
	summary["cue_count"] = json.RawMessage("2")
	summary["media_end_milliseconds"], summary["media_span_milliseconds"] = json.RawMessage("71000"), json.RawMessage("71000")
	fields["document"], _ = json.Marshal(summary)
	var stats map[string]json.RawMessage
	json.Unmarshal(fields["stats"], &stats)
	stats["cue_count"], stats["media_span_milliseconds"] = json.RawMessage("2"), json.RawMessage("71000")
	fields["stats"], _ = json.Marshal(stats)
	if declaredTiming {
		fields["media_timing"] = json.RawMessage(`{"duration_milliseconds":90000}`)
	}
	raw, _ = json.Marshal(fields)
	if err = subtitles.ValidateDocument(raw); err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSuppliedWindowTranscriptReuseDoesNotAuthorizeAcousticReuse(t *testing.T) {
	digest := strings.Repeat("a", 64)
	plan := processing.ProcessingPlan{Identity: processing.PlanIdentity{SourceDigest: digest}, Tracks: []processing.TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 16000, EndSample: 16000, StartNumerator: "0", StartDenominator: "1"}}}
	previous := catalog.Recording{SourceDigest: digest, SourceMap: json.RawMessage(`{"policy":"supplied-document-source-clock;no-retiming","embedded_tracks":[]}`)}
	if err := compatibleWindowTranscriptReuse(previous, plan); err != nil {
		t.Fatal(err)
	}
	if compatibleWindowReuse(previous, plan) == nil {
		t.Fatal("supplied document authorized acoustic reuse")
	}
	previous.SourceDigest = strings.Repeat("b", 64)
	if compatibleWindowTranscriptReuse(previous, plan) == nil {
		t.Fatal("changed source reused")
	}
	previous.SourceDigest = digest
	previous.SourceMap, _ = json.Marshal(windowSourceMap{Policy: "windowed-canonical-source-clock-v1", SourceDigest: digest, Tracks: plan.Tracks})
	plan.Tracks = append([]processing.TrackPlan(nil), plan.Tracks...)
	plan.Tracks[0].StreamIndex = 1
	if compatibleWindowTranscriptReuse(previous, plan) == nil {
		t.Fatal("generated track change bypassed fence")
	}
}

func TestLongSuppliedAdmissionRunsElectedWindowPipeline(t *testing.T) {
	for _, declared := range []bool{false, true} {
		t.Run(map[bool]string{false: "implicit-native-clock", true: "declared-native-clock"}[declared], func(t *testing.T) {
			exerciseSuppliedWindowImport(t, fixtureSubtitles{}, declared)
		})
	}
}
func TestNativeLongSuppliedAdmissionRunsElectedWindowPipeline(t *testing.T) {
	executable := os.Getenv("CUESON_EXECUTABLE")
	if executable == "" {
		t.Skip("pinned native Cueson not elected")
	}
	binary, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(binary)
	driver, err := subtitles.New(subtitles.Tool{Executable: executable, ExecutableSHA256: hex.EncodeToString(digest[:])})
	if err != nil {
		t.Fatal(err)
	}
	exerciseSuppliedWindowImport(t, driver, true)
}
func exerciseSuppliedWindowImport(t *testing.T, engine subtitleEngine, declared bool) {
	t.Helper()
	a := configuredApp(t)
	doc := suppliedWindowDocumentFixture(t, declared)
	var calls atomic.Int32
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: engine, PrepareWindows: func(_ context.Context, entry catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
			if entry.DurationUS == nil || *entry.DurationUS != 61000000 {
				t.Error("long recording metadata lost")
			}
			track := processing.TrackPlan{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 61 * 16000, EndSample: 61 * 16000, StartNumerator: "0", StartDenominator: "1"}
			plan, err := processing.BuildWindowPlan(identity, options, []processing.TrackPlan{track})
			if err != nil {
				return nil, err
			}
			return &windowExecution{Plan: plan, Close: func() error { return nil }, Window: func(_ context.Context, index int) (*audioExecution, error) {
				window := plan.Windows[index]
				return &audioExecution{SourceMap: window.Mapping, Provenance: map[string]any{}, Diagnostics: []processing.Diagnostic{}, Close: func() error { return nil },
					Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
						t.Error("supplied transcript recognized again")
						return processing.RecognitionResult{}, contracts.Fail("operation_failed")
					},
					Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
						calls.Add(1)
						start, _, err := processing.WindowCoreUS(window)
						if err != nil {
							return processing.DiarizationResult{}, err
						}
						return processing.DiarizationFromTurns([]processing.Turn{{StartUS: start + 100000, EndUS: start + 500000, Label: "voice"}}, false, window.Mapping)
					},
				}, nil
			}}, nil
		}}, nil
	}
	pipelineID := contracts.ID()
	configuredResult(t, realRequest(a, "pipelines.set", pipelineID, map[string]any{"expected_revision": 0, "pipeline": map[string]any{"id": pipelineID, "name": "Supplied long import", "preset": "local", "configuration": localDefinition()}}))
	selection, err := a.freezeImportSelection(library.Options{}, library.Options{PipelineID: &pipelineID}, library.Options{}, true)
	if err != nil {
		t.Fatal(err)
	}
	source, _ := filepath.Abs("../../tests/fixtures/media/speech.flac")
	artifacts, err := a.artifactService()
	if err != nil {
		t.Fatal(err)
	}
	pub, err := artifacts.Publish(a.ctx, contracts.ID(), source, "original-media")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := a.Catalog.EnqueueWork(a.ctx, contracts.ID(), "media.import", json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, parent.ID, a.Session, leaseTTL)
	if err != nil {
		t.Fatal(err)
	}
	duration := int64(61000000)
	facts, _ := json.Marshal(library.Facts{Streams: []library.Stream{{Index: 0, Kind: "audio", SampleRate: "16000"}}, DurationUS: &duration, Extension: ".flac"})
	metadata, _ := json.Marshal(library.Metadata{CaptureState: "unsupported", Reports: []library.Extraction{}, Observations: []library.Observation{}})
	entry := catalog.LibraryEntry{ID: contracts.ID(), AssetID: pub.ArtifactID, Title: "Long supplied import", Class: "audio", Mode: "copy", SourceLocator: source, Digest: pub.Digest, Size: pub.Size, OriginalPublicationID: &pub.ID, DurationUS: &duration, Facts: facts, Metadata: metadata, Dates: json.RawMessage(`{"observations":[],"policy":"owner-first"}`), ReportPublicationIDs: json.RawMessage(`[]`)}
	recording := &catalog.Recording{ID: entry.ID, SourceDigest: entry.Digest, SourceRevision: 1, State: "ready", Document: doc, DocumentDigest: documentHash(doc), SourceMap: json.RawMessage(`{"policy":"supplied-document-source-clock;no-retiming","embedded_tracks":[]}`), Provenance: json.RawMessage(`{"mode":"import"}`), Diagnostics: json.RawMessage(`[]`)}
	entry, _, err = a.Catalog.CommitAdmission(a.ctx, claim, 0, 0, entry, recording)
	if err != nil {
		t.Fatal(err)
	}
	if entry.SubtitlePublicationID != nil {
		t.Fatal("admitted document retained legacy subtitle publication")
	}
	ids, err := a.queueImportProcessing(a.ctx, claim, 0, entry.ID, selection)
	if err != nil || len(ids) != 1 {
		t.Fatal(ids, err)
	}
	done := awaitWork(t, a, ids[0])
	if done.State != "succeeded" {
		t.Fatal(done.State, string(done.Result))
	}
	if calls.Load() != 3 {
		t.Fatal("long import did not run expected windows", calls.Load())
	}
	current, err := a.Catalog.Recording(a.ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	var before, after map[string]json.RawMessage
	json.Unmarshal(doc, &before)
	json.Unmarshal(current.Document, &after)
	if !bytes.Equal(before["media_timing"], after["media_timing"]) {
		t.Fatal("supplied media clock replaced", string(after["media_timing"]))
	}
	beforeCues, err := subtitles.DocumentCues(doc)
	if err != nil {
		t.Fatal(err)
	}
	afterCues, err := subtitles.DocumentCues(current.Document)
	if err != nil || !reflect.DeepEqual(beforeCues, afterCues) {
		t.Fatal("supplied cue timing changed", beforeCues, afterCues, err)
	}
	if !bytes.Contains(current.Document, []byte("Later supplied text.")) {
		t.Fatal("late supplied text removed")
	}
	var accepted windowSourceMap
	if json.Unmarshal(current.SourceMap, &accepted) != nil || !accepted.SuppliedDocument {
		t.Fatal("supplied timing authority lost", string(current.SourceMap))
	}
	// A subsequent acoustic reuse still binds exact selected tracks.
	plan := processing.ProcessingPlan{Identity: processing.PlanIdentity{SourceDigest: entry.Digest}, Tracks: accepted.Tracks}
	if err = compatibleWindowReuse(current, plan); err != nil {
		t.Fatal("same acoustic observations cannot be reused", err)
	}
	plan.Tracks = append([]processing.TrackPlan(nil), plan.Tracks...)
	plan.Tracks[0].StreamIndex = 1
	if compatibleWindowReuse(current, plan) == nil {
		t.Fatal("new supplied flag bypassed acoustic track fence")
	}
}

func TestSuppliedTranscriptCannotBypassDiarizationReuseFence(t *testing.T) {
	digest := strings.Repeat("a", 64)
	entry := catalog.LibraryEntry{ID: contracts.ID(), Revision: 1, Digest: digest}
	previous := catalog.Recording{State: "ready", SourceDigest: digest, SourceMap: json.RawMessage(`{"policy":"supplied-document-source-clock;no-retiming","embedded_tracks":[]}`), DocumentDigest: strings.Repeat("b", 64)}
	a := &App{recordingFactory: func() (*recordingExecution, error) {
		return &recordingExecution{PrepareWindows: func(_ context.Context, _ catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
			track := processing.TrackPlan{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 16000, EndSample: 16000, StartNumerator: "0", StartDenominator: "1"}
			plan, err := processing.BuildWindowPlan(identity, options, []processing.TrackPlan{track})
			if err != nil {
				return nil, err
			}
			return &windowExecution{Plan: plan, Close: func() error { return nil }, Window: func(context.Context, int) (*audioExecution, error) {
				t.Fatal("unfenced reuse started inference")
				return nil, nil
			}}, nil
		}}, nil
	}}
	for _, mode := range []string{"reuse", "supplied", "generate"} {
		input := recordingPayload{Expected: 1, Options: RecordingOptions{Transcription: mode, Diarization: "reuse", Windows: &processing.WindowOptions{}}}
		if _, err := a.processWindowedRecording(context.Background(), catalog.Work{}, input, entry, previous); err == nil || contracts.SafeError(err).Code != "conflict" {
			t.Fatal(mode, err)
		}
	}
}
