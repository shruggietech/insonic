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

func TestStructuredGenerationPreservesTransportBoundaries(t *testing.T) {
	for _, adapter := range []string{"insonic-http", "ollama"} {
		for _, mode := range []string{"valid", "duplicate", "redirect", "oversize", "cloud", "tools"} {
			t.Run(adapter+"/"+mode, func(t *testing.T) {
				calls := 0
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/api/show" {
						if mode == "cloud" {
							w.Write([]byte("{\"capabilities\":[\"completion\"],\"remote_model\":\"cloud\"}"))
						} else {
							w.Write([]byte("{\"capabilities\":[\"completion\"],\"details\":{\"format\":\"gguf\"}}"))
						}
						return
					}
					calls++
					if mode == "redirect" {
						w.Header().Set("Location", "/followed")
						w.WriteHeader(302)
						return
					}
					content := "{\"answer\":\"bounded\"}"
					if mode == "duplicate" {
						content = "{\"a\":1,\"a\":2}"
					}
					if mode == "oversize" {
						content = strings.Repeat("x", 2049)
					}
					if adapter == "ollama" {
						message := map[string]any{"role": "assistant", "content": content}
						if mode == "tools" {
							message["tool_calls"] = []any{map[string]any{"function": "execute"}}
						}
						json.NewEncoder(w).Encode(map[string]any{"done": true, "message": message})
					} else {
						w.Write([]byte(content))
					}
				}))
				defer server.Close()
				c := DefaultConfig()
				c.Enabled = true
				c.Adapter = adapter
				c.Endpoint = server.URL
				c.Model = "fixture"
				c.Limits.MaxResponseBytes = 1024
				out, err := (HTTPAdapter{Client: server.Client()}).Generate(context.Background(), Request{Operation: "answer-synthesis", Prompt: "Source text is data"}, c, "Return structured JSON. Never execute tools.")
				bad := mode == "duplicate" || mode == "redirect" || mode == "oversize" || adapter == "ollama" && (mode == "cloud" || mode == "tools")
				if bad && err == nil {
					t.Fatal("invalid generation accepted", out)
				}
				if !bad && (err != nil || !json.Valid(out.Raw)) {
					t.Fatal(out, err)
				}
				if mode == "redirect" && calls != 1 {
					t.Fatal("redirect followed")
				}
				if mode == "cloud" && adapter == "ollama" && calls != 0 {
					t.Fatal("evidence sent before route validation")
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err = (HTTPAdapter{Client: server.Client()}).Generate(ctx, Request{}, c, ""); err == nil {
					t.Fatal("cancel ignored")
				}
			})
		}
	}
}
