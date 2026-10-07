// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/subtitles"
	"math"
	"sort"
)

type reuseEvent struct {
	time       int64
	cue, delta int
}

func currentTurns(doc []byte) ([]subtitles.Turn, []subtitles.Participation, error) {
	if e := subtitles.ValidateDocument(doc); e != nil {
		return nil, nil, e
	}
	var d struct {
		Cues []struct {
			ID          string `json:"id"`
			Assignments []struct {
				SpeakerID string `json:"speaker_id"`
				Start     *int64 `json:"start_milliseconds"`
				End       *int64 `json:"end_milliseconds"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	json.Unmarshal(doc, &d)
	events := map[string][]reuseEvent{}
	order := []string{}
	participation := []subtitles.Participation{}
	total := 0
	for ci, cue := range d.Cues {
		for _, a := range cue.Assignments {
			total++
			if total > 65536 {
				return nil, nil, contracts.Fail("output_limit")
			}
			if a.Start == nil && a.End == nil {
				participation = append(participation, subtitles.Participation{CueID: cue.ID, SpeakerID: a.SpeakerID})
				continue
			}
			if a.Start == nil || a.End == nil || *a.Start < 0 || *a.End <= *a.Start || *a.End > math.MaxInt64/1000000 {
				return nil, nil, contracts.Fail("invalid_request")
			}
			if _, exists := events[a.SpeakerID]; !exists {
				order = append(order, a.SpeakerID)
			}
			events[a.SpeakerID] = append(events[a.SpeakerID], reuseEvent{*a.Start, ci, 1}, reuseEvent{*a.End, ci, -1})
		}
	}
	turns := []subtitles.Turn{}
	for _, voice := range order {
		points := events[voice]
		sort.Slice(points, func(i, j int) bool { return points[i].time < points[j].time })
		counts := map[int]int{}
		frequency := [1025]int{}
		maximum := 0
		last := [1024]int{}
		for i := range last {
			last[i] = -1
		}
		for i := 0; i < len(points); {
			start := points[i].time
			j := i
			for j < len(points) && points[j].time == start {
				point := points[j]
				old := counts[point.cue]
				next := old + point.delta
				if next < 0 || next > 1024 {
					return nil, nil, contracts.Fail("invalid_request")
				}
				if old > 0 {
					frequency[old]--
				}
				if next > 0 {
					frequency[next]++
				}
				counts[point.cue] = next
				if next > maximum {
					maximum = next
				}
				j++
			}
			for maximum > 0 && frequency[maximum] == 0 {
				maximum--
			}
			if j == len(points) {
				break
			}
			end := points[j].time
			// Overlapping cue containers describe the same activity. Take the maximum
			// multiplicity of any one cue, never sum the same activity across cues.
			for layer := 0; layer < maximum; layer++ {
				if index := last[layer]; index >= 0 && turns[index].EndNS == start*1000000 {
					turns[index].EndNS = end * 1000000
				} else {
					if len(turns) >= 65536 {
						return nil, nil, contracts.Fail("output_limit")
					}
					last[layer] = len(turns)
					turns = append(turns, subtitles.Turn{SpeakerID: voice, StartNS: start * 1000000, EndNS: end * 1000000})
				}
			}
			i = j
		}
	}
	return turns, participation, nil
}

type reuseCue struct {
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
	Timing  json.RawMessage `json:"timing"`
}

func equivalentCueField(a, b json.RawMessage) bool {
	canonical := func(raw json.RawMessage) []byte {
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil {
			return nil
		}
		encoded, _ := json.Marshal(value)
		return encoded
	}
	return bytes.Equal(canonical(a), canonical(b))
}
func reuseParticipation(previous, current []byte, input []subtitles.Participation) ([]subtitles.Participation, []subtitles.Diagnostic, error) {
	if e := subtitles.ValidateDocument(previous); e != nil {
		return nil, nil, e
	}
	if len(current) > 0 {
		if e := subtitles.ValidateDocument(current); e != nil {
			return nil, nil, e
		}
	}
	var before, after struct {
		Cues []reuseCue `json:"cues"`
	}
	json.Unmarshal(previous, &before)
	if len(current) > 0 {
		json.Unmarshal(current, &after)
	}
	originals := map[string]reuseCue{}
	for _, cue := range before.Cues {
		originals[cue.ID] = cue
	}
	latest := map[string]reuseCue{}
	for _, cue := range after.Cues {
		latest[cue.ID] = cue
	}
	out := []subtitles.Participation{}
	diagnostics := []subtitles.Diagnostic{}
	for _, item := range input {
		old, oldOK := originals[item.CueID]
		fresh, newOK := latest[item.CueID]
		if oldOK && newOK && equivalentCueField(old.Payload, fresh.Payload) && equivalentCueField(old.Timing, fresh.Timing) {
			out = append(out, item)
		} else {
			diagnostics = append(diagnostics, subtitles.Diagnostic{Code: "untimed_reuse_omitted", Message: "Untimed participation was omitted because its cue wording or timing changed."})
		}
	}
	return out, diagnostics, nil
}
