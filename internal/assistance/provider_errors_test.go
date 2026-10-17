// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestProviderFailureMatrix(t *testing.T) {
	for _, adapter := range []string{"insonic-http", "ollama"} {
		t.Run(adapter, func(t *testing.T) {
			protocol := "provider_protocol"
			if adapter == "ollama" {
				protocol = "ollama_protocol"
			}
			cases := []struct {
				name, code, reason string
				status             int
			}{
				{"auth", "unavailable", "provider_auth", 401}, {"forbidden", "unavailable", "provider_auth", 403},
				{"missing", "unavailable", "provider_missing", 404}, {"busy", "unavailable", "provider_busy", 429},
				{"overloaded", "unavailable", "provider_busy", 503}, {"failure", "operation_failed", "provider_failure", 500},
				{"redirect", "unavailable", "provider_redirect", 302},
				{"malformed", "operation_failed", protocol, 200}, {"duplicate", "operation_failed", protocol, 200},
				{"invalid-proposal", "operation_failed", protocol, 200},
				{"trailing", "operation_failed", protocol, 200}, {"truncated", "operation_failed", protocol, 200},
				{"short-body", "operation_failed", protocol, 200}, {"oversize", "output_limit", "provider_response_limit", 200},
				{"deadline", "unavailable", "provider_timeout", 200}, {"body-deadline", "unavailable", "provider_timeout", 200},
			}
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					var followed atomic.Int32
					s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Path == "/followed" {
							followed.Add(1)
							return
						}
						if r.URL.Path == "/api/show" {
							w.Write([]byte(`{"capabilities":["completion"],"details":{"format":"gguf"}}`))
							return
						}
						if tc.name == "deadline" {
							select {
							case <-r.Context().Done():
							case <-time.After(500 * time.Millisecond):
							}
							return
						}
						if tc.name == "body-deadline" {
							w.WriteHeader(200)
							w.(http.Flusher).Flush()
							select {
							case <-r.Context().Done():
							case <-time.After(500 * time.Millisecond):
							}
							return
						}
						if tc.name == "short-body" {
							w.Header().Set("Content-Length", "500")
						}
						if tc.status == 302 {
							w.Header().Set("Location", "/followed")
						}
						w.WriteHeader(tc.status)
						body := "private-body-secret"
						switch tc.name {
						case "duplicate":
							body = `{"contract_version":"1","contract_version":"1"}`
						case "invalid-proposal":
							body = `{"contract_version":"1","query":{},"explanation":"invalid"}`
							if adapter == "ollama" {
								body = `{"done":true,"message":{"role":"assistant","content":"{}"}}`
							}
						case "trailing":
							body = "{}{}"
						case "truncated":
							body = "{"
						case "oversize":
							body = strings.Repeat("x", 1025)
						}
						w.Write([]byte(body))
					}))
					defer s.Close()
					c := DefaultConfig()
					c.Enabled = true
					c.Adapter = adapter
					c.Endpoint = s.URL
					c.Model = "fixture"
					c.Limits.MaxResponseBytes = 1024
					c.Limits.TimeoutMS = 100
					_, err := (HTTPAdapter{Client: s.Client()}).Propose(context.Background(), Request{Prompt: "private-source-secret"}, c)
					code, reason := tc.code, tc.reason
					if adapter == "ollama" && tc.status != 200 {
						code = "unavailable"
						switch tc.status {
						case 401, 403:
							reason = "ollama_auth"
						case 404:
							reason = "ollama_missing"
						case 429, 503:
							reason = "ollama_busy"
						case 302:
							reason = "ollama_redirect"
						default:
							reason = "ollama_provider"
						}
					}
					assertProviderError(t, err, code, reason)
					if followed.Load() != 0 {
						t.Fatal("redirect followed")
					}
				})
			}
			for _, cancelled := range []bool{false, true} {
				t.Run("pre-context-"+strconv.FormatBool(cancelled), func(t *testing.T) {
					ctx, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					defer stop()
					code, reason := "unavailable", "provider_timeout"
					if cancelled {
						ctx, stop = context.WithCancel(context.Background())
						stop()
						code, reason = "cancelled", ""
					}
					c := DefaultConfig()
					c.Enabled = true
					c.Adapter = adapter
					c.Endpoint = "http://127.0.0.1:1"
					c.Model = "fixture"
					_, err := (HTTPAdapter{Client: &http.Client{}}).Propose(ctx, Request{}, c)
					assertProviderError(t, err, code, reason)
				})
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			endpoint := "http://" + listener.Addr().String()
			listener.Close()
			c := DefaultConfig()
			c.Enabled = true
			c.Adapter = adapter
			c.Endpoint = endpoint
			c.Model = "fixture"
			_, err = (HTTPAdapter{Client: &http.Client{}}).Propose(context.Background(), Request{}, c)
			reason := "provider_unavailable"
			if adapter == "ollama" {
				reason = "ollama_unavailable"
			}
			assertProviderError(t, err, "unavailable", reason)
		})
	}
}
func assertProviderError(t *testing.T, err error, code, reason string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected provider error")
	}
	e := contracts.SafeError(err)
	if e.Code != code || e.Reason != reason {
		t.Fatalf("got %s/%s, want %s/%s", e.Code, e.Reason, code, reason)
	}
	raw, _ := json.Marshal(e)
	for _, marker := range []string{"private-body-secret", "private-source-secret", "127.0.0.1", "unexpected EOF"} {
		if strings.Contains(string(raw), marker) {
			t.Fatal("external diagnostic leaked", string(raw))
		}
	}
	if reason != "" && e.Message == contracts.Fail(code).Message {
		t.Fatal("missing corrective message")
	}
}
