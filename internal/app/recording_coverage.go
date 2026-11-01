// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
	"math"
)

type recordingCueSource struct {
	CueID  string               `json:"cue_id"`
	Source processing.CueSource `json:"source"`
}
type recordingCoverageProvenance struct {
	Evidence     *processing.CoverageEvidence `json:"coverage_evidence"`
	CueSources   []recordingCueSource         `json:"cue_sources"`
	VoiceSources []windowVoiceSource          `json:"voice_sources"`
}

func recordingCoverageInput(r catalog.Recording) (processing.CoverageInput, error) {
	in := processing.CoverageInput{RecordingID: r.ID, SourceDigest: r.SourceDigest, SourceRevision: r.SourceRevision, DocumentDigest: r.DocumentDigest, DocumentRevision: r.Revision, SourceMapDigest: documentHash(r.SourceMap), Cues: []processing.CoverageCue{}, Limitations: []string{"canonical_audio_bytes_not_reverified"}}
	var source windowSourceMap
	if json.Unmarshal(r.SourceMap, &source) != nil {
		return in, contracts.Fail("invalid_request")
	}
	if source.Policy == "windowed-canonical-source-clock-v1" {
		if source.SourceDigest != r.SourceDigest {
			return in, contracts.Condition("conflict", "source_map_incompatible")
		}
		in.Tracks = source.Tracks
	} else {
		var legacy processing.SourceMap
		if json.Unmarshal(r.SourceMap, &legacy) == nil && legacy.SampleRate > 0 && legacy.SampleCount >= 0 && legacy.StartDenominator != "" {
			in.Tracks = []processing.TrackPlan{{StreamIndex: legacy.StreamIndex, Channel: legacy.Channel, SourceSampleRate: legacy.SampleRate, SourceSampleCount: legacy.SampleCount, StartNumerator: legacy.StartNumerator, StartDenominator: legacy.StartDenominator, EndSample: legacy.SampleCount}}
			in.Limitations = append(in.Limitations, "legacy_mapped_audio_basis")
		}
	}
	var p recordingCoverageProvenance
	if json.Unmarshal(r.Provenance, &p) != nil {
		return in, contracts.Fail("invalid_request")
	}
	in.Evidence = p.Evidence
	if r.State != "ready" {
		in.Limitations = append(in.Limitations, "accepted_timed_document_unavailable")
		return in, nil
	}
	if err := subtitles.ValidateDocument(r.Document); err != nil {
		return in, err
	}
	cues, err := evidence.ParseCues(r.Document)
	if err != nil {
		return in, err
	}
	links := map[string]processing.CueSource{}
	for _, s := range p.CueSources {
		if _, ok := links[s.CueID]; ok {
			return in, contracts.Fail("invalid_request")
		}
		links[s.CueID] = s.Source
	}
	voices := map[string]int{}
	for _, v := range p.VoiceSources {
		if lane, ok := voices[v.LocalSpeakerID]; ok && lane != v.TrackOrdinal {
			return in, contracts.Fail("invalid_request")
		}
		voices[v.LocalSpeakerID] = v.TrackOrdinal
	}
	for _, c := range cues {
		if c.StartMS < 0 || c.EndMS > math.MaxInt64/1000 {
			return in, contracts.Fail("invalid_timing")
		}
		out := processing.CoverageCue{ID: c.ID, Text: c.Text, StartUS: c.StartMS * 1000, EndUS: c.EndMS * 1000, TrackOrdinal: -1, VoiceIDs: c.Voices}
		if s, ok := links[c.ID]; ok {
			if s.TrackOrdinal < 0 || s.TrackOrdinal >= len(in.Tracks) {
				return in, contracts.Fail("invalid_request")
			}
			track := in.Tracks[s.TrackOrdinal]
			if track.StreamIndex != s.StreamIndex || (track.Channel == nil) != (s.Channel == nil) || track.Channel != nil && *track.Channel != *s.Channel {
				return in, contracts.Fail("invalid_request")
			}
			out.TrackOrdinal = s.TrackOrdinal
			out.WindowIDs = s.WindowIDs
			out.Source = &processing.CoverageSpan{StartUS: s.StartUS, EndUS: s.EndUS}
		} else if len(in.Tracks) == 1 {
			out.TrackOrdinal = 0
		} else {
			lane := -1
			ambiguous := false
			for _, v := range c.Voices {
				at, ok := voices[v]
				if !ok || at < 0 || at >= len(in.Tracks) {
					ambiguous = true
					break
				}
				if lane >= 0 && lane != at {
					ambiguous = true
					break
				}
				lane = at
			}
			if !ambiguous {
				out.TrackOrdinal = lane
			}
		}
		in.Cues = append(in.Cues, out)
	}
	return in, nil
}
func (a *App) recordingCoverage(req contracts.Request) (any, error) {
	var options processing.CoverageOptions
	if len(req.Data) > 0 && strictPayload(req.Data, &options) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	r, e := a.Catalog.Recording(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	in, e := recordingCoverageInput(r)
	if e != nil {
		return nil, e
	}
	entry, readErr := a.Catalog.Library(a.ctx, req.ItemID)
	if readErr != nil {
		in.Limitations = append(in.Limitations, "canonical_audio_authority_unavailable")
	} else if entry.Revision != r.SourceRevision || entry.Digest != r.SourceDigest {
		in.Limitations = append(in.Limitations, "accepted_recording_source_stale")
	}
	return processing.EvaluateCoverage(in, options)
}
