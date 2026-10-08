// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/evidence"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestExploreHTTPExtractionUsesElectedProtocol(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Chunk evidence.Chunk `json:"chunk"`
		}
		if json.NewDecoder(r.Body).Decode(&input) != nil || len(input.Chunk.Cues) == 0 {
			t.Error("missing cue context")
			w.WriteHeader(400)
			return
		}
		cue := input.Chunk.Cues[0]
		_ = json.NewEncoder(w).Encode(map[string]any{"contract_version": "1", "assertions": []evidence.Assertion{{Subject: "source", Relation: "states", Object: "conditional claim", Polarity: "negative", Modality: "conditional", Conditions: []string{"unless condition"}, CueIDs: []string{cue.ID}}}})
	}))
	defer server.Close()
	a := configuredApp(t)
	entry := desktopMedia(t, a, false)
	doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
	rec := desktopRecording(t, a, entry, 0, doc)
	cfg := evidence.DefaultConfig()
	cfg.Adapter = "insonic-http"
	cfg.Endpoint = server.URL
	cfg.Model = "fixture-contract"
	r := realRequest(a, "evidence.extract", entry.ID, map[string]any{"expected_revision": rec.Revision, "document_digest": rec.DocumentDigest, "configuration": cfg})
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	id := r.Result.(map[string]any)["work_id"].(string)
	deadline := time.Now().Add(4 * time.Second)
	for {
		work, e := a.Catalog.Work(a.ctx, id)
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
	if len(assertions) != 1 || assertions[0].Polarity != "negative" || assertions[0].Modality != "conditional" {
		t.Fatal(assertions)
	}
}
