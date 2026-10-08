package evidence

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSelectedProtocolAndTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Operation string `json:"operation"`
			Chunk     Chunk  `json:"chunk"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Operation != "assertion-extraction" || len(body.Chunk.Cues) == 0 {
			t.Error("missing selected cue request")
		}
		json.NewEncoder(w).Encode(map[string]any{"contract_version": "1", "assertions": []Assertion{{Subject: "maintainer", Relation: "approval", Object: "proposal", Polarity: "negative", Modality: "conditional", Conditions: []string{"until revised"}, CueIDs: []string{body.Chunk.Cues[0].ID}}}})
	}))
	defer server.Close()
	cfg := DefaultConfig()
	cfg.Adapter = "insonic-http"
	cfg.Endpoint = server.URL
	out, e := Extract(context.Background(), testCues(), cfg, HTTPAdapter{})
	if e != nil || len(out.Assertions) == 0 || out.Assertions[0].Polarity != "negative" {
		t.Fatal(out, e)
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(30 * time.Millisecond) }))
	defer slow.Close()
	cfg.Endpoint = slow.URL
	cfg.TimeoutMS = 1
	out, e = Extract(context.Background(), testCues(), cfg, HTTPAdapter{})
	if e != nil || len(out.Diagnostics) == 0 || len(out.Assertions) != 0 {
		t.Fatal("failed route wasn't diagnosed", out, e)
	}
}
func TestHTTPNoRedirectAndNoFallback(t *testing.T) {
	called := false
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 302) }))
	defer redirect.Close()
	cfg := DefaultConfig()
	cfg.Adapter = "insonic-http"
	cfg.Endpoint = redirect.URL
	out, e := Extract(context.Background(), testCues(), cfg, HTTPAdapter{})
	if e != nil || called || len(out.Assertions) > 0 || len(out.Diagnostics) == 0 {
		t.Fatal(out, e, called)
	}
}
