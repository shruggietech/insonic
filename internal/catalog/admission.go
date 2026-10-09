// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"strconv"
)

func admissionOperation(work string, ordinal int) string {
	return operationID("admission-current", work, strconv.Itoa(ordinal))
}

// CommitAdmission accepts media and its selected transcript in one revision.
// Candidate artifacts are verified first; current references and cleanup
// obligations change only under this live work fence.
func (s *Store) CommitAdmission(ctx context.Context, claim Work, ordinal int, expectedRecording int64, entry LibraryEntry, r *Recording, itemResult ...json.RawMessage) (LibraryEntry, *Recording, error) {
	return s.CommitElectedAdmission(ctx, claim, ordinal, expectedRecording, entry, r, AdmissionElection{}, itemResult...)
}
func (s *Store) CommitElectedAdmission(ctx context.Context, claim Work, ordinal int, expectedRecording int64, entry LibraryEntry, r *Recording, election AdmissionElection, itemResult ...json.RawMessage) (LibraryEntry, *Recording, error) {
	if election.ExpectedRosterRevision < 0 || election.ReplaceAudio && r == nil || election.RosterPolicy != "" && election.RosterPolicy != "retain" && election.RosterPolicy != "clear" || election.ReplaceAudio && election.SpeakerIDs != nil || !election.ReplaceAudio && election.RosterPolicy != "" {
		return entry, r, contracts.Fail("invalid_request")
	}
	if len(itemResult) > 1 || len(itemResult) == 1 && !referenceResult(itemResult[0]) || ordinal < 0 || ordinal >= 10000 || expectedRecording < 0 || !validLibrary(entry) || (r != nil && (r.ID != entry.ID || r.SourceDigest != entry.Digest)) {
		return entry, r, contracts.Fail("invalid_request")
	}
	expectedLibrary := entry.Revision
	entry.Revision = 0
	var elected *Recording
	if r != nil {
		copy := *r
		copy.Revision = 0
		copy.SourceRevision = 0
		elected = &copy
	}
	intentFields := []any{"admission", ordinal, expectedLibrary, expectedRecording, entry, elected}
	if election.ReplaceAudio || election.ExpectedRosterRevision != 0 || election.RosterPolicy != "" || election.SpeakerIDs != nil {
		intentFields = append(intentFields, election)
	}
	digest, e := intent(intentFields)
	if e != nil {
		return entry, r, e
	}
	op := admissionOperation(claim.ID, ordinal)
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, e := s.workAuthority(ctx, tx, claim); e != nil {
			return e
		}
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			if e = s.domainTx(ctx, tx, "Library", entry.ID, &entry); e != nil {
				return e
			}
			if r != nil {
				return s.domainTx(ctx, tx, "Recordings", r.ID, r)
			}
			return nil
		}
		var old LibraryEntry
		oldErr := s.domainTx(ctx, tx, "Library", entry.ID, &old)
		if oldErr != nil && oldErr != sql.ErrNoRows {
			return oldErr
		}
		if election.ReplaceAudio && (oldErr != nil || expectedLibrary == 0) {
			return contracts.Fail("conflict")
		}
		if e = s.saveElectedLibrary(ctx, tx, &entry, expectedLibrary, rev, election.ReplaceAudio); e != nil {
			return e
		}
		if old.SubtitlePublicationID != nil && entry.SubtitlePublicationID == nil {
			if e = s.queueRemoved(ctx, tx, entry.ID, publicationList(old.SubtitlePublicationID), publicationList(nil), rev+1); e != nil {
				return e
			}
			// The legacy asset row is not an independent retained subtitle authority.
			if _, e = s.exec(ctx, tx, "DELETE FROM media_asset WHERE workspace_id=? AND id=? AND role='subtitle-source' AND NOT EXISTS(SELECT 1 FROM library_entry WHERE workspace_id=? AND subtitle_publication_id=?)", s.workspace, *old.SubtitlePublicationID, s.workspace, *old.SubtitlePublicationID); e != nil {
				return e
			}
		}
		result, e := s.currentReceiptResult(ctx, tx, "media_id", entry.ID, entry.Revision, libraryDigest(entry))
		if e != nil {
			return e
		}
		result["library_digest"] = result["record_digest"]
		delete(result, "record_digest")
		result["admission_ordinal"] = ordinal
		result["admission_work_id"] = claim.ID
		result["state"] = "imported"
		if expectedLibrary > 0 {
			result["state"] = "replaced"
		}
		if r != nil {
			r.SourceDigest = entry.Digest
			r.SourceRevision = entry.Revision
			if !validRecording(*r) {
				return contracts.Fail("invalid_request")
			}
			// Imported equal tokens alone cannot carry a prior identity election.
			if election.ReplaceAudio {
				if _, e = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND recording_id=?", s.workspace, r.ID); e != nil {
					return e
				}
			}
			if _, e = s.exec(ctx, tx, "DELETE FROM speaker_mapping WHERE workspace_id=? AND recording_id=? AND document_digest<>?", s.workspace, r.ID, r.DocumentDigest); e != nil {
				return e
			}
			if e = s.saveRecording(ctx, tx, expectedRecording, r, rev); e != nil {
				return e
			}
			recordingResult, e := s.currentReceiptResult(ctx, tx, "recording_id", r.ID, r.Revision, recordingDigest(*r))
			if e != nil {
				return e
			}
			result["recording_id"] = r.ID
			result["recording_digest"] = recordingResult["record_digest"]
			result["document_digest"] = r.DocumentDigest
			mappings, e := s.recordingMappings(ctx, tx, r.ID)
			if e != nil {
				return e
			}
			result["mapping_digest"], e = intent(mappings)
			if e != nil {
				return e
			}
			// Both current domains share the same cleanup entry and revision.
			result["cleanup_ids"] = recordingResult["cleanup_ids"]
		}
		roster, e := s.rosterTx(ctx, tx, entry.ID)
		if e != nil {
			return e
		}
		if election.ReplaceAudio {
			if roster.Revision != election.ExpectedRosterRevision || roster.Declared && election.RosterPolicy == "" {
				return contracts.Fail("conflict")
			}
			if election.RosterPolicy != "" {
				roster, e = s.saveRoster(ctx, tx, entry.ID, election.ExpectedRosterRevision, rev, election.RosterPolicy, nil)
			}
		} else if election.SpeakerIDs != nil {
			if expectedLibrary != 0 {
				return contracts.Fail("invalid_request")
			}
			roster, e = s.saveRoster(ctx, tx, entry.ID, 0, rev, "replace", election.SpeakerIDs)
		}
		if e != nil {
			return e
		}
		if roster.Declared {
			result["roster_proofs"] = map[string]string{entry.ID: rosterDigest(roster)}
		}
		if len(itemResult) == 1 {
			var detail map[string]json.RawMessage
			if strict(itemResult[0], &detail) != nil {
				return contracts.Fail("invalid_request")
			}
			if r != nil {
				detail["recording_revision"], _ = json.Marshal(r.Revision)
				detail["document_digest"], _ = json.Marshal(r.DocumentDigest)
			}
			result["admission_result"] = detail
			if e = s.scrubAdmissionInput(ctx, tx, claim, ordinal, entry.ID, op, result); e != nil {
				return e
			}
		}
		_, e = s.accept(ctx, tx, op, digest, rev, result)
		return e
	})
	return entry, r, e
}
func (s *Store) AcceptedAdmissionWork(ctx context.Context, work string, ordinal int) (json.RawMessage, bool, error) {
	if !contracts.ValidID(work) || ordinal < 0 || ordinal >= 10000 {
		return nil, false, contracts.Fail("invalid_request")
	}
	var raw string
	e := s.db.QueryRowContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, admissionOperation(work, ordinal)).Scan(&raw)
	if e == sql.ErrNoRows {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, sanitize(e)
	}
	return json.RawMessage(raw), true, nil
}

// SkipAdmission seals a successful collision result without changing current media.
func (s *Store) SkipAdmission(ctx context.Context, claim Work, ordinal int, itemResult json.RawMessage) error {
	if ordinal < 0 || ordinal >= 10000 || !referenceResult(itemResult) {
		return contracts.Fail("invalid_request")
	}
	var detail struct {
		MediaID string `json:"media_id"`
		State   string `json:"state"`
	}
	if json.Unmarshal(itemResult, &detail) != nil || !contracts.ValidID(detail.MediaID) || detail.State != "skipped" {
		return contracts.Fail("invalid_request")
	}
	digest, err := intent([]any{"admission-skip", ordinal, itemResult})
	if err != nil {
		return err
	}
	op := admissionOperation(claim.ID, ordinal)
	return s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, err := s.workAuthority(ctx, tx, claim); err != nil {
			return err
		}
		_, ok, err := s.replay(ctx, tx, op, digest)
		if err != nil || ok {
			return err
		}
		result := map[string]any{"admission_work_id": claim.ID, "admission_ordinal": ordinal, "admission_result": itemResult, "state": "skipped"}
		if err = s.scrubAdmissionInput(ctx, tx, claim, ordinal, detail.MediaID, op, result); err != nil {
			return err
		}
		_, err = s.accept(ctx, tx, op, digest, rev, result)
		return err
	})
}

func importInputProof(payload json.RawMessage) (string, []string, error) {
	var envelope map[string]json.RawMessage
	if strict(payload, &envelope) != nil {
		return "", nil, contracts.Fail("invalid_request")
	}
	var items []json.RawMessage
	if strict(envelope["items"], &items) != nil || len(items) == 0 || len(items) > 10000 {
		return "", nil, contracts.Fail("invalid_request")
	}
	delete(envelope, "items")
	base, err := intent(envelope)
	if err != nil {
		return "", nil, err
	}
	digests := make([]string, len(items))
	for i, item := range items {
		digests[i], err = intent(item)
		if err != nil {
			return "", nil, err
		}
	}
	return base, digests, nil
}

type importInitialProof struct {
	IntentDigest   string   `json:"input_intent_digest"`
	EnvelopeDigest string   `json:"input_envelope_digest"`
	ItemDigests    []string `json:"item_intent_digests"`
}

func (s *Store) importInitial(ctx context.Context, tx *sql.Tx, w Work) (importInitialProof, string, error) {
	var raw, digest string
	err := s.row(ctx, tx, "SELECT result,digest FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, w.ID).Scan(&raw, &digest)
	var proof importInitialProof
	if err != nil || json.Unmarshal([]byte(raw), &proof) != nil {
		return proof, digest, contracts.Fail("invalid_request")
	}
	return proof, digest, nil
}
func (s *Store) validateWorkInput(ctx context.Context, tx *sql.Tx, w Work, initialDigest string) error {
	want, _ := intent([]any{"work", w.Kind, w.Payload})
	if want == initialDigest {
		return nil
	}
	if w.Kind == "recordings.match" {
		return s.validateScrubbedSpeakerWorkInput(ctx, tx, w, initialDigest)
	}
	if w.Kind != "media.import" && w.Kind != "library.import" {
		return contracts.Fail("invalid_request")
	}
	proof, digest, err := s.importInitial(ctx, tx, w)
	if err != nil || proof.IntentDigest != digest || digest != initialDigest {
		return contracts.Fail("invalid_request")
	}
	base, items, err := importInputProof(w.Payload)
	if err != nil || base != proof.EnvelopeDigest || len(items) != len(proof.ItemDigests) {
		return contracts.Fail("invalid_request")
	}
	var envelope struct {
		Items []json.RawMessage `json:"items"`
	}
	json.Unmarshal(w.Payload, &envelope)
	for ordinal, item := range envelope.Items {
		if items[ordinal] == proof.ItemDigests[ordinal] {
			continue
		}
		var marker map[string]string
		if strict(item, &marker) != nil || len(marker) != 3 || marker["source"] != "<accepted>" || marker["accepted_receipt"] != admissionOperation(w.ID, ordinal) || !contracts.ValidID(marker["record"]) {
			return contracts.Fail("invalid_request")
		}
		var raw string
		if err = s.row(ctx, tx, "SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?", s.workspace, marker["accepted_receipt"]).Scan(&raw); err != nil {
			return contracts.Fail("invalid_request")
		}
		var accepted struct {
			WorkID      string `json:"admission_work_id"`
			Ordinal     int    `json:"admission_ordinal"`
			Digest      string `json:"accepted_item_digest"`
			InputDigest string `json:"work_input_digest"`
			Result      struct {
				MediaID string `json:"media_id"`
			} `json:"admission_result"`
		}
		if json.Unmarshal([]byte(raw), &accepted) != nil || accepted.WorkID != w.ID || accepted.Ordinal != ordinal || accepted.Digest != proof.ItemDigests[ordinal] || accepted.InputDigest != initialDigest || accepted.Result.MediaID != marker["record"] {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}
func (s *Store) scrubAdmissionInput(ctx context.Context, tx *sql.Tx, claim Work, ordinal int, recording, op string, result map[string]any) error {
	work, err := s.workAuthority(ctx, tx, claim)
	if err != nil {
		return err
	}
	proof, initialDigest, err := s.importInitial(ctx, tx, work)
	if err != nil {
		return err
	}
	// Historical fixture work without an import-manifest retains its original proof.
	if proof.IntentDigest == "" {
		return nil
	}
	if err = s.validateWorkInput(ctx, tx, work, initialDigest); err != nil {
		return err
	}
	var envelope map[string]json.RawMessage
	if strict(work.Payload, &envelope) != nil {
		return contracts.Fail("invalid_request")
	}
	var items []json.RawMessage
	if strict(envelope["items"], &items) != nil || ordinal >= len(items) {
		return contracts.Fail("invalid_request")
	}
	result["accepted_item_digest"] = proof.ItemDigests[ordinal]
	result["work_input_digest"] = initialDigest
	items[ordinal], _ = json.Marshal(map[string]string{"source": "<accepted>", "accepted_receipt": op, "record": recording})
	envelope["items"], _ = json.Marshal(items)
	work.Payload, _ = json.Marshal(envelope)
	work.JournalReceiptID = op
	if !validWork(work) {
		return contracts.Fail("invalid_request")
	}
	if err = s.putDomain(ctx, tx, "Works", work); err != nil {
		return err
	}
	result["work_id"] = work.ID
	result["work_digest"] = workJournalDigest(work)
	return nil
}
