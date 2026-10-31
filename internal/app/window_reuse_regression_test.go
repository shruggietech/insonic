// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestImplicitSuppliedCurrentRejectsChangedTrackBeforeInference(t *testing.T) {
	for _, mode := range []string{"", "supplied"} {
		t.Run(mode, func(t *testing.T) {
			entry := catalog.LibraryEntry{ID: contracts.ID(), Digest: strings.Repeat("a", 64), Revision: 1}
			oldTrack := processing.TrackPlan{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 16000, EndSample: 16000, StartNumerator: "0", StartDenominator: "1"}
			raw, _ := json.Marshal(windowSourceMap{Policy: "windowed-canonical-source-clock-v1", SourceDigest: entry.Digest, Tracks: []processing.TrackPlan{oldTrack}})
			previous := catalog.Recording{State: "ready", SourceMap: raw, DocumentDigest: strings.Repeat("b", 64)}
			closed := false
			a := &App{recordingFactory: func() (*recordingExecution, error) {
				return &recordingExecution{PrepareWindows: func(_ context.Context, _ catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
					next := oldTrack
					next.StreamIndex = 1
					plan, err := processing.BuildWindowPlan(identity, options, []processing.TrackPlan{next})
					if err != nil {
						return nil, err
					}
					return &windowExecution{Plan: plan, Close: func() error { closed = true; return nil }, Window: func(context.Context, int) (*audioExecution, error) {
						t.Fatal("inference started before source compatibility check")
						return nil, nil
					}}, nil
				}}, nil
			}}
			input := recordingPayload{Expected: 1, Options: RecordingOptions{Transcription: mode, Diarization: "run", Windows: &processing.WindowOptions{}}}
			_, err := a.processWindowedRecording(context.Background(), catalog.Work{}, input, entry, previous)
			if err == nil || contracts.SafeError(err).Code != "conflict" || !closed {
				t.Fatal(err, closed)
			}
		})
	}
}

func TestNativeWindowCuesKeepSourceOrderAndPositiveClock(t *testing.T) {
	executable := os.Getenv("CUESON_EXECUTABLE")
	if executable == "" {
		t.Skip("pinned native Cueson not selected")
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
	w, err := workspace.Init(t.TempDir(), "native window source alignment")
	if err != nil {
		t.Fatal(err)
	}
	a, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	source, _ := filepath.Abs("../../tests/fixtures/media/speech.flac")
	id := seedLegacyMedia(t, a, source, "")
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: driver, PrepareWindows: func(_ context.Context, _ catalog.LibraryEntry, options processing.WindowOptions, identity processing.PlanIdentity) (*windowExecution, error) {
			tracks := []processing.TrackPlan{
				{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 16000, EndSample: 16000, StartNumerator: "10", StartDenominator: "1"},
				{StreamIndex: 1, SourceSampleRate: 16000, SourceSampleCount: 16000, EndSample: 16000, StartNumerator: "10", StartDenominator: "1"},
			}
			plan, err := processing.BuildWindowPlan(identity, options, tracks)
			if err != nil {
				return nil, err
			}
			return &windowExecution{Plan: plan, Close: func() error { return nil }, Window: func(_ context.Context, index int) (*audioExecution, error) {
				mapping := plan.Windows[index].Mapping
				text := []string{"First track.", "第二 track."}[index]
				return &audioExecution{SourceMap: mapping, Provenance: map[string]any{}, Diagnostics: []processing.Diagnostic{}, Close: func() error { return nil },
					Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
						return processing.RecognitionFromSegments([]processing.RecognitionSegment{{StartUS: 100000, EndUS: 800000, Text: text}}, false, mapping)
					},
					Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
						return processing.DiarizationFromTurns([]processing.Turn{{StartUS: 100000, EndUS: 800000, Label: "SPEAKER_00"}}, false, mapping)
					},
				}, nil
			}}, nil
		}}, nil
	}
	admitted := realRequest(a, "recordings.process", id, RecordingOptions{Transcription: "generate", Diarization: "run", RecognitionModelID: contracts.ID(), DiarizationModelID: contracts.ID(), Windows: &processing.WindowOptions{}})
	if admitted.Error != nil {
		t.Fatal(admitted.Error)
	}
	done := awaitWork(t, a, admitted.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		plans, _ := a.Catalog.ProcessingPlans(context.Background(), done.ID)
		t.Fatal(done.State, string(done.Result), "phase", done.Phase, "plans", len(plans))
	}
	current, err := a.Catalog.Recording(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Cues []struct {
			ID      string `json:"id"`
			Payload struct {
				PlainText string `json:"plain_text"`
			} `json:"payload"`
			Assignments []struct {
				ID string `json:"speaker_id"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if err = json.Unmarshal(current.Document, &document); err != nil || len(document.Cues) != 2 {
		t.Fatal(string(current.Document), err)
	}
	cues, err := subtitles.DocumentCues(current.Document)
	if err != nil {
		t.Fatal(err)
	}
	for index, cue := range document.Cues {
		if cue.Payload.PlainText != []string{"First track.", "第二 track."}[index] || len(cue.Assignments) != 1 || cues[index].StartMS != 10100 || cues[index].EndMS != 10800 {
			t.Fatal(document, cues)
		}
		raw, err := catalog.ResolveEvidenceSourceMap(current.SourceMap, current.Provenance, cue.ID, cue.Assignments[0].ID)
		var mapping processing.SourceMap
		if err != nil || json.Unmarshal(raw, &mapping) != nil || mapping.StreamIndex != index {
			t.Fatal(string(raw), err)
		}
	}
	if document.Cues[0].Assignments[0].ID == document.Cues[1].Assignments[0].ID {
		t.Fatal("independent track voices merged")
	}
}
