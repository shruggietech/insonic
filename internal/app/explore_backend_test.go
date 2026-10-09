// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/assistance"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/explore"
	"github.com/shruggietech/insonic/internal/graph"
	"os"
	"testing"
)

func exploreBackendSuite(t *testing.T, a *App, g graph.Adapter) {
	t.Helper()
	if e := g.EnsureSchema(a.ctx); e != nil {
		t.Fatal(e)
	}
	if e := g.BindWorkspace(a.ctx, a.Workspace.Config.WorkspaceID); e != nil {
		t.Fatal(e)
	}
	a.graphMu.Lock()
	a.Graph = g
	a.graphMu.Unlock()
	entry := desktopMedia(t, a, false)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	rec := desktopRecording(t, a, entry, 0, doc)
	published := realRequest(a, "graph.publish", "", nil)
	if published.Error != nil {
		t.Fatal("publish dispatch", published.Error)
	}
	snap, e := a.Catalog.Export(a.ctx)
	if e != nil {
		t.Fatal(e)
	}
	c, e := explore.Build(snap)
	if e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"media-list", "speaker-search", "model-list", "text-search", "time-range", "evidence-traverse", "graph-view"} {
		q := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: op, Pagination: &contracts.QueryPagination{Limit: 2}}}
		expected, e := explore.Query(c, c.Refs, q)
		if e != nil {
			t.Fatal(e)
		}
		response := realRequest(a, "query.run", "", q)
		if response.Error != nil {
			t.Fatal(op, response.Error)
		}
		actual := response.Result.(explore.Result)
		want, _ := json.Marshal(expected.Rows)
		got, _ := json.Marshal(actual.Rows)
		if string(want) != string(got) || expected.Total != actual.Total {
			t.Fatal(op, "backend parity", string(want), string(got))
		}
	}

	// Deterministic assistant output still executes against the real selected backend.
	q := testProposal().Query
	a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
		return testProposal(), nil
	}
	assisted := realRequest(a, "query.assist", "", map[string]any{"prompt": "list media", "mode": "auto-run", "configuration": assistanceConfig()})
	if assisted.Error != nil {
		t.Fatal("assistance backend", assisted.Error)
	}
	direct := realRequest(a, "query.run", "", q)
	if direct.Error != nil {
		t.Fatal(direct.Error)
	}
	assertCurrentQueryParity(t, assisted.Result.(assistanceResult).Execution, direct.Result)
	// Native output also passes selected-engine EXPLAIN before elected execution.
	counter := &countedAssistanceGraph{Adapter: g}
	a.Graph = counter
	dialect := "ladybug-cypher"
	if g.Capabilities()["adapter_id"] == "arcadedb" {
		dialect = "arcade-opencypher"
	}
	native := contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "native", Dialect: dialect, Text: "MATCH (n:Entity) RETURN n.entity_id AS id LIMIT 5"}}
	a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
		p := testProposal()
		p.Query = native
		return p, nil
	}

	suggestion := realRequest(a, "query.assist", "", map[string]any{"prompt": "suggest identities", "configuration": assistanceConfig()})
	if suggestion.Error != nil || counter.queries != 0 {
		t.Fatal("suggestion executed", suggestion.Error, counter.queries)
	}
	a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
		p := testProposal()
		p.Query = native
		p.Query.Definition.Text = "MATCH (n) DELETE n"
		return p, nil
	}
	rejected := realRequest(a, "query.assist", "", map[string]any{"prompt": "invalid", "mode": "auto-run", "configuration": assistanceConfig()})
	if rejected.Error == nil || counter.queries != 0 {
		t.Fatal("invalid output executed", counter.queries)
	}
	a.AssistanceFixture = func(context.Context, assistance.Request, assistance.Config) (assistance.Proposal, error) {
		p := testProposal()
		p.Query = native
		return p, nil
	}
	assisted = realRequest(a, "query.assist", "", map[string]any{"prompt": "list identities", "mode": "auto-run", "configuration": assistanceConfig()})
	if assisted.Error != nil {
		t.Fatal("native assistant", assisted.Error)
	}
	if counter.queries != 1 {
		t.Fatal("auto-run did not execute exactly once", counter.queries)
	}
	a.Graph = g

	// Decimal transport strings bind as real int64 values on each supported engine.
	paramText := "MATCH (n:Entity) RETURN $clock AS clock LIMIT 1"
	params := map[string]contracts.QueryParameter{"clock": {Type: "integer", Value: json.RawMessage(`"9223372036854775807"`)}}
	exact, e := g.Query(a.ctx, dialect, paramText, params)
	if e != nil || len(exact) != 1 {
		t.Fatal("exact integer binding", e, exact)
	}
	rawExact, _ := json.Marshal(exact)
	if !bytes.Contains(rawExact, []byte("9223372036854775807")) {
		t.Fatal("integer rounded before engine", string(rawExact))
	}
	a.AssistanceFixture = nil
	// Apply a current-document replacement, then ensure old reference IDs disappear.
	var parsed map[string]any
	decoder := json.NewDecoder(bytes.NewReader(doc))
	decoder.UseNumber()
	if e = decoder.Decode(&parsed); e != nil {
		t.Fatal(e)
	}
	parsed["cues"].([]any)[0].(map[string]any)["payload"].(map[string]any)["plain_text"] = "Corrected current source"

	replacement, _ := json.Marshal(parsed)
	next := desktopRecording(t, a, entry, rec.Revision, replacement)
	response := realRequest(a, "query.run", "", map[string]any{"definition": map[string]any{"mode": "normalized", "operation": "text-search", "filters": map[string]any{"text": "Corrected current source"}}})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	actual := response.Result.(explore.Result)
	if len(actual.Rows) != 1 || actual.Rows[0].DocumentDigest != next.DocumentDigest {
		t.Fatal("replacement not published", actual)
	}
	refs, e := g.ReadRefs(a.ctx, a.Workspace.Config.WorkspaceID)
	if e != nil {
		t.Fatal(e)
	}
	for _, n := range refs.Nodes {
		if n.Kind == "cue" && n.ID == "cue:"+entry.ID+":"+rec.DocumentDigest+":"+actual.Rows[0].CueID {
			t.Fatal("old cue remained")
		}
	}
	rebuilt := realRequest(a, "graph.rebuild", "", nil)
	if rebuilt.Error != nil {
		t.Fatal("rebuild dispatch", rebuilt.Error)
	}
}

type countedAssistanceGraph struct {
	graph.Adapter
	queries int
}

func (g *countedAssistanceGraph) Query(ctx context.Context, dialect, text string, parameters map[string]contracts.QueryParameter) (graph.Rows, error) {
	g.queries++
	return g.Adapter.Query(ctx, dialect, text, parameters)
}
