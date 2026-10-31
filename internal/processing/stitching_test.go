// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"fmt"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
	"testing"
)

func stitchFixture(t *testing.T) []WindowEvidence {
	t.Helper()
	track := TrackPlan{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 60 * 16000, StartNumerator: "0", StartDenominator: "1", EndSample: 60 * 16000}
	plan, err := BuildWindowPlan(windowTestIdentity(), WindowOptions{}, []TrackPlan{track})
	if err != nil {
		t.Fatal(err)
	}
	out := make([]WindowEvidence, len(plan.Windows))
	for i, w := range plan.Windows {
		out[i] = WindowEvidence{Window: w, OutputDigest: strings.Repeat("a", 64)}
	}
	return out
}
func TestStitchBoundaryWordOnceAndRealRepetitionPreserved(t *testing.T) {
	windows := stitchFixture(t)
	windows[0].Recognition.Segments = []RecognitionSegment{{StartUS: 29500000, EndUS: 30300000, Text: " Hello,", Words: []RecognitionWord{{StartUS: 29500000, EndUS: 30300000, Text: " Hello,"}}}}
	windows[1].Recognition.Segments = []RecognitionSegment{{StartUS: 1500000, EndUS: 2300000, Text: " Hello,", Words: []RecognitionWord{{StartUS: 1500000, EndUS: 2300000, Text: " Hello,"}}}, {StartUS: 4000000, EndUS: 4400000, Text: " Hello,", Words: []RecognitionWord{{StartUS: 4000000, EndUS: 4400000, Text: " Hello,"}}}}
	result, err := ReconcileWindows(windows)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(result.Recognition.SRT), "Hello,") != 2 || len(result.CueSources) != 2 || len(result.CueSources[0].WindowIDs) != 2 {
		t.Fatalf("%s %+v", result.Recognition.SRT, result.CueSources)
	}
	prior := result.Digest
	windows[1].OutputDigest = strings.Repeat("b", 64)
	result, err = ReconcileWindows(windows)
	if err != nil || result.Digest == prior {
		t.Fatal("output digest not bound", err)
	}
}
func TestStitchTracksAndSegmentOnlyAmbiguityRemain(t *testing.T) {
	windows := stitchFixture(t)
	windows[0].Recognition.Segments = []RecognitionSegment{{StartUS: 29500000, EndUS: 31000000, Text: "世界 one"}}
	windows[1].Recognition.Segments = []RecognitionSegment{{StartUS: 1500000, EndUS: 3000000, Text: "世界 two"}}
	result, err := ReconcileWindows(windows)
	if err != nil || !strings.Contains(string(result.Recognition.SRT), "世界 one") || !strings.Contains(string(result.Recognition.SRT), "世界 two") {
		t.Fatal(result, err)
	}
	found := false
	for _, d := range result.Diagnostics {
		found = found || d.Code == "window_overlap_ambiguous"
	}
	if !found {
		t.Fatal("missing ambiguity")
	}
	windows[1].Window.TrackOrdinal = 1
	windows[1].Recognition.Segments[0].Text = "世界 one"
	result, err = ReconcileWindows(windows)
	if err != nil || strings.Count(string(result.Recognition.SRT), "世界 one") != 2 {
		t.Fatal(result, err)
	}
}
func TestStitchDiarizationLabelsAreWindowScoped(t *testing.T) {
	windows := stitchFixture(t)
	for i := range windows {
		windows[i].Recognition.NoSpeech = true
		windows[i].Diarization.Turns = []Turn{{Label: "SPEAKER_00", StartUS: 29000000, EndUS: 31000000}}
	}
	result, err := ReconcileWindows(windows)
	if err != nil || len(result.Turns) != 2 || result.Turns[0].Label == result.Turns[1].Label || result.Turns[0].EndUS != 30000000 || result.Turns[1].StartUS != 30000000 {
		t.Fatal(result.Turns, err)
	}
}
func TestStitchRejectsDuplicateWindowAndMalformedWord(t *testing.T) {
	windows := stitchFixture(t)
	windows[1].Window.ID = windows[0].Window.ID
	if _, err := ReconcileWindows(windows); err == nil {
		t.Fatal("duplicate window accepted")
	}
	windows = stitchFixture(t)
	windows[0].Recognition.Segments = []RecognitionSegment{{StartUS: 0, EndUS: 1, Text: "x", Words: []RecognitionWord{{StartUS: 0, EndUS: 2, Text: "x"}}}}
	if _, err := ReconcileWindows(windows); err == nil {
		t.Fatal("invalid word accepted")
	}
}

func TestStitchPartialWordEvidencePreservesWholeSegment(t *testing.T) {
	windows := stitchFixture(t)
	windows[0].Recognition.Segments = []RecognitionSegment{{StartUS: 1000000, EndUS: 2000000, Text: "partial evidence remains", Words: []RecognitionWord{{StartUS: 1000000, EndUS: 1200000, Text: "partial"}}}}
	result, err := ReconcileWindows(windows)
	if err != nil || !strings.Contains(string(result.Recognition.SRT), "partial evidence remains") {
		t.Fatal(result, err)
	}
}
func TestStitchLargeSourceOffsetDoesNotBecomeDecodedDuration(t *testing.T) {
	windows := stitchFixture(t)
	for i := range windows {
		windows[i].Window.Mapping.StartNumerator = fmt.Sprint(1000000 + windows[i].Window.MappedStartSample/16000)
	}
	windows[0].Recognition.Segments = []RecognitionSegment{{StartUS: 0, EndUS: 1000000, Text: "source clock"}}
	result, err := ReconcileWindows(windows)
	if err != nil || !strings.Contains(string(result.Recognition.SRT), "277:46:40,000") {
		t.Fatal(string(result.Recognition.SRT), err)
	}
}
func TestStitchSameWindowRepetitionIsNeverDeduplicated(t *testing.T) {
	windows := stitchFixture(t)
	windows[0].Recognition.Segments = []RecognitionSegment{{StartUS: 1000000, EndUS: 2000000, Text: "yes yes", Words: []RecognitionWord{{StartUS: 1000000, EndUS: 1600000, Text: "yes"}, {StartUS: 1400000, EndUS: 2000000, Text: " yes"}}}}
	result, err := ReconcileWindows(windows)
	if err != nil || !strings.Contains(string(result.Recognition.SRT), "yes yes") {
		t.Fatal(result, err)
	}
}

func TestStitchAggregateExceedsPerWindowSegmentCount(t *testing.T) {
	windows := stitchFixture(t)
	for i := range windows {
		base := int64(0)
		if i == 1 {
			base = 2000000
		}
		for j := 0; j < 10001; j++ {
			start := base + int64(j)*2000
			windows[i].Recognition.Segments = append(windows[i].Recognition.Segments, RecognitionSegment{StartUS: start, EndUS: start + 1000, Text: "x"})
		}
	}
	result, err := ReconcileWindows(windows)
	if err != nil || len(result.CueSources) != 20002 || len(result.Recognition.Segments) != 20002 {
		t.Fatal(len(result.CueSources), err)
	}
}
func TestAggregateWordCapacityIsIndependentAndReportsCapacity(t *testing.T) {
	segments := make([]RecognitionSegment, 101)
	for i := range segments {
		start := int64(i) * 2000
		segment := RecognitionSegment{StartUS: start, EndUS: start + 1000, Text: strings.Repeat("x", 1000)}
		for j := 0; j < 1000; j++ {
			segment.Words = append(segment.Words, RecognitionWord{StartUS: start + int64(j), EndUS: start + int64(j) + 1, Text: "x"})
		}
		segments[i] = segment
	}
	duration := segments[len(segments)-1].EndUS
	if ValidateRecognitionSegments(segments, duration) == nil {
		t.Fatal("worker word cap was removed")
	}
	if _, err := renderAggregateRecognitionSegments(segments, false, duration); err != nil {
		t.Fatal("valid aggregate treated as malformed worker", err)
	}
	if _, err := validateAggregateCapacityFixture(); contracts.SafeError(err).Code != "output_limit" {
		t.Fatal("aggregate capacity misclassified", err)
	}
}
func validateAggregateCapacityFixture() (RecognitionResult, error) {
	return renderAggregateRecognitionSegments(make([]RecognitionSegment, maxReconciledAtoms+1), false, 1)
}
func TestStitchNoSpeechAndUntimedEvidenceAreDistinct(t *testing.T) {
	windows := stitchFixture(t)
	for i := range windows {
		windows[i].Recognition.NoSpeech = true
	}
	result, err := ReconcileWindows(windows)
	if err != nil || !result.Recognition.NoSpeech || len(result.Recognition.SRT) != 0 {
		t.Fatal(result, err)
	}
	windows[0].Recognition.NoSpeech = false
	result, err = ReconcileWindows(windows)
	if err != nil || result.Recognition.NoSpeech || len(result.Recognition.SRT) != 0 {
		t.Fatal("unrepresentable result relabeled no-speech", result, err)
	}
}

func TestStitchRetainsRecognitionDiarizationDisagreement(t *testing.T) {
	windows := stitchFixture(t)
	for i := range windows {
		windows[i].Recognition.Segments = []RecognitionSegment{{StartUS: 100000, EndUS: 200000, Text: "speech"}}
		windows[i].Diarization.NoSpeech = true
	}
	result, err := ReconcileWindows(windows)
	if err != nil {
		t.Fatal(err)
	}
	count := int64(0)
	for _, d := range result.Diagnostics {
		if d.Code == "recognition_diarization_disagreement" {
			count = d.Count
		}
	}
	if count != 2 {
		t.Fatal(result.Diagnostics)
	}
	// A reuse placeholder makes no fresh acoustic no-speech claim.
	for i := range windows {
		windows[i].Diarization.NoSpeech = false
	}
	result, err = ReconcileWindows(windows)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range result.Diagnostics {
		if d.Code == "recognition_diarization_disagreement" {
			t.Fatal("reuse marked disagreement")
		}
	}
}
