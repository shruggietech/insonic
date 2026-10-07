// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/shruggietech/insonic/internal/contracts"
)

type RecognitionSegment struct {
	StartUS int64  `json:"start_us"`
	EndUS   int64  `json:"end_us"`
	Text    string `json:"text"`
}

// MappedDuration validates the exact source map independently of any engine.
func MappedDuration(m SourceMap) (int64, error) {
	if m.SampleCount < 0 || m.SampleCount > 604800*16000 || m.SampleRate != 16000 || len(m.StartNumerator) > 128 || len(m.StartDenominator) > 128 || m.StartNumerator == "" || m.StartDenominator == "" {
		return 0, contracts.Fail("invalid_timing")
	}
	numerator, ok := new(big.Int).SetString(m.StartNumerator, 10)
	if !ok {
		return 0, contracts.Fail("invalid_timing")
	}
	denominator, ok := new(big.Int).SetString(m.StartDenominator, 10)
	if !ok || denominator.Sign() <= 0 {
		return 0, contracts.Fail("invalid_timing")
	}
	_ = numerator
	return m.SampleCount * 1000000 / m.SampleRate, nil
}
func ValidateMappedAudio(path string, m SourceMap) error {
	duration, e := MappedDuration(m)
	if e != nil {
		return e
	}
	// Permit exact fractional final samples while bounding the physical waveform.
	count, e := pcmInfo(path, duration+1)
	if e != nil {
		return e
	}
	if count != m.SampleCount {
		return contracts.Fail("conflict")
	}
	return nil
}
func srtStamp(ms int64) string {
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

// RecognitionFromSegments converts mapped-relative integer intervals to native
// SRT before Cueson encoding, using the same conservative source-clock policy.
func RecognitionFromSegments(segments []RecognitionSegment, noSpeech bool, m SourceMap) (RecognitionResult, error) {
	duration, e := MappedDuration(m)
	if e != nil {
		return RecognitionResult{}, e
	}
	if len(segments) > 20000 || noSpeech && len(segments) > 0 {
		return RecognitionResult{}, contracts.Fail("invalid_engine_output")
	}
	origin, _ := new(big.Rat).SetString(m.StartNumerator + "/" + m.StartDenominator)
	origin.Mul(origin, new(big.Rat).SetInt64(1000000))
	var out strings.Builder
	ds := []Diagnostic{}
	count := 0
	for _, s := range segments {
		if s.StartUS < 0 || s.EndUS <= s.StartUS || s.EndUS > duration || !utf8.ValidString(s.Text) || len(s.Text) > 65536 || strings.ContainsFunc(s.Text, func(r rune) bool { return unicode.IsControl(r) && r != '\r' && r != '\n' && r != '\t' }) {
			return RecognitionResult{}, contracts.Fail("invalid_engine_output")
		}
		text := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s.Text, "\r\n", "\n"), "\r", "\n"))
		if strings.Contains(text, "\n\n") {
			return RecognitionResult{}, contracts.Fail("invalid_engine_output")
		}
		if text == "" {
			continue
		}
		startRat := new(big.Rat).Add(origin, new(big.Rat).SetInt64(s.StartUS))
		startRat.Quo(startRat, new(big.Rat).SetInt64(1000))
		endRat := new(big.Rat).Add(origin, new(big.Rat).SetInt64(s.EndUS))
		endRat.Quo(endRat, new(big.Rat).SetInt64(1000))
		start, e := roundedRat(startRat, true)
		if e != nil {
			return RecognitionResult{}, e
		}
		end, e := roundedRat(endRat, false)
		if e != nil {
			return RecognitionResult{}, e
		}
		if start < 0 || end < 0 {
			ds = append(ds, Diagnostic{Code: "negative_source_interval_unrepresentable", Count: 1})
			continue
		}
		if start >= end {
			ds = append(ds, Diagnostic{Code: "submillisecond_segment_unrepresentable", Count: 1})
			continue
		}
		if start > math.MaxInt64/1000000 || end > math.MaxInt64/1000000 {
			return RecognitionResult{}, contracts.Fail("invalid_timing")
		}
		count++
		fmt.Fprintf(&out, "%d\n%s --> %s\n%s\n\n", count, srtStamp(start), srtStamp(end), text)
		if out.Len() > 8<<20 {
			return RecognitionResult{}, contracts.Fail("output_limit")
		}
	}
	return RecognitionResult{SRT: []byte(out.String()), NoSpeech: noSpeech, Diagnostics: ds, Provenance: map[string]any{"segment_count": len(segments), "source_clock_projection": "exact-source-origin;ceil-start/floor-end-millisecond"}}, nil
}
func DiarizationFromTurns(turns []Turn, noSpeech bool, m SourceMap) (DiarizationResult, error) {
	duration, e := MappedDuration(m)
	if e != nil {
		return DiarizationResult{}, e
	}
	if len(turns) > 20000 || noSpeech && len(turns) > 0 {
		return DiarizationResult{}, contracts.Fail("invalid_engine_output")
	}
	for _, t := range turns {
		if !validTurn(t, duration) {
			return DiarizationResult{}, contracts.Fail("invalid_engine_output")
		}
	}
	// Copy so callers retain mapped-relative evidence for optional pure quality.
	mapped := append([]Turn(nil), turns...)
	projected, ds, e := sourceTurns(mapped, m, nil)
	if e != nil {
		return DiarizationResult{}, e
	}
	return DiarizationResult{Turns: projected, NoSpeech: noSpeech, Diagnostics: ds, Provenance: map[string]any{"turn_count": len(turns), "source_clock_projection": "exact-source-origin;ceil-start/floor-end-microsecond"}}, nil
}
