// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"sort"
	"strings"
)

func (s *Store) resolveSpeakers(ctx context.Context, tx *sql.Tx, refs []string, active bool) ([]string, error) {
	if len(refs) > 1000 {
		return nil, contracts.Fail("input_limit")
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, ref := range refs {
		if !identityText(ref, 512) {
			return nil, contracts.Fail("invalid_request")
		}
		var matches []string
		var e error
		if contracts.ValidID(ref) {
			var sp Speaker
			e = s.domainTx(ctx, tx, "Speakers", ref, &sp)
			if e != nil || active && sp.State != "active" {
				return nil, contracts.Fail("conflict")
			}
			matches = []string{ref}
		} else {
			name := strings.TrimPrefix(ref, "name:")
			where := "(name=? OR id IN (SELECT speaker_id FROM speaker_alias WHERE workspace_id=? AND text=? AND state='active'))"
			if active {
				where += " AND state='active'"
			}
			matches, e = s.identityIDs(ctx, tx, "speaker", where, []any{name, s.workspace, name}, "", 2)
			if e != nil {
				return nil, e
			}
			if len(matches) != 1 {
				return nil, contracts.Fail("conflict")
			}
		}
		if !seen[matches[0]] {
			seen[matches[0]] = true
			ids = append(ids, matches[0])
		}
	}
	sort.Strings(ids)
	return ids, nil
}
func (s *Store) ResolveSpeakers(ctx context.Context, refs []string) ([]string, error) {
	tx, e := s.identityRead(ctx)
	if e != nil {
		return nil, sanitize(e)
	}
	defer tx.Rollback()
	ids, e := s.resolveSpeakers(ctx, tx, refs, true)
	return ids, sanitize(e)
}
func (s *Store) rosterTx(ctx context.Context, tx *sql.Tx, id string) (Roster, error) {
	out := Roster{RecordingID: id, Members: []Speaker{}}
	var parent LibraryEntry
	if e := s.domainTx(ctx, tx, "Library", id, &parent); e != nil {
		return out, e
	}
	var h RosterHeader
	e := s.domainTx(ctx, tx, "Rosters", id, &h)
	if e == sql.ErrNoRows {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	out.Declared = true
	out.Revision = h.Revision
	rows, e := tx.QueryContext(ctx, s.query("SELECT s.id,s.name,s.revision,s.state FROM roster_member m JOIN speaker s ON s.workspace_id=m.workspace_id AND s.id=m.speaker_id WHERE m.workspace_id=? AND m.recording_id=? ORDER BY s.id LIMIT 1001"), s.workspace, id)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var sp Speaker
		if e = rows.Scan(&sp.ID, &sp.Name, &sp.Revision, &sp.State); e != nil {
			return out, e
		}
		out.Members = append(out.Members, sp)
	}
	if len(out.Members) > 1000 {
		return out, contracts.Fail("invalid_request")
	}
	return out, rows.Err()
}
func (s *Store) Roster(ctx context.Context, id string) (Roster, error) {
	if !contracts.ValidID(id) {
		return Roster{}, contracts.Fail("invalid_request")
	}
	tx, e := s.identityRead(ctx)
	if e != nil {
		return Roster{}, sanitize(e)
	}
	defer tx.Rollback()
	r, e := s.rosterTx(ctx, tx, id)
	return r, sanitize(e)
}
func rosterIDs(r Roster) []string {
	ids := []string{}
	for _, sp := range r.Members {
		ids = append(ids, sp.ID)
	}
	return ids
}
func rosterDigest(r Roster) string {
	d, _ := intent([]any{r.RecordingID, r.Revision, rosterIDs(r)})
	return d
}
func (s *Store) saveRoster(ctx context.Context, tx *sql.Tx, id string, expected, rev int64, mode string, ids []string) (Roster, error) {
	old, e := s.rosterTx(ctx, tx, id)
	if e != nil {
		return old, e
	}
	if old.Revision != expected {
		return old, contracts.Fail("conflict")
	}
	set := map[string]bool{}
	if mode == "add" || mode == "remove" || mode == "retain" {
		for _, x := range rosterIDs(old) {
			set[x] = true
		}
	}
	switch mode {
	case "retain":
		return old, nil
	case "clear":
	case "add", "replace":
		resolved, e := s.resolveSpeakers(ctx, tx, ids, true)
		if e != nil {
			return old, e
		}
		for _, x := range resolved {
			set[x] = true
		}
	case "remove":
		for _, x := range ids {
			if !contracts.ValidID(x) {
				return old, contracts.Fail("invalid_request")
			}
			delete(set, x)
		}
	default:
		return old, contracts.Fail("invalid_request")
	}
	next := []string{}
	for id := range set {
		next = append(next, id)
	}
	sort.Strings(next)
	if len(next) > 1000 {
		return old, contracts.Fail("input_limit")
	}
	same := old.Declared && len(next) == len(old.Members)
	for i := range next {
		if same && next[i] != old.Members[i].ID {
			same = false
		}
	}
	if same {
		return old, nil
	}
	h := RosterHeader{ID: id, Revision: rev + 1}
	if e = s.putDomain(ctx, tx, "Rosters", h); e != nil {
		return old, e
	}
	if _, e = s.exec(ctx, tx, "DELETE FROM roster_member WHERE workspace_id=? AND recording_id=?", s.workspace, id); e != nil {
		return old, e
	}
	for _, sp := range next {
		if e = s.putDomain(ctx, tx, "RosterMembers", RosterMember{ID: operationID("roster-member", id, sp), RecordingID: id, SpeakerID: sp, Revision: h.Revision}); e != nil {
			return old, e
		}
	}
	return s.rosterTx(ctx, tx, id)
}
func (s *Store) MutateRoster(ctx context.Context, op, id string, expected int64, mode string, refs []string) (Roster, error) {
	if !contracts.ValidID(op) || !contracts.ValidID(id) || expected < 0 || len(refs) > 1000 || (mode != "add" && mode != "remove" && mode != "replace" && mode != "clear") || mode == "clear" && len(refs) > 0 || (mode == "add" || mode == "remove") && len(refs) == 0 {
		return Roster{}, contracts.Fail("invalid_request")
	}
	digest, e := intent([]any{"roster", id, expected, mode, refs})
	if e != nil {
		return Roster{}, e
	}
	var out Roster
	e = s.write(ctx, func(tx *sql.Tx, rev int64) error {
		_, ok, e := s.replay(ctx, tx, op, digest)
		if e != nil {
			return e
		}
		if ok {
			out, e = s.rosterTx(ctx, tx, id)
			return e
		}
		ids := []string{}
		if mode != "clear" {
			ids, e = s.resolveSpeakers(ctx, tx, refs, mode != "remove")
			if e != nil {
				return e
			}
		}
		out, e = s.saveRoster(ctx, tx, id, expected, rev, mode, ids)
		if e != nil {
			return e
		}
		_, e = s.accept(ctx, tx, op, digest, rev, map[string]any{"roster_proofs": map[string]string{id: rosterDigest(out)}})
		return e
	})
	return out, e
}
func (s *Store) validateRosterState(ctx context.Context, tx *sql.Tx, r Records) error {
	proofs := map[string]string{}
	rows, e := tx.QueryContext(ctx, s.query("SELECT result FROM operation_receipt WHERE workspace_id=? ORDER BY revision"), s.workspace)
	if e != nil {
		return e
	}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			break
		}
		var item struct {
			Proofs map[string]string `json:"roster_proofs"`
		}
		if json.Unmarshal([]byte(raw), &item) != nil {
			e = contracts.Fail("invalid_request")
			break
		}
		for id, d := range item.Proofs {
			proofs[id] = d
		}
	}
	err := rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	if err != nil {
		return err
	}
	headers := map[string]RosterHeader{}
	members := map[string][]RosterMember{}
	seen := map[string]bool{}
	for _, h := range r.Rosters {
		if h.Revision < 1 || headers[h.ID].ID != "" {
			return contracts.Fail("invalid_request")
		}
		headers[h.ID] = h
	}
	for _, m := range r.RosterMembers {
		h, ok := headers[m.RecordingID]
		key := m.RecordingID + "/" + m.SpeakerID
		if !ok || m.Revision != h.Revision || seen[key] || m.ID != operationID("roster-member", m.RecordingID, m.SpeakerID) {
			return contracts.Fail("invalid_request")
		}
		seen[key] = true
		members[m.RecordingID] = append(members[m.RecordingID], m)
	}
	for id, h := range headers {
		if len(members[id]) > 1000 {
			return contracts.Fail("invalid_request")
		}
		roster, e := s.rosterTx(ctx, tx, id)
		if e != nil || roster.Revision != h.Revision || len(roster.Members) != len(members[id]) || proofs[id] != rosterDigest(roster) {
			return contracts.Fail("invalid_request")
		}
		delete(proofs, id)
	}
	if len(proofs) > 0 {
		return contracts.Fail("invalid_request")
	}
	return nil
}
