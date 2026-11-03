// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	"testing"
)

func TestMixedSpeakerEvidenceIncludesLaterUnmappedVoices(t *testing.T) {
	r := Recording{SourceDigest: hash([]byte("source")), SourceMap: json.RawMessage(`{"stream_index":0,"channel":0,"start_numerator":"0","start_denominator":"1","source_sample_rate":16000,"source_sample_count":16000}`), Document: json.RawMessage(`{"cues":[{"id":"a","speaker_attributions":[{"speaker_id":"v","start_milliseconds":0,"end_milliseconds":100}]},{"id":"b","speaker_attributions":[{"speaker_id":"other","start_milliseconds":50,"end_milliseconds":150}]},{"id":"c","speaker_attributions":[{"speaker_id":"v","start_milliseconds":150,"end_milliseconds":250}]}]}`)}
	mixed, e := MixedSpeakerEvidence(r)
	if e != nil || !mixed["a/v"] || !mixed["b/other"] || mixed["c/v"] {
		t.Fatal(mixed, e)
	}
	r.SourceMap = json.RawMessage(`{"policy":"windowed-canonical-source-clock-v1","tracks":[{"stream_index":0,"channel":0,"start_numerator":"0","start_denominator":"1","source_sample_rate":16000,"source_sample_count":16000},{"stream_index":1,"channel":0,"start_numerator":"0","start_denominator":"1","source_sample_rate":16000,"source_sample_count":16000}]}`)
	r.Provenance = json.RawMessage(`{"voice_sources":[{"local_speaker_id":"v","track_ordinal":0},{"local_speaker_id":"other","track_ordinal":1}]}`)
	mixed, e = MixedSpeakerEvidence(r)
	if e != nil || len(mixed) != 0 {
		t.Fatal(mixed, e)
	}
}
