// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"sort"
)

type Candidate struct {
	SpeakerID, VersionID string
	Vector               []float64
}
type Decision struct {
	LocalSpeakerID string   `json:"local_speaker_id"`
	SpeakerID      string   `json:"speaker_id,omitempty"`
	VersionID      string   `json:"version_id,omitempty"`
	State          string   `json:"state"`
	Score          *float64 `json:"score,omitempty"`
	RunnerUpScore  *float64 `json:"runner_up_score,omitempty"`
	EvidenceUS     int64    `json:"evidence_us"`
}

func normalized(vector []float64) ([]float64, error) {
	if len(vector) < 1 || len(vector) > 4096 {
		return nil, contracts.Fail("invalid_embedding")
	}
	norm := 0.0
	for _, v := range vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, contracts.Fail("invalid_embedding")
		}
		norm += v * v
	}
	if math.IsInf(norm, 0) || norm < 1e-24 {
		return nil, contracts.Fail("invalid_embedding")
	}
	norm = math.Sqrt(norm)
	out := make([]float64, len(vector))
	for i, v := range vector {
		out[i] = v / norm
	}
	return out, nil
}
func Decide(local string, vector []float64, evidenceUS int64, candidates []Candidate, threshold, margin float64, minimumUS int64) (Decision, error) {
	out := Decision{LocalSpeakerID: local, State: "unknown", EvidenceUS: evidenceUS}
	if math.IsNaN(threshold) || math.IsInf(threshold, 0) || threshold < -1 || threshold > 1 || math.IsNaN(margin) || math.IsInf(margin, 0) || margin < 0 || margin > 2 || minimumUS < 1 {
		return out, contracts.Fail("invalid_request")
	}
	v, err := normalized(vector)
	if err != nil {
		return out, err
	}
	if evidenceUS < minimumUS || len(candidates) == 0 {
		return out, nil
	}
	type scored struct {
		candidate Candidate
		score     float64
	}
	scores := make([]scored, 0, len(candidates))
	for _, c := range candidates {
		n, err := normalized(c.Vector)
		if err != nil || len(n) != len(v) {
			return out, contracts.Fail("incompatible_version")
		}
		score := 0.0
		for i := range n {
			score += n[i] * v[i]
		}
		score = math.Max(-1, math.Min(1, score))
		scores = append(scores, scored{c, score})
	}
	sort.Slice(scores, func(i, j int) bool {
		if scores[i].score == scores[j].score {
			return scores[i].candidate.SpeakerID < scores[j].candidate.SpeakerID
		}
		return scores[i].score > scores[j].score
	})
	best := scores[0]
	out.Score = &best.score
	if len(scores) > 1 {
		runner := scores[1].score
		out.RunnerUpScore = &runner
	}
	if best.score < threshold {
		return out, nil
	}
	if len(scores) > 1 && (best.score == scores[1].score || best.score-scores[1].score < margin) {
		out.State = "ambiguous"
		return out, nil
	}
	out.State = "accepted"
	out.SpeakerID = best.candidate.SpeakerID
	out.VersionID = best.candidate.VersionID
	return out, nil
}
