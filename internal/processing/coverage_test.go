// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func coverageFixture() CoverageInput {
	tracks := []TrackPlan{{StreamIndex: 0, SourceSampleRate: 1000, SourceSampleCount: 10000, StartNumerator: "0", StartDenominator: "1", EndSample: 10000}, {StreamIndex: 1, SourceSampleRate: 1000, SourceSampleCount: 5000, StartNumerator: "0", StartDenominator: "1", EndSample: 5000}}
	return CoverageInput{RecordingID: "1b0c490d-2781-4b27-a978-ff1ecbbbe139", SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, DocumentDigest: strings.Repeat("b", 64), DocumentRevision: 2, SourceMapDigest: strings.Repeat("c", 64), Tracks: tracks, Evidence: &CoverageEvidence{Kind: "recording-coverage-evidence", Version: 1, SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, SourceMapDigest: strings.Repeat("c", 64), Observations: []CoverageObservation{{WindowID: "11f1ae21-17d1-40d0-a494-70b728e9dd1b", OutputDigest: strings.Repeat("d", 64), TrackOrdinal: 0, Core: CoverageSpan{0, 10000000}, RecognitionStatus: "unavailable", DiarizationStatus: "executed", DiarizationSpeech: []CoverageSpan{{1000000, 9000000}}}, {WindowID: "ae93fca9-dbf9-4258-b68c-a2754da6449e", OutputDigest: strings.Repeat("e", 64), TrackOrdinal: 1, Core: CoverageSpan{0, 5000000}, RecognitionStatus: "executed", DiarizationStatus: "executed", RecognitionNoSpeech: true, DiarizationNoSpeech: true}}}, Cues: []CoverageCue{{ID: "a", Text: "before", StartUS: 1000000, EndUS: 3000000, TrackOrdinal: 0}, {ID: "b", Text: "after", StartUS: 6000000, EndUS: 9000000, TrackOrdinal: 0}}}
}
func TestCoverageIndependentLanesAndKnownGap(t *testing.T) {
	in := coverageFixture()
	r, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Tracks) != 2 || r.Tracks[0].Metrics.SelectedAudioUS != 10000000 || r.Tracks[1].Metrics.SelectedAudioUS != 5000000 {
		t.Fatalf("lanes: %+v", r)
	}
	m := r.Tracks[0].Metrics
	if m.ObservedSpeechUS != 8000000 || m.UncoveredObservedSpeechUS != 3000000 || m.CueCoverageUS != 5000000 || m.UnknownActivityUS != 2000000 {
		t.Fatalf("metrics: %+v", m)
	}
	if r.Tracks[1].Metrics.ObservedNoSpeechUS != 5000000 || r.Tracks[1].SpeechCoverage.Fraction != nil {
		t.Fatal("silence is not a speech denominator")
	}
	if len(r.Regions) != 1 || r.Regions[0].Reason != "uncovered_speech" || r.Regions[0].StartUS != 3000000 || r.Regions[0].EndUS != 6000000 {
		t.Fatalf("gap: %+v", r.Regions)
	}
}
func TestCoverageUnknownAndConflictingSilence(t *testing.T) {
	in := coverageFixture()
	in.Evidence = nil
	r, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Tracks[0].Metrics.UnknownActivityUS != 10000000 || len(r.Regions) != 0 {
		t.Fatal("invented speech")
	}
	in = coverageFixture()
	in.Evidence.Observations[0].RecognitionStatus = "executed"
	in.Evidence.Observations[0].RecognitionNoSpeech = true
	r, e = EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if r.Tracks[0].Metrics.ObservedSpeechUS != 8000000 || r.Tracks[0].Metrics.ObservedNoSpeechUS != 2000000 {
		t.Fatal("silence overrode positive evidence")
	}
	if len(r.Limitations) == 0 {
		t.Fatal("missing disagreement")
	}
}
func TestCoverageDuplicatePaginationAndGenuineRepetition(t *testing.T) {
	in := coverageFixture()
	first, second, third := "11f1ae21-17d1-40d0-a494-70b728e9dd1b", "f4e6a76c-83a6-4ad7-bb3d-b5e748dc975e", "82271487-038a-40ae-88dd-b43a48cdeae8"
	for _, id := range []string{second, third} {
		v := in.Evidence.Observations[0]
		v.WindowID = id
		in.Evidence.Observations = append(in.Evidence.Observations, v)
	}
	in.Cues = []CoverageCue{{ID: "a", Text: "Hello!", StartUS: 1000000, EndUS: 2000000, TrackOrdinal: 0, WindowIDs: []string{first}}, {ID: "b", Text: "hello", StartUS: 1100000, EndUS: 2100000, TrackOrdinal: 0, WindowIDs: []string{second}}, {ID: "c", Text: "hello", StartUS: 3000000, EndUS: 4000000, TrackOrdinal: 0, WindowIDs: []string{third}}}
	full, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	page, e := EvaluateCoverage(in, CoverageOptions{Limit: 1})
	if e != nil {
		t.Fatal(e)
	}
	if full.Tracks[0].Metrics.DuplicateOverlapUS != 900000 || page.TotalRegions != full.TotalRegions || len(page.Regions) != 1 || page.Next == nil {
		t.Fatalf("pagination/duplicate: %+v", page)
	}
	if full.OptionsDigest != page.OptionsDigest {
		t.Fatal("pagination changed diagnostic basis")
	}
}
func TestCoverageRejectsMalformedAndStaleEvidence(t *testing.T) {
	in := coverageFixture()
	in.Evidence.SourceRevision++
	r, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil || r.Tracks[0].Metrics.UnknownActivityUS != 10000000 {
		t.Fatal("stale evidence counted")
	}
	in = coverageFixture()
	in.Evidence.Observations[0].Core.EndUS = -1
	if _, e = EvaluateCoverage(in, CoverageOptions{}); e == nil {
		t.Fatal("malformed accepted")
	}
	if _, e = EvaluateCoverage(coverageFixture(), CoverageOptions{GapToleranceUS: -1}); e == nil {
		t.Fatal("negative tolerance")
	}
}

func TestCoveragePreservesWordBoundariesAndPhysicalBounds(t *testing.T) {
	in := coverageFixture()
	in.Evidence = nil
	in.Tracks[0].StartSample = 3000
	in.Tracks[0].EndSample = 6000
	in.Cues = []CoverageCue{{ID: "a", Text: "now here", StartUS: 1000000, EndUS: 2000000, TrackOrdinal: 0}, {ID: "b", Text: "nowhere", StartUS: 1000000, EndUS: 2000000, TrackOrdinal: 0}}
	r, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range r.Regions {
		if v.Reason == "missing_audio" || v.Reason == "repeated_text" {
			t.Fatalf("invented defect: %+v", v)
		}
	}
	in.Cues[0].EndUS = 12000000
	r, e = EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, v := range r.Regions {
		if v.Reason == "missing_audio" {
			found = true
			if v.StartUS != 10000000 {
				t.Fatal(v)
			}
		}
	}
	if !found {
		t.Fatal("missing actual out-of-audio region")
	}
}

func TestCoverageReuseRetainsOriginalEvidence(t *testing.T) {
	in := coverageFixture()
	before, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil {
		t.Fatal(e)
	}
	empty := *in.Evidence
	empty.Observations = nil
	retained, e := RetainReusedCoverage(empty, in.Evidence, true, true)
	if e != nil {
		t.Fatal(e)
	}
	in.Evidence = &retained
	after, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil || before.Tracks[0].Metrics != after.Tracks[0].Metrics || before.EvidenceDigest != after.EvidenceDigest {
		t.Fatal("reuse lost speech evidence", e)
	}
}

func TestCoverageBoundsAndQuietControls(t *testing.T) {
	in := coverageFixture()
	in.Cues[0].EndUS = math.MaxInt64
	if _, e := EvaluateCoverage(in, CoverageOptions{}); e == nil {
		t.Fatal("overflow accepted")
	}
	in = coverageFixture()
	in.Evidence.Observations = make([]CoverageObservation, MaxCoverageObservations+1)
	if _, e := EvaluateCoverage(in, CoverageOptions{}); e == nil {
		t.Fatal("observation bound ignored")
	}
	in = coverageFixture()
	in.Cues = nil
	for i := 0; i < 1500; i++ {
		in.Cues = append(in.Cues, CoverageCue{ID: strconv.Itoa(i), Text: "repeat", StartUS: 1000000, EndUS: 2000000, TrackOrdinal: 0})
	}
	if _, e := EvaluateCoverage(in, CoverageOptions{}); e == nil {
		t.Fatal("comparison/diagnostic bound ignored")
	}
	in = coverageFixture()
	in.Evidence.Observations[0].DiarizationSpeech = nil
	in.Evidence.Observations[0].DiarizationNoSpeech = true
	in.Cues = nil
	r, e := EvaluateCoverage(in, CoverageOptions{})
	if e != nil || len(r.Regions) != 0 || r.Tracks[0].Metrics.ObservedNoSpeechUS != 10000000 {
		t.Fatal("quiet/music no-speech control diagnosed as missing text", e)
	}
}

func TestCoverageRepeatedTextNeedsNonemptyMatchingVoiceEvidence(t *testing.T) {
	for _, tc := range []struct {
		name        string
		left, right []string
		uncertain   bool
	}{
		{"both unresolved", nil, nil, true}, {"one unresolved", []string{"voice-a"}, nil, true},
		{"different voices", []string{"voice-a"}, []string{"voice-b"}, true},
		{"same local voice", []string{"voice-a"}, []string{"voice-a"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := coverageFixture()
			first := in.Evidence.Observations[0]
			second := first
			second.WindowID = "f4e6a76c-83a6-4ad7-bb3d-b5e748dc975e"
			in.Evidence.Observations = append(in.Evidence.Observations, second)
			in.Cues = []CoverageCue{{ID: "a", Text: "Hello", StartUS: 1000000, EndUS: 2000000, TrackOrdinal: 0, WindowIDs: []string{first.WindowID}, VoiceIDs: tc.left}, {ID: "b", Text: "Hello", StartUS: 1100000, EndUS: 2100000, TrackOrdinal: 0, WindowIDs: []string{second.WindowID}, VoiceIDs: tc.right}}
			report, err := EvaluateCoverage(in, CoverageOptions{})
			if err != nil {
				t.Fatal(err)
			}
			repeated := 0
			for _, region := range report.Regions {
				if region.Reason == "repeated_text" {
					repeated++
					if region.Uncertain != tc.uncertain {
						t.Fatalf("uncertain=%v, want %v", region.Uncertain, tc.uncertain)
					}
				}
			}
			if repeated != 1 {
				t.Fatal("repetition diagnosis disappeared")
			}
		})
	}
}
