// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWindowSourceResolutionRequiresKnownTrack(t *testing.T) {
	source := json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"stream_index":1,"source_sample_rate":48000,"source_sample_count":480000,"start_numerator":"1","start_denominator":"3"},{"stream_index":4,"channel":1,"source_sample_rate":16000,"source_sample_count":160000,"start_numerator":"2","start_denominator":"1"}]}`)
	provenance := json.RawMessage(`{"cue_sources":[{"cue_id":"cue-a","source":{"track_ordinal":1,"stream_index":4,"channel":1}}],"voice_sources":[{"local_speaker_id":"voice-a","track_ordinal":0}]}`)
	got, err := ResolveCueSourceMap(source, provenance, "cue-a")
	if err != nil || !bytes.Contains(got, []byte(`"stream_index":4`)) {
		t.Fatal(string(got), err)
	}
	got, err = ResolveEvidenceSourceMap(source, provenance, "cue-a", "voice-a")
	if err != nil || !bytes.Contains(got, []byte(`"stream_index":1`)) {
		t.Fatal(string(got), err)
	}
	if _, err = ResolveCueSourceMap(source, provenance, "unknown"); err == nil {
		t.Fatal("unknown multitrack source guessed")
	}
}
func TestWindowSourceResolutionPreservesLegacyAndSingleTrack(t *testing.T) {
	legacy := json.RawMessage(`{"stream_index":7,"channel":null}`)
	got, err := ResolveCueSourceMap(legacy, nil, "cue")
	if err != nil || !bytes.Equal(got, legacy) {
		t.Fatal(string(got), err)
	}
	single := json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"stream_index":3,"source_sample_rate":48000,"source_sample_count":480000,"start_numerator":"1","start_denominator":"3"}]}`)
	got, err = ResolveCueSourceMap(single, nil, "supplied")
	if err != nil || !bytes.Contains(got, []byte(`"stream_index":3`)) || !bytes.Contains(got, []byte(`"sample_count":160000`)) {
		t.Fatal(string(got), err)
	}
}
func TestWindowSourceResolutionRejectsForgedSelection(t *testing.T) {
	source := json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"stream_index":1,"source_sample_rate":48000,"source_sample_count":480000,"start_numerator":"0","start_denominator":"1"}]}`)
	bad := json.RawMessage(`{"cue_sources":[{"cue_id":"cue-a","source":{"track_ordinal":0,"stream_index":7}}]}`)
	if _, err := ResolveCueSourceMap(source, bad, "cue-a"); err == nil {
		t.Fatal("forged stream accepted")
	}
}

func TestWindowSourceResolutionRejectsMissingOrInvalidSelectedFields(t *testing.T) {
	source := json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"stream_index":0,"source_sample_rate":16000,"source_sample_count":16000,"start_numerator":"0","start_denominator":"1"}]}`)
	for _, raw := range []string{
		`{"voice_sources":[{"local_speaker_id":"v"}],"cue_sources":[{"cue_id":"c","source":{"track_ordinal":0,"stream_index":0}}]}`,
		`{"voice_sources":[{"local_speaker_id":"v","track_ordinal":-1}],"cue_sources":[{"cue_id":"c","source":{"track_ordinal":0,"stream_index":0}}]}`,
		`{"cue_sources":[{"cue_id":"c","source":{"stream_index":0}}]}`,
		`{"cue_sources":[{"cue_id":"c","source":{"track_ordinal":0}}]}`,
	} {
		if _, err := ResolveEvidenceSourceMap(source, json.RawMessage(raw), "c", "v"); err == nil {
			t.Fatalf("invalid selection accepted: %s", raw)
		}
	}
	missing := json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"source_sample_rate":16000,"source_sample_count":16000,"start_numerator":"0","start_denominator":"1"}]}`)
	if _, err := ResolveCueSourceMap(missing, nil, "c"); err == nil {
		t.Fatal("missing stream guessed")
	}
}
