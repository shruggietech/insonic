// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/schemas"
)

type answerGraphCatalog struct {
	catalog.Catalog
	checkpoint graph.Checkpoint
}

func (c answerGraphCatalog) GraphStatus(context.Context) (map[string]int64, error) {
	return map[string]int64{"pending_events": 0, "checkpoint": c.checkpoint.Sequence, "next_sequence": c.checkpoint.Sequence, "generation": c.checkpoint.Generation}, nil
}
func (answerGraphCatalog) Export(context.Context) (catalog.Snapshot, error) {
	panic("answer graph traversal hydrated the whole catalog")
}

type answerFixtureGraph struct {
	graph.Adapter
	checkpoint graph.Checkpoint
	edges      []graph.Edge
}

func (g *answerFixtureGraph) Close() {}
func (g *answerFixtureGraph) ReadCheckpoint(context.Context, string, string) (graph.Checkpoint, error) {
	return g.checkpoint, nil
}
func (g *answerFixtureGraph) Neighborhood(_ context.Context, r graph.NeighborhoodRequest) (graph.NeighborhoodPage, error) {
	if r.Checkpoint != g.checkpoint {
		return graph.NeighborhoodPage{}, contracts.Fail("conflict")
	}
	p := graph.NeighborhoodPage{Edges: []graph.Edge{}, Rows: 1}
	for _, edge := range g.edges {
		if edge.From == r.Node || edge.To == r.Node {
			p.NodePresent = true
		}
		if edge.ID <= r.After || len(r.Kinds) > 0 && !slices.Contains(r.Kinds, edge.Kind) {
			continue
		}
		if (r.Direction == "out" && edge.From == r.Node) || (r.Direction == "in" && edge.To == r.Node) || (r.Direction == "both" && (edge.From == r.Node || edge.To == r.Node)) {
			p.Edges = append(p.Edges, edge)
		}
	}
	sort.Slice(p.Edges, func(i, j int) bool { return p.Edges[i].ID < p.Edges[j].ID })
	if len(p.Edges) >= r.Limit {
		p.Edges = p.Edges[:r.Limit]
		p.Next = p.Edges[len(p.Edges)-1].ID
	}
	raw, _ := json.Marshal(p.Edges)
	p.Bytes = len(raw)
	p.Rows += len(p.Edges)
	return p, nil
}

func TestAnswerMultiHopAndStaleGraphCandidates(t *testing.T) {
	a := configuredApp(t)
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	one := desktopMedia(t, a, false)
	r1 := desktopRecording(t, a, one, 0, []byte(strings.ReplaceAll(string(doc), "Research source.", "alpha launch decision")))
	two := desktopMedia(t, a, false)
	r2 := desktopRecording(t, a, two, 0, []byte(strings.ReplaceAll(string(doc), "Research source.", "beta supporting context")))
	three := desktopMedia(t, a, false)
	desktopRecording(t, a, three, 0, []byte(strings.ReplaceAll(string(doc), "Research source.", "disconnected context")))
	c1 := "cue:" + one.ID + ":" + r1.DocumentDigest + ":cue-000000"
	c2 := "cue:" + two.ID + ":" + r2.DocumentDigest + ":cue-000000"
	stale := "cue:" + two.ID + ":" + strings.Repeat("f", 64) + ":cue-000000"
	cp := graph.Checkpoint{Generation: 1, Sequence: 1, CatalogRevision: 1}
	g := &answerFixtureGraph{checkpoint: cp, edges: []graph.Edge{{ID: "a", From: "assertion:one", To: c1, Kind: "supported-by"}, {ID: "b", From: "assertion:one", To: "concept:shared", Kind: "subject"}, {ID: "c", From: "assertion:two", To: "concept:shared", Kind: "subject"}, {ID: "d", From: "assertion:two", To: c2, Kind: "supported-by"}, {ID: "e", From: "assertion:two", To: stale, Kind: "supported-by"}}}
	a.graphMu.Lock()
	a.Graph = g
	a.Catalog = answerGraphCatalog{a.Catalog, cp}
	a.graphMu.Unlock()
	zero := 0
	r := answers.Request{Version: contracts.Version, Question: "launch", Filters: answers.Filters{Text: "alpha"}, Traversal: &contracts.QueryTraversal{Direction: "both", MaxDepth: 4}, Budget: answers.Budget{NeighborCues: &zero, Rounds: 64}}
	out, err := a.prepareAnswer(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Package.Items) != 2 {
		t.Fatal("missing cross-recording evidence or admitted disconnected source", out.Package)
	}
	found := false
	for _, item := range out.Package.Items {
		if item.ID == c2 {
			for _, path := range item.Paths {
				if len(path.Edges) == 4 && path.Nodes[0] == c1 && path.Nodes[4] == c2 {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("exact multi-hop route not retained", out.Package.Items)
	}
	if !slices.Contains(out.Package.Diagnostics, "graph_authority_missing") {
		t.Fatal("stale projection hidden", out.Package)
	}
	for _, item := range out.Package.Items {
		if item.ID == stale {
			t.Fatal("stale candidate admitted")
		}
	}
	if out.Usage.Rows > out.Package.Request.Budget.Rows || out.Usage.SourceBytes > out.Package.Request.Budget.SourceBytes || out.Usage.Rounds > 64 || out.Usage.Edges > out.Package.Request.Budget.Edges {
		t.Fatal("aggregate bounds", out.Usage)
	}
}

// The same test runs inside both actual normalized graph backend campaigns.
func answerBackendSuite(t *testing.T, a *App, recording string) {
	t.Helper()
	zero := 0
	r := answers.Request{Version: contracts.Version, Question: "Research source", Filters: answers.Filters{MediaIDs: []string{recording}}, Traversal: &contracts.QueryTraversal{Direction: "both", MaxDepth: 4, RelationshipTypes: []string{"supported-by"}}, Budget: answers.Budget{NeighborCues: &zero}}
	response := realRequest(a, "answers.prepare", "", r)
	if response.Error != nil {
		t.Fatal("answer dispatch", response.Error)
	}
	view := response.Result.(answerResult)
	if view.Freshness != "current" || view.Package.Observation.GraphState != "current" || len(view.Package.Items) != 1 {
		t.Fatal("native answer authority", view)
	}
	raw, _ := json.Marshal(view.Package)
	if err := schemas.ValidateAnswerPackage(raw); err != nil {
		t.Fatal("native package schema", err)
	}
	response = realRequest(a, "answers.prepare", "", r)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	cached := response.Result.(answerResult)
	if !cached.CacheHit || cached.Package.ID != view.Package.ID {
		t.Fatal("native compatible cache", cached)
	}
	for _, op := range []string{"answers.show", "answers.export"} {
		response = realRequest(a, op, view.Package.ID, nil)
		if response.Error != nil || response.Result.(answerResult).Freshness != "current" {
			t.Fatal(op, response)
		}
	}
}
