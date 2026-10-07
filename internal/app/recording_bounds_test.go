// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"path/filepath"
	"testing"
)

func TestZeroDurationCuesBecomeNullCurrentResult(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "zero cues")
	if e != nil {
		t.Fatal(e)
	}
	a, e := New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	raw, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	var d map[string]json.RawMessage
	json.Unmarshal(raw, &d)
	var cues []map[string]json.RawMessage
	json.Unmarshal(d["cues"], &cues)
	cues[0]["timing"] = json.RawMessage(`{"start_milliseconds":0,"end_milliseconds":0,"duration_milliseconds":0}`)
	d["cues"], _ = json.Marshal(cues)
	d["document"] = json.RawMessage(`{"cue_count":1,"media_start_milliseconds":0,"media_end_milliseconds":0,"media_span_milliseconds":0,"has_word_level_timing":false}`)
	d["stats"] = json.RawMessage(`{"cue_count":1,"diagnostic_count":0,"warning_count":0,"error_count":0,"has_word_level_timing":false,"media_span_milliseconds":0}`)
	doc, _ := json.Marshal(d)
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	os.WriteFile(mapped, []byte("fixture"), 0600)
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			return &audioExecution{Path: mapped, SourceMap: map[string]any{}, Provenance: map[string]any{}, Diagnostics: []any{}, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				return processing.DiarizationResult{Turns: []processing.Turn{{Label: "voice", StartUS: 0, EndUS: 1000}}}, nil
			}, Close: func() error { return nil }}, nil
		}}, nil
	}
	id := importRecordingFixture(t, a)
	response := realRequest(a, "recordings.process", id, RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID()})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	done := awaitWork(t, a, response.Result.(map[string]any)["work_id"].(string))
	if done.State != "succeeded" {
		t.Fatal(string(done.Result))
	}
	assert := func() {
		r, e := a.Catalog.Recording(context.Background(), id)
		if e != nil || r.State != "no-timed-subtitles" || string(r.Document) != "null" || r.DocumentDigest != "" {
			t.Fatalf("fabricated timing %+v %v", r, e)
		}
	}
	assert()
	response = realRequest(a, "recordings.assemble", id, AssemblyInput{Turns: []processing.Turn{}})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	assert()
}

type resolvingCatalog struct {
	catalog.Catalog
	value catalog.ResolvedSegment
}

func (c resolvingCatalog) ResolveSegment(context.Context, string, int64) (catalog.ResolvedSegment, error) {
	return c.value, nil
}
func TestSegmentLookupIsScopedToRequestedRecording(t *testing.T) {
	recording, id := contracts.ID(), contracts.ID()
	a := &App{ctx: context.Background(), Catalog: resolvingCatalog{value: catalog.ResolvedSegment{Reference: catalog.Segment{ID: id, Revision: 1, RecordingID: recording}, Cue: json.RawMessage(`{"id":"current"}`)}}}
	data, _ := json.Marshal(map[string]any{"segment_id": id, "segment_revision": 1})
	req := contracts.Request{Operation: "recordings.resolve-segment", ItemID: recording, Data: data}
	if !recordingRequestValid(req) {
		t.Fatal("valid request rejected")
	}
	out, e := a.recordingDispatch(req)
	if e != nil || out.(catalog.ResolvedSegment).Reference.RecordingID != recording {
		t.Fatal(out, e)
	}
	req.ItemID = contracts.ID()
	if _, e = a.recordingDispatch(req); e == nil {
		t.Fatal("cross-recording lookup accepted")
	}
	req.Data = json.RawMessage(`{"segment_id":"bad","segment_revision":1}`)
	if _, e = a.recordingDispatch(req); e == nil {
		t.Fatal("invalid reference accepted")
	}
}
