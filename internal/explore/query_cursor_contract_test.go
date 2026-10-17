// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"encoding/base64"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func requireCursorError(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || contracts.SafeError(err).Code != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}
func TestCursorBindsEffectivePageLimit(t *testing.T) {
	c := pagingCorpus(131, false)
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view"}}
	first, e := Query(c, c.Refs, q)
	if e != nil || first.NextCursor == "" {
		t.Fatal(e)
	}
	q.Definition.Pagination = &contracts.QueryPagination{Limit: 100, Cursor: first.NextCursor}
	if _, e := Query(c, c.Refs, q); e != nil {
		t.Fatal("explicit default differs from effective default", e)
	}
	q.Definition.Pagination.Limit = 500
	_, e = Query(c, c.Refs, q)
	requireCursorError(t, e, "conflict")
}
func TestStaleCursorConflictsBeforeCurrentOffsetBounds(t *testing.T) {
	for _, shrink := range []string{"rows", "edges"} {
		t.Run(shrink, func(t *testing.T) {
			c := pagingCorpus(12, true)
			q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view", Pagination: &contracts.QueryPagination{Limit: 2}}}
			first, e := Query(c, c.Refs, q)
			if e != nil {
				t.Fatal(e)
			}
			q.Definition.Pagination.Cursor = first.NextCursor
			if shrink == "edges" {
				c.Refs.Edges = c.Refs.Edges[:1]
			} else {
				for id := range c.Rows {
					if id != "node:0000" {
						delete(c.Rows, id)
					}
				}
				c.Refs.Nodes = c.Refs.Nodes[:1]
				c.Refs.Edges = nil
			}
			_, e = Query(c, c.Refs, q)
			requireCursorError(t, e, "conflict")
		})
	}
}
func TestCurrentCursorStillRejectsInvalidOffsets(t *testing.T) {
	c := pagingCorpus(12, false)
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view", Pagination: &contracts.QueryPagination{Limit: 2}}}
	first, e := Query(c, c.Refs, q)
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(first.NextCursor)
	var cur cursor
	json.Unmarshal(raw, &cur)
	for _, bad := range []cursor{{Digest: cur.Digest, Offset: -1}, {Digest: cur.Digest, Offset: 10000}, {Digest: cur.Digest, EdgeOffset: -1}, {Digest: cur.Digest, EdgeOffset: 10000}} {
		raw, _ := json.Marshal(bad)
		q.Definition.Pagination.Cursor = base64.RawURLEncoding.EncodeToString(raw)
		_, e := Query(c, c.Refs, q)
		requireCursorError(t, e, "invalid_request")
	}
}
