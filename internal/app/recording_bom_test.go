// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestNativeFormatRecognizesBOMWithoutChangingSource(t *testing.T) {
	for _, source := range []string{"WEBVTT\n\n", "\xef\xbb\xbfWEBVTT\r\n\r\n", " \nWEBVTT\n\n", "\xef\xbb\xbf1\n00:00:00,000 --> 00:00:01,000\nText\n"} {
		raw := []byte(source)
		before := bytes.Clone(raw)
		want := "vtt"
		if bytes.Contains(raw, []byte("00:00:00,000")) {
			want = "srt"
		}
		if got := nativeFormat(raw, ""); got != want {
			t.Fatalf("detected %q want %q", got, want)
		}
		if got := nativeFormat(raw, "srt"); got != "srt" {
			t.Fatal("explicit format election changed")
		}
		if !bytes.Equal(raw, before) {
			t.Fatal("format sniffing changed source bytes")
		}
	}
}

type bomPreservingSubtitles struct {
	subtitleEngine
	input []byte
	calls *atomic.Int32
}

func (s bomPreservingSubtitles) Ingest(ctx context.Context, input []byte, format string) (json.RawMessage, error) {
	if format != "vtt" || !bytes.Equal(input, s.input) {
		return nil, contracts.Fail("invalid_request")
	}
	s.calls.Add(1)
	return s.subtitleEngine.Ingest(ctx, input, format)
}

func TestSuppliedBOMWebVTTAutoDetectsForProcessingAndAssembly(t *testing.T) {
	input := []byte("\xef\xbb\xbfWEBVTT\r\n\r\n00:00:00.000 --> 00:00:01.000\r\nPreserved supplied text.\r\n\r\n")
	fixture, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var engine subtitleEngine = fixtureSubtitles{fixture}
	native := os.Getenv("CUESON_EXECUTABLE") != ""
	if native {
		path := os.Getenv("CUESON_EXECUTABLE")
		binary, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		digest := sha256.Sum256(binary)
		driver, e := subtitles.New(subtitles.Tool{Executable: path, ExecutableSHA256: hex.EncodeToString(digest[:])})
		if e != nil {
			t.Fatal(e)
		}
		engine = driver
	}
	var calls atomic.Int32
	engine = bomPreservingSubtitles{engine, input, &calls}
	w, e := workspace.Init(t.TempDir(), "BOM WebVTT")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	if e = os.WriteFile(mapped, []byte("deterministic mapped audio"), 0600); e != nil {
		t.Fatal(e)
	}
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: engine, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{}, Provenance: map[string]any{}, Diagnostics: []any{}, Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
				return processing.RecognitionResult{}, contracts.Fail("operation_failed")
			}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				return processing.DiarizationResult{Turns: []processing.Turn{{Label: "voice", StartUS: 0, EndUS: 500000}}}, nil
			}, Close: func() error { return nil }}, nil
		}}, nil
	}
	source, _ := filepath.Abs("../../tests/fixtures/media/speech.flac")
	subtitle := filepath.Join(t.TempDir(), "supplied.vtt")
	if e = os.WriteFile(subtitle, input, 0600); e != nil {
		t.Fatal(e)
	}
	admission := realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: source, Subtitle: subtitle, NewEntry: true}}})
	if admission.Error != nil {
		t.Fatal(admission.Error)
	}
	done := awaitWork(t, a, admission.Result.(map[string]any)["work_id"].(string))
	var imported library.ImportResult
	if json.Unmarshal(done.Result, &imported) != nil || len(imported.Items) != 1 {
		t.Fatal("admission", string(done.Result))
	}
	id := imported.Items[0].MediaID
	assert := func() {
		current, e := a.Catalog.Recording(context.Background(), id)
		if e != nil || current.State != "ready" {
			t.Fatal("BOM WebVTT was rejected", current.State, e)
		}
		if native {
			var document struct {
				Format string `json:"format"`
				Source struct {
					Assets []struct {
						Data string `json:"data_base64"`
					} `json:"assets"`
				} `json:"source"`
			}
			if json.Unmarshal(current.Document, &document) != nil || document.Format != "webvtt" || len(document.Source.Assets) != 1 {
				t.Fatal("native WebVTT document identity")
			}
			retained, e := base64.StdEncoding.DecodeString(document.Source.Assets[0].Data)
			if e != nil || !bytes.Equal(retained, input) {
				t.Fatal("native source envelope changed supplied bytes")
			}
		}
	}
	processed := realRequest(a, "recordings.process", id, RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID()})
	if processed.Error != nil {
		t.Fatal(processed.Error)
	}
	done = awaitWork(t, a, processed.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatal("processing", string(done.Result))
	}
	assert()
	assembled := realRequest(a, "recordings.assemble", id, AssemblyInput{Turns: []processing.Turn{{Label: "voice", StartUS: 0, EndUS: 500000}}})
	if assembled.Error != nil {
		t.Fatal(assembled.Error)
	}
	assert()
	if calls.Load() != 2 {
		t.Fatal("both supplied default routes were not exercised", calls.Load())
	}
	retained, e := os.ReadFile(subtitle)
	if e != nil || !bytes.Equal(retained, input) {
		t.Fatal("original subtitle was modified")
	}
}
