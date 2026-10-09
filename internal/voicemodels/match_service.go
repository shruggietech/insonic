// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"io"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func (s *Service) ValidateMatch(ctx context.Context, o MatchOptions) (MatchPayload, error) {
	out := MatchPayload{Options: o}
	if !contracts.ValidID(o.RecordingID) || !text(o.ModelID, 256) || o.ModelDigest != "" && !digestPattern.MatchString(o.ModelDigest) || ValidateAdapter(o.Adapter) != nil || o.Adapter.ID != "pyannote-profile" || math.IsNaN(o.Threshold) || math.IsInf(o.Threshold, 0) || o.Threshold < -1 || o.Threshold > 1 || math.IsNaN(o.Margin) || math.IsInf(o.Margin, 0) || o.Margin < 0 || o.Margin > 2 || o.MinEvidenceUS < 1 || o.MinEvidenceUS > 604800000000 {
		return out, contracts.Fail("invalid_request")
	}
	var err error
	out.Options.Adapter.Limits, err = normalizeLimits(o.Adapter.Limits)
	if err != nil {
		return out, err
	}
	out.Snapshot, err = s.Catalog.FreezeSpeakerMatching(ctx, o.RecordingID)
	return out, err
}

type voiceSpan struct {
	start, end int64
	cueID      string
}

func (s *Service) ExecuteMatch(ctx context.Context, claim catalog.Work) (any, error) {
	var payload MatchPayload
	if claim.Kind != "recordings.match" || strict(claim.Payload, &payload) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if _, err := s.Catalog.RenewWork(ctx, claim, 30*time.Second); err != nil {
		return nil, err
	}
	options := payload.Options
	if !contracts.ValidID(options.ModelID) || !digestPattern.MatchString(options.ModelDigest) || ValidateAdapter(options.Adapter) != nil || options.Adapter.ID != "pyannote-profile" || payload.Snapshot.Recording.ID != options.RecordingID {
		return nil, contracts.Fail("invalid_request")
	}
	recording, err := s.Catalog.Recording(ctx, options.RecordingID)
	if err != nil {
		return nil, err
	}
	frozen := payload.Snapshot.Recording
	if recording.Revision != frozen.Revision || recording.SourceRevision != frozen.SourceRevision || recording.SourceDigest != frozen.SourceDigest || recording.DocumentDigest != frozen.DocumentDigest {
		return nil, contracts.Fail("conflict")
	}
	var document struct {
		Cues []struct {
			ID          string `json:"id"`
			Assignments []struct {
				ID    string `json:"speaker_id"`
				Start *int64 `json:"start_milliseconds"`
				End   *int64 `json:"end_milliseconds"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(recording.Document, &document) != nil {
		return nil, contracts.Fail("conflict")
	}
	voices := map[string][]voiceSpan{}
	cueIDs := map[string][]string{}
	for _, cue := range document.Cues {
		for _, assignment := range cue.Assignments {
			if !contracts.ValidLocalSpeakerID(assignment.ID) {
				return nil, contracts.Fail("invalid_request")
			}
			if _, ok := voices[assignment.ID]; !ok {
				voices[assignment.ID] = []voiceSpan{}
			}
			if !contains(cueIDs[assignment.ID], cue.ID) {
				cueIDs[assignment.ID] = append(cueIDs[assignment.ID], cue.ID)
			}
			if assignment.Start != nil && assignment.End != nil && *assignment.End > *assignment.Start {
				voices[assignment.ID] = append(voices[assignment.ID], voiceSpan{*assignment.Start, *assignment.End, cue.ID})
			}
		}
	}
	if len(voices) > 1000 {
		return nil, contracts.Fail("output_limit")
	}
	candidates := []Candidate{}
	missing := append([]string{}, payload.Snapshot.Missing...)
	for index, output := range payload.Snapshot.Outputs {
		if index >= len(payload.Snapshot.Profiles) {
			return nil, contracts.Fail("conflict")
		}
		head := payload.Snapshot.Profiles[index]
		profile, err := s.readProfile(ctx, output)
		if err != nil {
			if ctx.Err() != nil {
				return nil, contracts.Fail("cancelled")
			}
			missing = append(missing, head.ID)
			continue
		}
		if profile.ModelID != options.ModelID || profile.ModelDigest != options.ModelDigest || profile.EvidenceUS < options.MinEvidenceUS {
			missing = append(missing, head.ID)
			continue
		}
		candidates = append(candidates, Candidate{SpeakerID: head.ID, VersionID: head.VersionID, Vector: profile.Vector})
	}
	directory, err := s.scratch()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	var audio *Audio
	if len(candidates) > 0 {
		entry, err := s.Catalog.Library(ctx, options.RecordingID)
		if err != nil {
			return nil, err
		}
		if entry.Digest != recording.SourceDigest {
			return nil, contracts.Fail("conflict")
		}
		var source processing.SourceMap
		if json.Unmarshal(recording.SourceMap, &source) != nil {
			return nil, contracts.Fail("conflict")
		}
		if s.Prepare == nil || s.Embed == nil && s.EmbedBatch == nil {
			return nil, contracts.Fail("unavailable")
		}
		audio, err = s.Prepare(ctx, entry, processing.AudioOptions{StreamIndex: &source.StreamIndex, Channel: source.Channel})
		if err != nil {
			return nil, err
		}
		defer audio.Close()
	}
	names := []string{}
	for local := range voices {
		names = append(names, local)
	}
	sort.Strings(names)
	decisions := []catalog.SpeakerMatchDecision{}
	reports := []Decision{}
	for _, local := range names {
		manual := contains(payload.Snapshot.ManualVoices, local)
		report := Decision{LocalSpeakerID: local, State: "unknown", EvidenceUS: observedVoiceDuration(voices[local])}
		noVector := false
		if manual {
			report.State = "manual-preserved"
		} else if len(candidates) > 0 {
			vector, duration, err := s.voiceEmbedding(ctx, audio, voices[local], options, directory)
			if err != nil {
				return nil, err
			}
			report.EvidenceUS = duration
			noVector = len(vector) == 0
			if len(vector) > 0 {
				compatible := []Candidate{}
				for _, candidate := range candidates {
					if len(candidate.Vector) == len(vector) {
						compatible = append(compatible, candidate)
					} else if !contains(missing, candidate.SpeakerID) {
						missing = append(missing, candidate.SpeakerID)
					}
				}
				report, err = Decide(local, vector, duration, compatible, options.Threshold, options.Margin, options.MinEvidenceUS)
				if err != nil {
					return nil, err
				}
			}
		}
		diagnostics := json.RawMessage(`[]`)
		if !manual && len(candidates) == 0 {
			diagnostics = json.RawMessage(`["no_eligible_roster_profile"]`)
		} else if noVector {
			if report.EvidenceUS < options.MinEvidenceUS {
				diagnostics = json.RawMessage(`["insufficient_voice_evidence"]`)
			} else {
				diagnostics = json.RawMessage(`["no_usable_voice_embedding"]`)
			}
		}
		acceptedSpeaker := report.SpeakerID
		if report.State == "manual-preserved" {
			acceptedSpeaker = ""
		}
		decisions = append(decisions, catalog.SpeakerMatchDecision{LocalSpeakerID: local, SpeakerID: acceptedSpeaker, State: report.State, Score: report.Score, RunnerUpScore: report.RunnerUpScore, CueIDs: cueIDs[local], Diagnostics: diagnostics})
		reports = append(reports, report)
	}
	provenance, _ := json.Marshal(map[string]any{"adapter": options.Adapter.ID, "contract_version": options.Adapter.ContractVersion, "model_id": options.ModelID, "model_digest": options.ModelDigest, "threshold": options.Threshold, "ambiguity_margin": options.Margin, "min_evidence_us": options.MinEvidenceUS, "roster_revision": payload.Snapshot.Roster.Revision, "snapshot_digest": payload.Snapshot.Digest, "method": "duration-weighted-normalized-embedding;cosine"})
	mappings, err := s.Catalog.CommitSpeakerMatching(ctx, claim, payload.Snapshot, decisions, provenance)
	if err != nil {
		return nil, err
	}
	return map[string]any{"recording_id": options.RecordingID, "decisions": reports, "missing_profiles": missing, "mapping_count": len(mappings), "state": "accepted"}, nil
}

func observedVoiceDuration(spans []voiceSpan) int64 {
	ordered := append([]voiceSpan(nil), spans...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].start == ordered[j].start {
			return ordered[i].end < ordered[j].end
		}
		return ordered[i].start < ordered[j].start
	})
	var duration int64
	var previous voiceSpan
	initialized := false
	for _, span := range ordered {
		if !initialized {
			previous = span
			initialized = true
			continue
		}
		if span.start <= previous.end {
			if span.end > previous.end {
				previous.end = span.end
			}
		} else {
			duration += (previous.end - previous.start) * 1000
			previous = span
		}
	}
	if initialized {
		duration += (previous.end - previous.start) * 1000
	}
	return duration
}
func (s *Service) voiceEmbedding(ctx context.Context, audio *Audio, spans []voiceSpan, options MatchOptions, directory string) ([]float64, int64, error) {
	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start == spans[j].start {
			return spans[i].end < spans[j].end
		}
		return spans[i].start < spans[j].start
	})
	merged := []voiceSpan{}
	for _, span := range spans {
		if len(merged) > 0 && span.start <= merged[len(merged)-1].end {
			if span.end > merged[len(merged)-1].end {
				merged[len(merged)-1].end = span.end
			}
		} else {
			merged = append(merged, span)
		}
	}
	vector := []float64{}
	duration := int64(0)
	limits, _ := normalizeLimits(options.Adapter.Limits)
	total := int64(0)
	inputs := []PreparedInput{}
	var availableUS int64
	for _, span := range merged {
		availableUS += (span.end - span.start) * 1000
	}
	if availableUS < options.MinEvidenceUS {
		return nil, availableUS, nil
	}
	for _, span := range merged {
		start := new(big.Rat).SetFrac(big.NewInt(span.start), big.NewInt(1000))
		end := new(big.Rat).SetFrac(big.NewInt(span.end), big.NewInt(1000))
		first, last, err := sampleBounds(start.RatString(), end.RatString(), audio.SourceMap)
		if err != nil {
			return nil, duration, err
		}
		size := (last - first) * 2
		total += size + 44
		if size > 256<<20 || total > limits.MaxInputBytes {
			return nil, duration, contracts.Fail("input_limit")
		}
		f, err := os.Open(audio.Path)
		if err != nil {
			return nil, duration, contracts.Fail("unavailable")
		}
		offset, n, err := wavData(f)
		if err != nil || last*2 > n {
			f.Close()
			return nil, duration, contracts.Fail("invalid_audio")
		}
		pcm := make([]byte, size)
		_, err = io.ReadFull(io.NewSectionReader(f, offset+first*2, size), pcm)
		f.Close()
		if err != nil {
			return nil, duration, contracts.Fail("invalid_audio")
		}
		path := filepath.Join(directory, fmt.Sprintf("match-%s.wav", contracts.ID()))
		if err = writeWAV(path, pcm); err != nil {
			return nil, duration, err
		}
		mapping := audio.SourceMap
		mapping.SampleCount = last - first
		mapping.StartNumerator = "0"
		mapping.StartDenominator = "1"
		us := (last - first) * 1000000 / audio.SourceMap.SampleRate
		inputs = append(inputs, PreparedInput{Path: path, SourceMap: mapping, DurationUS: us})
	}
	for start := 0; start < len(inputs); start += 64 {
		end := start + 64
		if end > len(inputs) {
			end = len(inputs)
		}
		vectors, err := s.embedInputs(ctx, inputs[start:end], options.ModelID, options.ModelDigest)
		if err != nil {
			return nil, duration, err
		}
		for index, input := range inputs[start:end] {
			if vectors[index] == nil {
				os.Remove(input.Path)
				continue
			}
			values, err := normalized(vectors[index])
			if err != nil {
				return nil, duration, err
			}
			if len(vector) == 0 {
				vector = make([]float64, len(values))
			}
			if len(vector) != len(values) {
				return nil, duration, contracts.Fail("incompatible_version")
			}
			for i, v := range values {
				vector[i] += v * float64(input.DurationUS)
			}
			duration += input.DurationUS
			os.Remove(input.Path)
		}
	}
	if len(vector) == 0 {
		return vector, availableUS, nil
	}
	vector, err := normalized(vector)
	return vector, duration, err
}
