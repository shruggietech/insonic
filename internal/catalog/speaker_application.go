// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"sort"
)

type ContextualLineage struct {
	MaxCues           int             `json:"max_cues"`
	Version           int             `json:"version"`
	Class             string          `json:"evidence_class"`
	OperationID       string          `json:"operation_id"`
	DiscoveryWorkID   string          `json:"discovery_work_id"`
	DiscoveryRevision int64           `json:"discovery_revision"`
	DiscoveryDigest   string          `json:"discovery_digest"`
	RecordingRevision int64           `json:"recording_revision"`
	SourceDigest      string          `json:"source_digest"`
	SourceMapDigest   string          `json:"source_map_digest"`
	SpeakerRevision   int64           `json:"speaker_identity_revision"`
	Policy            sd.Policy       `json:"policy"`
	Evidence          []sd.Hypothesis `json:"evidence"`
}
type MappingRevision struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
}
type SpeakerApplication struct {
	OperationID      string            `json:"operation_id"`
	RecordingID      string            `json:"recording_id"`
	Applied          int               `json:"applied"`
	Preserved        int               `json:"preserved"`
	Retracted        int               `json:"retracted"`
	Unknown          int               `json:"unknown"`
	Ambiguous        int               `json:"ambiguous"`
	Conflicting      int               `json:"conflicting"`
	SpeakerIDs       []string          `json:"speaker_ids"`
	MappingRevisions []MappingRevision `json:"mapping_revisions"`
}

func IndependentMapping(m SpeakerMapping) bool {
	return m.Origin == "manual" || m.Origin == "contextual"
}

func (s *Store) ApplySpeakerDiscovery(ctx context.Context, op, id string, expected int64, claim *Work) (out SpeakerApplication, e error) {
	if !contracts.ValidID(op) || !contracts.ValidID(id) || expected < 1 {
		return out, contracts.Fail("invalid_request")
	}
	digest, _ := intent([]any{"speaker-application", id, expected})
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		receipt, replayed, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if replayed {
			var result struct {
				Application SpeakerApplication `json:"speaker_application"`
			}
			if json.Unmarshal(receipt.Result, &result) != nil {
				return contracts.Fail("invalid_request")
			}
			out = result.Application
			return nil
		}
		if claim != nil {
			if _, err = s.workAuthority(ctx, tx, *claim); err != nil {
				return err
			}
		}
		var discovery SpeakerDiscovery
		if err = s.domainTx(ctx, tx, "Discoveries", id, &discovery); err != nil {
			return err
		}
		if discovery.Revision != expected {
			return contracts.Fail("conflict")
		}
		frozen, err := s.discoveryCurrentTx(ctx, tx, discovery)
		if err != nil {
			return err
		}
		var a sd.Aggregate
		json.Unmarshal(discovery.Aggregate, &a)
		independent, err := sd.IndependentContext(frozen.Context, a.Hypotheses, a.Policy)
		if err != nil {
			return err
		}
		eligible := map[string]sd.Decision{}
		for _, d := range independent {
			if d.Outcome == "supported-contextual" && d.SpeakerID != nil {
				eligible[d.VoiceID] = d
			}
		}
		var r Recording
		if err = s.domainTx(ctx, tx, "Recordings", id, &r); err != nil {
			return err
		}
		old, err := s.recordingMappings(ctx, tx, id)
		if err != nil {
			return err
		}
		byVoice := map[string]SpeakerMapping{}
		for _, m := range old {
			byVoice[m.LocalSpeakerID] = m
		}
		out = SpeakerApplication{OperationID: op, RecordingID: id, SpeakerIDs: []string{}, MappingRevisions: []MappingRevision{}}
		sourceMap, _ := canonical(r.SourceMap)
		people := map[string]bool{}
		changed := false
		for _, decision := range a.Decisions {
			existing, exists := byVoice[decision.VoiceID]
			if exists && existing.Origin == "manual" {
				out.Preserved++
				continue
			}
			d, ok := eligible[decision.VoiceID]
			if !ok || decision.SpeakerID == nil || decision.Outcome == "conflicting" || decision.Outcome == "ambiguous" || *d.SpeakerID != *decision.SpeakerID {
				if exists && existing.Origin == "contextual" && decision.Outcome == "conflicting" {
					if _, err = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND id=?", s.workspace, existing.ID); err != nil {
						return err
					}
					out.Retracted++
					changed = true
				}
				if exists && existing.Origin == "contextual" && decision.Outcome != "conflicting" {
					out.Preserved++
				}
				switch decision.Outcome {
				case "ambiguous":
					out.Ambiguous++
				case "conflicting":
					out.Conflicting++
				default:
					out.Unknown++
				}
				continue
			}
			var sp Speaker
			if err = s.domainTx(ctx, tx, "Speakers", *d.SpeakerID, &sp); err != nil {
				return err
			}
			if sp.State != "active" {
				return contracts.Fail("conflict")
			}
			evidence := []sd.Hypothesis{}
			for _, h := range a.Hypotheses {
				if h.VoiceID == d.VoiceID && h.Class == "self-identification" && h.Attributable && !h.Uncertain {
					evidence = append(evidence, h)
				}
			}
			lineage := ContextualLineage{int(discovery.MaxCues), 1, "independent-contextual", op, discovery.WorkID, discovery.Revision, hash(discovery.Aggregate), r.Revision, r.SourceDigest, hash(sourceMap), sp.Revision, a.Policy, evidence}
			raw, _ := json.Marshal(lineage)
			m := SpeakerMapping{ID: operationID("speaker-mapping", id, d.VoiceID), RecordingID: id, LocalSpeakerID: d.VoiceID, SpeakerID: *d.SpeakerID, DocumentDigest: r.DocumentDigest, Revision: rev + 1, Origin: "contextual", Provenance: raw}
			if !validSpeakerMapping(m) {
				return contracts.Fail("output_limit")
			}
			// An unchanged independently grounded assignment keeps its revision and corpus.
			if exists && existing.Origin == "contextual" && existing.SpeakerID == m.SpeakerID {
				if err = s.contextualMappingCurrentTx(ctx, tx, existing); err == nil {
					m = existing
				} else {
					return err
				}
			} else {
				if err = s.putDomain(ctx, tx, "SpeakerMappings", m); err != nil {
					return err
				}
				changed = true
			}
			out.Applied++
			people[m.SpeakerID] = true
			out.MappingRevisions = append(out.MappingRevisions, MappingRevision{m.ID, m.Revision})
		}
		for person := range people {
			out.SpeakerIDs = append(out.SpeakerIDs, person)
		}
		sort.Strings(out.SpeakerIDs)
		if changed {
			if err = s.reconcileRecordingEvidence(ctx, tx, r, r, rev+1); err != nil {
				return err
			}
		}
		mappings, err := s.recordingMappings(ctx, tx, id)
		if err != nil {
			return err
		}
		md, _ := intent(mappings)
		result, err := s.currentReceiptResult(ctx, tx, "mapping_recording_id", id, rev+1, recordingDigest(r))
		if err != nil {
			return err
		}
		result["mapping_digest"] = md
		result["speaker_application"] = out
		result["cleanup_entry_id"] = id
		_, err = s.accept(ctx, tx, op, digest, rev, result)
		return err
	})
	return
}
func (s *Store) RetractSpeakerDiscovery(ctx context.Context, op, id string, expected int64, targets []MappingRevision) (out SpeakerApplication, e error) {
	if !contracts.ValidID(op) || !contracts.ValidID(id) || expected < 1 || len(targets) < 1 || len(targets) > 1000 {
		return out, contracts.Fail("invalid_request")
	}
	targets = append([]MappingRevision{}, targets...)
	sort.Slice(targets, func(i, j int) bool { return targets[i].ID < targets[j].ID })
	for i, t := range targets {
		if !contracts.ValidID(t.ID) || t.Revision < 1 || i > 0 && targets[i-1].ID == t.ID {
			return out, contracts.Fail("invalid_request")
		}
	}
	digest, _ := intent([]any{"speaker-retraction", id, expected, targets})
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		receipt, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if ok {
			var result struct {
				Application SpeakerApplication `json:"speaker_application"`
			}
			json.Unmarshal(receipt.Result, &result)
			out = result.Application
			return nil
		}
		var r Recording
		if err = s.domainTx(ctx, tx, "Recordings", id, &r); err != nil {
			return err
		}
		if r.Revision != expected {
			return contracts.Fail("conflict")
		}
		for _, t := range targets {
			var m SpeakerMapping
			if err = s.domainTx(ctx, tx, "SpeakerMappings", t.ID, &m); err != nil {
				return err
			}
			if m.RecordingID != id || m.Revision != t.Revision || m.Origin != "contextual" {
				return contracts.Fail("conflict")
			}
		}
		for _, t := range targets {
			if _, err = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND id=?", s.workspace, t.ID); err != nil {
				return err
			}
		}
		if err = s.reconcileRecordingEvidence(ctx, tx, r, r, rev+1); err != nil {
			return err
		}
		mappings, err := s.recordingMappings(ctx, tx, id)
		if err != nil {
			return err
		}
		md, _ := intent(mappings)
		out = SpeakerApplication{OperationID: op, RecordingID: id, Retracted: len(targets), SpeakerIDs: []string{}, MappingRevisions: []MappingRevision{}}
		result, err := s.currentReceiptResult(ctx, tx, "mapping_recording_id", id, rev+1, recordingDigest(r))
		if err != nil {
			return err
		}
		result["mapping_digest"] = md
		result["speaker_application"] = out
		result["cleanup_entry_id"] = id
		_, err = s.accept(ctx, tx, op, digest, rev, result)
		return err
	})
	return
}
func (s *Store) contextualMappingCurrentTx(ctx context.Context, tx *sql.Tx, m SpeakerMapping) error {
	var p ContextualLineage
	if m.Origin != "contextual" || strict(m.Provenance, &p) != nil || p.Version != 1 || p.MaxCues < 1 || p.MaxCues > 500 || p.Class != "independent-contextual" || !contracts.ValidID(p.OperationID) || !contracts.ValidID(p.DiscoveryWorkID) || p.DiscoveryRevision < 1 || !digestPattern.MatchString(p.DiscoveryDigest) || len(p.Evidence) == 0 {
		return contracts.Fail("invalid_request")
	}
	var r Recording
	if e := s.domainTx(ctx, tx, "Recordings", m.RecordingID, &r); e != nil {
		return e
	}
	raw, _ := canonical(r.SourceMap)
	var sp Speaker
	if e := s.domainTx(ctx, tx, "Speakers", m.SpeakerID, &sp); e != nil {
		return e
	}
	if r.Revision != p.RecordingRevision || r.DocumentDigest != m.DocumentDigest || r.SourceDigest != p.SourceDigest || hash(raw) != p.SourceMapDigest || sp.Revision != p.SpeakerRevision || sp.State != "active" {
		return contracts.Fail("conflict")
	}
	context, _, err := recordingDiscoveryContext(r, p.MaxCues)
	if err != nil {
		return err
	}
	if err = sd.ValidateReferences(context, p.Evidence); err != nil {
		return err
	}
	identity, err := s.speakerTx(ctx, tx, m.SpeakerID)
	if err != nil {
		return err
	}
	aliases := []sd.Alias{}
	for _, a := range identity.Aliases {
		if a.State == "active" {
			aliases = append(aliases, sd.Alias{Text: a.Text, Language: a.Language, Scope: a.Scope})
		}
	}
	context.Identities = []sd.Identity{{ID: sp.ID, Name: sp.Name, Aliases: aliases}}
	decisions, err := sd.IndependentContext(context, p.Evidence, p.Policy)
	if err != nil {
		return err
	}
	supported := false
	for _, d := range decisions {
		if d.VoiceID == m.LocalSpeakerID && d.Outcome == "supported-contextual" && d.SpeakerID != nil && *d.SpeakerID == m.SpeakerID {
			supported = true
		}
	}
	if !supported {
		return contracts.Fail("invalid_request")
	}
	for _, h := range p.Evidence {
		if !sd.ValidHypothesis(h) || h.VoiceID != m.LocalSpeakerID || h.Class != "self-identification" || !h.Attributable || h.Uncertain || len(h.Roots) == 0 {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

func speakerOutputCurrent(v SpeakerOutput) bool {
	var m struct {
		State string `json:"state"`
	}
	return json.Unmarshal(v.Metadata, &m) == nil && m.State == "current"
}

// Identity/source changes retire contextual mappings before dependent output and
// work reconciliation. Absence of a discovery alone never retracts a mapping.
func (s *Store) reconcileContextualMappings(ctx context.Context, tx *sql.Tx, op string, revision int64, result map[string]json.RawMessage) error {
	relevant := false
	for _, key := range []string{"speaker_proofs", "recording_id", "mapping_recording_id", "media_id", "accepted"} {
		if _, ok := result[key]; ok {
			relevant = true
		}
	}
	if !relevant {
		return nil
	}
	// Historical identity attestation runs before origin columns are migrated.
	var schemaVersion int
	if err := s.row(ctx, tx, "SELECT schema_version FROM workspace WHERE id=?", s.workspace).Scan(&schemaVersion); err != nil {
		return err
	}
	if schemaVersion < 9 {
		return nil
	}
	cursor := ""
	changed := map[string]bool{}
	for {
		ids, e := s.identityIDs(ctx, tx, "speaker_mapping", "origin='contextual'", nil, cursor, 100)
		if e != nil {
			return e
		}
		for _, id := range ids {
			var m SpeakerMapping
			if e = s.domainTx(ctx, tx, "SpeakerMappings", id, &m); e != nil {
				return e
			}
			if e = s.contextualMappingCurrentTx(ctx, tx, m); e == nil {
				continue
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if e != sql.ErrNoRows {
				if _, ok := e.(*contracts.Error); !ok {
					return e
				}
			}
			if _, e = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND id=?", s.workspace, id); e != nil {
				return e
			}
			changed[m.RecordingID] = true
		}
		if len(ids) < 100 {
			break
		}
		cursor = ids[len(ids)-1]
	}
	if len(changed) == 0 {
		return nil
	}
	proofs := map[string]string{}
	var invalidated []string
	json.Unmarshal(result["invalidated_dataset_ids"], &invalidated)
	for id := range changed {
		if e := s.invalidateRecordingDatasets(ctx, tx, id, revision); e != nil {
			return e
		}
		ms, e := s.recordingMappings(ctx, tx, id)
		if e != nil {
			return e
		}
		proofs[id], _ = intent(ms)
		ids, e := s.invalidatedDatasetIDs(ctx, tx, id, revision)
		if e != nil {
			return e
		}
		invalidated = append(invalidated, ids...)
	}
	sort.Strings(invalidated)
	result["contextual_mapping_proofs"], _ = json.Marshal(proofs)
	result["invalidated_dataset_ids"], _ = json.Marshal(invalidated)
	result["graph_dirty"] = json.RawMessage("true")
	return nil
}

func (s *Store) validateContextualReceiptTx(ctx context.Context, tx *sql.Tx, m SpeakerMapping) error {
	var p ContextualLineage
	if strict(m.Provenance, &p) != nil {
		return contracts.Fail("invalid_request")
	}
	var raw string
	if e := s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, p.OperationID).Scan(&raw); e != nil {
		return e
	}
	var receipt struct {
		Application SpeakerApplication `json:"speaker_application"`
	}
	if json.Unmarshal([]byte(raw), &receipt) != nil || receipt.Application.OperationID != p.OperationID || receipt.Application.RecordingID != m.RecordingID {
		return contracts.Fail("invalid_request")
	}
	for _, ref := range receipt.Application.MappingRevisions {
		if ref.ID == m.ID && ref.Revision == m.Revision {
			return nil
		}
	}
	return contracts.Fail("invalid_request")
}
