// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"sort"
)

type SpeakerDiscovery struct {
	ID                string          `json:"id"`
	WorkID            string          `json:"work_id"`
	Revision          int64           `json:"revision"`
	RecordingRevision int64           `json:"recording_revision"`
	DocumentDigest    string          `json:"document_digest"`
	SourceEpoch       string          `json:"source_epoch"`
	MaxCues           int64           `json:"max_cues"`
	MatchingWorkID    *string         `json:"matching_work_id"`
	Aggregate         json.RawMessage `json:"aggregate"`
}
type SpeakerDiscoverySnapshot struct {
	RecordingID       string      `json:"recording_id"`
	RecordingRevision int64       `json:"recording_revision"`
	DocumentDigest    string      `json:"document_digest"`
	SourceEpoch       string      `json:"source_epoch"`
	MaxCues           int         `json:"max_cues"`
	MatchingWorkID    string      `json:"matching_work_id,omitempty"`
	ExpectedRevision  int64       `json:"expected_revision"`
	Coverage          sd.Coverage `json:"coverage"`
	Context           sd.Context  `json:"-"`
}

func validSpeakerDiscovery(v SpeakerDiscovery) bool {
	var a sd.Aggregate
	return contracts.ValidID(v.ID) && contracts.ValidID(v.WorkID) && v.Revision > 0 && v.RecordingRevision > 0 && digestPattern.MatchString(v.DocumentDigest) && digestPattern.MatchString(v.SourceEpoch) && v.MaxCues > 0 && v.MaxCues <= 500 && (v.MatchingWorkID == nil || contracts.ValidID(*v.MatchingWorkID)) && len(v.Aggregate) <= sd.MaxBytes && strict(v.Aggregate, &a) == nil && sd.ValidAggregate(a)
}
func (s *Store) FreezeSpeakerDiscovery(ctx context.Context, id string, maxCues int, matchingWorkID string) (out SpeakerDiscoverySnapshot, e error) {
	if !contracts.ValidID(id) || maxCues < 1 || maxCues > 500 || (matchingWorkID != "" && !contracts.ValidID(matchingWorkID)) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	out, e = s.freezeSpeakerDiscoveryTx(ctx, tx, id, maxCues, matchingWorkID)
	return out, sanitize(e)
}
func (s *Store) freezeSpeakerDiscoveryTx(ctx context.Context, tx *sql.Tx, id string, maxCues int, matchID string) (out SpeakerDiscoverySnapshot, e error) {
	out.RecordingID = id
	out.MaxCues = maxCues
	out.MatchingWorkID = matchID
	var size int64
	if e = s.row(ctx, tx, "SELECT "+s.answerBytes("document")+" FROM current_recording WHERE workspace_id=? AND id=?", s.workspace, id).Scan(&size); e != nil {
		return
	}
	if size > 32<<20 {
		return out, contracts.Fail("input_limit")
	}
	var r Recording
	if e = s.domainTx(ctx, tx, "Recordings", id, &r); e != nil {
		return
	}
	if r.State != "ready" {
		return out, contracts.Condition("conflict", "transcript_required")
	}
	if e = s.validateRecordingSource(ctx, tx, r, false); e != nil {
		return
	}
	out.RecordingRevision = r.Revision
	out.DocumentDigest = r.DocumentDigest
	var old SpeakerDiscovery
	if err := s.domainTx(ctx, tx, "Discoveries", id, &old); err == nil {
		out.ExpectedRevision = old.Revision
	} else if err != sql.ErrNoRows {
		return out, err
	}
	roster, e := s.rosterTx(ctx, tx, id)
	if e != nil {
		return out, e
	}
	mappings, e := s.recordingMappings(ctx, tx, id)
	if e != nil {
		return out, e
	}
	speakers, e := s.identityRecords(ctx, tx, "Speakers", "state='active'", nil, 1001)
	if e != nil {
		return out, e
	}
	aliases, e := s.identityRecords(ctx, tx, "Aliases", "state='active'", nil, 16001)
	if e != nil {
		return out, e
	}
	if len(speakers.Speakers) > 1000 || len(aliases.Aliases) > 16000 {
		return out, contracts.Fail("input_limit")
	}
	out.Context = sd.Context{Cues: []sd.Cue{}, Voices: []string{}, Identities: []sd.Identity{}, Owners: map[string]string{}}
	rosterIDs := map[string]bool{}
	for _, sp := range roster.Members {
		rosterIDs[sp.ID] = true
	}
	aliasMap := map[string][]sd.Alias{}
	for _, a := range aliases.Aliases {
		aliasMap[a.SpeakerID] = append(aliasMap[a.SpeakerID], sd.Alias{Text: a.Text, Language: a.Language, Scope: a.Scope})
	}
	for _, sp := range speakers.Speakers {
		a := aliasMap[sp.ID]
		if a == nil {
			a = []sd.Alias{}
		}
		out.Context.Identities = append(out.Context.Identities, sd.Identity{ID: sp.ID, Name: sp.Name, Aliases: a, Roster: rosterIDs[sp.ID]})
	}
	if b, _ := json.Marshal(out.Context.Identities); len(b) > sd.MaxBytes {
		return out, contracts.Fail("input_limit")
	}
	ownerAuthority := []Speaker{}
	activeIDs := map[string]bool{}
	for _, sp := range speakers.Speakers {
		activeIDs[sp.ID] = true
	}
	for _, m := range mappings {
		if m.Origin == "manual" {
			out.Context.Owners[m.LocalSpeakerID] = m.SpeakerID
			if !activeIDs[m.SpeakerID] {
				var sp Speaker
				if err := s.domainTx(ctx, tx, "Speakers", m.SpeakerID, &sp); err != nil {
					return out, err
				}
				ownerAuthority = append(ownerAuthority, sp)
				out.Context.OwnerIdentities = append(out.Context.OwnerIdentities, sd.Identity{ID: sp.ID, Name: sp.Name})
				activeIDs[sp.ID] = true
			}
			out.Context.OwnerEvidence = append(out.Context.OwnerEvidence, sd.OwnerEvidence{MappingID: m.ID, Revision: m.Revision, VoiceID: m.LocalSpeakerID})
		}
	}
	var doc struct {
		Cues []struct {
			ID      string `json:"id"`
			Payload struct {
				Text string `json:"plain_text"`
			} `json:"payload"`
			Attributions []struct {
				ID string `json:"speaker_id"`
			} `json:"speaker_attributions"`
		} `json:"cues"`
	}
	if json.Unmarshal(r.Document, &doc) != nil {
		return out, contracts.Fail("invalid_request")
	}
	out.Coverage.Total = len(doc.Cues)
	voices := map[string]bool{}
	cueIDs := map[string]bool{}
	for _, cue := range doc.Cues {
		if cue.ID == "" || cueIDs[cue.ID] {
			return out, contracts.Fail("invalid_request")
		}
		cueIDs[cue.ID] = true
		for _, a := range cue.Attributions {
			voices[a.ID] = true
		}
		if len(out.Context.Cues) >= maxCues {
			continue
		}
		c := sd.Cue{ID: cue.ID, Text: cue.Payload.Text, Voices: []string{}, Roots: []string{}}
		for _, a := range cue.Attributions {
			c.Voices = append(c.Voices, a.ID)
		}
		sort.Strings(c.Voices)
		out.Context.Cues = append(out.Context.Cues, c)
	}
	for voice := range voices {
		out.Context.Voices = append(out.Context.Voices, voice)
	}
	sort.Strings(out.Context.Voices)
	if len(voices) > 1000 {
		return out, contracts.Fail("input_limit")
	}
	if e = discoveryRoots(r, &out.Context); e != nil {
		return out, e
	}
	out.Coverage.Selected = len(out.Context.Cues)
	out.Coverage.Omitted = out.Coverage.Total - out.Coverage.Selected
	out.Context.Acoustic, e = s.speakerDiscoveryAcousticTx(ctx, tx, id, matchID)
	if e != nil {
		return out, e
	}
	out.SourceEpoch, _ = intent([]any{r.ID, recordingDigest(r), r.Revision, speakers.Speakers, aliases.Aliases, roster, mappings, ownerAuthority, out.Context.Acoustic, maxCues})
	return
}
func matchingSelection(v SpeakerDiscovery) string {
	if v.MatchingWorkID != nil {
		return *v.MatchingWorkID
	}
	return ""
}
func (s *Store) discoveryCurrentTx(ctx context.Context, tx *sql.Tx, v SpeakerDiscovery) (SpeakerDiscoverySnapshot, error) {
	if !validSpeakerDiscovery(v) {
		return SpeakerDiscoverySnapshot{}, contracts.Fail("invalid_request")
	}
	f, e := s.freezeSpeakerDiscoveryTx(ctx, tx, v.ID, int(v.MaxCues), matchingSelection(v))
	if e != nil {
		return f, e
	}
	if f.SourceEpoch != v.SourceEpoch || f.RecordingRevision != v.RecordingRevision || f.DocumentDigest != v.DocumentDigest {
		return f, contracts.Condition("conflict", "source_authority_changed")
	}
	var a sd.Aggregate
	if strict(v.Aggregate, &a) != nil {
		return f, contracts.Fail("invalid_request")
	}
	if e = validateDiscoveryAggregate(f, a); e != nil {
		return f, e
	}
	return f, nil
}
func validateDiscoveryAggregate(f SpeakerDiscoverySnapshot, a sd.Aggregate) error {
	if !sd.ValidAggregate(a) || a.Coverage.Total != f.Coverage.Total || a.Coverage.Selected > f.Coverage.Selected || a.Coverage.Omitted != a.Coverage.Total-a.Coverage.Selected || sd.Digest(a.Acoustic) != sd.Digest(f.Context.Acoustic) {
		return contracts.Fail("invalid_request")
	}
	if sd.Digest(a.OwnerEvidence) != sd.Digest(f.Context.OwnerEvidence) {
		return contracts.Fail("invalid_request")
	}
	selected := f.Context
	selected.Cues = selected.Cues[:a.Coverage.Selected]
	if e := sd.ValidateReferences(selected, a.Hypotheses); e != nil {
		return e
	}
	decisions, e := sd.Resolve(selected, a.Hypotheses, a.Policy)
	if e != nil || sd.Digest(decisions) != sd.Digest(a.Decisions) {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func (s *Store) SpeakerDiscovery(ctx context.Context, id string) (out SpeakerDiscovery, e error) {
	if !contracts.ValidID(id) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	if e = s.domainTx(ctx, tx, "Discoveries", id, &out); e != nil {
		return SpeakerDiscovery{}, sanitize(e)
	}
	if _, e = s.discoveryCurrentTx(ctx, tx, out); e != nil {
		return SpeakerDiscovery{}, sanitize(e)
	}
	return out, nil
}
func (s *Store) AcceptSpeakerDiscovery(ctx context.Context, claim Work, frozen SpeakerDiscoverySnapshot, a sd.Aggregate) (out Work, e error) {
	if !contracts.ValidID(frozen.RecordingID) || frozen.MaxCues < 1 || frozen.MaxCues > 500 || frozen.ExpectedRevision < 0 || (frozen.MatchingWorkID != "" && !contracts.ValidID(frozen.MatchingWorkID)) || !sd.ValidAggregate(a) {
		return out, contracts.Fail("invalid_request")
	}
	raw, _ := json.Marshal(a)
	result, _ := json.Marshal(map[string]any{"accepted": true, "recording_id": frozen.RecordingID, "discovery_digest": hash(raw), "source_epoch": frozen.SourceEpoch})
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if err := s.domainTx(ctx, tx, "Works", claim.ID, &out); err != nil {
			return err
		}
		if out.Kind != "speakers.discover" {
			return contracts.Fail("invalid_request")
		}
		if out.State == "succeeded" {
			prior, priorErr := canonical(out.Result)
			desired, desiredErr := canonical(result)
			if priorErr == nil && desiredErr == nil && bytes.Equal(prior, desired) {
				return nil
			}
			return contracts.Fail("conflict")
		}
		w, err := s.workAuthority(ctx, tx, claim)
		if err != nil {
			return err
		}
		var payload struct {
			RecordingID      string `json:"recording_id"`
			SourceEpoch      string `json:"source_epoch"`
			MaxCues          int    `json:"max_cues"`
			ExpectedRevision int64  `json:"expected_revision"`
			MatchingWorkID   string `json:"matching_work_id"`
		}
		if json.Unmarshal(w.Payload, &payload) != nil || payload.RecordingID != frozen.RecordingID || payload.SourceEpoch != frozen.SourceEpoch || payload.MaxCues != frozen.MaxCues || payload.ExpectedRevision != frozen.ExpectedRevision || payload.MatchingWorkID != frozen.MatchingWorkID {
			return contracts.Fail("conflict")
		}
		f, err := s.freezeSpeakerDiscoveryTx(ctx, tx, frozen.RecordingID, frozen.MaxCues, frozen.MatchingWorkID)
		if err != nil {
			return err
		}
		if f.SourceEpoch != frozen.SourceEpoch || f.ExpectedRevision != frozen.ExpectedRevision {
			return contracts.Condition("conflict", "source_authority_changed")
		}
		if err = validateDiscoveryAggregate(f, a); err != nil {
			return err
		}
		if _, err = s.workAuthority(ctx, tx, claim); err != nil {
			return err
		}
		v := SpeakerDiscovery{ID: f.RecordingID, WorkID: w.ID, Revision: rev + 1, RecordingRevision: f.RecordingRevision, DocumentDigest: f.DocumentDigest, SourceEpoch: f.SourceEpoch, MaxCues: int64(f.MaxCues), Aggregate: raw}
		if f.MatchingWorkID != "" {
			id := f.MatchingWorkID
			v.MatchingWorkID = &id
		}
		if err = s.putDomain(ctx, tx, "Discoveries", v); err != nil {
			return err
		}
		w.State = "succeeded"
		w.Phase = "complete"
		w.LeaseUntil = 0
		w.Result = result
		w.Error = ""
		w.JournalReceiptID = contracts.ID()
		if !validWork(w) {
			return contracts.Fail("invalid_request")
		}
		if err = s.putDomain(ctx, tx, "Works", w); err != nil {
			return err
		}
		wd := workJournalDigest(w)
		proof, _ := intent(v)
		_, err = s.accept(ctx, tx, w.JournalReceiptID, wd, rev, map[string]any{"work_id": w.ID, "state": w.State, "generation": w.Generation, "work_digest": wd, "discovery_proofs": map[string]string{v.ID: proof}})
		out = w
		return err
	})
	return out, e
}
func (s *Store) ResolveSpeakerDiscovery(ctx context.Context, op, id string, expected int64, p sd.Policy) (out SpeakerDiscovery, e error) {
	if !contracts.ValidID(op) || !contracts.ValidID(id) || expected < 1 {
		return out, contracts.Fail("invalid_request")
	}
	p, e = sd.NormalizePolicy(p)
	if e != nil {
		return
	}
	digest, _ := intent([]any{"speaker-resolution", id, expected, p})
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, replayed, err := s.replay(ctx, tx, op, digest)
		if err != nil {
			return err
		}
		if err = s.domainTx(ctx, tx, "Discoveries", id, &out); err != nil {
			return err
		}
		f, err := s.discoveryCurrentTx(ctx, tx, out)
		if err != nil {
			return err
		}
		if replayed {
			return nil
		}
		if out.Revision != expected {
			return contracts.Fail("conflict")
		}
		var a sd.Aggregate
		json.Unmarshal(out.Aggregate, &a)
		a.Policy = p
		a.Decisions, err = sd.Resolve(f.Context, a.Hypotheses, p)
		if err != nil {
			return err
		}
		out.Aggregate, _ = json.Marshal(a)
		out.Revision = rev + 1
		if !validSpeakerDiscovery(out) {
			return contracts.Fail("output_limit")
		}
		if err = s.putDomain(ctx, tx, "Discoveries", out); err != nil {
			return err
		}
		proof, _ := intent(out)
		_, err = s.accept(ctx, tx, op, digest, rev, map[string]any{"discovery_proofs": map[string]string{id: proof}})
		return err
	})
	if e != nil {
		return SpeakerDiscovery{}, e
	}
	return
}
