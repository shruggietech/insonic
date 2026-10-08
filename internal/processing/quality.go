// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"math"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
)

// QualityConfig controls optional aggregate analysis, never document acceptance.
type QualityConfig struct {
	Enabled            *bool    `json:"enabled,omitempty"`
	MinSpeechCoverage  *float64 `json:"min_speech_coverage,omitempty"`
	MaxOverlapFraction *float64 `json:"max_overlap_fraction,omitempty"`
	ShortTurnUS        *int64   `json:"short_turn_us,omitempty"`
}

func ValidateQuality(c QualityConfig) error {
	for _, value := range []*float64{c.MinSpeechCoverage, c.MaxOverlapFraction} {
		if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value > 1) {
			return contracts.Fail("invalid_request")
		}
	}
	if c.ShortTurnUS != nil && (*c.ShortTurnUS < 1 || *c.ShortTurnUS > 60000000) {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func validTurn(t Turn, durationUS int64) bool {
	return t.Label != "" && len(t.Label) <= 256 && utf8.ValidString(t.Label) && !strings.ContainsFunc(t.Label, unicode.IsControl) && t.StartUS >= 0 && t.EndUS > t.StartUS && t.EndUS <= durationUS
}

// Quality measures distinct active voices, so duplicate same-voice intervals
// cannot manufacture overlap. It validates timing even when analysis is disabled.
func Quality(turns []Turn, durationUS int64, c QualityConfig) ([]Diagnostic, error) {
	if ValidateQuality(c) != nil || durationUS < 0 || len(turns) > 20000 {
		return nil, contracts.Fail("invalid_request")
	}
	type event struct {
		at    int64
		label string
		delta int
	}
	events := make([]event, 0, len(turns)*2)
	shortUS := int64(250000)
	if c.ShortTurnUS != nil {
		shortUS = *c.ShortTurnUS
	}
	short := int64(0)
	for _, t := range turns {
		if !validTurn(t, durationUS) {
			return nil, contracts.Fail("invalid_engine_output")
		}
		events = append(events, event{t.StartUS, t.Label, 1}, event{t.EndUS, t.Label, -1})
		if t.EndUS-t.StartUS < shortUS {
			short++
		}
	}
	if c.Enabled != nil && !*c.Enabled {
		return []Diagnostic{}, nil
	}
	sort.Slice(events, func(i, j int) bool { return events[i].at < events[j].at })
	labels := map[string]int{}
	active := 0
	last, speech, overlap := int64(0), int64(0), int64(0)
	for _, e := range events {
		delta := e.at - last
		if active > 0 {
			speech += delta
		}
		if active > 1 {
			overlap += delta
		}
		before := labels[e.label]
		labels[e.label] += e.delta
		if before == 0 && labels[e.label] > 0 {
			active++
		}
		if before > 0 && labels[e.label] == 0 {
			active--
		}
		last = e.at
	}
	ds := []Diagnostic{}
	if durationUS > 0 {
		coverage, over := float64(speech)/float64(durationUS), float64(overlap)/float64(durationUS)
		ds = append(ds, Diagnostic{Code: "speech_coverage_fraction", Value: &coverage}, Diagnostic{Code: "overlap_fraction", Value: &over})
		if c.MinSpeechCoverage != nil && coverage < *c.MinSpeechCoverage {
			ds = append(ds, Diagnostic{Code: "speech_coverage_below_threshold", Count: 1})
		}
		if c.MaxOverlapFraction != nil && over > *c.MaxOverlapFraction {
			ds = append(ds, Diagnostic{Code: "overlap_above_threshold", Count: 1})
		}
	}
	if short > 0 {
		ds = append(ds, Diagnostic{Code: "short_turns", Count: short})
	}
	if len(turns) == 0 {
		ds = append(ds, Diagnostic{Code: "no_speech", Count: 1})
	}
	return ds, nil
}

// ApplyDiarizationQuality replaces the worker's legacy fixed-quality aggregates
// before source-clock projection. Physical model-boundary diagnostics survive.
func ApplyDiarizationQuality(r DiarizationResult, durationUS int64, config *QualityConfig) (DiarizationResult, error) {
	c := QualityConfig{}
	if config != nil {
		c = *config
	}
	diagnostics, e := Quality(r.Turns, durationUS, c)
	if e != nil {
		return DiarizationResult{}, e
	}
	keep := []Diagnostic{}
	for _, d := range r.Diagnostics {
		switch d.Code {
		case "speech_coverage_fraction", "overlap_fraction", "short_turns", "no_speech", "speech_coverage_below_threshold", "overlap_above_threshold":
			continue
		}
		keep = append(keep, d)
	}
	r.Diagnostics = append(keep, diagnostics...)
	if r.Provenance == nil {
		r.Provenance = map[string]any{}
	}
	r.Provenance["quality_diagnostics_enabled"] = c.Enabled == nil || *c.Enabled
	return r, nil
}
