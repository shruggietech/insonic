// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"math"
	"sort"
)

func (s *Store) SpeakerDiscoveryAcoustic(ctx context.Context, id, workID string) (out sd.AcousticEvidence, e error) {
	if !contracts.ValidID(id) || (workID != "" && !contracts.ValidID(workID)) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	out, e = s.speakerDiscoveryAcousticTx(ctx, tx, id, workID)
	return out, sanitize(e)
}
func (s *Store) speakerDiscoveryAcousticTx(ctx context.Context, tx *sql.Tx, id, workID string) (out sd.AcousticEvidence, e error) {
	out = sd.AcousticEvidence{Availability: "unavailable", WorkID: workID, Decisions: []sd.AcousticDecision{}, EligibleProfiles: []string{}}
	var recording Recording
	if e = s.domainTx(ctx, tx, "Recordings", id, &recording); e != nil {
		return
	}
	mappings, e := s.recordingMappings(ctx, tx, id)
	if e != nil {
		return out, e
	}
	if workID == "" {
		for _, m := range mappings {
			if m.Origin != "automatic" {
				continue
			}
			var p struct {
				WorkID string `json:"work_id"`
			}
			if json.Unmarshal(m.Provenance, &p) != nil || !contracts.ValidID(p.WorkID) {
				out.Availability = "stale"
				return out, nil
			}
			if workID != "" && workID != p.WorkID {
				out.Availability = "stale"
				return out, nil
			}
			workID = p.WorkID
		}
	}
	if workID == "" {
		expr := "json_extract(w.payload,'$.options.recording_id')"
		if s.backend == "postgresql" {
			expr = "(w.payload::jsonb->'options'->>'recording_id')"
		}
		e = s.row(ctx, tx, "SELECT w.id FROM work_operation w JOIN operation_receipt r ON r.workspace_id=w.workspace_id AND r.id=w.journal_receipt_id WHERE w.workspace_id=? AND w.kind='recordings.match' AND w.state='succeeded' AND "+expr+"=? ORDER BY r.revision DESC,w.id DESC LIMIT 1", s.workspace, id).Scan(&workID)
		if e == sql.ErrNoRows {
			return out, nil
		}
		if e != nil {
			return out, e
		}
	}
	out.WorkID = workID
	stale := func() (sd.AcousticEvidence, error) {
		return sd.AcousticEvidence{Availability: "stale", WorkID: workID, Decisions: []sd.AcousticDecision{}, EligibleProfiles: []string{}}, nil
	}
	var w Work
	if e = s.domainTx(ctx, tx, "Works", workID, &w); e == sql.ErrNoRows {
		return stale()
	}
	if e != nil {
		return out, e
	}
	if w.Kind != "recordings.match" || w.State != "succeeded" {
		return stale()
	}
	var payload struct {
		Snapshot SpeakerMatchingSnapshot `json:"snapshot"`
		Options  struct {
			RecordingID string `json:"recording_id"`
		} `json:"options"`
	}
	var result struct {
		RecordingID string `json:"recording_id"`
		State       string `json:"state"`
		Decisions   []struct {
			LocalSpeakerID string   `json:"local_speaker_id"`
			SpeakerID      string   `json:"speaker_id"`
			VersionID      string   `json:"version_id"`
			State          string   `json:"state"`
			Score          *float64 `json:"score"`
			RunnerUpScore  *float64 `json:"runner_up_score"`
		} `json:"decisions"`
	}
	if json.Unmarshal(w.Payload, &payload) != nil || payload.Options.RecordingID != id || payload.Snapshot.Recording.ID != id || payload.Snapshot.Digest != matchingSnapshotDigest(payload.Snapshot) || json.Unmarshal(w.Result, &result) != nil || result.RecordingID != id || result.State != "accepted" || len(result.Decisions) > 1000 {
		return stale()
	}
	var raw string
	if e = s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.JournalReceiptID).Scan(&raw); e == sql.ErrNoRows {
		return stale()
	}
	if e != nil {
		return out, e
	}
	var journal struct {
		WorkDigest string `json:"work_digest"`
	}
	if json.Unmarshal([]byte(raw), &journal) != nil || journal.WorkDigest != workJournalDigest(w) {
		return stale()
	}
	if e = s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, operationID("speaker-matching", w.ID, id)).Scan(&raw); e == sql.ErrNoRows {
		return stale()
	}
	if e != nil {
		return out, e
	}
	var receipt struct {
		Snapshot string `json:"matching_snapshot_digest"`
		Mappings string `json:"mapping_digest"`
	}
	md, _ := intent(mappings)
	if json.Unmarshal([]byte(raw), &receipt) != nil || receipt.Snapshot != payload.Snapshot.Digest || receipt.Mappings != md {
		return stale()
	}
	current, e := s.freezeSpeakerMatchingTx(ctx, tx, id)
	if e != nil {
		if _, ok := e.(*contracts.Error); ok {
			return stale()
		}
		return out, e
	}
	current.MappingDigest = payload.Snapshot.MappingDigest
	if matchingSnapshotDigest(current) != payload.Snapshot.Digest {
		return stale()
	}
	target, e := discoveryIntervals(recording)
	if e != nil {
		return out, e
	}
	profiles := map[string]SpeakerProfile{}
	for _, p := range current.Profiles {
		profiles[p.ID] = p
		out.EligibleProfiles = append(out.EligibleProfiles, p.ID)
	}
	currentMappings := map[string]SpeakerMapping{}
	for _, m := range mappings {
		currentMappings[m.LocalSpeakerID] = m
	}
	voices := documentSpeakers(recording.Document)
	seen := map[string]bool{}
	// Cache current dataset validation per output; independence remains voice-relative.
	datasets := map[string]SpeakerDataset{}
	grounded := map[string]bool{}
	lineage := map[string][]discoveryInterval{}
	for _, v := range current.Outputs {
		var metadata struct {
			State string `json:"state"`
		}
		if json.Unmarshal(v.Metadata, &metadata) != nil || metadata.State != "current" {
			continue
		}
		dataset, err := s.speakerDatasetTx(ctx, tx, v.DatasetID)
		if err != nil {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			if err != sql.ErrNoRows {
				if _, ok := err.(*contracts.Error); !ok {
					return out, err
				}
			}
			continue
		}
		if dataset.Dataset.State != "current" || len(dataset.References) == 0 || len(dataset.References) > SpeakerDatasetReferenceLimit {
			continue
		}
		valid := true
		cache := evidenceRecordingCache{}
		maps := map[string]SpeakerMapping{}
		intervalCache := map[string][]discoveryInterval{}
		spans := []discoveryInterval{}
		for _, ref := range dataset.References {
			if err = s.validateCurrentReferenceTx(ctx, tx, ref, cache, maps); err != nil {
				valid = false
				break
			}
			intervals, ok := intervalCache[ref.RecordingID]
			if !ok {
				intervals, err = discoveryIntervals(cache[ref.RecordingID].Recording)
				if err != nil {
					valid = false
					break
				}
				intervalCache[ref.RecordingID] = intervals
			}
			found := false
			for _, interval := range intervals {
				if interval.cue == ref.CueID && interval.voice == ref.LocalSpeakerID {
					spans = append(spans, interval)
					found = true
				}
			}
			if !found {
				valid = false
				break
			}
		}
		if valid {
			datasets[v.ID] = dataset
			grounded[v.ID] = true
			lineage[v.ID] = spans
		}
	}
	independent := func(version, voice string) bool {
		if !grounded[version] {
			return false
		}
		own := []discoveryInterval{}
		for _, v := range target {
			if v.voice == voice {
				own = append(own, v)
			}
		}
		if len(own) == 0 {
			return false
		}
		for _, enrolled := range lineage[version] {
			for _, evaluated := range own {
				if discoveryOverlap(enrolled, evaluated) {
					return false
				}
			}
		}
		return true
	}
	for _, d := range result.Decisions {
		if !voices[d.LocalSpeakerID] || seen[d.LocalSpeakerID] {
			return stale()
		}
		seen[d.LocalSpeakerID] = true
		for _, score := range []*float64{d.Score, d.RunnerUpScore} {
			if score != nil && (math.IsNaN(*score) || math.IsInf(*score, 0) || *score < -1 || *score > 1) {
				return stale()
			}
		}
		value := sd.AcousticDecision{VoiceID: d.LocalSpeakerID, State: d.State, SpeakerID: d.SpeakerID, Score: d.Score, RunnerUpScore: d.RunnerUpScore, Reason: "no_independent_profile_lineage"}
		switch d.State {
		case "accepted":
			p, ok := profiles[d.SpeakerID]
			if !ok || p.VersionID != d.VersionID {
				return stale()
			}
			m, ok := currentMappings[d.LocalSpeakerID]
			if !ok || m.Origin != "automatic" || m.SpeakerID != d.SpeakerID {
				return stale()
			}
			var proof struct {
				WorkID    string               `json:"work_id"`
				VersionID string               `json:"model_version_id"`
				Decision  SpeakerMatchDecision `json:"decision"`
			}
			if json.Unmarshal(m.Provenance, &proof) != nil || proof.WorkID != w.ID || proof.VersionID != p.VersionID || proof.Decision.State != "accepted" || sd.Digest(proof.Decision.Score) != sd.Digest(d.Score) || sd.Digest(proof.Decision.RunnerUpScore) != sd.Digest(d.RunnerUpScore) {
				return stale()
			}
			value.State = "matched"
			value.ProfileID = p.VersionID
			value.Independent = independent(p.VersionID, d.LocalSpeakerID)
		case "ambiguous":
			value.Independent = len(profiles) > 0
			for _, p := range profiles {
				if !independent(p.VersionID, d.LocalSpeakerID) {
					value.Independent = false
				}
			}
		case "unknown", "manual-preserved":
		default:
			return stale()
		}
		if value.Independent {
			value.Reason = "current_manual_enrollment_without_source_overlap"
		}
		out.Decisions = append(out.Decisions, value)
	}
	if len(seen) != len(voices) {
		return stale()
	}
	out.Availability = "current"
	sort.Strings(out.EligibleProfiles)
	sort.Slice(out.Decisions, func(i, j int) bool { return out.Decisions[i].VoiceID < out.Decisions[j].VoiceID })
	// Dataset epochs and reference hashes fence current enrollment without copying it.
	lineageProofs := map[string]string{}
	for id, d := range datasets {
		lineageProofs[id] = sd.Digest(d)
	}
	out.Digest = sd.Digest([]any{workJournalDigest(w), receipt, current.Digest, lineageProofs, out.Decisions})
	return out, nil
}
