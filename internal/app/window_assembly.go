// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
)

type windowSourceMap struct {
	Policy       string                 `json:"policy"`
	SourceDigest string                 `json:"source_digest"`
	Tracks       []processing.TrackPlan `json:"tracks"`
}
type windowVoiceSource struct {
	LocalSpeakerID string `json:"local_speaker_id"`
	TrackOrdinal   int    `json:"track_ordinal"`
}

func windowTrackMapping(t processing.TrackPlan) processing.SourceMap {
	first := (t.StartSample*16000 + t.SourceSampleRate - 1) / t.SourceSampleRate
	last := t.EndSample * 16000 / t.SourceSampleRate
	origin, _ := new(big.Rat).SetString(t.StartNumerator + "/" + t.StartDenominator)
	origin.Add(origin, new(big.Rat).SetFrac64(first, 16000))
	return processing.SourceMap{StartNumerator: origin.Num().String(), StartDenominator: origin.Denom().String(), SampleRate: 16000, SampleCount: max(int64(0), last-first), StreamIndex: t.StreamIndex, Channel: t.Channel, Policy: "windowed-source-range"}
}
func compatibleWindowReuse(previous catalog.Recording, plan processing.ProcessingPlan) error {
	var old windowSourceMap
	if json.Unmarshal(previous.SourceMap, &old) == nil && old.Policy == "windowed-canonical-source-clock-v1" {
		if old.SourceDigest != plan.Identity.SourceDigest || len(old.Tracks) != len(plan.Tracks) {
			return contracts.Condition("conflict", "source_map_incompatible")
		}
		for i, t := range old.Tracks {
			a, _ := json.Marshal(t)
			b, _ := json.Marshal(plan.Tracks[i])
			if string(a) != string(b) {
				return contracts.Condition("conflict", "source_map_incompatible")
			}
		}
		return nil
	}
	if len(plan.Tracks) != 1 {
		return contracts.Condition("conflict", "source_map_incompatible")
	}
	var legacy processing.SourceMap
	if strictPayload(previous.SourceMap, &legacy) != nil {
		return contracts.Condition("conflict", "source_map_incompatible")
	}
	now := windowTrackMapping(plan.Tracks[0])
	now.Policy = legacy.Policy
	if compatibleReuseMap(previous.SourceMap, now) != nil {
		return contracts.Condition("conflict", "source_map_incompatible")
	}
	return nil
}

// Use the same native validator with a source-lane constraint for generated cues.
func assembleWindowDocument(ctx context.Context, engine subtitleEngine, doc []byte, duration *int64, turns []subtitles.Turn, participation []subtitles.Participation, cueTracks map[string]int, voiceTracks map[string]int) (subtitles.Assembly, error) {
	if len(cueTracks) == 0 {
		return engine.Assemble(ctx, doc, duration, turns, participation)
	}
	if scoped, ok := engine.(interface {
		AssembleScoped(context.Context, []byte, *int64, []subtitles.Turn, []subtitles.Participation, map[string]int, map[string]int) (subtitles.Assembly, error)
	}); ok {
		return scoped.AssembleScoped(ctx, doc, duration, turns, participation, cueTracks, voiceTracks)
	}
	return subtitles.Assembly{}, contracts.Fail("unsupported_capability")
}
