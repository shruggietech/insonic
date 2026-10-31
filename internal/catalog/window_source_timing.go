// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	"math/big"

	"github.com/shruggietech/insonic/internal/contracts"
)

// Windowed transcript clocks include each selected track's rational source
// origin. Their final endpoint is not the physical elapsed container duration.
func windowSourceDurationMS(raw json.RawMessage, sourceDigest string) (*int64, error) {
	var envelope struct {
		Policy           string `json:"policy"`
		SourceDigest     string `json:"source_digest"`
		SuppliedDocument bool   `json:"supplied_document,omitempty"`
		Tracks           []struct {
			StreamIndex       int    `json:"stream_index"`
			Channel           *int   `json:"channel"`
			SourceSampleRate  int64  `json:"source_sample_rate"`
			SourceSampleCount int64  `json:"source_sample_count"`
			StartNumerator    string `json:"start_numerator"`
			StartDenominator  string `json:"start_denominator"`
			StartSample       int64  `json:"start_sample"`
			EndSample         int64  `json:"end_sample"`
		} `json:"tracks"`
	}
	if len(raw) > MaxProcessingPlanBytes || strict(raw, &envelope) != nil || envelope.Policy != "windowed-canonical-source-clock-v1" || envelope.SourceDigest != sourceDigest || len(envelope.Tracks) < 1 || len(envelope.Tracks) > 64 {
		return nil, contracts.Fail("invalid_request")
	}
	maximum := new(big.Rat)
	for _, track := range envelope.Tracks {
		if track.StreamIndex < 0 || track.StreamIndex > 65535 || track.Channel != nil && (*track.Channel < 0 || *track.Channel > 63) || track.SourceSampleRate < 1 || track.SourceSampleRate > 768000 || track.SourceSampleCount < 0 || track.SourceSampleCount > 604800*track.SourceSampleRate || track.StartSample < 0 || track.EndSample < track.StartSample || track.EndSample > track.SourceSampleCount || len(track.StartNumerator) > 128 || len(track.StartDenominator) > 128 {
			return nil, contracts.Fail("invalid_timing")
		}
		numerator, numOK := new(big.Int).SetString(track.StartNumerator, 10)
		denominator, denOK := new(big.Int).SetString(track.StartDenominator, 10)
		if !numOK || !denOK || denominator.Sign() <= 0 {
			return nil, contracts.Fail("invalid_timing")
		}
		if track.StartSample == track.EndSample {
			continue
		}
		end := new(big.Rat).SetFrac(numerator, denominator)
		end.Add(end, new(big.Rat).SetFrac64(track.EndSample, track.SourceSampleRate))
		if end.Cmp(maximum) > 0 {
			maximum.Set(end)
		}
	}
	if maximum.Sign() <= 0 {
		return nil, nil
	}
	maximum.Mul(maximum, big.NewRat(1_000_000_000, 1))
	nanoseconds, remainder := new(big.Int), new(big.Int)
	nanoseconds.QuoRem(maximum.Num(), maximum.Denom(), remainder)
	if remainder.Sign() != 0 {
		nanoseconds.Add(nanoseconds, big.NewInt(1))
	}
	if !nanoseconds.IsInt64() {
		return nil, contracts.Fail("invalid_timing")
	}
	milliseconds := nanoseconds.Int64() / 1_000_000
	return &milliseconds, nil
}
