// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"unicode"

	"github.com/shruggietech/insonic/internal/contracts"
)

const maxReconciledAtoms = 1000000

// WindowEvidence retains worker-relative recognition evidence and already
// source-projected diarization. OutputDigest binds the verified checkpoint bytes.
type WindowEvidence struct {
	Window       WindowPlan        `json:"window"`
	OutputDigest string            `json:"output_sha256"`
	Recognition  RecognitionResult `json:"recognition"`
	Diarization  DiarizationResult `json:"diarization"`
}
type CueSource struct {
	Ordinal      int      `json:"ordinal"`
	TrackOrdinal int      `json:"track_ordinal"`
	StreamIndex  int      `json:"stream_index"`
	Channel      *int     `json:"channel,omitempty"`
	WindowIDs    []string `json:"window_ids"`
	StartUS      int64    `json:"start_us"`
	EndUS        int64    `json:"end_us"`
}
type VoiceSource struct {
	Label        string `json:"label"`
	TrackOrdinal int    `json:"track_ordinal"`
	WindowID     string `json:"window_id"`
}

type ReconciledResult struct {
	VoiceSources []VoiceSource
	Recognition  RecognitionResult
	Turns        []Turn
	CueSources   []CueSource
	Diagnostics  []Diagnostic
	Digest       string
}
type stitchAtom struct {
	start, end            int64
	text                  string
	key                   string
	window, segment, word int
	lane                  int
	core                  bool
	margin                int64
	timedWord             bool
	sources               []string
}
type stitchCore struct{ start, end int64 }

func projectWindowUS(mapping SourceMap, value int64, ceil bool) (int64, error) {
	origin, ok := new(big.Rat).SetString(mapping.StartNumerator + "/" + mapping.StartDenominator)
	if !ok {
		return 0, contracts.Fail("invalid_timing")
	}
	origin.Mul(origin, big.NewRat(1000000, 1))
	origin.Add(origin, big.NewRat(value, 1))
	return roundedRat(origin, ceil)
}
func overlapToken(text string) string {
	// Matching ignores punctuation/case but retained output is always original
	// text. Time overlap and distinct windows are independently required.
	return strings.Map(func(r rune) rune {
		if unicode.IsPunct(r) || unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, text)
}
func temporalDuplicate(a, b stitchAtom) bool {
	overlap := min(a.end, b.end) - max(a.start, b.start)
	return overlap > 0 && overlap*2 >= min(a.end-a.start, b.end-b.start)
}

// ReconcileWindows joins only temporally corroborated observations in the same
// track/channel lane. Ambiguous alternatives remain visible with diagnostics.
func ReconcileWindows(windows []WindowEvidence) (ReconciledResult, error) {
	out := ReconciledResult{Turns: []Turn{}, CueSources: []CueSource{}, Diagnostics: []Diagnostic{}}
	if len(windows) > MaxPlanWindows {
		return out, contracts.Fail("output_limit")
	}
	ordered := append([]WindowEvidence(nil), windows...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Window.Ordinal < ordered[j].Window.Ordinal })
	seen := map[string]bool{}
	seenOrdinals := map[int]bool{}
	seenVoices := map[string]bool{}
	cores := map[int][]stitchCore{}
	atoms := []stitchAtom{}
	observations := []map[string]any{}
	noSpeech := true
	for wi, e := range ordered {
		w := e.Window
		if !contracts.ValidID(w.ID) || seen[w.ID] || seenOrdinals[w.Ordinal] || w.Ordinal < 0 || !pinnedDigest.MatchString(e.OutputDigest) || w.TrackOrdinal < 0 {
			return out, contracts.Fail("invalid_engine_output")
		}
		seen[w.ID] = true
		seenOrdinals[w.Ordinal] = true
		duration, err := MappedDuration(w.Mapping)
		if err != nil {
			return out, err
		}
		a, b, err := WindowCoreUS(w)
		if err != nil || b < a {
			return out, contracts.Fail("invalid_timing")
		}
		ca, err := projectWindowUS(w.Mapping, a, true)
		if err != nil {
			return out, err
		}
		cb, err := projectWindowUS(w.Mapping, b, false)
		if err != nil {
			return out, err
		}
		cores[w.TrackOrdinal] = append(cores[w.TrackOrdinal], stitchCore{ca, cb})
		if ValidateRecognitionSegments(e.Recognition.Segments, duration) != nil || e.Recognition.NoSpeech && len(e.Recognition.Segments) > 0 {
			return out, contracts.Fail("invalid_engine_output")
		}
		if len(e.Recognition.Segments) > 0 && !e.Recognition.NoSpeech && e.Diarization.NoSpeech {
			out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "recognition_diarization_disagreement", Count: 1})
		}
		if !e.Recognition.NoSpeech {
			noSpeech = false
		}
		if len(e.Recognition.Segments) == 0 && len(strings.TrimSpace(string(e.Recognition.SRT))) > 0 {
			return out, contracts.Fail("invalid_engine_output")
		}
		observations = append(observations, map[string]any{"window_id": w.ID, "output_sha256": e.OutputDigest, "recognition": e.Recognition.Provenance, "diarization": e.Diarization.Provenance})
		add := func(start, end int64, text string, segment, word int, timed bool) error {
			if strings.TrimSpace(text) == "" {
				return nil
			}
			if len(atoms) >= maxReconciledAtoms {
				return contracts.Fail("output_limit")
			}
			first, err := projectWindowUS(w.Mapping, start, true)
			if err != nil {
				return err
			}
			last, err := projectWindowUS(w.Mapping, end, false)
			if err != nil {
				return err
			}
			if first < 0 || last < 0 {
				out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "negative_source_interval_unrepresentable", Count: 1})
				return nil
			}
			if first >= last {
				out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "word_interval_unrepresentable", Count: 1})
				return nil
			}
			midpoint := start + (end-start)/2
			atoms = append(atoms, stitchAtom{start: first, end: last, text: text, key: overlapToken(text), window: wi, segment: segment, word: word, lane: w.TrackOrdinal, core: midpoint >= a && midpoint < b, margin: min(start, duration-end), timedWord: timed, sources: []string{w.ID}})
			return nil
		}
		for si, segment := range e.Recognition.Segments {
			usableWords := len(segment.Words) > 0
			var wordText strings.Builder
			for _, word := range segment.Words {
				wordText.WriteString(word.Text)
			}
			if strings.Join(strings.Fields(wordText.String()), " ") != strings.Join(strings.Fields(segment.Text), " ") {
				usableWords = false
			}
			for _, word := range segment.Words {
				if word.StartUS >= word.EndUS {
					usableWords = false
				}
			}
			if !usableWords {
				if err := add(segment.StartUS, segment.EndUS, segment.Text, si, -1, false); err != nil {
					return out, err
				}
				out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "window_word_timing_unavailable", Count: 1})
			} else {
				for i, word := range segment.Words {
					if err := add(word.StartUS, word.EndUS, word.Text, si, i, true); err != nil {
						return out, err
					}
				}
			}
		}
		contextStart, err := projectWindowUS(w.Mapping, 0, true)
		if err != nil {
			return out, err
		}
		contextEnd, err := projectWindowUS(w.Mapping, duration, false)
		if err != nil {
			return out, err
		}
		if e.Diarization.NoSpeech && len(e.Diarization.Turns) > 0 {
			return out, contracts.Fail("invalid_engine_output")
		}
		for _, turn := range e.Diarization.Turns {
			if turn.StartUS < 0 || turn.StartUS < contextStart || turn.EndUS > contextEnd || turn.EndUS <= turn.StartUS || !contracts.ValidTextLength(turn.Label, 1, 256) {
				return out, contracts.Fail("invalid_engine_output")
			}
			start, end := max(turn.StartUS, ca), min(turn.EndUS, cb)
			if start >= end || end <= 0 {
				continue
			}
			start = max(int64(0), start)
			// Labels are local to each inference window. Repeated raw labels do not
			// establish an acoustic identity across windows or tracks.
			sum := sha256.Sum256([]byte(w.ID + "\x00" + turn.Label))
			label := "window-" + hex.EncodeToString(sum[:16])
			out.Turns = append(out.Turns, Turn{Label: label, StartUS: start, EndUS: end})
			if !seenVoices[label] {
				seenVoices[label] = true
				out.VoiceSources = append(out.VoiceSources, VoiceSource{Label: label, TrackOrdinal: w.TrackOrdinal, WindowID: w.ID})
			}
		}
		out.Diagnostics = append(out.Diagnostics, e.Recognition.Diagnostics...)
		out.Diagnostics = append(out.Diagnostics, e.Diarization.Diagnostics...)
	}
	for lane, intervals := range cores {
		sort.Slice(intervals, func(i, j int) bool { return intervals[i].start < intervals[j].start })
		cores[lane] = intervals
	}
	sort.Slice(atoms, func(i, j int) bool {
		a, b := atoms[i], atoms[j]
		if a.lane != b.lane {
			return a.lane < b.lane
		}
		if a.start != b.start {
			return a.start < b.start
		}
		if a.window != b.window {
			return a.window < b.window
		}
		if a.segment != b.segment {
			return a.segment < b.segment
		}
		return a.word < b.word
	})
	retained := []stitchAtom{}
	// An active sweep limits comparison to the current temporal overlap, rather
	// than comparing every token in a long recording with every other token.
	active := []int{}
	comparisons := 0
	lane := -1
	for _, atom := range atoms {
		intervals := cores[atom.lane]
		coreIndex := sort.Search(len(intervals), func(i int) bool { return intervals[i].end > atom.start })
		eligible := coreIndex < len(intervals) && intervals[coreIndex].start < atom.end
		if !eligible {
			continue
		}
		if atom.lane != lane {
			active = nil
			lane = atom.lane
		}
		next := active[:0]
		matched := -1
		ambiguous := false
		bestOverlap := int64(-1)
		for _, index := range active {
			comparisons++
			if comparisons > 20000000 {
				return out, contracts.Fail("output_limit")
			}
			prior := retained[index]
			if prior.end <= atom.start {
				continue
			}
			next = append(next, index)
			if len(next) > 20000 {
				return out, contracts.Fail("output_limit")
			}
			if prior.window == atom.window {
				continue
			}
			sameWindow := false
			for _, id := range prior.sources {
				if id == ordered[atom.window].Window.ID {
					sameWindow = true
					break
				}
			}
			if sameWindow {
				continue
			}
			overlap := min(prior.end, atom.end) - max(prior.start, atom.start)
			if overlap <= 0 {
				continue
			}
			if prior.key != "" && prior.key == atom.key && prior.timedWord == atom.timedWord && temporalDuplicate(prior, atom) {
				if overlap > bestOverlap {
					matched = index
					bestOverlap = overlap
				}
			} else {
				ambiguous = true
			}
		}
		active = next
		if matched >= 0 {
			prior := retained[matched]
			ids := append(append([]string(nil), prior.sources...), atom.sources...)
			sort.Strings(ids)
			if atom.core && !prior.core || atom.core == prior.core && atom.margin > prior.margin {
				retained[matched] = atom
			}
			retained[matched].sources = ids
			out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "window_overlap_duplicate_reconciled", Count: 1})
			continue
		}
		if ambiguous {
			out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "window_overlap_ambiguous", Count: 1})
		}
		retained = append(retained, atom)
		active = append(active, len(retained)-1)
	}
	sort.Slice(retained, func(i, j int) bool {
		a, b := retained[i], retained[j]
		if a.start != b.start {
			return a.start < b.start
		}
		if a.lane != b.lane {
			return a.lane < b.lane
		}
		if a.window != b.window {
			return a.window < b.window
		}
		if a.segment != b.segment {
			return a.segment < b.segment
		}
		return a.word < b.word
	})
	segments := []RecognitionSegment{}
	var last *stitchAtom
	for _, atom := range retained {
		// Tokens remain in recognizer order. Split at lane/window/segment changes
		// rather than joining unrelated simultaneous speech into one caption.
		if last != nil && last.window == atom.window && last.segment == atom.segment && last.timedWord && atom.timedWord && atom.word == last.word+1 && last.lane == atom.lane {
			i := len(segments) - 1
			segments[i].Text += atom.text
			segments[i].EndUS = max(segments[i].EndUS, atom.end)
			segments[i].Words = append(segments[i].Words, RecognitionWord{StartUS: atom.start, EndUS: atom.end, Text: atom.text})
			out.CueSources[i].EndUS = segments[i].EndUS
			for _, id := range atom.sources {
				found := false
				for _, prior := range out.CueSources[i].WindowIDs {
					found = found || prior == id
				}
				if !found {
					out.CueSources[i].WindowIDs = append(out.CueSources[i].WindowIDs, id)
				}
			}
			sort.Strings(out.CueSources[i].WindowIDs)
		} else {
			segment := RecognitionSegment{StartUS: atom.start, EndUS: atom.end, Text: atom.text}
			if atom.timedWord {
				segment.Words = []RecognitionWord{{StartUS: atom.start, EndUS: atom.end, Text: atom.text}}
			}
			segments = append(segments, segment)
			w := ordered[atom.window].Window
			out.CueSources = append(out.CueSources, CueSource{Ordinal: len(segments) - 1, TrackOrdinal: atom.lane, StreamIndex: w.StreamIndex, Channel: w.Channel, WindowIDs: atom.sources, StartUS: atom.start, EndUS: atom.end})
		}
		copy := atom
		last = &copy
	}
	// Cueson stores milliseconds. Discard only intervals its native contract
	// cannot represent, keeping the provenance ordinals exactly aligned.
	representable := segments[:0]
	sources := out.CueSources[:0]
	for i, segment := range segments {
		if segment.StartUS/1000+boolInt(segment.StartUS%1000 != 0) >= segment.EndUS/1000 {
			out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "submillisecond_segment_unrepresentable", Count: 1})
			continue
		}
		source := out.CueSources[i]
		source.Ordinal = len(representable)
		representable = append(representable, segment)
		sources = append(sources, source)
	}
	out.CueSources = sources
	maximum := int64(0)
	for _, segment := range representable {
		maximum = max(maximum, segment.EndUS)
	}
	// Render absolute source microseconds without treating a source offset as
	// decoded duration or multiplying it into a potentially overflowing count.
	result, err := renderAggregateRecognitionSegments(representable, noSpeech && len(representable) == 0, maximum)
	if err != nil {
		return out, err
	}
	encoded, err := json.Marshal(struct {
		Version int
		Windows []WindowEvidence
	}{1, ordered})
	if err != nil {
		return out, contracts.Fail("invalid_engine_output")
	}
	digest := sha256.Sum256(encoded)
	out.Digest = hex.EncodeToString(digest[:])
	result.Provenance = map[string]any{"reconciliation_version": 1, "reconciliation_sha256": out.Digest, "window_count": len(ordered), "window_observations": observations, "source_clock_projection": "exact-source-origin;ceil-start/floor-end-millisecond"}
	out.Diagnostics = compactStitchDiagnostics(append(out.Diagnostics, result.Diagnostics...))
	result.Diagnostics = out.Diagnostics
	out.Recognition = result
	sort.Slice(out.Turns, func(i, j int) bool {
		a, b := out.Turns[i], out.Turns[j]
		if a.StartUS != b.StartUS {
			return a.StartUS < b.StartUS
		}
		if a.EndUS != b.EndUS {
			return a.EndUS < b.EndUS
		}
		return a.Label < b.Label
	})
	return out, nil
}
func compactStitchDiagnostics(input []Diagnostic) []Diagnostic {
	output := []Diagnostic{}
	indices := map[string]int{}
	for _, item := range input {
		if item.Value != nil {
			output = append(output, item)
			continue
		}
		key := fmt.Sprint(item.Code)
		if index, ok := indices[key]; ok {
			output[index].Count += item.Count
		} else {
			indices[key] = len(output)
			output = append(output, item)
		}
	}
	return output
}

func boolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// ValidateWindowEvidence checks checkpoint content before it can be accepted.
func ValidateWindowEvidence(e WindowEvidence) error {
	_, err := ReconcileWindows([]WindowEvidence{e})
	return err
}
