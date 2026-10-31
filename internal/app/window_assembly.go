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
	SuppliedDocument bool                   `json:"supplied_document,omitempty"`
	Policy           string                 `json:"policy"`
	SourceDigest     string                 `json:"source_digest"`
	Tracks           []processing.TrackPlan `json:"tracks"`
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

// suppliedWindowTranscript identifies text whose native timing was admitted
// independently of an acoustic track selection. It does not authorize reuse of
// diarization observations from a different source or track.
func suppliedWindowTranscript(previous catalog.Recording) bool {
	var source struct {
		Policy           string `json:"policy"`
		SuppliedDocument bool   `json:"supplied_document"`
	}
	return json.Unmarshal(previous.SourceMap, &source) == nil && (source.Policy == "supplied-document-source-clock;no-retiming" || source.SuppliedDocument)
}
func compatibleWindowTranscriptReuse(previous catalog.Recording, plan processing.ProcessingPlan) error {
	if suppliedWindowTranscript(previous) {
		if previous.SourceDigest != plan.Identity.SourceDigest {
			return contracts.Condition("conflict", "source_map_incompatible")
		}
		return nil
	}
	return compatibleWindowReuse(previous, plan)
}

// Preserve the native supplied-document media clock, including its absence.
// Acoustic turns are already validated against their selected source windows.
func assembleSuppliedWindowDocument(ctx context.Context, engine subtitleEngine, doc []byte, turns []subtitles.Turn, participation []subtitles.Participation, cueTracks, voiceTracks map[string]int) (subtitles.Assembly, error) {
	result, err := assembleWindowDocument(ctx, engine, doc, nil, turns, participation, cueTracks, voiceTracks)
	if err != nil {
		return result, err
	}
	var before, after map[string]json.RawMessage
	if json.Unmarshal(doc, &before) != nil || json.Unmarshal(result.Document, &after) != nil {
		return subtitles.Assembly{}, contracts.Fail("invalid_request")
	}
	if timing, ok := before["media_timing"]; ok {
		after["media_timing"] = timing
	} else {
		delete(after, "media_timing")
	}
	result.Document, err = json.Marshal(after)
	if err != nil {
		return subtitles.Assembly{}, contracts.Fail("invalid_request")
	}
	if validator, ok := engine.(interface {
		Validate(context.Context, []byte) error
	}); ok {
		err = validator.Validate(ctx, result.Document)
	} else {
		err = subtitles.ValidateDocument(result.Document)
	}
	if err != nil {
		return subtitles.Assembly{}, err
	}
	return result, nil
}
