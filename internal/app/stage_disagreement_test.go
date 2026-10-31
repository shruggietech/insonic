// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync/atomic"
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

func TestNativeIntegrityFailurePreservesAcceptedRecordingAndAttemptCause(t *testing.T) {
	a := configuredApp(t)
	doc, err := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if err != nil {
		t.Fatal(err)
	}
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	if err = os.WriteFile(mapped, []byte("existing decoded candidate"), 0600); err != nil {
		t.Fatal(err)
	}
	var fail atomic.Bool
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{}, Provenance: map[string]any{}, Diagnostics: []any{}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				if fail.Load() {
					return processing.DiarizationResult{}, contracts.WithDeviceAttempt(contracts.Fail("checksum_mismatch"), contracts.DeviceAttempt{RequestedDevice: "auto", AttemptedDevices: []string{"cuda"}})
				}
				return processing.DiarizationResult{Turns: []processing.Turn{{Label: "fixture", StartUS: 0, EndUS: 500000}}, Provenance: map[string]any{}}, nil
			}, Close: func() error { return nil }}, nil
		}}, nil
	}
	id := importRecordingFixture(t, a)
	options := RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID(), Attribution: processing.DiarizationOptions{Device: "auto"}}
	accepted := realRequest(a, "recordings.process", id, options)
	if accepted.Error != nil {
		t.Fatal(accepted.Error)
	}
	first := awaitWork(t, a, accepted.Result.(map[string]any)["work_id"].(string))
	if first.State != "succeeded" {
		t.Fatalf("initial acceptance failed: %s", first.Result)
	}
	before, err := a.Catalog.Recording(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	fail.Store(true)
	options.Transcription = "reuse"
	accepted = realRequest(a, "recordings.process", id, options)
	if accepted.Error != nil {
		t.Fatal(accepted.Error)
	}
	failed := awaitWork(t, a, accepted.Result.(map[string]any)["work_id"].(string))
	var result struct {
		Cause *contracts.Error `json:"cause"`
	}
	if json.Unmarshal(failed.Result, &result) != nil || failed.State != "failed" || result.Cause == nil || result.Cause.Code != "checksum_mismatch" || result.Cause.DeviceAttempt == nil || result.Cause.DeviceAttempt.RequestedDevice != "auto" || len(result.Cause.DeviceAttempt.AttemptedDevices) != 1 || result.Cause.DeviceAttempt.AttemptedDevices[0] != "cuda" || result.Cause.DeviceAttempt.FallbackReason != "" {
		t.Fatalf("failed work lost integrity cause: %s", failed.Result)
	}
	after, err := a.Catalog.Recording(context.Background(), id)
	if err != nil || before.Revision != after.Revision || before.DocumentDigest != after.DocumentDigest || !bytes.Equal(before.Document, after.Document) || !bytes.Equal(before.Provenance, after.Provenance) {
		t.Fatal("integrity failure changed accepted recording", err)
	}
	if before.MappedAudioPublicationID == nil || after.MappedAudioPublicationID == nil || *before.MappedAudioPublicationID != *after.MappedAudioPublicationID {
		t.Fatal("integrity failure replaced accepted audio")
	}
	publication, err := a.Catalog.Publication(context.Background(), *before.MappedAudioPublicationID)
	if err != nil || publication.State != "available" {
		t.Fatal("integrity failure retired accepted audio", err)
	}
}
