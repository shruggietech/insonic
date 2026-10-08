// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"math/big"
)

func compatibleReuseMap(previous json.RawMessage, current any) error {
	var old processing.SourceMap
	if strictPayload(previous, &old) != nil {
		return contracts.Fail("conflict")
	}
	now, e := audioSourceMap(&audioExecution{SourceMap: current})
	if e != nil {
		return e
	}
	if _, e = processing.MappedDuration(old); e != nil {
		return e
	}
	if _, e = processing.MappedDuration(now); e != nil {
		return e
	}
	a, _ := new(big.Rat).SetString(old.StartNumerator + "/" + old.StartDenominator)
	b, _ := new(big.Rat).SetString(now.StartNumerator + "/" + now.StartDenominator)
	sameChannel := old.Channel == nil && now.Channel == nil || old.Channel != nil && now.Channel != nil && *old.Channel == *now.Channel
	if a.Cmp(b) != 0 || old.SampleCount != now.SampleCount || old.SampleRate != now.SampleRate || old.StreamIndex != now.StreamIndex || !sameChannel || old.Policy != now.Policy {
		return contracts.Fail("conflict")
	}
	return nil
}

// Reused quality is evaluated on the sole current Cueson assignments. Those
// intervals are millisecond-quantized source-clock evidence, not raw turns.
func reusedQuality(turns []subtitles.Turn, source any, config *processing.QualityConfig) ([]processing.Diagnostic, error) {
	m, e := audioSourceMap(&audioExecution{SourceMap: source})
	if e != nil {
		return nil, e
	}
	duration, e := processing.MappedDuration(m)
	if e != nil {
		return nil, e
	}
	origin, _ := new(big.Rat).SetString(m.StartNumerator + "/" + m.StartDenominator)
	origin.Mul(origin, new(big.Rat).SetInt64(1000000))
	relative := func(ns int64) (int64, error) {
		r := new(big.Rat).Sub(new(big.Rat).SetFrac64(ns, 1000), origin)
		v := new(big.Int).Quo(r.Num(), r.Denom())
		if !v.IsInt64() {
			return 0, contracts.Fail("invalid_timing")
		}
		n := v.Int64()
		if n < -1000 || n > duration+1000 {
			return 0, contracts.Fail("invalid_timing")
		}
		return max(int64(0), min(n, duration)), nil
	}
	input := make([]processing.Turn, 0, len(turns))
	for _, turn := range turns {
		start, e := relative(turn.StartNS)
		if e != nil {
			return nil, e
		}
		end, e := relative(turn.EndNS)
		if e != nil {
			return nil, e
		}
		if end <= start {
			continue
		}
		input = append(input, processing.Turn{Label: turn.SpeakerID, StartUS: start, EndUS: end})
	}
	c := processing.QualityConfig{}
	if config != nil {
		c = *config
	}
	return processing.Quality(input, duration, c)
}
