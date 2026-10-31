// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"math/big"
)

// ResolveCueSourceMap projects the current recording's per-cue source selection.
// The original complete SourceMap remains the digest-fencing authority.
func ResolveCueSourceMap(sourceMap, provenance json.RawMessage, cueID string) (json.RawMessage, error) {
	return ResolveEvidenceSourceMap(sourceMap, provenance, cueID, "")
}

// ResolveEvidenceSourceMap prefers the assigned voice's known track, then the
// cue's track, then an unambiguous single track. It never guesses stream zero.
func ResolveEvidenceSourceMap(sourceMap, provenance json.RawMessage, cueID, localSpeakerID string) (json.RawMessage, error) {
	var envelope struct {
		Policy string `json:"policy"`
		Tracks []struct {
			StreamIndex       *int   `json:"stream_index"`
			Channel           *int   `json:"channel"`
			SourceSampleRate  int64  `json:"source_sample_rate"`
			SourceSampleCount int64  `json:"source_sample_count"`
			StartNumerator    string `json:"start_numerator"`
			StartDenominator  string `json:"start_denominator"`
		} `json:"tracks"`
	}
	if ValidateJSON(sourceMap) != nil || json.Unmarshal(sourceMap, &envelope) != nil {
		return nil, contracts.Fail("conflict")
	}
	if envelope.Policy != "windowed-canonical-source-clock-v1" {
		return append(json.RawMessage(nil), sourceMap...), nil
	}
	if len(envelope.Tracks) < 1 || len(envelope.Tracks) > 64 {
		return nil, contracts.Condition("conflict", "source_map_incompatible")
	}
	var p struct {
		Cues []struct {
			CueID  string `json:"cue_id"`
			Source struct {
				TrackOrdinal *int `json:"track_ordinal"`
				StreamIndex  *int `json:"stream_index"`
				Channel      *int `json:"channel"`
			} `json:"source"`
		} `json:"cue_sources"`
		Voices []struct {
			LocalSpeakerID string `json:"local_speaker_id"`
			TrackOrdinal   *int   `json:"track_ordinal"`
		} `json:"voice_sources"`
	}
	if len(provenance) > 0 && (ValidateJSON(provenance) != nil || json.Unmarshal(provenance, &p) != nil) {
		return nil, contracts.Fail("conflict")
	}
	selected := -1
	if localSpeakerID != "" {
		for _, voice := range p.Voices {
			if voice.LocalSpeakerID == localSpeakerID {
				if voice.TrackOrdinal == nil || *voice.TrackOrdinal < 0 || *voice.TrackOrdinal >= len(envelope.Tracks) {
					return nil, contracts.Fail("conflict")
				}
				if selected != -1 && selected != *voice.TrackOrdinal {
					return nil, contracts.Fail("conflict")
				}
				selected = *voice.TrackOrdinal
			}
		}
	}
	if selected < 0 {
		for _, cue := range p.Cues {
			if cue.CueID == cueID {
				if cue.Source.TrackOrdinal == nil || cue.Source.StreamIndex == nil {
					return nil, contracts.Fail("conflict")
				}
				if selected != -1 && selected != *cue.Source.TrackOrdinal {
					return nil, contracts.Fail("conflict")
				}
				selected = *cue.Source.TrackOrdinal
				if selected < 0 || selected >= len(envelope.Tracks) {
					return nil, contracts.Fail("conflict")
				}
				track := envelope.Tracks[selected]
				if track.StreamIndex == nil || *track.StreamIndex != *cue.Source.StreamIndex || (track.Channel == nil) != (cue.Source.Channel == nil) || track.Channel != nil && *track.Channel != *cue.Source.Channel {
					return nil, contracts.Fail("conflict")
				}
			}
		}
	}
	if selected < 0 && len(envelope.Tracks) == 1 {
		selected = 0
	}
	if selected < 0 || selected >= len(envelope.Tracks) {
		return nil, contracts.Condition("conflict", "source_map_incompatible")
	}
	track := envelope.Tracks[selected]
	origin, ok := new(big.Rat).SetString(track.StartNumerator + "/" + track.StartDenominator)
	denominator, denOK := new(big.Int).SetString(track.StartDenominator, 10)
	if !ok || !denOK || denominator.Sign() <= 0 || track.StreamIndex == nil || *track.StreamIndex < 0 || track.Channel != nil && (*track.Channel < 0 || *track.Channel > 63) || track.SourceSampleRate < 1 || track.SourceSampleCount < 0 {
		return nil, contracts.Fail("conflict")
	}
	count := new(big.Int).Mul(big.NewInt(track.SourceSampleCount), big.NewInt(16000))
	count.Quo(count, big.NewInt(track.SourceSampleRate))
	if !count.IsInt64() {
		return nil, contracts.Fail("input_limit")
	}
	return json.Marshal(map[string]any{"start_numerator": origin.Num().String(), "start_denominator": origin.Denom().String(), "sample_count": count.Int64(), "sample_rate": 16000, "stream_index": track.StreamIndex, "channel": track.Channel, "policy": "windowed-cue-source-clock-v1"})
}
