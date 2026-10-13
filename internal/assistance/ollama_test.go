// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOllamaNativeProposalAndCloudElection(t *testing.T) {
	cloud := false
	chats := 0
	var format any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			if cloud {
				w.Write([]byte("{\"capabilities\":[\"completion\"],\"remote_host\":\"https://ollama.com\",\"remote_model\":\"fixture\"}"))
			} else {
				w.Write([]byte("{\"capabilities\":[\"completion\"],\"details\":{\"format\":\"gguf\"}}"))
			}
		case "/api/chat":
			chats++
			var in map[string]any
			json.NewDecoder(r.Body).Decode(&in)
			format = in["format"]
			if in["stream"] != false {
				t.Error("streaming")
			}
			content := "{\"contract_version\":\"1\",\"query\":{\"definition\":{\"mode\":\"normalized\",\"operation\":\"media-list\",\"pagination\":{\"limit\":5}},\"parameters\":{}},\"explanation\":\"List media\"}"
			json.NewEncoder(w).Encode(map[string]any{"done": true, "message": map[string]any{"role": "assistant", "content": content}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	c := DefaultConfig()
	c.Enabled = true
	c.Adapter = "ollama"
	c.Endpoint = server.URL
	c.Model = "fixture"
	a := Ollama{Client: server.Client()}
	if _, e := a.Propose(context.Background(), Request{Prompt: "List five media"}, c); e != nil || format != "json" || chats != 1 {
		t.Fatal(e, format, chats)
	}
	cloud = true
	if _, e := a.Propose(context.Background(), Request{Prompt: "private"}, c); e == nil || chats != 1 {
		t.Fatal("silent cloud route", e, chats)
	}
	c.Route = "hosted"
	if _, e := a.Propose(context.Background(), Request{Prompt: "List five media"}, c); e != nil || format != nil || chats != 2 {
		t.Fatal("elected cloud", e, format, chats)
	}
}
func TestOllamaMetadataBoundedAndNoPrompt(t *testing.T) {
	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if strings.Contains(r.URL.Path, "chat") {
			t.Error("inference during metadata")
		}
		switch r.URL.Path {
		case "/api/version":
			w.Write([]byte("{\"version\":\"0.99\"}"))
		case "/api/tags":
			w.Write([]byte("{\"models\":[{\"name\":\"fixture\",\"size\":42,\"digest\":\"abc\"}]}"))
		case "/api/show":
			w.Write([]byte("{\"capabilities\":[\"completion\"],\"details\":{\"format\":\"gguf\"}}"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	result, e := (Ollama{Client: server.Client()}).Inspect(context.Background(), server.URL, "", "")
	if e != nil || result.Version != "0.99" || len(result.Models) != 1 || result.Models[0].InferenceRoute != "local" || hits != 3 {
		t.Fatal(result, e, hits)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = (Ollama{Client: server.Client()}).Inspect(ctx, server.URL, "", ""); e == nil {
		t.Fatal("ignored cancellation")
	}
}

func TestOllamaRedirectOutputCapAndRedaction(t *testing.T) {
	for _, kind := range []string{"redirect", "oversize", "auth", "malformed"} {
		hits := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			switch kind {
			case "redirect":
				http.Redirect(w, r, "/followed", 302)
			case "oversize":
				w.Write([]byte(strings.Repeat("x", (1<<20)+1)))
			case "auth":
				http.Error(w, "private-provider-secret", 401)
			case "malformed":
				w.Write([]byte("{\"version\":\"x\",\"version\":\"y\"}"))
			}
		}))
		_, e := (Ollama{Client: server.Client()}).Inspect(context.Background(), server.URL, "", "")
		server.Close()
		if e == nil || strings.Contains(e.Error(), "private-provider-secret") || hits != 1 {
			t.Fatal(kind, e, hits)
		}
	}
}
func TestExternalHintsAreNotContactedOrEchoed(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "http://user:private-secret@external.example:11434")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, _ := (Ollama{}).Discover(ctx, DiscoveryInput{}, DefaultConfig())
	raw, _ := json.Marshal(out)
	if strings.Contains(string(raw), "private-secret") {
		t.Fatal(string(raw))
	}
	if endpoint, e := normalizeHint("0.0.0.0:11435"); e != nil || endpoint != "http://127.0.0.1:11435" {
		t.Fatal(endpoint, e)
	}
	if endpoint, e := normalizeHint("[::]:11435"); e != nil || endpoint != "http://[::1]:11435" {
		t.Fatal(endpoint, e)
	}
}

func TestDiscoveryRetainsVersionEvidenceOnMetadataFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			w.Write([]byte("{\"version\":\"fixture\"}"))
			return
		}
		w.Write([]byte("malformed"))
	}))
	defer server.Close()
	c := DefaultConfig()
	c.Adapter = "ollama"
	c.Endpoint = server.URL
	out, e := (Ollama{Client: server.Client()}).Discover(context.Background(), DiscoveryInput{}, c)
	if e != nil {
		t.Fatal(e)
	}
	for _, candidate := range out.Candidates {
		if candidate.Endpoint == server.URL {
			if !candidate.Reachable || !candidate.Running || candidate.ProbeState != "metadata-failed" {
				t.Fatal(candidate)
			}
			return
		}
	}
	t.Fatal("missing candidate")
}
