// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestHostedRecordingLoopbackNoFallbackAndReuseFence(t *testing.T) {
	a := configuredApp(t)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	raw := make([]byte, 32044)
	copy(raw, "RIFF")
	binary.LittleEndian.PutUint32(raw[4:], 32036)
	copy(raw[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(raw[16:], 16)
	binary.LittleEndian.PutUint16(raw[20:], 1)
	binary.LittleEndian.PutUint16(raw[22:], 1)
	binary.LittleEndian.PutUint32(raw[24:], 16000)
	binary.LittleEndian.PutUint32(raw[28:], 32000)
	binary.LittleEndian.PutUint16(raw[32:], 2)
	binary.LittleEndian.PutUint16(raw[34:], 16)
	copy(raw[36:], "data")
	binary.LittleEndian.PutUint32(raw[40:], 32000)
	mapped := filepath.Join(t.TempDir(), "fixture.wav")
	if os.WriteFile(mapped, raw, 0600) != nil {
		t.Fatal("wave fixture")
	}
	var fail atomic.Bool
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if fail.Load() {
			w.WriteHeader(503)
			return
		}
		if r.ParseMultipartForm(1<<20) != nil {
			t.Error("multipart")
			return
		}
		defer r.MultipartForm.RemoveAll()
		var request struct {
			Operation string `json:"operation"`
		}
		if json.Unmarshal([]byte(r.FormValue("request")), &request) != nil {
			t.Error("request")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if request.Operation == "transcription" {
			w.Write([]byte(`{"contract_version":"1","operation":"transcription","segments":[{"start_us":0,"end_us":500000,"text":"fixture"}],"no_speech":false}`))
		} else {
			w.Write([]byte(`{"contract_version":"1","operation":"diarization","turns":[{"label":"voice","start_us":0,"end_us":500000}],"no_speech":false}`))
		}
	}))
	defer server.Close()
	a.hostedClient = server.Client()
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(_ context.Context, _ catalog.LibraryEntry, o processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: processing.SourceMap{StartNumerator: "0", StartDenominator: "1", SampleCount: 16000, SampleRate: 16000, Channel: o.Channel, Policy: "selected-audio"}, Diagnostics: []any{}, Provenance: map[string]any{}, Close: func() error { return nil }, Recognize: func(context.Context, string, processing.RecognitionOptions) (processing.RecognitionResult, error) {
				t.Error("local recognition fallback")
				return processing.RecognitionResult{}, contracts.Fail("operation_failed")
			}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				t.Error("local diarization fallback")
				return processing.DiarizationResult{}, contracts.Fail("operation_failed")
			}}, nil
		}}, nil
	}
	media := importRecordingFixture(t, a)
	id := contracts.ID()
	stage := map[string]any{"adapter": "insonic-http", "contract_version": "1", "mode": "hosted", "endpoint": server.URL, "remote_model": "synthetic", "capabilities": []string{"transcription", "diarization"}, "supports_hints": true, "max_hint_bytes": 200}
	configuredResult(t, realRequest(a, "pipelines.set", id, map[string]any{"expected_revision": 0, "pipeline": map[string]any{"id": id, "name": "loopback", "preset": "connected", "configuration": map[string]any{"recognition": stage, "diarization": stage, "quality": map[string]any{"enabled": false}}}}))
	run := func(input map[string]any) catalog.Work {
		v := configuredResult(t, realRequest(a, "recordings.process", media, input))
		return awaitWork(t, a, v["work_id"].(string))
	}
	input := map[string]any{"pipeline_id": id, "transcription": "generate", "diarization": "run"}
	fail.Store(true)
	if run(input).State != "failed" {
		t.Fatal("hosted error succeeded")
	}
	fail.Store(false)
	if run(input).State != "succeeded" {
		t.Fatal("hosted election failed")
	}
	record, e := a.Catalog.Recording(a.ctx, media)
	if e != nil {
		t.Fatal(e)
	}
	var ds []processing.Diagnostic
	if json.Unmarshal(record.Diagnostics, &ds) != nil {
		t.Fatal("diagnostics")
	}
	for _, d := range ds {
		if d.Code == "speech_coverage_fraction" {
			t.Fatal("quality disable ignored")
		}
	}
	before := calls.Load()
	input["diarization"] = "reuse"
	input["audio"] = map[string]int{"channel": 1}
	if run(input).State != "failed" || calls.Load() != before {
		t.Fatal("changed mapping ran inference/replaced current")
	}
	current, e := a.Catalog.Recording(a.ctx, media)
	if e != nil || current.Revision != record.Revision {
		t.Fatal("rejected reuse replaced current")
	}
	delete(input, "audio")
	if run(input).State != "succeeded" {
		t.Fatal("compatible reuse rejected")
	}
}
