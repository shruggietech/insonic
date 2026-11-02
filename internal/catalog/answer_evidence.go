// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
)

type AnswerEpoch struct {
	Digest      string
	Revision    int64
	Rows, Bytes int
	Complete    bool
}
type AnswerPageRequest struct {
	MediaIDs                         []string
	RecordingID, CueID, Cursor, Zone string
	FirstOrdinal, LastOrdinal        *int64
	Limit, MaxBytes                  int
}
type AnswerPage struct {
	Items             []answers.Item
	Rows, Bytes       int
	Next, LimitReason string
}
type answerCursor struct {
	Recording string
	Ordinal   int64
	Scope     string
}

// Only compiled column expressions enter these queries. All caller values are
// parameters. Text projections make both SQL engines produce the same identity.
func (s *Store) answerObject(fields ...string) string {
	parts := []string{}
	for i := 0; i < len(fields); i += 2 {
		parts = append(parts, "'"+fields[i]+"'", "COALESCE(''||CAST("+fields[i+1]+" AS TEXT),'')")
	}
	fn := "json_object"
	if s.backend == "postgresql" {
		fn = "jsonb_build_object"
	}
	return "CAST(" + fn + "(" + strings.Join(parts, ",") + ") AS TEXT)"
}
func (s *Store) answerArray(query string) string {
	fn := "json_group_array(value)"
	if s.backend == "postgresql" {
		fn = "json_agg(value)"
	}
	return "COALESCE(CAST((SELECT " + fn + " FROM (" + query + ") answer_values) AS TEXT),'[]')"
}
func (s *Store) answerBytes(expression string) string {
	if s.backend == "postgresql" {
		return "octet_length(" + expression + ")"
	}
	return "length(CAST(" + expression + " AS BLOB))"
}

// Bound payloads before they leave the database. The window runs over an
// already row-limited projection, never over all transcript text in the library.
func (s *Store) answerRows(ctx context.Context, tx *sql.Tx, query string, args []any, maxBytes int) (out []map[string]string, rowsRead, bytesRead int, limited bool, err error) {
	q := "WITH candidates AS (" + query + "), sized AS (SELECT ord,raw," + s.answerBytes("raw") + " AS bytes FROM candidates) SELECT CASE WHEN SUM(bytes) OVER (ORDER BY ord ROWS UNBOUNDED PRECEDING)<=? THEN raw ELSE NULL END FROM sized ORDER BY ord"
	args = append(args, maxBytes)
	rows, err := tx.QueryContext(ctx, s.query(q), args...)
	if err != nil {
		return nil, 0, 0, false, err
	}
	defer rows.Close()
	for rows.Next() {
		rowsRead++
		var raw sql.NullString
		if err = rows.Scan(&raw); err != nil {
			return
		}
		if !raw.Valid {
			limited = true
			break
		}
		var value map[string]string
		if err = json.Unmarshal([]byte(raw.String), &value); err != nil {
			return
		}
		bytesRead += len(raw.String)
		out = append(out, value)
	}
	err = rows.Err()
	return
}
func answerScope(ids []string) error {
	if len(ids) > 500 {
		return contracts.Fail("input_limit")
	}
	for _, id := range ids {
		if !contracts.ValidID(id) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

// AnswerEpoch hashes bounded current semantic authority, excluding work,
// artifact and cache-index writes. It also detects restored equal revisions
// with different identities. An incomplete scan cannot authorize cache reuse.
func (s *Store) AnswerEpoch(ctx context.Context, mediaIDs []string, maxRows, maxBytes int) (out AnswerEpoch, err error) {
	tx, err := s.identityRead(ctx)
	if err != nil {
		return out, sanitize(err)
	}
	defer tx.Rollback()
	out, err = s.answerEpochTx(ctx, tx, mediaIDs, maxRows, maxBytes)
	if err != nil {
		return out, err
	}
	return out, sanitize(tx.Commit())
}
func (s *Store) answerEpochTx(ctx context.Context, tx *sql.Tx, mediaIDs []string, maxRows, maxBytes int) (out AnswerEpoch, err error) {
	if answerScope(mediaIDs) != nil || maxRows < 1 || maxRows > 100000 || maxBytes < 1 || maxBytes > 8<<20 {
		return out, contracts.Fail("invalid_request")
	}
	if err = s.row(ctx, tx, "SELECT revision FROM workspace WHERE id=?", s.workspace).Scan(&out.Revision); err != nil {
		return out, sanitize(err)
	}
	out.Rows = 1
	if maxRows == 1 {
		return out, nil
	}
	definitions := []struct {
		table, scope string
		fields       []string
	}{
		{"current_recording", "id", []string{"id", "id", "revision", "revision", "source", "source_digest", "source_revision", "source_revision", "state", "state", "document", "document_digest", "map", "source_map", "provenance", "provenance"}},
		{"library_entry", "id", []string{"id", "id", "revision", "revision", "digest", "digest", "facts", "facts", "dates", "dates"}},
		{"speaker_mapping", "recording_id", []string{"id", "id", "revision", "revision", "recording", "recording_id", "local", "local_speaker_id", "speaker", "speaker_id", "document", "document_digest", "origin", "origin"}},
		{"current_extraction", "id", []string{"id", "id", "revision", "revision", "recording_revision", "recording_revision", "document", "document_digest", "assertions", "assertions", "provenance", "provenance"}},
		{"speaker", "", []string{"id", "id", "revision", "revision", "name", "name", "state", "state"}},
		{"context_term", "", []string{"id", "id", "revision", "revision", "canonical", "canonical", "variants", "variants", "state", "state"}},
	}
	queries, args := []string{}, []any{}
	for i, d := range definitions {
		q := "SELECT '" + strconv.Itoa(i) + ":'||id AS ord," + s.answerObject(d.fields...) + " AS raw FROM " + d.table + " WHERE workspace_id=?"
		args = append(args, s.workspace)
		if len(mediaIDs) > 0 && d.scope != "" {
			marks, values := selectedIDs(mediaIDs)
			q += " AND " + d.scope + " IN (" + marks + ")"
			args = append(args, values...)
		}
		queries = append(queries, q)
	}
	q := strings.Join(queries, " UNION ALL ") + " ORDER BY ord LIMIT ?"
	args = append(args, maxRows-1)
	values, n, b, limited, err := s.answerRows(ctx, tx, q, args, maxBytes)
	out.Rows += n
	out.Bytes = b
	if err != nil {
		return out, sanitize(err)
	}
	out.Complete = !limited && n < maxRows-1
	if out.Complete {
		h := sha256.New()
		for _, v := range values {
			for key, value := range v {
				if json.Valid([]byte(value)) {
					if canonicalValue, e := canonical([]byte(value)); e == nil {
						v[key] = string(canonicalValue)
					}
				}
			}
			raw, _ := json.Marshal(v)
			h.Write(raw)
			h.Write([]byte{'\n'})
		}
		out.Digest = hex.EncodeToString(h.Sum(nil))
	}
	return out, nil
}

func (s *Store) AnswerCandidates(ctx context.Context, request AnswerPageRequest) (out AnswerPage, err error) {
	out.Items = []answers.Item{}
	if answerScope(request.MediaIDs) != nil || request.RecordingID != "" && !contracts.ValidID(request.RecordingID) || !contracts.ValidTextLength(request.CueID, 0, 1024) || request.Limit < 1 || request.Limit > 500 || request.MaxBytes < 1 || request.MaxBytes > 8<<20 {
		return out, contracts.Fail("invalid_request")
	}
	if request.FirstOrdinal != nil && *request.FirstOrdinal < 0 || request.LastOrdinal != nil && *request.LastOrdinal < 0 {
		return out, contracts.Fail("invalid_request")
	}
	scope := request
	scope.Cursor = ""
	scope.Limit = 0
	scope.MaxBytes = 0
	cursor := answerCursor{Scope: answers.Digest(scope), Ordinal: -1}
	if request.Cursor != "" {
		raw, e := base64.RawURLEncoding.DecodeString(request.Cursor)
		if e != nil || len(raw) > 4096 || strict(raw, &cursor) != nil || cursor.Scope != answers.Digest(scope) || !contracts.ValidID(cursor.Recording) || cursor.Ordinal < 0 {
			return out, contracts.Fail("invalid_request")
		}
	}
	tx, err := s.identityRead(ctx)
	if err != nil {
		return out, sanitize(err)
	}
	defer tx.Rollback()
	cid := s.evidenceJSONText("c.value", "id")
	ordinal := s.evidenceOrdinal("c")
	mapping := s.answerObject("local", "m.local_speaker_id", "speaker", "m.speaker_id", "revision", "m.revision", "speaker_revision", "p.revision", "speaker_state", "p.state", "speaker_name", "p.name")
	mappings := s.answerArray("SELECT " + mapping + " AS value FROM speaker_mapping m JOIN speaker p ON p.workspace_id=m.workspace_id AND p.id=m.speaker_id WHERE m.workspace_id=r.workspace_id AND m.recording_id=r.id AND m.document_digest=r.document_digest ORDER BY m.local_speaker_id")
	aFrom := "json_each(e.assertions) a"
	support := "json_each(a.value,'$.cue_ids') support"
	if s.backend == "postgresql" {
		aFrom = "jsonb_array_elements(e.assertions::jsonb) a(value)"
		support = "jsonb_array_elements_text(a.value->'cue_ids') support(value)"
	}
	assertions := s.answerArray("SELECT CAST(a.value AS TEXT) AS value FROM current_extraction e JOIN " + aFrom + " ON true WHERE e.workspace_id=r.workspace_id AND e.id=r.id AND e.recording_revision=r.revision AND e.document_digest=r.document_digest AND EXISTS(SELECT 1 FROM " + support + " WHERE support.value=" + cid + ") ORDER BY " + s.evidenceJSONText("a.value", "id"))
	count := "json_array_length(r.document,'$.cues')"
	if s.backend == "postgresql" {
		count = "jsonb_array_length(r.document::jsonb->'cues')"
	}
	unique := "(SELECT COUNT(*) FROM (SELECT 1 FROM (SELECT 1) seed " + s.evidenceJSONCues("r.document", "duplicate_cue") + " WHERE " + s.evidenceJSONText("duplicate_cue.value", "id") + "=" + cid + " LIMIT 2) cue_matches)"
	projection := s.answerObject("cue_count", count, "cue_matches", unique, "id", "r.id", "revision", "r.revision", "document", "r.document_digest", "source", "r.source_digest", "map", "r.source_map", "provenance", "r.provenance", "ordinal", ordinal, "cue", "c.value", "mappings", mappings, "assertions", assertions, "facts", "l.facts", "dates", "l.dates")
	// Select one indexed recording before expanding its cues. A page never
	// orders or materializes cue payloads across the whole library.
	chosen := "SELECT r.* FROM current_recording r WHERE r.workspace_id=? AND r.state='ready' AND " + count + ">0"
	args := []any{s.workspace}
	if len(request.MediaIDs) > 0 {
		marks, values := selectedIDs(request.MediaIDs)
		chosen += " AND r.id IN (" + marks + ")"
		args = append(args, values...)
	}
	if request.RecordingID != "" {
		chosen += " AND r.id=?"
		args = append(args, request.RecordingID)
	}
	if request.FirstOrdinal != nil {
		chosen += " AND " + count + ">?"
		args = append(args, *request.FirstOrdinal)
	}
	if request.Cursor != "" {
		chosen += " AND (r.id>? OR (r.id=? AND " + count + ">?))"
		args = append(args, cursor.Recording, cursor.Recording, cursor.Ordinal+1)
	}
	chosen += " ORDER BY r.id LIMIT 1"
	q := "SELECT ROW_NUMBER() OVER (ORDER BY r.id," + ordinal + ") AS ord," + projection + " AS raw FROM (" + chosen + ") r JOIN library_entry l ON l.workspace_id=r.workspace_id AND l.id=r.id " + s.evidenceJSONCues("r.document", "c") + " WHERE 1=1"
	if request.CueID != "" {
		q += " AND " + cid + "=?"
		args = append(args, request.CueID)
	}
	if request.FirstOrdinal != nil {
		q += " AND " + ordinal + ">=?"
		args = append(args, *request.FirstOrdinal)
	}
	if request.LastOrdinal != nil {
		q += " AND " + ordinal + "<=?"
		args = append(args, *request.LastOrdinal)
	}
	if request.Cursor != "" {
		q += " AND (r.id>? OR (r.id=? AND " + ordinal + ">?))"
		args = append(args, cursor.Recording, cursor.Recording, cursor.Ordinal)
	}
	q += " ORDER BY r.id," + ordinal + " LIMIT ?"
	args = append(args, request.Limit)
	values, n, b, limited, err := s.answerRows(ctx, tx, q, args, request.MaxBytes)
	out.Rows = n
	out.Bytes = b
	if err != nil {
		return out, sanitize(err)
	}
	for _, value := range values {
		if value["cue_matches"] != "1" {
			return out, contracts.Condition("conflict", "ambiguous_cue_identity")
		}
		item, e := s.answerItem(value, request.Zone)
		if e != nil {
			return out, e
		}
		out.Items = append(out.Items, item)
		cursor.Recording = item.Reference.RecordingID
		cursor.Ordinal, _ = strconv.ParseInt(item.Reference.Ordinal, 10, 64)
	}
	if limited {
		out.LimitReason = "source_bytes"
	} else if request.LastOrdinal != nil && len(values) == request.Limit {
		count, _ := strconv.ParseInt(values[len(values)-1]["cue_count"], 10, 64)
		if cursor.Ordinal < min(*request.LastOrdinal, count-1) {
			out.LimitReason = "neighbor_rows"
		}
	}
	if len(values) > 0 {
		raw, _ := json.Marshal(cursor)
		out.Next = base64.RawURLEncoding.EncodeToString(raw)
	}
	return out, sanitize(tx.Commit())
}

func answerRawArray(raw string) ([]json.RawMessage, error) {
	var items []json.RawMessage
	if json.Unmarshal([]byte(raw), &items) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	for i, v := range items {
		if len(v) > 0 && v[0] == '"' {
			var text string
			if json.Unmarshal(v, &text) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			items[i] = json.RawMessage(text)
		}
	}
	return items, nil
}
func (s *Store) answerItem(v map[string]string, zone string) (out answers.Item, err error) {
	cues, err := evidence.ParseCues(json.RawMessage(`{"cues":[` + v["cue"] + `]}`))
	if err != nil || len(cues) != 1 {
		return out, contracts.Fail("invalid_request")
	}
	cue := cues[0]
	canonicalMap, err := canonical([]byte(v["map"]))
	if err != nil {
		return out, err
	}
	out = answers.Item{ID: "cue:" + v["id"] + ":" + v["document"] + ":" + cue.ID, Text: cue.Text, Family: "source:" + v["source"], Lineage: "source-digest;shared-upstream-origin-unknown", LocalSpeakerIDs: cue.Voices, Paths: []answers.Path{}, Reasons: []string{}, Reference: answers.Reference{WorkspaceID: s.workspace, RecordingID: v["id"], RecordingRevision: v["revision"], DocumentDigest: v["document"], SourceDigest: v["source"], SourceMapDigest: hash(canonicalMap), CueID: cue.ID, Ordinal: v["ordinal"]}}
	identity := map[string]json.RawMessage{}
	for _, key := range []string{"map", "provenance", "mappings", "facts", "dates", "assertions"} {
		if key == "mappings" || key == "assertions" {
			values, e := answerRawArray(v[key])
			if e != nil {
				return out, e
			}
			normalized, _ := json.Marshal(values)
			v[key] = string(normalized)
		}
		raw, e := canonical([]byte(v[key]))
		if e != nil {
			return out, e
		}
		identity[key] = raw
	}
	out.Reference.IdentityDigest = answers.Digest(identity)
	if cue.Timed {
		if cue.StartMS < 0 || cue.EndMS <= cue.StartMS || cue.EndMS > math.MaxInt64/1000 {
			return out, contracts.Fail("invalid_request")
		}
		out.Reference.Source = &answers.Interval{StartUS: strconv.FormatInt(cue.StartMS*1000, 10), EndUS: strconv.FormatInt(cue.EndMS*1000, 10)}
	}
	resolvedMap, e := ResolveCueSourceMap([]byte(v["map"]), []byte(v["provenance"]), cue.ID)
	if e == nil {
		var routing struct {
			Stream  *int `json:"stream_index"`
			Channel *int `json:"channel"`
		}
		if json.Unmarshal(resolvedMap, &routing) == nil {
			out.Reference.StreamIndex = routing.Stream
			out.Reference.Channel = routing.Channel
		}
	} else {
		out.Reasons = append(out.Reasons, "source-route-unavailable")
	}
	var facts struct {
		Canonical *struct {
			SourceDigest string `json:"source_digest"`
			Tracks       []struct {
				Index       *int `json:"index"`
				SourceIndex *int `json:"source_index"`
			} `json:"tracks"`
		} `json:"canonical"`
	}
	if json.Unmarshal([]byte(v["facts"]), &facts) == nil && facts.Canonical != nil && answers.ValidDigest(facts.Canonical.SourceDigest) {
		out.Family = "source:" + facts.Canonical.SourceDigest
		out.Lineage = "canonical-origin"
		if out.Reference.StreamIndex != nil {
			for _, track := range facts.Canonical.Tracks {
				if track.Index != nil && track.SourceIndex != nil && *track.Index == *out.Reference.StreamIndex && *track.SourceIndex >= 0 {
					index := *track.SourceIndex
					out.OriginStreamIndex = &index
					break
				}
			}
		}
		if out.OriginStreamIndex == nil {
			out.Reasons = append(out.Reasons, "origin-stream-unresolved")
		}
	}
	mappings, e := answerRawArray(v["mappings"])
	if e != nil {
		return out, e
	}
	for _, raw := range mappings {
		var m map[string]string
		if json.Unmarshal(raw, &m) != nil {
			return out, contracts.Fail("invalid_request")
		}
		if slices.Contains(cue.Voices, m["local"]) {
			out.SpeakerIDs = append(out.SpeakerIDs, m["speaker"])
		}
	}
	slices.Sort(out.SpeakerIDs)
	out.SpeakerIDs = slices.Compact(out.SpeakerIDs)
	statements, e := answerRawArray(v["assertions"])
	if e != nil {
		return out, e
	}
	for _, raw := range statements {
		var a evidence.Assertion
		if json.Unmarshal(raw, &a) != nil {
			return out, contracts.Fail("invalid_request")
		}
		out.Statements = append(out.Statements, a)
	}
	out.RecordingDate, out.RecordingDateThrough, out.DateKnown, err = answers.DateSpan([]byte(v["dates"]), zone)
	return out, err
}

func (s *Store) ResolveAnswerEvidence(ctx context.Context, reference answers.Reference, maxBytes int) (answers.Item, error) {
	if answers.ValidateReference(reference) != nil || reference.WorkspaceID != s.workspace {
		return answers.Item{}, contracts.Fail("invalid_request")
	}
	page, err := s.AnswerCandidates(ctx, AnswerPageRequest{RecordingID: reference.RecordingID, CueID: reference.CueID, Limit: 2, MaxBytes: maxBytes})
	if err != nil {
		return answers.Item{}, err
	}
	if page.LimitReason != "" {
		return answers.Item{}, contracts.Fail("input_limit")
	}
	if len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0].Reference, reference) {
		return answers.Item{}, contracts.Condition("conflict", "source_authority_changed")
	}
	return page.Items[0], nil
}
