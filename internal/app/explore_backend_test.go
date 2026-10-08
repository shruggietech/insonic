// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"encoding/json"
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
