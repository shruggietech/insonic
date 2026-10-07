// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/speakers"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"
)

func TestSpeakerSelectionPageBoundariesDoNotChangeCorpus(t *testing.T) {
	a := configuredApp(t)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	mapped := filepath.Join(t.TempDir(), "mapped.wav")
	if os.WriteFile(mapped, []byte("synthetic fixture, no engine"), 0600) != nil {
		t.Fatal("mapped fixture")
	}
	var offset atomic.Int64
	a.recordingFactory = func() (*recordingExecution, error) {
		return &recordingExecution{Subtitles: fixtureSubtitles{doc}, Prepare: func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*audioExecution, error) {
			start := offset.Load()
			return &audioExecution{Path: mapped, SourceMap: processing.SourceMap{StartNumerator: "0", StartDenominator: "1", SampleRate: 16000, SampleCount: 16000, StreamIndex: 0, Policy: "fixture"}, Provenance: map[string]any{}, Diagnostics: []any{}, Close: func() error { return nil }, Diarize: func(context.Context, string, processing.DiarizationOptions) (processing.DiarizationResult, error) {
				return processing.DiarizationResult{Turns: []processing.Turn{{Label: "voice", StartUS: start, EndUS: start + 500000}}}, nil
			}}, nil
		}}, nil
	}
	person, e := a.Catalog.PutSpeaker(a.ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Person"}})
	if e != nil {
		t.Fatal(e)
	}
	for _, start := range []int64{0, 0, 250000} {
		offset.Store(start)
		media := importRecordingFixture(t, a)
		queued := configuredResult(t, realRequest(a, "recordings.process", media, RecordingOptions{Transcription: "supplied", Diarization: "run", DiarizationModelID: contracts.ID()}))
		done := awaitWork(t, a, queued["work_id"].(string))
		if done.State != "succeeded" {
			t.Fatal(done.Result)
		}
		record, e := a.Catalog.Recording(a.ctx, media)
		if e != nil {
			t.Fatal(e)
		}
		turns, _, e := currentTurns(record.Document)
		if e != nil || len(turns) != 1 {
			t.Fatal("assignment fixture", e)
		}
		if _, e = a.Catalog.SetSpeakerMapping(a.ctx, contracts.ID(), record.Revision, catalog.SpeakerMapping{RecordingID: media, LocalSpeakerID: turns[0].SpeakerID, SpeakerID: person.Speaker.ID, DocumentDigest: record.DocumentDigest}); e != nil {
			t.Fatal(e)
		}
	}
	type summary struct {
		spans       []string
		duplicates  int
		diagnostics map[string]int
	}
	collect := func(limit int) summary {
		out := summary{diagnostics: map[string]int{}}
		cursor := ""
		for page := 0; page < 10; page++ {
			value := configuredResult(t, realRequest(a, "speakers.select", person.Speaker.ID, map[string]any{"limit": limit, "cursor": cursor}))
			raw, _ := json.Marshal(value["selection"])
			var selection speakers.Selection
			if json.Unmarshal(raw, &selection) != nil {
				t.Fatal("selection shape")
			}
			out.duplicates += selection.Duplicates
			for _, d := range selection.Diagnostics {
				out.diagnostics[d.Code] += d.Count
			}
			for _, item := range selection.Items {
				for _, span := range item.Intervals {
					out.spans = append(out.spans, span.Start+":"+span.End)
				}
			}
			cursor = value["next_cursor"].(string)
			if cursor == "" {
				sort.Strings(out.spans)
				return out
			}
		}
		t.Fatal("unbounded pagination")
		return out
	}
	whole := collect(100)
	if len(whole.spans) != 2 || whole.duplicates != 1 || whole.diagnostics["overlapping_evidence"] != 1 {
		t.Fatal("global fixture lacks duplicate/overlap", whole)
	}
	for _, limit := range []int{1, 2} {
		paged := collect(limit)
		if !reflect.DeepEqual(whole, paged) {
			t.Fatalf("page size%d changes corpus: whole=%+v paged=%+v", limit, whole, paged)
		}
	}
}
