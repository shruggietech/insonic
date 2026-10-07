// SPDX-License-Identifier: Apache-2.0
package speakers

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"math/big"
	"sort"
	"strconv"
)

func integer(value int) string { return strconv.Itoa(value) }

type SourceInterval struct {
	Start string `json:"start_seconds"`
	End   string `json:"end_seconds"`
}
type SelectedEvidence struct {
	Reference catalog.CurrentReference `json:"reference"`
	Intervals []SourceInterval         `json:"source_intervals"`
	Untimed   bool                     `json:"untimed"`
}
type Selection struct {
	Items       []SelectedEvidence `json:"items"`
	Diagnostics []Diagnostic       `json:"diagnostics"`
	Duplicates  int                `json:"duplicates"`
	Partial     bool               `json:"partial"`
}

// Select operates on freshly resolved ephemeral evidence. It never substitutes
// cue coverage for an untimed attribution, and never persists copied intervals.
func Select(evidence []catalog.ResolvedEvidence) (out Selection, e error) {
	return selectEvidence(evidence, nil)
}

// SelectWithPrior uses catalog comparisons against all earlier current evidence
// in the fenced selection, so neither deduplication nor diagnostics restart at
// a page boundary. The callback supplies no copied intervals or assignments.
func SelectWithPrior(evidence []catalog.ResolvedEvidence, compare func(catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error)) (Selection, error) {
	if compare == nil {
		return Selection{}, contracts.Fail("invalid_request")
	}
	return selectEvidence(evidence, compare)
}

func selectEvidence(evidence []catalog.ResolvedEvidence, compare func(catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error)) (out Selection, e error) {
	if len(evidence) > 100 {
		return out, contracts.Fail("invalid_request")
	}
	out.Items = []SelectedEvidence{}
	out.Diagnostics = []Diagnostic{}
	counts := map[string]int{}
	seen := map[string]bool{}
	voices := map[string]map[string]bool{}
	type timed struct {
		source     string
		start, end *big.Rat
	}
	intervals := []timed{}
	for _, resolved := range evidence {
		ref := resolved.Reference
		var cue struct {
			ID          string `json:"id"`
			Assignments []struct {
				ID    string `json:"speaker_id"`
				Start *int64 `json:"start_milliseconds"`
				End   *int64 `json:"end_milliseconds"`
			} `json:"speaker_attributions"`
		}
		var source struct {
			Numerator   string `json:"start_numerator"`
			Denominator string `json:"start_denominator"`
			Stream      *int   `json:"stream_index"`
			Channel     *int   `json:"channel"`
		}
		if json.Unmarshal(resolved.Cue, &cue) != nil || json.Unmarshal(resolved.SourceMap, &source) != nil || cue.ID != ref.CueID || resolved.Mapping.SpeakerID != ref.SpeakerID {
			return out, contracts.Fail("invalid_request")
		}
		flags := map[int]catalog.PriorEvidenceComparison{}
		if compare != nil {
			comparisons, err := compare(ref)
			if err != nil {
				return out, err
			}
			matchedCount := 0
			for _, assignment := range cue.Assignments {
				if assignment.ID == ref.LocalSpeakerID {
					matchedCount++
				}
			}
			if len(comparisons) != matchedCount {
				return out, contracts.Fail("conflict")
			}
			localVoices := int64(0)
			firstReference := false
			for index, comparison := range comparisons {
				ordinal := comparison.AssignmentOrdinal
				if ordinal < 0 || ordinal >= len(cue.Assignments) || cue.Assignments[ordinal].ID != ref.LocalSpeakerID || comparison.LocalVoices < 1 {
					return out, contracts.Fail("conflict")
				}
				if _, exists := flags[ordinal]; exists {
					return out, contracts.Fail("conflict")
				}
				assignment := cue.Assignments[ordinal]
				if assignment.Start == nil && (comparison.Duplicate || comparison.Overlap) {
					return out, contracts.Fail("conflict")
				}
				if index == 0 {
					localVoices = comparison.LocalVoices
					firstReference = comparison.FirstReference
				} else if localVoices != comparison.LocalVoices || firstReference != comparison.FirstReference {
					return out, contracts.Fail("conflict")
				}
				flags[ordinal] = comparison
			}
			// All assignment comparisons repeat the reference-level facts. Count
			// them once, only on the globally first reference of this recording.
			if firstReference && localVoices > 1 {
				counts["multiple_local_voices"]++
			}
		}
		item := SelectedEvidence{Reference: ref, Intervals: []SourceInterval{}}
		matched := false
		channel := "downmix"
		if source.Channel != nil {
			channel = integer(*source.Channel)
		}
		sourceKey := ref.SourceDigest + ":" + ref.RecordingID
		if source.Stream != nil {
			sourceKey = ref.SourceDigest + ":" + integer(*source.Stream) + ":" + channel
		} else {
			counts["source_stream_unknown"]++
		}
		key := ref.SpeakerID + ":" + ref.RecordingID
		if voices[key] == nil {
			voices[key] = map[string]bool{}
		}
		voices[key][ref.LocalSpeakerID] = true
		for ordinal, a := range cue.Assignments {
			if a.ID != ref.LocalSpeakerID {
				continue
			}
			matched = true
			if a.Start == nil || a.End == nil {
				if a.Start != nil || a.End != nil {
					return out, contracts.Fail("invalid_request")
				}
				item.Untimed = true
				counts["untimed_participation"]++
				continue
			}
			if *a.Start < 0 || *a.End <= *a.Start {
				return out, contracts.Fail("invalid_request")
			}
			start := new(big.Rat).SetFrac(big.NewInt(*a.Start), big.NewInt(1000))
			end := new(big.Rat).SetFrac(big.NewInt(*a.End), big.NewInt(1000))
			if compare != nil {
				flag := flags[ordinal]
				if flag.Duplicate {
					out.Duplicates++
					continue
				}
				if flag.Overlap {
					counts["overlapping_evidence"]++
				}
			} else {
				dedup := sourceKey + ":" + start.RatString() + ":" + end.RatString()
				if seen[dedup] {
					out.Duplicates++
					continue
				}
				seen[dedup] = true
				for _, prior := range intervals {
					if prior.source == sourceKey && start.Cmp(prior.end) < 0 && end.Cmp(prior.start) > 0 {
						counts["overlapping_evidence"]++
						break
					}
				}
				intervals = append(intervals, timed{sourceKey, start, end})
			}
			item.Intervals = append(item.Intervals, SourceInterval{start.RatString(), end.RatString()})
		}
		if !matched {
			return out, contracts.Fail("conflict")
		}
		if item.Untimed || len(item.Intervals) > 0 {
			out.Items = append(out.Items, item)
		}
	}
	if compare == nil {
		for _, set := range voices {
			if len(set) > 1 {
				counts["multiple_local_voices"]++
			}
		}
	}
	if out.Duplicates > 0 {
		counts["duplicate_evidence"] = out.Duplicates
	}
	codes := []string{}
	for code := range counts {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{code, counts[code]})
	}
	return
}
