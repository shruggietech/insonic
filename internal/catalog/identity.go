// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"math"
	"net"
	"net/url"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

func identityText(text string, max int) bool {
	return strings.TrimSpace(text) != "" && len(text) <= max && utf8.ValidString(text) && strings.IndexFunc(text, unicode.IsControl) < 0
}
func identityState(state string) bool        { return state == "active" || state == "inactive" }
func optionalText(text string, max int) bool { return text == "" || identityText(text, max) }
func validAlias(a SpeakerAlias) bool {
	return contracts.ValidID(a.ID) && contracts.ValidID(a.SpeakerID) && identityText(a.Text, 512) && optionalText(a.Language, 80) && optionalText(a.Scope, 256) && identityState(a.State) && referenceOnlyOptions(a.Provenance)
}
func validTerm(t Term) bool {
	var variants []string
	if !contracts.ValidID(t.ID) || !identityText(t.Canonical, 512) || !optionalText(t.Language, 80) || !optionalText(t.Context, 256) || !identityState(t.State) || !referenceOnlyOptions(t.Provenance) || strict(t.Variants, &variants) != nil || len(variants) > 128 {
		return false
	}
	for _, v := range variants {
		if !identityText(v, 512) {
			return false
		}
	}
	return (t.SpeakerID == nil || contracts.ValidID(*t.SpeakerID)) && (t.AliasID == nil || contracts.ValidID(*t.AliasID))
}
func validPipeline(p Pipeline) bool {
	if !contracts.ValidID(p.ID) || !identityText(p.Name, 512) || (p.Preset != "local" && p.Preset != "connected" && p.Preset != "custom") || len(p.Configuration) > 1<<20 || !nonsecret(p.Configuration) || schemas.ValidatePipelineConfiguration(p.Configuration) != nil {
		return false
	}
	var config struct {
		Recognition struct {
			Mode         string `json:"mode"`
			Endpoint     string `json:"endpoint"`
			CredentialID string `json:"credential_id"`
		} `json:"recognition"`
		Diarization struct {
			Mode         string `json:"mode"`
			Endpoint     string `json:"endpoint"`
			CredentialID string `json:"credential_id"`
		} `json:"diarization"`
		SpeakerTraining struct {
			Adapter struct {
				Mode         string `json:"mode"`
				Endpoint     string `json:"endpoint"`
				CredentialID string `json:"credential_id"`
			} `json:"adapter"`
		} `json:"speaker_training"`
	}
	if json.Unmarshal(p.Configuration, &config) != nil {
		return false
	}
	for _, stage := range []struct{ endpoint, credential string }{{config.Recognition.Endpoint, config.Recognition.CredentialID}, {config.Diarization.Endpoint, config.Diarization.CredentialID}, {config.SpeakerTraining.Adapter.Endpoint, config.SpeakerTraining.Adapter.CredentialID}} {
		if stage.endpoint == "" {
			continue
		}
		u, e := url.Parse(stage.endpoint)
		if e != nil || !u.IsAbs() || u.Hostname() == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsFunc(stage.endpoint, unicode.IsSpace) || (u.Scheme != "http" && u.Scheme != "https") {
			return false
		}
		if stage.credential != "" && u.Scheme == "http" {
			ip := net.ParseIP(u.Hostname())
			if !strings.EqualFold(u.Hostname(), "localhost") && (ip == nil || !ip.IsLoopback()) {
				return false
			}
		}
	}
	processing := config.Recognition.Mode != "" || config.Diarization.Mode != ""
	training := config.SpeakerTraining.Adapter.Mode
	return p.Preset == "custom" || p.Preset == "local" && (!processing || config.Recognition.Mode == "local" && config.Diarization.Mode == "local") && (training == "" || training == "local" || training == "embedding") || p.Preset == "connected" && (!processing || config.Recognition.Mode == "hosted" && config.Diarization.Mode == "hosted") && (training == "" || training == "hosted")
}
func defaultRaw(raw json.RawMessage, value string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(value)
	}
	return raw
}
func (s *Store) identityRead(ctx context.Context) (*sql.Tx, error) {
	options := &sql.TxOptions{ReadOnly: true}
	if s.backend == "postgresql" {
		options.Isolation = sql.LevelRepeatableRead
	}
	return s.db.BeginTx(ctx, options)
}
func (s *Store) Pipeline(ctx context.Context, id string) (out Pipeline, e error) {
	e = s.readOne(ctx, "Pipelines", id, &out)
	return
}
func (s *Store) Term(ctx context.Context, id string) (out Term, e error) {
	e = s.readOne(ctx, "Terms", id, &out)
	return
}
func (s *Store) speakerTx(ctx context.Context, tx *sql.Tx, id string) (out SpeakerIdentity, e error) {
	if e = s.domainTx(ctx, tx, "Speakers", id, &out.Speaker); e != nil {
		return
	}
	records, err := s.identityRecords(ctx, tx, "Aliases", "speaker_id=?", []any{id}, 1001)
	if err != nil {
		return out, err
	}
	if len(records.Aliases) > 1000 {
		return out, contracts.Fail("invalid_request")
	}
	out.Aliases = records.Aliases
	if out.Aliases == nil {
		out.Aliases = []SpeakerAlias{}
	}
	return

}
func (s *Store) Speaker(ctx context.Context, id string) (out SpeakerIdentity, e error) {
	if !contracts.ValidID(id) {
		return out, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	out, e = s.speakerTx(ctx, tx, id)
	if e == nil {
		e = tx.Commit()
	}
	return out, sanitize(e)
}
func pageLimit(cursor string, limit int) (int, error) {
	if cursor != "" && !contracts.ValidID(cursor) {
		return 0, contracts.Fail("invalid_request")
	}
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 100 {
		return 0, contracts.Fail("invalid_request")
	}
	return limit, nil
}
func (s *Store) identityIDs(ctx context.Context, tx *sql.Tx, table, where string, args []any, cursor string, limit int) ([]string, error) {
	q := "SELECT id FROM " + table + " WHERE workspace_id=?"
	parameters := []any{s.workspace}
	if where != "" {
		q += " AND (" + where + ")"
		parameters = append(parameters, args...)
	}
	if cursor != "" {
		q += " AND id>?"
		parameters = append(parameters, cursor)
	}
	q += " ORDER BY id LIMIT ?"
	parameters = append(parameters, limit)
	rows, e := tx.QueryContext(ctx, s.query(q), parameters...)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (s *Store) Pipelines(ctx context.Context, cursor string, limit int) (out []Pipeline, next string, e error) {
	limit, e = pageLimit(cursor, limit)
	if e != nil {
		return
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return nil, "", sanitize(e)
	}
	defer tx.Rollback()
	ids, e := s.identityIDs(ctx, tx, "current_pipeline", "", nil, cursor, limit+1)
	if e != nil {
		return nil, "", sanitize(e)
	}
	if len(ids) > limit {
		ids = ids[:limit]
		next = ids[len(ids)-1]
	}
	out = []Pipeline{}
	for _, id := range ids {
		var p Pipeline
		if e = s.domainTx(ctx, tx, "Pipelines", id, &p); e != nil {
			return nil, "", sanitize(e)
		}
		if !boundedPage(append(out, p)) {
			if len(out) == 0 {
				return nil, "", contracts.Fail("output_limit")
			}
			next = out[len(out)-1].ID
			break
		}
		out = append(out, p)
	}
	e = sanitize(tx.Commit())
	return
}
func (s *Store) Terms(ctx context.Context, cursor string, limit int) (out []Term, next string, e error) {
	limit, e = pageLimit(cursor, limit)
	if e != nil {
		return
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return nil, "", sanitize(e)
	}
	defer tx.Rollback()
	ids, e := s.identityIDs(ctx, tx, "context_term", "", nil, cursor, limit+1)
	if e != nil {
		return nil, "", sanitize(e)
	}
	if len(ids) > limit {
		ids = ids[:limit]
		next = ids[len(ids)-1]
	}
	out = []Term{}
	for _, id := range ids {
		var t Term
		if e = s.domainTx(ctx, tx, "Terms", id, &t); e != nil {
			return nil, "", sanitize(e)
		}
		if !boundedPage(append(out, t)) {
			if len(out) == 0 {
				return nil, "", contracts.Fail("output_limit")
			}
			next = out[len(out)-1].ID
			break
		}
		out = append(out, t)
	}
	e = sanitize(tx.Commit())
	return
}
func (s *Store) Speakers(ctx context.Context, cursor string, limit int) (out []SpeakerIdentity, next string, e error) {
	limit, e = pageLimit(cursor, limit)
	if e != nil {
		return
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return nil, "", sanitize(e)
	}
	defer tx.Rollback()
	ids, e := s.identityIDs(ctx, tx, "speaker", "", nil, cursor, limit+1)
	if e != nil {
		return nil, "", sanitize(e)
	}
	if len(ids) > limit {
		ids = ids[:limit]
		next = ids[len(ids)-1]
	}
	out = []SpeakerIdentity{}
	for _, id := range ids {
		p, err := s.speakerTx(ctx, tx, id)
		if err != nil {
			return nil, "", sanitize(err)
		}
		if !boundedPage(append(out, p)) {
			if len(out) == 0 {
				return nil, "", contracts.Fail("output_limit")
			}
			next = out[len(out)-1].Speaker.ID
			break
		}
		out = append(out, p)
	}
	e = sanitize(tx.Commit())
	return
}
func validIdentityMutation(op string, expected int64) bool {
	return contracts.ValidID(op) && expected >= 0 && expected < math.MaxInt64
}
func (s *Store) identityCAS(ctx context.Context, tx *sql.Tx, table, id string, expected int64) error {
	var revision int64
	e := s.row(ctx, tx, "SELECT revision FROM "+table+" WHERE workspace_id=? AND id=?", s.workspace, id).Scan(&revision)
	if e == sql.ErrNoRows && expected == 0 {
		return nil
	}
	if e != nil && e != sql.ErrNoRows {
		return e
	}
	if e == sql.ErrNoRows || revision != expected {
		return contracts.Fail("conflict")
	}
	return nil
}
func (s *Store) PutPipeline(ctx context.Context, op string, expected int64, p Pipeline) (Pipeline, error) {
	p.Revision = 0
	if !validIdentityMutation(op, expected) || !validPipeline(p) {
		return p, contracts.Fail("invalid_request")
	}
	p.Configuration, _ = canonical(p.Configuration)
	digest, e := intent([]any{"pipeline", expected, p})
	if e != nil {
		return p, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			p.Revision = r.Revision
			return nil
		}
		if e = s.identityCAS(ctx, tx, "current_pipeline", p.ID, expected); e != nil {
			return e
		}
		p.Revision = rev + 1
		if e = s.putDomain(ctx, tx, "Pipelines", p); e != nil {
			return e
		}
		proof, _ := intent(p)
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"pipeline_proofs": map[string]string{p.ID: proof}})
		return e
	})
	return p, e
}
func normalizeIdentity(identity SpeakerIdentity) SpeakerIdentity {
	identity.Aliases = append([]SpeakerAlias{}, identity.Aliases...)
	identity.Speaker.Revision = 0
	if identity.Speaker.State == "" {
		identity.Speaker.State = "active"
	}
	if identity.Aliases == nil {
		identity.Aliases = []SpeakerAlias{}
	}
	for i := range identity.Aliases {
		a := &identity.Aliases[i]
		if a.State == "" {
			a.State = "active"
		}
		a.Provenance = defaultRaw(a.Provenance, "{}")
		a.Provenance, _ = canonical(a.Provenance)
	}
	sort.Slice(identity.Aliases, func(i, j int) bool { return identity.Aliases[i].ID < identity.Aliases[j].ID })
	return identity
}
func (s *Store) PutSpeaker(ctx context.Context, op string, expected int64, identity SpeakerIdentity) (SpeakerIdentity, error) {
	identity = normalizeIdentity(identity)
	if !validIdentityMutation(op, expected) || !contracts.ValidID(identity.Speaker.ID) || !identityText(identity.Speaker.Name, 512) || !identityState(identity.Speaker.State) || len(identity.Aliases) > 1000 {
		return identity, contracts.Fail("invalid_request")
	}
	raw, err := json.Marshal(identity)
	if err != nil || len(raw) > 256<<10 {
		return identity, contracts.Fail("input_limit")
	}
	seen := map[string]bool{}
	for _, a := range identity.Aliases {
		if !validAlias(a) || a.SpeakerID != identity.Speaker.ID || seen[a.ID] {
			return identity, contracts.Fail("invalid_request")
		}
		seen[a.ID] = true
	}
	digest, e := intent([]any{"speaker", expected, identity})
	if e != nil {
		return identity, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			identity.Speaker.Revision = r.Revision
			return nil
		}
		if e = s.identityCAS(ctx, tx, "speaker", identity.Speaker.ID, expected); e != nil {
			return e
		}
		// Retain linked alias identity; deactivation expresses its withdrawal.
		ids, e := s.identityIDs(ctx, tx, "speaker_alias", "speaker_id=?", []any{identity.Speaker.ID}, "", 1001)
		if e != nil {
			return e
		}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			var linked bool
			if e = s.row(ctx, tx, "SELECT EXISTS(SELECT 1 FROM context_term WHERE workspace_id=? AND alias_id=?)", s.workspace, id).Scan(&linked); e != nil {
				return e
			}
			if linked {
				return contracts.Fail("conflict")
			}
			if _, e = s.exec(ctx, tx, "DELETE FROM speaker_alias WHERE workspace_id=? AND id=?", s.workspace, id); e != nil {
				return e
			}
		}
		identity.Speaker.Revision = rev + 1
		if e = s.putDomain(ctx, tx, "Speakers", identity.Speaker); e != nil {
			return e
		}
		for _, a := range identity.Aliases {
			var owner string
			err := s.row(ctx, tx, "SELECT speaker_id FROM speaker_alias WHERE workspace_id=? AND id=?", s.workspace, a.ID).Scan(&owner)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == nil && owner != a.SpeakerID {
				return contracts.Fail("conflict")
			}
			if e = s.putDomain(ctx, tx, "Aliases", a); e != nil {
				return e
			}
		}
		proof, _ := intent(identity)
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"speaker_proofs": map[string]string{identity.Speaker.ID: proof}})
		return e
	})
	return identity, e
}
func (s *Store) PutTerm(ctx context.Context, op string, expected int64, t Term) (Term, error) {
	t.Revision = 0
	if t.State == "" {
		t.State = "active"
	}
	t.Variants = defaultRaw(t.Variants, "[]")
	t.Provenance = defaultRaw(t.Provenance, "{}")
	if !validIdentityMutation(op, expected) || !validTerm(t) {
		return t, contracts.Fail("invalid_request")
	}
	t.Variants, _ = canonical(t.Variants)
	t.Provenance, _ = canonical(t.Provenance)
	digest, e := intent([]any{"term", expected, t})
	if e != nil {
		return t, e
	}
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		r, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			t.Revision = r.Revision
			return nil
		}
		if e = s.identityCAS(ctx, tx, "context_term", t.ID, expected); e != nil {
			return e
		}
		if t.AliasID != nil {
			var a SpeakerAlias
			if e = s.domainTx(ctx, tx, "Aliases", *t.AliasID, &a); e != nil {
				return e
			}
			if t.SpeakerID != nil && *t.SpeakerID != a.SpeakerID {
				return contracts.Fail("invalid_request")
			}
		}
		t.Revision = rev + 1
		if e = s.putDomain(ctx, tx, "Terms", t); e != nil {
			return e
		}
		proof, _ := intent(t)
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"term_proofs": map[string]string{t.ID: proof}})
		return e
	})
	return t, e
}

// Names, aliases and terms share one repeatable read and one catalog revision.
func (s *Store) ContextInputs(ctx context.Context, filter ContextFilter) (out ContextSnapshot, e error) {
	if !optionalText(filter.Language, 80) || !optionalText(filter.Context, 256) || len(filter.SpeakerIDs) > 1000 {
		return out, contracts.Fail("invalid_request")
	}
	unique := map[string]bool{}
	for _, id := range filter.SpeakerIDs {
		if !contracts.ValidID(id) {
			return out, contracts.Fail("invalid_request")
		}
		unique[id] = true
	}
	filter.SpeakerIDs = []string{}
	for id := range unique {
		filter.SpeakerIDs = append(filter.SpeakerIDs, id)
	}
	sort.Strings(filter.SpeakerIDs)
	out.Filter = filter
	out.Speakers = []SpeakerIdentity{}
	out.Terms = []Term{}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return out, sanitize(e)
	}
	defer tx.Rollback()
	if e = s.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", s.workspace).Scan(&out.Revision); e != nil {
		return out, sanitize(e)
	}
	if len(filter.SpeakerIDs) > 0 {
		marks, args := selectedIDs(filter.SpeakerIDs)
		speakers, err := s.identityRecords(ctx, tx, "Speakers", "id IN ("+marks+")", args, 1001)
		if err != nil {
			return out, sanitize(err)
		}
		if len(speakers.Speakers) != len(filter.SpeakerIDs) {
			return out, contracts.Fail("not_found")
		}
		aliases, err := s.identityRecords(ctx, tx, "Aliases", "speaker_id IN ("+marks+")", args, 100001)
		if err != nil {
			return out, sanitize(err)
		}
		bySpeaker := map[string][]SpeakerAlias{}
		for _, a := range aliases.Aliases {
			bySpeaker[a.SpeakerID] = append(bySpeaker[a.SpeakerID], a)
		}
		for _, sp := range speakers.Speakers {
			a := bySpeaker[sp.ID]
			if a == nil {
				a = []SpeakerAlias{}
			}
			out.Speakers = append(out.Speakers, SpeakerIdentity{sp, a})
		}
	}

	// Global terms and terms linked to selected identities are selected in SQL.
	where := "speaker_id IS NULL AND alias_id IS NULL"
	args := []any{}
	if len(filter.SpeakerIDs) > 0 {
		marks := strings.TrimSuffix(strings.Repeat("?,", len(filter.SpeakerIDs)), ",")
		where = "(" + where + ") OR speaker_id IN (" + marks + ") OR alias_id IN (SELECT id FROM speaker_alias WHERE workspace_id=? AND speaker_id IN (" + marks + "))"
		for _, id := range filter.SpeakerIDs {
			args = append(args, id)
		}
		args = append(args, s.workspace)
		for _, id := range filter.SpeakerIDs {
			args = append(args, id)
		}
	}
	records, e := s.identityRecords(ctx, tx, "Terms", where, args, 10001)
	if e != nil {
		return out, sanitize(e)
	}
	if len(records.Terms) > 10000 {
		return out, contracts.Fail("input_limit")
	}
	out.Terms = records.Terms
	if out.Terms == nil {
		out.Terms = []Term{}
	}
	if !boundedPage(out) {
		return out, contracts.Fail("input_limit")
	}
	e = sanitize(tx.Commit())
	return

}
