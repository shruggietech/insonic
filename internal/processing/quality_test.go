// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"strings"
	"testing"
)

func TestQualitySweepsVoicesInsteadOfCountingDuplicateIntervals(t *testing.T) {
	turns := []Turn{{"A", 0, 500000}, {"A", 250000, 750000}, {"B", 500000, 1000000}, {"C", 1000000, 1100000}}
	diagnostics, err := Quality(turns, 2000000, QualityConfig{})
	if err != nil {
		t.Fatal(err)
	}
	metrics := map[string]float64{}
	counts := map[string]int64{}
	for _, d := range diagnostics {
		if d.Value != nil {
			metrics[d.Code] = *d.Value
		}
		counts[d.Code] = d.Count
	}
	if metrics["speech_coverage_fraction"] != 0.55 || metrics["overlap_fraction"] != 0.125 || counts["short_turns"] != 1 {
		t.Fatalf("quality metrics: %+v %+v", metrics, counts)
	}
	disabled := false
	if ds, err := Quality(turns, 2000000, QualityConfig{Enabled: &disabled}); err != nil || len(ds) != 0 {
		t.Fatal("disabled quality still emitted diagnostics")
	}
	if _, err := Quality([]Turn{{"A", -1, 1}}, 2000000, QualityConfig{Enabled: &disabled}); err == nil {
		t.Fatal("quality disabling bypassed timing integrity")
	}
}
func TestRecognitionProjectionAndHints(t *testing.T) {
	mapping := SourceMap{StartNumerator: "1", StartDenominator: "3", SampleRate: 16000, SampleCount: 16000}
	r, err := RecognitionFromSegments([]RecognitionSegment{{0, 500000, "hello"}}, false, mapping)
	if err != nil || !strings.Contains(string(r.SRT), "00:00:00,334 --> 00:00:00,833") {
		t.Fatal("source clock not projected conservatively")
	}
	if _, err := RecognitionFromSegments([]RecognitionSegment{{0, 1000001, "hello"}}, false, mapping); err == nil {
		t.Fatal("out of range segment accepted")
	}
	if _, err := ValidateRecognitionOptions(RecognitionOptions{Hints: []string{strings.Repeat("x", 201)}}, 200); err == nil {
		t.Fatal("hint truncation accepted silently")
	}
	if _, err := ValidateRecognitionOptions(RecognitionOptions{Hints: []string{"name"}, ContextDigest: strings.Repeat("0", 64)}, 200); err == nil {
		t.Fatal("mismatched frozen context accepted")
	}
	if opts, err := ValidateRecognitionOptions(RecognitionOptions{Hints: []string{"name"}}, 200); err != nil || opts.ContextDigest != HintsDigest([]string{"name"}) {
		t.Fatal("hint digest not normalized")
	}
}
