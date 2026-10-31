// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
)

func windowSourceFixture() json.RawMessage {
	return json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"stream_index":1,"source_sample_rate":16000,"source_sample_count":16000,"start_sample":0,"end_sample":16000,"start_numerator":"0","start_denominator":"1"},{"stream_index":4,"channel":1,"source_sample_rate":16000,"source_sample_count":16000,"start_sample":0,"end_sample":16000,"start_numerator":"0","start_denominator":"1"}]}`)
}
func TestMatchingPreparesIndependentCueTracksAndCombinesVectors(t *testing.T) {
	s, _ := fixture(t)
	prepared := []int{}
	s.Prepare = func(_ context.Context, _ catalog.LibraryEntry, options processing.AudioOptions) (*Audio, error) {
		if options.StreamIndex == nil {
			t.Fatal("stream omitted")
		}
		stream := *options.StreamIndex
		if stream == 1 && options.Channel != nil || stream == 4 && (options.Channel == nil || *options.Channel != 1) {
			t.Fatal("wrong channel", options)
		}
		prepared = append(prepared, stream)
		path := filepath.Join(t.TempDir(), "track.wav")
		if err := writeWAV(path, make([]byte, 32000)); err != nil {
			t.Fatal(err)
		}
		return &Audio{Path: path, SourceMap: processing.SourceMap{StartNumerator: "0", StartDenominator: "1", SampleRate: 16000, SampleCount: 16000, StreamIndex: stream, Channel: options.Channel}}, nil
	}
	s.Embed = func(_ context.Context, _ string, mapping processing.SourceMap, _, _ string) ([]float64, error) {
		if mapping.StreamIndex == 1 {
			return []float64{1, 0}, nil
		}
		return []float64{0, 1}, nil
	}
	r := catalog.Recording{SourceMap: windowSourceFixture(), Provenance: json.RawMessage(`{"cue_sources":[{"cue_id":"a","source":{"track_ordinal":0,"stream_index":1}},{"cue_id":"b","source":{"track_ordinal":1,"stream_index":4,"channel":1}}]}`)}
	spans := []voiceSpan{{0, 500, "a"}, {0, 500, "b"}}
	cache := map[string]*Audio{}
	options := MatchOptions{Adapter: builtin(), MinEvidenceUS: 750000}
	vector, duration, _, err := s.voiceEmbeddingBySource(context.Background(), r, catalog.LibraryEntry{}, "voice", spans, options, t.TempDir(), cache)
	if err != nil || duration != 1000000 || len(vector) != 2 || math.Abs(vector[0]-math.Sqrt(.5)) > 1e-9 || math.Abs(vector[1]-math.Sqrt(.5)) > 1e-9 || !reflect.DeepEqual(prepared, []int{1, 4}) {
		t.Fatal(vector, duration, prepared, err)
	}
	// Reuse is per stream/channel, so repeated voices do not prepare either track again.
	if _, _, _, err = s.voiceEmbeddingBySource(context.Background(), r, catalog.LibraryEntry{}, "voice", spans, options, t.TempDir(), cache); err != nil || len(prepared) != 2 {
		t.Fatal(prepared, err)
	}
	r.Provenance = json.RawMessage(`{}`)
	if _, _, _, err = s.voiceEmbeddingBySource(context.Background(), r, catalog.LibraryEntry{}, "voice", spans, options, t.TempDir(), cache); err == nil {
		t.Fatal("unknown multitrack voice guessed")
	}
}
func TestEnrollmentPreparesEachSelectedCueTrack(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	r, local := recording(t, db, 2)
	r.SourceMap = windowSourceFixture()
	var envelope map[string]any
	json.Unmarshal(r.SourceMap, &envelope)
	envelope["source_digest"] = r.SourceDigest
	r.SourceMap, _ = json.Marshal(envelope)
	var document map[string]json.RawMessage
	json.Unmarshal(r.Document, &document)
	document["media_timing"] = json.RawMessage(`{"duration_milliseconds":1000}`)
	r.Document, _ = json.Marshal(document)
	r.DocumentDigest = hash(r.Document)
	r.Provenance = json.RawMessage(`{"cue_sources":[{"cue_id":"cue-000000","source":{"track_ordinal":0,"stream_index":1}},{"cue_id":"cue-000001","source":{"track_ordinal":1,"stream_index":4,"channel":1}}]}`)
	var err error
	r, err = db.CommitRecording(ctx, work(t, db, "recordings.process", map[string]any{}), r.Revision, r)
	if err != nil {
		t.Fatal(err)
	}
	speaker, err := db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Grounded"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.SetSpeakerMapping(ctx, contracts.ID(), r.Revision, catalog.SpeakerMapping{RecordingID: r.ID, LocalSpeakerID: local, SpeakerID: speaker.Speaker.ID, DocumentDigest: r.DocumentDigest}); err != nil {
		t.Fatal(err)
	}
	dataset, err := s.CreateDataset(ctx, contracts.ID(), DatasetOptions{SpeakerID: speaker.Speaker.ID})
	if err != nil {
		t.Fatal(err)
	}
	original := s.Prepare
	prepared := []int{}
	s.Prepare = func(ctx context.Context, entry catalog.LibraryEntry, options processing.AudioOptions) (*Audio, error) {
		if options.StreamIndex == nil {
			t.Fatal("stream omitted")
		}
		prepared = append(prepared, *options.StreamIndex)
		audio, err := original(ctx, entry, options)
		if audio != nil {
			audio.SourceMap.StreamIndex = *options.StreamIndex
		}
		return audio, err
	}
	inputs, _, err := s.prepare(ctx, dataset, TrainOptions{Adapter: builtin()}, t.TempDir())
	if err != nil || len(inputs) != 2 || !reflect.DeepEqual(prepared, []int{1, 4}) || inputs[0].SourceMap.StreamIndex != 1 || inputs[1].SourceMap.StreamIndex != 4 {
		t.Fatal(inputs, prepared, err)
	}
}
