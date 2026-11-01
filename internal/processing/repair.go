// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
)

type RepairPolicy struct {
	MaxAttempts   int   `json:"max_attempts,omitempty"`
	MaxWindows    int   `json:"max_windows,omitempty"`
	MaxDurationUS int64 `json:"max_duration_us,omitempty"`
}
type RepairTarget struct {
	TrackOrdinal int `json:"track_ordinal"`
	CoverageSpan
}
type RepairWindow struct {
	ID           string       `json:"window_id"`
	TrackOrdinal int          `json:"track_ordinal"`
	Core         CoverageSpan `json:"core"`
	Context      CoverageSpan `json:"context"`
}
type RepairSelection struct {
	Targets       []RepairTarget `json:"targets"`
	Scopes        []RepairTarget `json:"scopes"`
	Windows       []RepairWindow `json:"windows"`
	RemovedCueIDs []string       `json:"removed_cue_ids"`
}

func NormalizeRepairPolicy(p RepairPolicy) (RepairPolicy, error) {
	if p.MaxAttempts == 0 {
		p.MaxAttempts = 1
	}
	if p.MaxWindows == 0 {
		p.MaxWindows = 64
	}
	if p.MaxDurationUS == 0 {
		p.MaxDurationUS = 300000000
	}
	if p.MaxAttempts < 1 || p.MaxAttempts > 3 || p.MaxWindows < 1 || p.MaxWindows > MaxPlanWindows || p.MaxDurationUS < 1 || p.MaxDurationUS > 604800000000 {
		return p, contracts.Fail("invalid_request")
	}
	return p, nil
}
func mergeRepairTargets(targets []RepairTarget) []RepairTarget {
	out := append([]RepairTarget{}, targets...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].TrackOrdinal != out[j].TrackOrdinal {
			return out[i].TrackOrdinal < out[j].TrackOrdinal
		}
		return out[i].StartUS < out[j].StartUS
	})
	n := 0
	for _, v := range out {
		if n > 0 && out[n-1].TrackOrdinal == v.TrackOrdinal && v.StartUS <= out[n-1].EndUS {
			out[n-1].EndUS = max(out[n-1].EndUS, v.EndUS)
		} else {
			out[n] = v
			n++
		}
	}
	return out[:n]
}
func RepairContains(scopes []RepairTarget, lane int, start, end int64) bool {
	for _, s := range scopes {
		if s.TrackOrdinal == lane && start >= s.StartUS && end <= s.EndUS {
			return true
		}
	}
	return false
}
func repairIntersects(scopes []RepairTarget, lane int, span CoverageSpan) bool {
	for _, s := range scopes {
		if s.TrackOrdinal == lane {
			if _, ok := coverageClip(s.CoverageSpan, span); ok {
				return true
			}
		}
	}
	return false
}

// NormalizeRepairSelection expands initial targets to elected window cores and
// complete intersecting accepted cues. Secondary windows provide any necessary
// boundary context without recursively expanding every adjacent window core.
func NormalizeRepairSelection(plan ProcessingPlan, cues []CoverageCue, targets []RepairTarget, policy RepairPolicy) (RepairSelection, error) {
	out := RepairSelection{Targets: []RepairTarget{}, Scopes: []RepairTarget{}, Windows: []RepairWindow{}, RemovedCueIDs: []string{}}
	p, e := NormalizeRepairPolicy(policy)
	if e != nil {
		return out, e
	}
	if len(targets) == 0 || len(targets) > MaxCoverageRegions || len(cues) > 65536 || len(plan.Windows) > MaxPlanWindows || len(plan.Tracks) > 64 {
		return out, contracts.Fail("input_limit")
	}
	bounds := make([]CoverageSpan, len(plan.Tracks))
	for i, t := range plan.Tracks {
		bounds[i], e = TrackCoverageSpan(t)
		if e != nil {
			return out, e
		}
	}
	for _, t := range targets {
		if t.TrackOrdinal < 0 || t.TrackOrdinal >= len(bounds) || t.StartUS < bounds[t.TrackOrdinal].StartUS || t.EndUS > bounds[t.TrackOrdinal].EndUS || t.EndUS <= t.StartUS {
			return out, contracts.Fail("invalid_request")
		}
	}
	out.Targets = mergeRepairTargets(targets)
	if len(out.Targets) > 32 {
		return out, contracts.Fail("input_limit")
	}
	all := make([]RepairWindow, len(plan.Windows))
	for i, w := range plan.Windows {
		core, err := WindowCoverageSpan(w)
		if err != nil {
			return out, err
		}
		duration, err := MappedDuration(w.Mapping)
		if err != nil {
			return out, err
		}
		start, err := projectWindowUS(w.Mapping, 0, true)
		if err != nil {
			return out, err
		}
		end, err := projectWindowUS(w.Mapping, duration, false)
		if err != nil {
			return out, err
		}
		all[i] = RepairWindow{ID: w.ID, TrackOrdinal: w.TrackOrdinal, Core: core, Context: CoverageSpan{start, end}}
		if repairIntersects(out.Targets, w.TrackOrdinal, core) {
			out.Scopes = append(out.Scopes, RepairTarget{TrackOrdinal: w.TrackOrdinal, CoverageSpan: core})
		}
	}
	out.Scopes = mergeRepairTargets(out.Scopes)
	if len(out.Scopes) == 0 {
		return out, contracts.Fail("invalid_request")
	}
	sorted := append([]CoverageCue{}, cues...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].TrackOrdinal != sorted[j].TrackOrdinal {
			return sorted[i].TrackOrdinal < sorted[j].TrackOrdinal
		}
		return sorted[i].StartUS < sorted[j].StartUS
	})
	// A sweep merges intersecting cue components. It does not iterate the entire
	// cue array once per expansion, even for many mutually overlapping captions.
	components := []RepairTarget{}
	for _, c := range sorted {
		if c.TrackOrdinal < 0 {
			for _, s := range out.Scopes {
				if _, ok := coverageClip(s.CoverageSpan, CoverageSpan{c.StartUS, c.EndUS}); ok {
					return out, contracts.Condition("conflict", "source_map_incompatible")
				}
			}
			continue
		}
		v := RepairTarget{TrackOrdinal: c.TrackOrdinal, CoverageSpan: CoverageSpan{c.StartUS, c.EndUS}}
		n := len(components)
		if n > 0 && components[n-1].TrackOrdinal == v.TrackOrdinal && v.StartUS < components[n-1].EndUS {
			components[n-1].EndUS = max(components[n-1].EndUS, v.EndUS)
		} else {
			components = append(components, v)
		}
	}
	if len(out.Scopes) > 32 {
		return out, contracts.Fail("input_limit")
	}
	initialScopes := append([]RepairTarget{}, out.Scopes...)
	// Cue components are disjoint. Expanding through one cannot reach another;
	// compare against the fixed initial scopes to retain O(cues * 32) work.
	for _, component := range components {
		if repairIntersects(initialScopes, component.TrackOrdinal, component.CoverageSpan) {
			out.Scopes = append(out.Scopes, component)
		}
	}
	out.Scopes = mergeRepairTargets(out.Scopes)
	if len(out.Scopes) > 32 {
		return out, contracts.Fail("input_limit")
	}
	var duration int64
	for _, s := range out.Scopes {
		if s.TrackOrdinal < 0 || s.TrackOrdinal >= len(bounds) || s.StartUS < bounds[s.TrackOrdinal].StartUS || s.EndUS > bounds[s.TrackOrdinal].EndUS {
			return out, contracts.Condition("conflict", "source_map_incompatible")
		}
		duration += s.EndUS - s.StartUS
		if duration > p.MaxDurationUS {
			return out, contracts.Fail("output_limit")
		}
	}
	for _, c := range cues {
		if c.TrackOrdinal < 0 {
			for _, scope := range out.Scopes {
				if _, ok := coverageClip(scope.CoverageSpan, CoverageSpan{c.StartUS, c.EndUS}); ok {
					return out, contracts.Condition("conflict", "source_map_incompatible")
				}
			}
		}
		if repairIntersects(out.Scopes, c.TrackOrdinal, CoverageSpan{c.StartUS, c.EndUS}) {
			if !RepairContains(out.Scopes, c.TrackOrdinal, c.StartUS, c.EndUS) {
				return out, contracts.Fail("conflict")
			}
			out.RemovedCueIDs = append(out.RemovedCueIDs, c.ID)
		}
	}
	for _, w := range all {
		if repairIntersects(out.Scopes, w.TrackOrdinal, w.Core) {
			out.Windows = append(out.Windows, w)
			if len(out.Windows) > p.MaxWindows {
				return out, contracts.Fail("output_limit")
			}
		}
	}
	sort.Strings(out.RemovedCueIDs)
	return out, nil
}
func RepairTargetsFromRanges(tracks []TrackPlan, ranges []TrackSelection) ([]RepairTarget, error) {
	if len(ranges) == 0 || len(ranges) > 32 {
		return nil, contracts.Fail("input_limit")
	}
	out := []RepairTarget{}
	for _, r := range ranges {
		lane := -1
		for i, t := range tracks {
			if trackKey(t.StreamIndex, t.Channel) == trackKey(r.StreamIndex, r.Channel) {
				lane = i
				break
			}
		}
		if lane < 0 || r.EndSample == nil || r.StartSample < tracks[lane].StartSample || *r.EndSample > tracks[lane].EndSample || *r.EndSample <= r.StartSample {
			return nil, contracts.Fail("invalid_request")
		}
		t := tracks[lane]
		t.StartSample = r.StartSample
		t.EndSample = *r.EndSample
		span, e := TrackCoverageSpan(t)
		if e != nil {
			return nil, e
		}
		out = append(out, RepairTarget{TrackOrdinal: lane, CoverageSpan: span})
	}
	return mergeRepairTargets(out), nil
}
func CompareCoverage(before, after CoverageReport) (bool, string) {
	if before.SourceDigest != after.SourceDigest || before.SourceRevision != after.SourceRevision || before.SourceMapDigest != after.SourceMapDigest || before.EvidenceDigest != after.EvidenceDigest || before.OptionsDigest != after.OptionsDigest || len(before.Tracks) != len(after.Tracks) {
		return false, "incomparable_evidence"
	}
	improved := false
	for i, b := range before.Tracks {
		a := after.Tracks[i]
		if b.TrackOrdinal != a.TrackOrdinal || b.Selected != a.Selected || b.Metrics.ObservedSpeechUS != a.Metrics.ObservedSpeechUS {
			return false, "incomparable_evidence"
		}
		x, y := b.Metrics, a.Metrics
		if y.UncoveredObservedSpeechUS > x.UncoveredObservedSpeechUS || y.DuplicateOverlapUS > x.DuplicateOverlapUS || y.BoundaryInconsistencies > x.BoundaryInconsistencies || y.CueCoverageUS < x.CueCoverageUS || y.RepeatedTextOverlapUS > x.RepeatedTextOverlapUS {
			return false, "protected_metric_worsened"
		}
		improved = improved || y.UncoveredObservedSpeechUS < x.UncoveredObservedSpeechUS || (y.DuplicateOverlapUS < x.DuplicateOverlapUS && y.RepeatedTextOverlapUS < x.RepeatedTextOverlapUS) || y.BoundaryInconsistencies < x.BoundaryInconsistencies
	}
	if !improved {
		return false, "not_improved"
	}
	return true, "improved"
}
