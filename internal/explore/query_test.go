// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func corpusFixture(t *testing.T) Corpus {
	t.Helper()
	s := catalog.Snapshot{Revision: 17}
	for i := 0; i < 63; i++ {
		id := StableID("media", fmt.Sprint(i))
		s.Records.Library = append(s.Records.Library, catalog.LibraryEntry{ID: id, Revision: 1, Title: fmt.Sprintf("Recording %03d", i), Dates: json.RawMessage(`{"selected":{"literal":"2026-10-01","precision":"day","interpretation_state":"date-only"}}`)})
	}
	doc := json.RawMessage(`{"cues":[{"id":"a","payload":{"plain_text":"We may not proceed unless ready."},"timing":{"start_milliseconds":10,"end_milliseconds":20}}]}`)
	id := s.Records.Library[0].ID
	s.Records.Recordings = []catalog.Recording{{ID: id, Revision: 5, DocumentDigest: "current", Document: doc}}
	c, e := Build(s)
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestWholeCalendarPaginationAndCurrentCursor(t *testing.T) {
	c := corpusFixture(t)
	p := &contracts.QueryPagination{Limit: 25}
	seen := map[string]bool{}
	for {
		r, e := Calendar(c, contracts.CalendarFilter{From: "2026-10-01", Through: "2026-10-01", Timezone: "America/New_York"}, p)
		if e != nil {
			t.Fatal(e)
		}
		if r.Total != 63 {
			t.Fatal(r.Total)
		}
		for _, row := range r.Rows {
			if seen[row.ID] {
				t.Fatal("duplicate", row.ID)
			}
			seen[row.ID] = true
			if row.RecordingDate != "2026-10-01" {
				t.Fatal("day shifted")
			}
		}
		if r.NextCursor == "" {
			break
		}
		p.Cursor = r.NextCursor
	}
	if len(seen) != 63 {
		t.Fatal("library page imposed", len(seen))
	}
	r, e := Calendar(c, contracts.CalendarFilter{IncludeUndated: true}, &contracts.QueryPagination{Limit: 1})
	if e != nil {
		t.Fatal(e)
	}
	for id, row := range c.Rows {
		if row.Kind == "media" {
			row.Label = "Changed"
			c.Rows[id] = row
			break
		}
	}
	if _, e = Calendar(c, contracts.CalendarFilter{IncludeUndated: true}, &contracts.QueryPagination{Limit: 1, Cursor: r.NextCursor}); e == nil {
		t.Fatal("stale cursor accepted")
	}
}
func TestCurrentQueryExactTimesAndMissingGraphRefs(t *testing.T) {
	c := corpusFixture(t)
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "text-search", Filters: contracts.QueryFilter{Text: "unless"}, Pagination: &contracts.QueryPagination{Limit: 10}}}
	r, e := Query(c, c.Refs, q)
	if e != nil || len(r.Rows) != 1 || r.Rows[0].StartUS != "10000" || r.Rows[0].EndUS != "20000" {
		t.Fatal(r, e)
	}
	old := c.Refs
	c.Records.Recordings[0].DocumentDigest = "replacement"
	c.Records.Recordings[0].Document = json.RawMessage(`{"cues":[{"id":"b","payload":{"plain_text":"Replacement"}}]}`)
	next, e := Build(catalog.Snapshot{Records: c.Records})
	if e != nil {
		t.Fatal(e)
	}
	r, e = Query(next, old, q)
	if e != nil || len(r.Rows) != 0 {
		t.Fatal("obsolete source returned", r, e)
	}
	q.Definition.Operation = "time-range"
	q.Definition.Filters = contracts.QueryFilter{SourceInterval: &contracts.SourceInterval{StartUS: 20000, EndUS: 30000}}
	r, e = Query(c, c.Refs, q)
	if e != nil || len(r.Rows) != 0 {
		t.Fatal("half-open boundary", r, e)
	}
}
