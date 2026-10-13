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

func TestSelectedModelMetadataFailurePreservesEvidenceAndCorrection(t *testing.T) {
	for _, phase := range []string{"auth", "malformed", "oversize", "redirect"} {
		t.Run(phase, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/version":
					w.Write([]byte("{\"version\":\"fixture\"}"))
				case "/api/tags":
					w.Write([]byte("{\"models\":[{\"name\":\"selected\",\"digest\":\"abc\",\"size\":42}]}"))
				case "/api/show":
					switch phase {
					case "auth":
						http.Error(w, "private-provider-secret", 401)
					case "malformed":
						w.Write([]byte("{\"capabilities\":[\"completion\"],\"capabilities\":[]}"))
					case "oversize":
						w.Write([]byte(strings.Repeat("x", (1<<20)+1)))
					case "redirect":
						http.Redirect(w, r, "/must-not-follow", 302)
					}
				default:
					t.Error("unexpected request", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			a := Ollama{Client: server.Client()}
			info, e := a.Inspect(context.Background(), server.URL, "", "selected")
			if e == nil || info.Version != "fixture" || len(info.Models) != 1 || info.Models[0].DetailState != "failed" || info.Models[0].Correction == "" {
				t.Fatal(info, e)
			}
			c := DefaultConfig()
			c.Adapter = "ollama"
			c.Endpoint = server.URL
			c.Model = "selected"
			out, e := a.Discover(context.Background(), DiscoveryInput{}, c)
			if e != nil {
				t.Fatal(e)
			}
			found := false
			for _, candidate := range out.Candidates {
				if candidate.Endpoint == server.URL {
					found = true
					if !candidate.Running || !candidate.Reachable || candidate.ProbeState != "metadata-failed" || candidate.Correction == "" || candidate.Info == nil || len(candidate.Info.Models) != 1 {
						t.Fatal(candidate)
					}
				}
			}
			if !found {
				t.Fatal("missing selected candidate")
			}
			raw, _ := json.Marshal(out)
			if strings.Contains(string(raw), "private-provider-secret") {
				t.Fatal("provider text disclosed")
			}
		})
	}
}

func TestUnselectedModelFailureKeepsOtherModelsAndCorrection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			w.Write([]byte("{\"version\":\"fixture\"}"))
		case "/api/tags":
			w.Write([]byte("{\"models\":[{\"name\":\"unavailable\"},{\"name\":\"selected\"}]}"))
		case "/api/show":
			var input struct{ Model string }
			json.NewDecoder(r.Body).Decode(&input)
			if input.Model == "unavailable" {
				http.Error(w, "private-provider-secret", 401)
			} else {
				w.Write([]byte("{\"capabilities\":[\"completion\"],\"details\":{\"format\":\"gguf\"}}"))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	info, e := (Ollama{Client: server.Client()}).Inspect(context.Background(), server.URL, "", "selected")
	if e != nil || len(info.Models) != 2 || info.Models[0].Correction == "" || info.Models[0].Compatible || !info.Models[1].Compatible {
		t.Fatal(info, e)
	}
}
