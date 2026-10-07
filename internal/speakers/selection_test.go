package speakers

import (
	"encoding/json"
	"errors"
	"github.com/shruggietech/insonic/internal/catalog"
	"testing"
)

func selectionPageEvidence(recording, cueID, voice string, assignments string) catalog.ResolvedEvidence {
	ref := catalog.CurrentReference{RecordingID: recording, CueID: cueID, SpeakerID: "person", LocalSpeakerID: voice, SourceDigest: "shared-source"}
	return catalog.ResolvedEvidence{Reference: ref, Mapping: catalog.SpeakerMapping{SpeakerID: "person"}, SourceMap: json.RawMessage(`{"stream_index":0,"channel":null}`), Cue: json.RawMessage(`{"id":"` + cueID + `","speaker_attributions":` + assignments + `}`)}
}

func TestSelectionGlobalComparisonsSurvivePageBoundaries(t *testing.T) {
	first := selectionPageEvidence("a", "cue1", "voice1", `[{"speaker_id":"voice1","start_milliseconds":100,"end_milliseconds":400}]`)
	second := selectionPageEvidence("b", "cue2", "voice2", `[{"speaker_id":"other","start_milliseconds":0,"end_milliseconds":99},{"speaker_id":"voice2","start_milliseconds":100,"end_milliseconds":400},{"speaker_id":"voice2","start_milliseconds":300,"end_milliseconds":500},{"speaker_id":"voice2"}]`)
	third := selectionPageEvidence("b", "cue3", "voice3", `[{"speaker_id":"voice3","start_milliseconds":600,"end_milliseconds":700}]`)
	compare := func(ref catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error) {
		switch ref.CueID {
		case "cue1":
			return []catalog.PriorEvidenceComparison{{AssignmentOrdinal: 0, LocalVoices: 1, FirstReference: true}}, nil
		case "cue2":
			return []catalog.PriorEvidenceComparison{{AssignmentOrdinal: 1, Duplicate: true, LocalVoices: 2, FirstReference: true}, {AssignmentOrdinal: 2, Overlap: true, LocalVoices: 2, FirstReference: true}, {AssignmentOrdinal: 3, LocalVoices: 2, FirstReference: true}}, nil
		default:
			return []catalog.PriorEvidenceComparison{{AssignmentOrdinal: 0, LocalVoices: 2, FirstReference: false}}, nil
		}
	}
	whole, e := SelectWithPrior([]catalog.ResolvedEvidence{first, second, third}, compare)
	if e != nil {
		t.Fatal(e)
	}
	merged := Selection{Items: []SelectedEvidence{}, Diagnostics: []Diagnostic{}}
	counts := map[string]int{}
	for _, evidence := range []catalog.ResolvedEvidence{first, second, third} {
		page, e := SelectWithPrior([]catalog.ResolvedEvidence{evidence}, compare)
		if e != nil {
			t.Fatal(e)
		}
		merged.Items = append(merged.Items, page.Items...)
		merged.Duplicates += page.Duplicates
		for _, diagnostic := range page.Diagnostics {
			counts[diagnostic.Code] += diagnostic.Count
		}
	}
	if whole.Duplicates != 1 || merged.Duplicates != 1 || len(whole.Items) != 3 || len(merged.Items) != 3 || len(merged.Items[1].Intervals) != 1 || !merged.Items[1].Untimed {
		t.Fatal("paginated duplicate handling changed selected source evidence")
	}
	if merged.Items[1].Intervals[0].Start != "3/10" || counts["overlapping_evidence"] != 1 || counts["multiple_local_voices"] != 1 || counts["untimed_participation"] != 1 {
		t.Fatal("global quality counts reset or were counted per assignment")
	}
	for _, diagnostic := range whole.Diagnostics {
		if counts[diagnostic.Code] != diagnostic.Count {
			t.Fatal("diagnostics depend on page size")
		}
	}
	standalone, _ := Select([]catalog.ResolvedEvidence{second})
	if standalone.Duplicates != 0 || len(standalone.Items[0].Intervals) != 2 {
		t.Fatal("standalone fixture selection unexpectedly used external comparison state")
	}
}

func TestSelectionRejectsMismatchedOrdinalComparisonsAndPreservesErrors(t *testing.T) {
	evidence := selectionPageEvidence("entry", "cue", "voice", `[{"speaker_id":"other","start_milliseconds":0,"end_milliseconds":100},{"speaker_id":"voice","start_milliseconds":100,"end_milliseconds":200},{"speaker_id":"voice"}]`)
	for _, flags := range [][]catalog.PriorEvidenceComparison{
		nil,
		{{AssignmentOrdinal: 0, LocalVoices: 1}, {AssignmentOrdinal: 2, LocalVoices: 1}},
		{{AssignmentOrdinal: 1, LocalVoices: 1}, {AssignmentOrdinal: 1, LocalVoices: 1}},
		{{AssignmentOrdinal: 1, LocalVoices: 1}, {AssignmentOrdinal: 2, Duplicate: true, LocalVoices: 1}},
		{{AssignmentOrdinal: 1, LocalVoices: 0}, {AssignmentOrdinal: 2, LocalVoices: 0}},
	} {
		_, e := SelectWithPrior([]catalog.ResolvedEvidence{evidence}, func(catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error) { return flags, nil })
		if e == nil {
			t.Fatal("inconsistent catalog comparison accepted")
		}
	}
	sentinel := errors.New("synthetic stale comparison")
	if _, e := SelectWithPrior([]catalog.ResolvedEvidence{evidence}, func(catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error) { return nil, sentinel }); !errors.Is(e, sentinel) {
		t.Fatal("comparison epoch failure ignored")
	}
	if _, e := SelectWithPrior(make([]catalog.ResolvedEvidence, 101), func(catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error) {
		t.Fatal("comparison invoked beyond bounded page")
		return nil, nil
	}); e == nil {
		t.Fatal("reference bound ignored")
	}
}

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
