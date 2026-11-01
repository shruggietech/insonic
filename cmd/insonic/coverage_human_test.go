// SPDX-License-Identifier: Apache-2.0
package main

import (
	"strings"
	"testing"
)

func TestCoverageHumanNamesDenominatorsAndLimits(t *testing.T) {
	var b humanBuffer
	v := map[string]any{"kind": "recording-coverage", "recording_id": "fixture", "document_revision": 2, "tracks": []any{map[string]any{"track_ordinal": 0, "metrics": map[string]any{"selected_audio_us": 10000000, "cue_coverage_us": 5000000, "observed_speech_us": 8000000, "uncovered_observed_speech_us": 3000000, "unknown_activity_us": 2000000, "observed_no_speech_us": 0}}}, "total_regions": 1, "limitations": []any{"acoustic_evidence_unavailable"}}
	if !humanCoverageSummary(&b, v) {
		t.Fatal("not handled")
	}
	s := b.String()
	for _, want := range []string{"selected audio", "observed speech", "unknown activity", "acoustic_evidence_unavailable"} {
		if !strings.Contains(s, want) {
			t.Fatal(s)
		}
	}
}

func TestRepairWorkHumanDistinguishesUnchanged(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext.Path = []string{"work", "wait"}
	for _, status := range []string{"accepted", "unchanged"} {
		var b humanBuffer
		result := map[string]any{"kind": "recording-repair", "status": status, "reason": "not_improved", "attempts": map[string]any{"window": 2}, "selection": map[string]any{"windows": []any{1}, "scopes": []any{1}}}
		if !humanWorkSummary(&b, map[string]any{"id": "10000000-0000-4000-8000-000000000001", "state": "succeeded", "kind": "recordings.repair", "result": result}, false) {
			t.Fatal("not handled")
		}
		for _, want := range []string{"Repair: " + status, "reason: not_improved", "1 windows; 2 attempts"} {
			if !strings.Contains(b.String(), want) {
				t.Fatal(b.String())
			}
		}
	}
}
