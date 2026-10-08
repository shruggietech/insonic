// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"math"
	"reflect"
	"strconv"
	"strings"
)

type Extraction struct {
	ID                string          `json:"id"`
	Revision          int64           `json:"revision"`
	RecordingRevision int64           `json:"recording_revision"`
	DocumentDigest    string          `json:"document_digest"`
	Assertions        json.RawMessage `json:"assertions"`
	Provenance        json.RawMessage `json:"provenance"`
	Diagnostics       json.RawMessage `json:"diagnostics"`
}
type SavedQuery struct {
	ID          string          `json:"id"`
	Revision    int64           `json:"revision"`
	Definition  json.RawMessage `json:"definition"`
	Validations json.RawMessage `json:"validations"`
}
type GraphLayout struct {
	ID       string          `json:"id"`
	QueryID  string          `json:"query_id"`
	Revision int64           `json:"revision"`
	Layout   json.RawMessage `json:"layout"`
}

func validSaved(q SavedQuery) bool {
	var input contracts.QueryInput
	var validations []map[string]json.RawMessage
	if strict(q.Validations, &validations) != nil || len(validations) > 32 || !referenceMetadata(q.Validations) {
		return false
	}
	if input.Parameters == nil {
		input.Parameters = map[string]contracts.QueryParameter{}
	}
	if contracts.DecodeExplore(q.Definition, &input) != nil {
		return false
	}
	if input.Parameters == nil {
		input.Parameters = map[string]contracts.QueryParameter{}
	}
	document, _ := json.Marshal(map[string]any{"kind": "graph-query", "schema_version": contracts.Version, "workspace_id": "00000000-0000-4000-8000-000000000001", "query_id": q.ID, "revision": q.Revision, "title": input.Title, "access_mode": "read-only", "definition": input.Definition, "parameters": input.Parameters, "validations": q.Validations})
	if schemas.ValidateGraphQuery(document) != nil {
		return false
	}
	return contracts.ValidID(q.ID) && q.Revision > 0 && nonsecret(q.Definition) && referenceMetadata(q.Definition) && contracts.DecodeExplore(q.Definition, &input) == nil && strings.TrimSpace(input.Title) != "" && input.Validate() == nil
}
func validLayout(v GraphLayout) bool {
	var input struct {
		Positions map[string]struct {
			X float64 `json:"x"`
			Y float64 `json:"y"`
		} `json:"positions"`
		Paused bool   `json:"paused"`
		Filter string `json:"filter"`
		Limit  int    `json:"limit"`
	}
	if strict(v.Layout, &input) != nil {
		return false
	}
	for id, p := range input.Positions {
		if len(id) < 1 || len(id) > 1024 || math.IsNaN(p.X) || math.IsNaN(p.Y) || math.IsInf(p.X, 0) || math.IsInf(p.Y, 0) || math.Abs(p.X) > 1000000 || math.Abs(p.Y) > 1000000 {
			return false
		}
	}
	return contracts.ValidID(v.ID) && contracts.ValidID(v.QueryID) && v.Revision > 0 && nonsecret(v.Layout) && referenceMetadata(v.Layout) && strict(v.Layout, &input) == nil && len(input.Positions) <= 1000 && len(input.Filter) <= 512 && input.Limit >= 0 && input.Limit <= 500
}
func validExtraction(v Extraction) bool {
	return contracts.ValidID(v.ID) && v.Revision > 0 && v.RecordingRevision > 0 && digestPattern.MatchString(v.DocumentDigest) && len(v.Assertions) <= 512<<10 && referenceMetadata(v.Assertions) && referenceMetadata(v.Provenance) && referenceMetadata(v.Diagnostics)
}
func (s *Store) Extraction(ctx context.Context, id string) (v Extraction, e error) {
	e = s.readOne(ctx, "Extractions", id, &v)
	return
}
func (s *Store) CommitExtraction(ctx context.Context, claim Work, v Extraction) (Extraction, error) {
	v.Revision = 0
	digest, e := intent(v)
	if e != nil {
		return v, e
	}
	op := operationID("extraction-current", claim.ID, v.ID)
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		if _, e := s.workAuthority(ctx, tx, claim); e != nil {
			return e
		}
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Extractions", v.ID, &v)
		}
		var current Recording
		if e = s.domainTx(ctx, tx, "Recordings", v.ID, &current); e != nil {
			return e
		}
		if current.Revision != v.RecordingRevision || current.DocumentDigest != v.DocumentDigest || validateExtractionCues(v, current) != nil {
			return contracts.Fail("conflict")
		}
		v.Revision = rev + 1
		if validateExtractionCues(v, current) != nil {
			return contracts.Fail("invalid_request")
		}
		if !validExtraction(v) {
			return contracts.Fail("invalid_request")
		}
		if e = s.putDomain(ctx, tx, "Extractions", v); e != nil {
			return e
		}
		proof, _ := intent(v)
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"extraction_proofs": map[string]string{v.ID: proof}, "graph_dirty": true})
		return e
	})
	return v, e
}
func (s *Store) AcceptedExtractionWork(ctx context.Context, job, id string) (json.RawMessage, bool, error) {
	var raw string
	e := s.db.QueryRowContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? AND id=?"), s.workspace, operationID("extraction-current", job, id)).Scan(&raw)
	if e == sql.ErrNoRows {
		return nil, false, nil
	}
	return json.RawMessage(raw), e == nil, sanitize(e)
}
func (s *Store) SavedQueries(ctx context.Context, id string) ([]SavedQuery, error) {
	if id != "" && !contracts.ValidID(id) {
		return nil, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return nil, sanitize(e)
	}
	defer tx.Rollback()
	r, e := s.readRecords(ctx, tx, "SavedQueries")
	if e != nil {
		return nil, sanitize(e)
	}
	out := []SavedQuery{}
	for _, q := range r.SavedQueries {
		if id == "" || q.ID == id {
			out = append(out, q)
		}
	}
	return out, sanitize(tx.Commit())
}
func (s *Store) PutSavedQuery(ctx context.Context, op string, expected int64, q SavedQuery) (SavedQuery, error) {
	if !validIdentityMutation(op, expected) {
		return q, contracts.Fail("invalid_request")
	}
	q.Revision = expected + 1
	if len(q.Validations) == 0 {
		q.Validations = json.RawMessage(`[]`)
	}
	q.Validations, _ = canonical(q.Validations)
	q.Definition, _ = canonical(q.Definition)
	if !validSaved(q) {
		return q, contracts.Fail("invalid_request")
	}
	digest, e := intent([]any{"saved-query", expected, q.ID, q.Revision, q.Definition})
	if e != nil {
		return q, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			var definition, validations string
			if e := s.row(ctx, tx, "SELECT definition,validations FROM saved_query WHERE workspace_id=? AND id=? AND revision=?", s.workspace, q.ID, q.Revision).Scan(&definition, &validations); e != nil {
				return e
			}
			q.Definition = json.RawMessage(definition)
			q.Validations = json.RawMessage(validations)
			return nil
		}
		var latest sql.NullInt64
		if e = s.row(ctx, tx, "SELECT max(revision) FROM saved_query WHERE workspace_id=? AND id=?", s.workspace, q.ID).Scan(&latest); e != nil {
			return e
		}
		if latest.Int64 != expected {
			return contracts.Fail("conflict")
		}
		if e = s.insertRecords(ctx, tx, Records{SavedQueries: []SavedQuery{q}}); e != nil {
			return e
		}
		proof, _ := intent(q)
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"query_proofs": map[string]string{q.ID + ":" + strconv.FormatInt(q.Revision, 10): proof}})
		return e
	})
	return q, e
}
func (s *Store) GraphLayout(ctx context.Context, id string) (v GraphLayout, e error) {
	e = s.readOne(ctx, "Layouts", id, &v)
	return
}
func (s *Store) PutGraphLayout(ctx context.Context, op string, expected int64, v GraphLayout) (GraphLayout, error) {
	if !validIdentityMutation(op, expected) {
		return v, contracts.Fail("invalid_request")
	}
	v.Revision = 0
	digest, e := intent([]any{"graph-layout", expected, v})
	if e != nil {
		return v, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			return s.domainTx(ctx, tx, "Layouts", v.ID, &v)
		}
		if e = s.identityCAS(ctx, tx, "graph_layout", v.ID, expected); e != nil {
			return e
		}
		var count int
		if e = s.row(ctx, tx, "SELECT count(*) FROM saved_query WHERE workspace_id=? AND id=?", s.workspace, v.QueryID).Scan(&count); e != nil {
			return e
		}
		if count == 0 {
			return contracts.Fail("not_found")
		}
		v.Revision = rev + 1
		if !validLayout(v) {
			return contracts.Fail("invalid_request")
		}
		if e = s.putDomain(ctx, tx, "Layouts", v); e != nil {
			return e
		}
		proof, _ := intent(v)
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"layout_proofs": map[string]string{v.ID: proof}})
		return e
	})
	return v, e
}
func (s *Store) validateExplorationState(ctx context.Context, tx *sql.Tx) error {
	expected := map[string]map[string]string{"query_proofs": {}, "layout_proofs": {}, "extraction_proofs": {}}
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var r map[string]json.RawMessage
		if strict([]byte(raw), &r) != nil {
			e = contracts.Fail("invalid_request")
			break
		}
		for key, m := range expected {
			if raw, ok := r[key]; ok {
				var p map[string]string
				if strict(raw, &p) != nil {
					e = contracts.Fail("invalid_request")
					break
				}
				for id, d := range p {
					m[id] = d
				}
			}
		}
	}
	re := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if re != nil {
		return re
	}
	actual := map[string]map[string]string{"query_proofs": {}, "layout_proofs": {}, "extraction_proofs": {}}
	r, e := s.readRecords(ctx, tx)
	if e != nil {
		return e
	}
	queryVersions := map[string]int64{}
	for _, q := range r.SavedQueries {
		if q.Revision != queryVersions[q.ID]+1 {
			return contracts.Fail("invalid_request")
		}
		queryVersions[q.ID] = q.Revision
		if !validSaved(q) {
			return contracts.Fail("invalid_request")
		}
		actual["query_proofs"][q.ID+":"+strconv.FormatInt(q.Revision, 10)], _ = intent(q)
	}
	for _, v := range r.Layouts {
		if !validLayout(v) || queryVersions[v.QueryID] == 0 {
			return contracts.Fail("invalid_request")
		}
		actual["layout_proofs"][v.ID], _ = intent(v)
	}
	for _, v := range r.Extractions {
		if !validExtraction(v) {
			return contracts.Fail("invalid_request")
		}
		var current Recording
		if e = s.domainTx(ctx, tx, "Recordings", v.ID, &current); e != nil {
			return e
		}
		if current.Revision != v.RecordingRevision || current.DocumentDigest != v.DocumentDigest || validateExtractionCues(v, current) != nil {
			return contracts.Fail("invalid_request")
		}
		actual["extraction_proofs"][v.ID], _ = intent(v)
	}
	for _, m := range expected {
		for id, d := range m {
			if d == "" {
				delete(m, id)
			}
		}
	}
	if !reflect.DeepEqual(expected, actual) {
		return contracts.Fail("invalid_request")
	}
	return nil
}
