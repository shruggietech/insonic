package speakers

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"testing"
)

func TestSelectionSourceClockDedupUntimedAndOverlap(t *testing.T) {
	ref := catalog.CurrentReference{RecordingID: "first", CueID: "cue", SpeakerID: "person", LocalSpeakerID: "voice", SourceDigest: "source"}
	evidence := catalog.ResolvedEvidence{Reference: ref, Mapping: catalog.SpeakerMapping{SpeakerID: "person"}, SourceMap: json.RawMessage(`{"start_numerator":"10","start_denominator":"1","stream_index":0,"channel":null}`), Cue: json.RawMessage(`{"id":"cue","speaker_attributions":[{"speaker_id":"voice","start_milliseconds":10100,"end_milliseconds":10400},{"speaker_id":"voice"}]}`)}
	second := evidence
	second.Reference.RecordingID = "second"
	second.Reference.LocalSpeakerID = "other"
	second.Cue = json.RawMessage(`{"id":"cue","speaker_attributions":[{"speaker_id":"other","start_milliseconds":10100,"end_milliseconds":10400},{"speaker_id":"other","start_milliseconds":10300,"end_milliseconds":10500}]}`)
	got, e := Select([]catalog.ResolvedEvidence{evidence, second})
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Items) != 2 || got.Items[0].Intervals[0].Start != "101/10" || got.Items[0].Intervals[0].End != "52/5" || !got.Items[0].Untimed || got.Duplicates != 1 {
		t.Fatal(got)
	}
	codes := map[string]bool{}
	for _, d := range got.Diagnostics {
		codes[d.Code] = true
	}
	if !codes["overlapping_evidence"] || !codes["untimed_participation"] {
		t.Fatal(got)
	}
}
func TestSuppliedSourceClockWithoutInferredStream(t *testing.T) {
	ref := catalog.CurrentReference{RecordingID: "entry", CueID: "cue", SpeakerID: "person", LocalSpeakerID: "voice", SourceDigest: "source"}
	resolved := catalog.ResolvedEvidence{Reference: ref, Mapping: catalog.SpeakerMapping{SpeakerID: "person"}, SourceMap: json.RawMessage(`{"policy":"supplied-subtitle-source-clock"}`), Cue: json.RawMessage(`{"id":"cue","speaker_attributions":[{"speaker_id":"voice","start_milliseconds":500,"end_milliseconds":750}]}`)}
	got, e := Select([]catalog.ResolvedEvidence{resolved})
	if e != nil || len(got.Items) != 1 || got.Items[0].Intervals[0].Start != "1/2" || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "source_stream_unknown" {
		t.Fatal(got, e)
	}
}
