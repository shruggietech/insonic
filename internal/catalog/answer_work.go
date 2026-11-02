// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
)

// AnswerCandidate is a compact provisional publication reference, not accepted output.
type AnswerCandidate struct {
	PublicationID string   `json:"publication_id"`
	PackageID     string   `json:"package_id"`
	Digest        string   `json:"digest"`
	Size          int64    `json:"size"`
	SourceEpoch   string   `json:"source_epoch"`
	MediaIDs      []string `json:"media_ids,omitempty"`
}

func (c AnswerCandidate) Valid() bool {
	return contracts.ValidID(c.PublicationID) && contracts.ValidID(c.PackageID) && answers.ValidDigest(c.Digest) && answers.ValidDigest(c.SourceEpoch) && c.Size > 0 && c.Size <= answers.MaxPackageBytes && answerScope(c.MediaIDs) == nil
}

// AcceptAnswer serializes source freshness, work cancellation/lease authority,
// publication integrity and terminal success. A lost response replays actual state.
func (s *Store) AcceptAnswer(ctx context.Context, claim Work, c AnswerCandidate, result json.RawMessage) (out Work, err error) {
	var reported struct {
		Candidate AnswerCandidate `json:"candidate"`
		Accepted  bool            `json:"accepted"`
		AnswerID  string          `json:"answer_id"`
	}
	if !c.Valid() || !referenceResult(result) || json.Unmarshal(result, &reported) != nil || !reported.Accepted || reported.AnswerID != c.PublicationID || !reflect.DeepEqual(reported.Candidate, c) {
		return out, contracts.Fail("invalid_request")
	}
	err = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if e := s.domainTx(ctx, tx, "Works", claim.ID, &out); e != nil {
			return e
		}
		if out.Kind != "answers.request" {
			return contracts.Fail("invalid_request")
		}
		if out.State == "succeeded" {
			if reflect.DeepEqual(out.Result, result) {
				return nil
			}
			return contracts.Fail("conflict")
		}
		w, e := s.workAuthority(ctx, tx, claim)
		if e != nil {
			return e
		}
		var frozen struct {
			PublicationID string          `json:"publication_id"`
			Request       answers.Request `json:"request"`
		}
		if json.Unmarshal(w.Payload, &frozen) != nil || frozen.PublicationID != c.PublicationID || !reflect.DeepEqual(frozen.Request.Filters.MediaIDs, c.MediaIDs) {
			return contracts.Fail("conflict")
		}
		p, e := s.publicationTx(ctx, tx, c.PublicationID)
		if e != nil {
			return e
		}
		if p.State != "available" || p.Kind != answers.GroundedKind || p.Digest != c.Digest || p.Size != c.Size || p.Verification != "sha256-readback" || !p.References[claim.ID] {
			return contracts.Fail("conflict")
		}
		epoch, e := s.answerEpochTx(ctx, tx, c.MediaIDs, 100000, 8<<20)
		if e != nil {
			return e
		}
		if !epoch.Complete || epoch.Digest != c.SourceEpoch {
			return contracts.Condition("conflict", "source_authority_changed")
		}
		// The bounded source scan must not outlive the work lease inside this transaction.
		if _, e = s.workAuthority(ctx, tx, claim); e != nil {
			return e
		}
		w.State = "succeeded"
		w.Phase = "complete"
		w.LeaseUntil = 0
		w.Result = result
		w.Error = ""
		if e = s.recordWork(ctx, tx, &w, rev); e != nil {
			return e
		}
		out = w
		return nil
	})
	return out, err
}
