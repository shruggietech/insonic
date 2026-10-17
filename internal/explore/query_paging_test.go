// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"strings"
	"testing"
	"time"
)

func pagingCorpus(n int, dense bool) Corpus {
	c := Corpus{Rows: map[string]Row{}}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("node:%04d", i)
		c.Rows[id] = Row{ID: id, Kind: "cue", Text: "paging match"}
		c.Refs.Nodes = append(c.Refs.Nodes, graph.Node{ID: id, Kind: "cue"})
	}
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			if !dense && j != n-1 {
				continue
			}
			c.Refs.Edges = append(c.Refs.Edges, graph.Edge{ID: fmt.Sprintf("edge:%04d:%04d", i, j), From: fmt.Sprintf("node:%04d", i), To: fmt.Sprintf("node:%04d", j), Kind: "contains-cue"})
		}
	}
	return c
}
func TestGraphPagesRetrieveEveryRelationship(t *testing.T) {
	for _, dense := range []bool{false, true} {
		n := 131
		if dense {
			n = 35
		}
		c := pagingCorpus(n, dense)
		for _, limit := range []int{1, 2, 100, 500} {
			t.Run(fmt.Sprint(dense, "/", limit), func(t *testing.T) {
				q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view", Pagination: &contracts.QueryPagination{Limit: limit}}}
				rows, edges := map[string]bool{}, map[string]bool{}
				for page := 0; ; page++ {
					if page > 20000 {
						t.Fatal("cursor made no progress")
					}
					r, e := Query(c, c.Refs, q)
					if e != nil {
						t.Fatal(e)
					}
					if len(r.Rows) > limit || len(r.Edges) > limit {
						t.Fatal("stream limit exceeded")
					}
					raw, _ := json.Marshal(r)
					if len(raw) > 512<<10 {
						t.Fatal("byte limit exceeded")
					}
					for _, v := range r.Rows {
						if rows[v.ID] {
							t.Fatal("duplicate row", v.ID)
						}
						rows[v.ID] = true
					}
					for _, v := range r.Edges {
						if edges[v.ID] {
							t.Fatal("duplicate edge", v.ID)
						}
						edges[v.ID] = true
					}
					if r.NextCursor == "" {
						break
					}
					if len(r.Rows)+len(r.Edges) == 0 {
						t.Fatal("empty nonterminal page")
					}
					q.Definition.Pagination.Cursor = r.NextCursor
				}
				if len(rows) != len(c.Rows) || len(edges) != len(c.Refs.Edges) {
					t.Fatalf("retrieved %d/%d rows, %d/%d edges", len(rows), len(c.Rows), len(edges), len(c.Refs.Edges))
				}
			})
		}
	}
}
func TestQueryByteBoundedContinuation(t *testing.T) {
	c := pagingCorpus(8, false)
	for id, r := range c.Rows {
		r.Text = strings.Repeat("x", 110000)
		c.Rows[id] = r
	}
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view", Pagination: &contracts.QueryPagination{Limit: 500}}}
	seen := 0
	for i := 0; ; i++ {
		if i > 20 {
			t.Fatal("no progress")
		}
		r, e := Query(c, c.Refs, q)
		if e != nil {
			t.Fatal(e)
		}
		seen += len(r.Rows)
		if r.NextCursor == "" {
			break
		}
		q.Definition.Pagination.Cursor = r.NextCursor
	}
	if seen != 8 {
		t.Fatal(seen)
	}
}
func TestSearchMoreThanFiveHundred(t *testing.T) {
	c := pagingCorpus(607, false)
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "text-search", Filters: contracts.QueryFilter{Text: "paging"}, Pagination: &contracts.QueryPagination{Limit: 100}}}
	seen := map[string]bool{}
	for {
		r, e := Query(c, c.Refs, q)
		if e != nil {
			t.Fatal(e)
		}
		for _, v := range r.Rows {
			if seen[v.ID] {
				t.Fatal("duplicate")
			}
			seen[v.ID] = true
		}
		if r.NextCursor == "" {
			break
		}
		q.Definition.Pagination.Cursor = r.NextCursor
	}
	if len(seen) != 607 {
		t.Fatal(len(seen))
	}
}
func TestCalendarAdvertisedZones(t *testing.T) {
	c := corpusFixture(t)
	for _, zone := range []string{"local", "Local", "UTC", "Z", "America/New_York", "+02:00", "-05:30"} {
		q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "media-list", Filters: contracts.QueryFilter{RecordingDates: &contracts.CalendarFilter{Timezone: zone}}}}
		if e := q.Validate(); e != nil {
			t.Errorf("validate %s: %v", zone, e)
		}
		if _, e := Query(c, c.Refs, q); e != nil {
			t.Errorf("query %s: %v", zone, e)
		}
		if _, e := Calendar(c, contracts.CalendarFilter{Timezone: zone}, nil); e != nil {
			t.Errorf("calendar %s: %v", zone, e)
		}
	}
}

func TestCalendarPinnedZoneDayMembership(t *testing.T) {
	s := catalog.Snapshot{Revision: 1}
	for i, iso := range []string{"2026-03-08T04:30:00Z", "2026-03-08T05:30:00Z", "2026-03-08T06:30:00Z", "2026-03-08T07:30:00Z"} {
		instant, _ := time.Parse(time.RFC3339, iso)
		raw, _ := json.Marshal(map[string]any{"selected": map[string]any{"precision": "instant", "resolved": map[string]any{"iso": iso, "unix_ns": instant.UnixNano()}}})
		s.Records.Library = append(s.Records.Library, catalog.LibraryEntry{ID: StableID("media", fmt.Sprint(i)), Revision: 1, Dates: raw})
	}
	c, e := Build(s)
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		zone string
		want int
	}{{"America/New_York", 3}, {"+02:00", 4}, {"-05:00", 3}, {"UTC", 4}} {
		f := contracts.CalendarFilter{From: "2026-03-08", Through: "2026-03-08", Timezone: tc.zone}
		r, e := Calendar(c, f, nil)
		if e != nil || r.Total != tc.want {
			t.Fatalf("zone %s total=%d want=%d: %v", tc.zone, r.Total, tc.want, e)
		}
	}
	for _, zone := range []string{"Bad/Zone", "+24:00", "+01:60"} {
		if _, e := Calendar(c, contracts.CalendarFilter{Timezone: zone}, nil); e == nil {
			t.Fatal("accepted invalid zone", zone)
		}
	}
}

func TestGraphContinuationWithEarlierHubAndSelfLoop(t *testing.T) {
	c := pagingCorpus(12, false)
	for i := range c.Refs.Edges {
		c.Refs.Edges[i].From = "node:0000"
		c.Refs.Edges[i].To = fmt.Sprintf("node:%04d", i+1)
	}
	c.Refs.Edges = append(c.Refs.Edges, graph.Edge{ID: "self", From: "node:0000", To: "node:0000", Kind: "self"})
	for _, limit := range []int{1, 2, 100, 500} {
		q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view", Pagination: &contracts.QueryPagination{Limit: limit}}}
		seen := map[string]bool{}
		for {
			r, e := Query(c, c.Refs, q)
			if e != nil {
				t.Fatal(e)
			}
			for _, edge := range r.Edges {
				if seen[edge.ID] {
					t.Fatal("repeated relationship")
				}
				seen[edge.ID] = true
			}
			if r.NextCursor == "" {
				break
			}
			q.Definition.Pagination.Cursor = r.NextCursor
		}
		if len(seen) != len(c.Refs.Edges) {
			t.Fatal("lost earlier hub/self edge")
		}
	}
}
