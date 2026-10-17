// SPDX-License-Identifier: Apache-2.0
package evidence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPAliases(t *testing.T) {
	for _, alias := range []string{"http", "insonic-http"} {
		c := DefaultConfig()
		c.Adapter = alias
		c.Endpoint = "http://127.0.0.1:1"
		normalized, err := Normalize(c)
		if err != nil || normalized.Adapter != "insonic-http" {
			t.Fatal(alias, normalized, err)
		}
	}
}

func TestHTTPRejectsOtherChunkForeignCueAndSpoofedClocks(t *testing.T) {
	cues := testCues()
	for _, spoof := range []bool{false, true} {
		t.Run(map[bool]string{false: "cue-ownership", true: "spoofed-clock"}[spoof], func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var input struct {
					Chunk Chunk `json:"chunk"`
				}
				json.NewDecoder(r.Body).Decode(&input)
				id := input.Chunk.Cues[0].ID
				assertions := []map[string]any{}
				for _, cue := range []string{id, "foreign-recording", cues[(input.Chunk.Index+1)%len(cues)].ID} {
					a := map[string]any{"subject": "source", "relation": "states", "object": "conditional claim", "polarity": "negative", "modality": "conditional", "conditions": []string{"unless revised"}, "cue_ids": []string{cue}}
					if spoof {
						a["source_start_us"] = "123456789"
					}
					assertions = append(assertions, a)
				}
				json.NewEncoder(w).Encode(map[string]any{"contract_version": "1", "assertions": assertions})
			}))
			defer s.Close()
			c := DefaultConfig()
			c.Adapter = "insonic-http"
			c.Endpoint = s.URL
			c.WindowCues = 1
			c.OverlapCues = 0
			out, err := Extract(context.Background(), cues, c, HTTPAdapter{})
			if err != nil || len(out.Diagnostics) == 0 {
				t.Fatal(out, err)
			}
			want := 3
			if spoof {
				want = 0
			}
			if len(out.Assertions) != want {
				t.Fatal(out)
			}
			for i, a := range out.Assertions {
				if len(a.CueIDs) != 1 || a.CueIDs[0] != cues[i].ID || a.ID == "" || a.Polarity != "negative" || len(a.Conditions) != 1 {
					t.Fatal(a)
				}
			}
		})
	}
}
func TestHTTPResponseShape(t *testing.T) {
	for _, raw := range []string{`{"contract_version":"1"}`, `{"contract_version":"1","assertions":null}`, `{"contract_version":"1","assertions":[],"assertions":[]}`, `{"contract_version":"1","assertions":[],"source_start_us":"0"}`, `{"contract_version":"1","assertions":[]}`} {
		t.Run(raw, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(raw)) }))
			defer s.Close()
			c := DefaultConfig()
			c.Adapter = "insonic-http"
			c.Endpoint = s.URL
			_, err := (HTTPAdapter{}).Extract(context.Background(), Chunk{}, c)
			valid := raw == `{"contract_version":"1","assertions":[]}`
			if valid != (err == nil) {
				t.Fatal(raw, err)
			}
		})
	}
}
