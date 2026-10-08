// SPDX-License-Identifier: Apache-2.0
package explore

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"testing"
)

func TestTraversalDepthDoesNotGrowDuringOneHop(t *testing.T) {
	c := Corpus{Rows: map[string]Row{}, Refs: graph.References{}}
	for _, id := range []string{"a", "b", "c", "d"} {
		c.Rows[id] = Row{ID: id, Kind: "cue", StartUS: map[string]string{"a": "100", "b": "20", "c": "3", "d": "2"}[id]}
		c.Refs.Nodes = append(c.Refs.Nodes, graph.Node{ID: id})
	}
	c.Refs.Edges = []graph.Edge{{ID: "1", From: "a", To: "b"}, {ID: "2", From: "b", To: "c"}, {ID: "3", From: "c", To: "d"}}
	q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "graph-view", Filters: contracts.QueryFilter{EntityIDs: []string{"a"}}, Traversal: &contracts.QueryTraversal{Direction: "out", MaxDepth: 1}}}
	r, e := Query(c, c.Refs, q)
	if e != nil || len(r.Rows) != 2 {
		t.Fatal(r, e)
	}
	q.Definition.Traversal = nil
	q.Definition.Filters = contracts.QueryFilter{}
	q.Definition.OrderBy = []contracts.QueryOrder{{Field: "cue.start_us", Direction: "asc"}}
	r, e = Query(c, c.Refs, q)
	if e != nil || r.Rows[0].ID != "d" || r.Rows[3].ID != "a" {
		t.Fatal("clock alias not numeric", r, e)
	}
}

func TestDisjointAssertionSpansDoNotManufactureGapEvidence(t *testing.T) {
	row := Row{ID: "assertion:a", Kind: "assertion", StartUS: "10", EndUS: "20", SourceSpans: []SourceSpan{{"a", "10", "20"}, {"b", "80", "90"}}}
	c := Corpus{Rows: map[string]Row{row.ID: row}, Refs: graph.References{Nodes: []graph.Node{{ID: row.ID}}}}
	for _, interval := range []struct {
		lo, hi int64
		count  int
	}{{40, 50, 0}, {80, 81, 1}} {
		q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "time-range", Filters: contracts.QueryFilter{SourceInterval: &contracts.SourceInterval{StartUS: interval.lo, EndUS: interval.hi}}}}
		r, e := Query(c, c.Refs, q)
		if e != nil || len(r.Rows) != interval.count {
			t.Fatal(r, e)
		}
	}
}
