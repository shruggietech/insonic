// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestRecordingStageDisagreementRetainsRecognizedText(t *testing.T) {
	for _, mode := range []string{"generate", "supplied"} {
		t.Run(mode, func(t *testing.T) {
			w, err := workspace.Init(t.TempDir(), "stage disagreement")
			if err != nil {
				t.Fatal(err)
			}
			a, err := New(w)
			if err != nil {
				t.Fatal(err)
			}
			defer a.Close()
			doc, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
			if err != nil {
				t.Fatal(err)
			}
			mapped := filepath.Join(t.TempDir(), "mapped.wav")
			if err = os.WriteFile(mapped, []byte("nonzero quiet speech fixture"), 0600); err != nil {
				t.Fatal(err)
			}
			calls := 0
			a.recordingFactory = func() (*recordingExecution, error) {
				return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
					return &audioExecution{Path: mapped, SourceMap: map[string]any{}, Provenance: map[string]any{}, Diagnostics: []any{},
						Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
							calls++
							return processing.RecognitionResult{SRT: []byte("1\n00:00:00,000 --> 00:00:00,500\nquiet speech\n"), Provenance: map[string]any{"adapter": "fixture"}}, nil
						},
						Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
							return processing.DiarizationResult{NoSpeech: true, Provenance: map[string]any{"adapter": "fixture"}}, nil
						}, Close: func() error { return nil }}, nil
				}}, nil
			}
			id := importRecordingFixture(t, a)
			request := realRequest(a, "recordings.process", id, RecordingOptions{Transcription: mode, Diarization: "run", RecognitionModelID: contracts.ID(), DiarizationModelID: contracts.ID()})
			if request.Error != nil {
				t.Fatal(request.Error)
			}
			done := awaitWork(t, a, request.Result.(map[string]any)["work_id"].(string))
			if done.State != "succeeded" {
				t.Fatalf("processing failed: %s", done.Result)
			}
			current, err := a.Catalog.Recording(context.Background(), id)
			if err != nil || current.State != "ready" || current.DocumentDigest == "" || bytes.Equal(current.Document, []byte("null")) {
				t.Fatalf("text discarded: %+v %v", current, err)
			}
			want := mode == "generate"
			if bytes.Contains(current.Diagnostics, []byte("recognition_diarization_disagreement")) != want || bytes.Contains(done.Result, []byte("Recognition returned text")) != want {
				t.Fatalf("wrong disagreement evidence: %s %s", current.Diagnostics, done.Result)
			}
			if want && calls != 1 || !want && calls != 0 {
				t.Fatal("unexpected recognition calls", calls)
			}
		})
	}
}
