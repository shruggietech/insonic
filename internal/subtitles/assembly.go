// SPDX-License-Identifier: Apache-2.0
package subtitles

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"

	"github.com/shruggietech/insonic/internal/contracts"
)

const nanosecondsPerMillisecond int64 = 1000000
const MaxTurns = 65536
const MaxAssignments = 65536

type Turn struct {
	SpeakerID string `json:"speaker_id"`
	StartNS   int64  `json:"start_ns"`
	EndNS     int64  `json:"end_ns"`
}
type Participation struct {
	CueID     string `json:"cue_id"`
	SpeakerID string `json:"speaker_id"`
}
type Diagnostic struct {
	Code    string `json:"code"`
	Pointer string `json:"pointer,omitempty"`
	Message string `json:"message"`
}
type Assembly struct {
	Document    json.RawMessage `json:"document"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
}
type Cue struct {
	ID      string `json:"id"`
	StartMS int64  `json:"start_milliseconds"`
	EndMS   int64  `json:"end_milliseconds"`
}

func DocumentCues(data []byte) ([]Cue, error) {
	if err := ValidateDocument(data); err != nil {
		return nil, err
	}
	var doc semanticDocument
	json.Unmarshal(data, &doc)
	cues := make([]Cue, len(doc.Cues))
	for i, cue := range doc.Cues {
		cues[i] = Cue{ID: cue.ID, StartMS: cue.Timing.Start, EndMS: cue.Timing.End}
	}
	return cues, nil
}

// AssembleDocument replaces only consumer fields. It preserves source/native
// observations and integer instants, while the exact source clock remains in
// the recording's external rational map. Engine turns are never persisted here.
func AssembleDocument(data []byte, durationNS *int64, turns []Turn, participation []Participation) (Assembly, error) {
	result := Assembly{Diagnostics: []Diagnostic{}}
	if err := ValidateDocument(data); err != nil {
		return result, err
	}
	if len(turns) > MaxTurns || len(participation) > MaxAssignments {
		return result, contracts.Fail("output_limit")
	}
	if durationNS != nil && *durationNS < 0 {
		return result, contracts.Fail("invalid_request")
	}
	for _, turn := range turns {
		if !contracts.ValidLocalSpeakerID(turn.SpeakerID) || turn.StartNS < 0 || turn.EndNS <= turn.StartNS || durationNS != nil && turn.EndNS > *durationNS {
			return result, contracts.Fail("invalid_request")
		}
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(data, &fields)
	var cues []map[string]json.RawMessage
	json.Unmarshal(fields["cues"], &cues)
	var doc semanticDocument
	json.Unmarshal(data, &doc)
	assignments := make([][]attribution, len(cues))
	cueIDs := map[string]int{}
	intervals := make([]cueInterval, 0, len(cues))
	for i, cue := range doc.Cues {
		cueIDs[cue.ID] = i
		delete(cues[i], "speaker_attributions")
		if cue.Timing.Start <= math.MaxInt64/nanosecondsPerMillisecond {
			start := cue.Timing.Start * nanosecondsPerMillisecond
			end := int64(math.MaxInt64)
			if cue.Timing.End <= math.MaxInt64/nanosecondsPerMillisecond {
				end = cue.Timing.End * nanosecondsPerMillisecond
			}
			if end > start {
				intervals = append(intervals, cueInterval{index: i, start: start, end: end})
			}
		}
	}
	sort.SliceStable(intervals, func(i, j int) bool { return intervals[i].start < intervals[j].start })
	tree := buildCueTree(intervals)
	total := 0
	appendDiagnostic := func(code, pointer, message string) error {
		if len(result.Diagnostics) >= 8192 {
			return contracts.Fail("output_limit")
		}
		result.Diagnostics = append(result.Diagnostics, Diagnostic{Code: code, Pointer: pointer, Message: message})
		return nil
	}
	for _, turn := range turns {
		intersections := []cueInterval{}
		tree.query(turn.StartNS, turn.EndNS, &intersections)
		coverage := make([]cueInterval, 0, len(intersections))
		for _, cue := range intersections {
			start, end := max(turn.StartNS, cue.start), min(turn.EndNS, cue.end)
			coverage = append(coverage, cueInterval{start: start, end: end})
			pointer := "/cues/" + integer(cue.index) + "/speaker_attributions/" + integer(len(assignments[cue.index]))
			if start != turn.StartNS || end != turn.EndNS {
				if err := appendDiagnostic("timing_clipped", pointer, "The turn was explicitly intersected with cue coverage."); err != nil {
					return result, err
				}
			}
			projectedStart := start / nanosecondsPerMillisecond
			if start%nanosecondsPerMillisecond != 0 {
				projectedStart++
			}
			projectedEnd := end / nanosecondsPerMillisecond
			if projectedEnd <= projectedStart {
				if err := appendDiagnostic("timing_collapsed", pointer, "Inward millisecond projection cannot represent this positive interval; no assignment was invented."); err != nil {
					return result, err
				}
				continue
			}
			if start%nanosecondsPerMillisecond != 0 || end%nanosecondsPerMillisecond != 0 {
				if err := appendDiagnostic("timing_rounded", pointer, "Endpoints were deliberately projected inward to integer milliseconds."); err != nil {
					return result, err
				}
			}
			if len(assignments[cue.index]) >= 1024 || total >= MaxAssignments {
				return result, contracts.Fail("output_limit")
			}
			assignments[cue.index] = append(assignments[cue.index], attribution{SpeakerID: turn.SpeakerID, Start: &projectedStart, End: &projectedEnd})
			total++
		}
		sort.Slice(coverage, func(i, j int) bool { return coverage[i].start < coverage[j].start })
		covered, end := int64(0), turn.StartNS
		for _, part := range coverage {
			if part.end <= end {
				continue
			}
			start := max(part.start, end)
			covered += part.end - start
			end = part.end
		}
		if covered < turn.EndNS-turn.StartNS {
			if err := appendDiagnostic("timing_uncovered", "/cues", "Some of the turn has no subtitle cue coverage; no cue or timestamp was invented."); err != nil {
				return result, err
			}
		}
	}
	for _, participant := range participation {
		index, ok := cueIDs[participant.CueID]
		if !ok || !contracts.ValidID(participant.SpeakerID) {
			return result, contracts.Fail("invalid_request")
		}
		if len(assignments[index]) >= 1024 || total >= MaxAssignments {
			return result, contracts.Fail("output_limit")
		}
		assignments[index] = append(assignments[index], attribution{SpeakerID: participant.SpeakerID})
		total++
	}
	for i, items := range assignments {
		if len(items) > 0 {
			encoded, err := json.Marshal(items)
			if err != nil {
				return result, contracts.Fail("invalid_request")
			}
			cues[i]["speaker_attributions"] = encoded
		}
	}
	fields["cues"], _ = json.Marshal(cues)
	delete(fields, "media_timing")
	if durationNS != nil {
		fields["media_timing"], _ = json.Marshal(semanticTiming{Duration: *durationNS / nanosecondsPerMillisecond})
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return result, contracts.Fail("invalid_request")
	}
	if len(encoded) > MaxDocumentBytes {
		return result, contracts.Fail("output_limit")
	}
	if err = ValidateDocument(encoded); err != nil {
		return result, err
	}
	result.Document = encoded
	return result, nil
}

type cueInterval struct {
	index      int
	start, end int64
}
type cueTree struct {
	cue                      cueInterval
	left, right              *cueTree
	minimumStart, maximumEnd int64
}

func buildCueTree(items []cueInterval) *cueTree {
	if len(items) == 0 {
		return nil
	}
	middle := len(items) / 2
	node := &cueTree{cue: items[middle], left: buildCueTree(items[:middle]), right: buildCueTree(items[middle+1:]), minimumStart: items[0].start, maximumEnd: items[middle].end}
	if node.left != nil {
		node.maximumEnd = max(node.maximumEnd, node.left.maximumEnd)
	}
	if node.right != nil {
		node.maximumEnd = max(node.maximumEnd, node.right.maximumEnd)
	}
	return node
}
func (node *cueTree) query(start, end int64, result *[]cueInterval) {
	if node == nil || node.maximumEnd <= start || node.minimumStart >= end {
		return
	}
	node.left.query(start, end, result)
	if node.cue.start < end && node.cue.end > start {
		*result = append(*result, node.cue)
	}
	node.right.query(start, end, result)
}
func integer(value int) string { return strconv.Itoa(value) }
