// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"fmt"
	"strings"
	"testing"
)

func TestRepairSelectsBoundedDisjointWindows(t *testing.T) {
	zero := int64(0)
	identity := PlanIdentity{SourceID: "13721a51-c02d-431f-a355-087f4be01efc", SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, ConfigurationDigest: strings.Repeat("b", 64)}
	plan, e := BuildWindowPlan(identity, WindowOptions{CoreUS: 1000000, ContextUS: &zero}, []TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 80000, StartNumerator: "0", StartDenominator: "1", EndSample: 80000}})
	if e != nil {
		t.Fatal(e)
	}
	selection, e := NormalizeRepairSelection(plan, nil, []RepairTarget{{TrackOrdinal: 0, CoverageSpan: CoverageSpan{100000, 500000}}, {TrackOrdinal: 0, CoverageSpan: CoverageSpan{4100000, 4500000}}}, RepairPolicy{})
	if e != nil || len(selection.Windows) != 2 || selection.Windows[0].ID != plan.Windows[0].ID || selection.Windows[1].ID != plan.Windows[4].ID {
		t.Fatalf("selected unrelated windows: %+v %v", selection, e)
	}
	if _, e = NormalizeRepairSelection(plan, nil, selection.Targets, RepairPolicy{MaxWindows: 1}); e == nil {
		t.Fatal("budget ignored")
	}
}
func TestRepairComparisonRequiresStableBasisAndNoLoss(t *testing.T) {
	in := coverageFixture()
	before, _ := EvaluateCoverage(in, CoverageOptions{})
	in.Cues = append(in.Cues, CoverageCue{ID: "fixed", Text: "middle", StartUS: 3000000, EndUS: 6000000, TrackOrdinal: 0})
	after, _ := EvaluateCoverage(in, CoverageOptions{})
	if ok, _ := CompareCoverage(before, after); !ok {
		t.Fatal("known gap not improved")
	}
	after.OptionsDigest = "other"
	if ok, _ := CompareCoverage(before, after); ok {
		t.Fatal("changed basis accepted")
	}
	after = before
	if ok, _ := CompareCoverage(before, after); ok {
		t.Fatal("same candidate accepted")
	}
}

func TestRepairComparisonCannotImproveByChangingWindowReferences(t *testing.T) {
	in := coverageFixture()
	before, _ := EvaluateCoverage(in, CoverageOptions{})
	before.Tracks[0].Metrics.DuplicateOverlapUS = 100
	before.Tracks[0].Metrics.RepeatedTextOverlapUS = 100
	after := before
	after.Tracks = append([]CoverageTrackReport{}, before.Tracks...)
	after.Tracks[0].Metrics.DuplicateOverlapUS = 0
	if ok, _ := CompareCoverage(before, after); ok {
		t.Fatal("lost corroboration counted as improvement")
	}
}

func TestRepairWholeCueClosureDoesNotCascadeThroughContextWindows(t *testing.T) {
	zero := int64(0)
	identity := PlanIdentity{SourceID: "13721a51-c02d-431f-a355-087f4be01efc", SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, ConfigurationDigest: strings.Repeat("b", 64)}
	plan, err := BuildWindowPlan(identity, WindowOptions{CoreUS: 1000000, ContextUS: &zero}, []TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 80000, StartNumerator: "0", StartDenominator: "1", EndSample: 80000}})
	if err != nil {
		t.Fatal(err)
	}
	cues := []CoverageCue{{ID: "crossing", TrackOrdinal: 0, StartUS: 1800000, EndUS: 2200000}, {ID: "outside", TrackOrdinal: 0, StartUS: 2300000, EndUS: 2500000}}
	targets := []RepairTarget{{TrackOrdinal: 0, CoverageSpan: CoverageSpan{1100000, 1200000}}}
	selected, err := NormalizeRepairSelection(plan, cues, targets, RepairPolicy{})
	if err != nil || len(selected.Windows) != 2 || len(selected.Scopes) != 1 || selected.Scopes[0].StartUS != 1000000 || selected.Scopes[0].EndUS != 2200000 || len(selected.RemovedCueIDs) != 1 || selected.RemovedCueIDs[0] != "crossing" {
		t.Fatalf("whole-cue scope: %+v %v", selected, err)
	}
	cues[0].TrackOrdinal = -1
	if _, err = NormalizeRepairSelection(plan, cues, targets, RepairPolicy{}); err == nil {
		t.Fatal("unresolved intersecting cue lane accepted")
	}
}

func TestRepairRejectsUnknownLaneReachedByWholeCueExpansion(t *testing.T) {
	zero := int64(0)
	plan, err := BuildWindowPlan(PlanIdentity{SourceID: "13721a51-c02d-431f-a355-087f4be01efc", SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, ConfigurationDigest: strings.Repeat("b", 64)}, WindowOptions{CoreUS: 1000000, ContextUS: &zero}, []TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 80000, StartNumerator: "0", StartDenominator: "1", EndSample: 80000}})
	if err != nil {
		t.Fatal(err)
	}
	cues := []CoverageCue{{ID: "crossing", TrackOrdinal: 0, StartUS: 1800000, EndUS: 2200000}, {ID: "unresolved", TrackOrdinal: -1, StartUS: 2100000, EndUS: 2300000}}
	if _, err = NormalizeRepairSelection(plan, cues, []RepairTarget{{TrackOrdinal: 0, CoverageSpan: CoverageSpan{1100000, 1200000}}}, RepairPolicy{}); err == nil {
		t.Fatal("expanded scope reached unresolved lane")
	}
}

func TestRepairManyIndependentCueComponents(t *testing.T) {
	zero := int64(0)
	plan, err := BuildWindowPlan(PlanIdentity{SourceID: "13721a51-c02d-431f-a355-087f4be01efc", SourceDigest: strings.Repeat("a", 64), SourceRevision: 1, ConfigurationDigest: strings.Repeat("b", 64)}, WindowOptions{CoreUS: 1000000, ContextUS: &zero}, []TrackPlan{{StreamIndex: 0, SourceSampleRate: 16000, SourceSampleCount: 80000, StartNumerator: "0", StartDenominator: "1", EndSample: 80000}})
	if err != nil {
		t.Fatal(err)
	}
	cues := make([]CoverageCue, 60000)
	for i := range cues {
		start := int64(i) * 50
		cues[i] = CoverageCue{ID: fmt.Sprint(i), TrackOrdinal: 0, StartUS: start, EndUS: start + 25}
	}
	selected, err := NormalizeRepairSelection(plan, cues, []RepairTarget{{TrackOrdinal: 0, CoverageSpan: CoverageSpan{100000, 200000}}}, RepairPolicy{})
	if err != nil || len(selected.RemovedCueIDs) != 20000 || len(selected.Windows) != 1 {
		t.Fatalf("dense scope: cues=%d windows=%d err=%v", len(selected.RemovedCueIDs), len(selected.Windows), err)
	}
}
