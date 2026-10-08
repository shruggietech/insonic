// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExploreExtractionDurableReplacementAndSavedDefinitions(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	doc, e := os.ReadFile("../subtitles/testdata/source.cueson.json")
	if e != nil {
		t.Fatal(e)
	}
	rec := desktopRecording(t, a, entry, 0, doc)
	in := map[string]any{"expected_revision": rec.Revision, "document_digest": rec.DocumentDigest, "configuration": evidence.DefaultConfig()}
	response := realRequest(a, "evidence.extract", entry.ID, in)
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	job := response.Result.(map[string]any)["work_id"].(string)
	deadline := time.Now().Add(4 * time.Second)
	for {
		work, e := a.Catalog.Work(a.ctx, job)
		if e != nil {
			t.Fatal(e)
		}
		if work.State == "succeeded" {
			break
		}
		if work.State == "failed" || time.Now().After(deadline) {
			t.Fatal(work)
		}
		time.Sleep(10 * time.Millisecond)
	}
	extraction, e := a.Catalog.Extraction(a.ctx, entry.ID)
	if e != nil {
		t.Fatal(e)
	}
	var assertions []evidence.Assertion
	_ = json.Unmarshal(extraction.Assertions, &assertions)
	if len(assertions) == 0 {
		t.Fatal("no source statements")
	}
	queryID := contracts.ID()
	saved := realRequest(a, "query.save", queryID, map[string]any{"expected_revision": 0, "query": map[string]any{"title": "Current evidence", "definition": map[string]any{"mode": "normalized", "operation": "text-search", "filters": map[string]any{"media_ids": []string{entry.ID}}}}})
	if saved.Error != nil {
		t.Fatal(saved.Error)
	}
	raw, _ := json.Marshal(saved.Result)
	if e = schemas.ValidateGraphQuery(raw); e != nil {
		t.Fatal("saved public contract", e, string(raw))
	}
	changed := desktopRecording(t, a, entry, rec.Revision, doc)
	if changed.Revision == rec.Revision {
		t.Fatal("replacement ignored")
	}
	if _, e = a.Catalog.Extraction(a.ctx, entry.ID); e == nil {
		t.Fatal("obsolete assertions retained")
	}
}
func TestExploreRuntimeContracts(t *testing.T) {
	for _, r := range []contracts.Request{
		{Operation: "timeline.calendar", Data: json.RawMessage(`{"filter":{"include_undated":true},"pagination":{"limit":25}}`)},
		{Operation: "query.run", Data: json.RawMessage(`{"definition":{"mode":"native","dialect":"ladybug-cypher","text":"MATCH (n:Entity) RETURN n.entity_id LIMIT 10"}}`)},
		{Operation: "query.save", ItemID: contracts.ID(), Data: json.RawMessage(`{"expected_revision":0,"query":{"title":"Search","definition":{"mode":"normalized","operation":"text-search","filters":{"text":"foo"}}}}`)},
	} {
		r.Kind = "runtime-request"
		r.Version = contracts.Version
		r.WorkspaceID = contracts.ID()
		r.RequestID = contracts.ID()
		raw, _ := json.Marshal(r)
		if e := schemas.ValidateRequest(raw); e != nil {
			t.Fatal(e, string(raw))
		}
	}
}

func TestPortableSearchRemainsAvailableWithoutGraphAndSavedMismatch(t *testing.T) {
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	desktopRecording(t, a, entry, 0, doc)
	a.Workspace.Config.Profiles.Graph.Adapter = "unavailable"
	response := realRequest(a, "query.run", "", map[string]any{"definition": map[string]any{"mode": "normalized", "operation": "text-search"}})
	if response.Error != nil {
		t.Fatal(response.Error)
	}
	value, _ := json.Marshal(response.Result)
	if !strings.Contains(string(value), "current-catalog-fallback") {
		t.Fatal(string(value))
	}
	a.Workspace.Config.Profiles.Graph.Adapter = "ladybugdb"
	id := contracts.ID()
	in := map[string]any{"expected_revision": 0, "query": map[string]any{"title": "Different engine", "definition": map[string]any{"mode": "native", "dialect": "arcade-sql", "text": "SELECT entity_id FROM Entity"}}}
	r := realRequest(a, "query.save", id, in)
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	raw, _ := json.Marshal(r.Result)
	if !strings.Contains(string(raw), "incompatible") || schemas.ValidateGraphQuery(raw) != nil {
		t.Fatal(string(raw))
	}
	if realRequest(a, "query.run", "", map[string]any{"saved_id": id}).Error == nil {
		t.Fatal("incompatible native query executed")
	}
}
