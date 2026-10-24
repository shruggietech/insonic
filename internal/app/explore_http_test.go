// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"github.com/shruggietech/insonic/schemas"
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
	for _, alias := range []string{"insonic-http", "http"} {
		t.Run(alias, func(t *testing.T) {
			a := configuredApp(t)
			entry := desktopMedia(t, a, false)
			doc, _ := os.ReadFile("../subtitles/testdata/source.cueson.json")
			rec := desktopRecording(t, a, entry, 0, doc)
			cfg := evidence.DefaultConfig()
			cfg.Adapter = alias
			cfg.Endpoint = server.URL
			cfg.Model = "fixture-contract"
			data, _ := json.Marshal(map[string]any{"expected_revision": rec.Revision, "document_digest": rec.DocumentDigest, "configuration": cfg})
			req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "evidence.extract", ItemID: entry.ID, Data: data}
			raw, _ := json.Marshal(req)
			if err := schemas.ValidateRequest(raw); err != nil {
				t.Fatal("public request schema rejected elected extractor", err)
			}
			r := a.Dispatch(req)
			if r.Error != nil {
				t.Fatal(r.Error)
			}
			id := r.Result.(map[string]any)["work_id"].(string)
			deadline := time.Now().Add(qualificationtiming.Duration("durable_work_seconds"))
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

		})
	}
}
