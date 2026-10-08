// SPDX-License-Identifier: Apache-2.0
package assistance

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfigLimitsAndElectedLocalRoute(t *testing.T) {
	c := DefaultConfig()
	c.Enabled = true
	c.Endpoint = "http://127.0.0.1:1234/query"
	c.Model = "fixture"
	if _, e := Normalize(c); e != nil {
		t.Fatal(e)
	}
	for _, url := range []string{"http://example.org/query", "https://user:pass@localhost/query", "http://localhost/query?token=x"} {
		bad := c
		bad.Endpoint = url
		if _, e := Normalize(bad); e == nil {
			t.Fatal(url)
		}
	}
	c.Limits.MaxRows = 501
	if _, e := Normalize(c); e == nil {
		t.Fatal("unbounded rows")
	}
}
func TestProviderStrictOutputRedirectTimeoutAndCI(t *testing.T) {
	query := `{"contract_version":"1","query":{"definition":{"mode":"normalized","operation":"media-list"}},"explanation":"List media"}`
	for _, raw := range []string{query, query + `{}`, strings.Replace(query, `"contract_version":"1"`, `"contract_version":"1","contract_version":"1"`, 1), strings.Replace(query, `"explanation"`, `"configuration"`, 1)} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(raw)) }))
		c := DefaultConfig()
		c.Enabled = true
		c.Endpoint = s.URL
		c.Model = "fixture"
		c, _ = Normalize(c)
		out, e := (HTTPAdapter{Client: s.Client()}).Propose(context.Background(), Request{ContractVersion: "1", Operation: "query-assistance", Model: c.Model, Prompt: "list"}, c)
		if (raw == query) != (e == nil) {
			t.Fatal(raw, out, e)
		}
		s.Close()
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/slow" {
			select {
			case <-r.Context().Done():
			case <-time.After(100 * time.Millisecond):
			}
			return
		}
		http.Redirect(w, r, "/slow", http.StatusFound)
	}))
	defer s.Close()
	c := DefaultConfig()
	c.Enabled = true
	c.Endpoint = s.URL
	c.Model = "fixture"
	c.Limits.TimeoutMS = 20
	c, _ = Normalize(c)
	if _, e := (HTTPAdapter{Client: s.Client()}).Propose(context.Background(), Request{}, c); e == nil {
		t.Fatal("redirect followed")
	}
	c.Endpoint = s.URL + "/slow"
	start := time.Now()
	if _, e := (HTTPAdapter{Client: s.Client()}).Propose(context.Background(), Request{}, c); e == nil || time.Since(start) > time.Second {
		t.Fatal(e)
	}
	t.Setenv("CI", "true")
	if _, e := (HTTPAdapter{}).Propose(context.Background(), Request{}, c); e == nil || e.(*contracts.Error).Code != "engine_ci_forbidden" {
		t.Fatal(e)
	}
}

type testSecrets struct {
	calls int
	value []byte
}

func (s *testSecrets) Resolve(context.Context, string) ([]byte, error) {
	s.calls++
	return s.value, nil
}
func TestCredentialAndOutputBounds(t *testing.T) {
	secret := &testSecrets{value: []byte("fixture-token")}
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") != "Bearer fixture-token" {
			t.Error("credential not applied")
		}
		w.WriteHeader(401)
		w.Write([]byte("private provider body fixture-token"))
	}))
	defer srv.Close()
	c := DefaultConfig()
	c.Enabled = true
	c.Endpoint = srv.URL
	c.Model = "fixture"
	c.CredentialID = contracts.ID()
	_, e := (HTTPAdapter{Client: srv.Client(), Secrets: secret}).Propose(context.Background(), Request{}, c)
	if e == nil || strings.Contains(e.Error(), "fixture-token") || strings.Contains(e.Error(), "private provider") {
		t.Fatal("provider body leaked", e)
	}
	if secret.calls != 1 || requests != 1 || string(secret.value) != "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00" {
		t.Fatal("credential lifecycle")
	}
	secret.value = []byte("fixture-token")
	t.Setenv("CI", "true")
	_, e = (HTTPAdapter{Secrets: secret}).Propose(context.Background(), Request{}, c)
	if e == nil || secret.calls != 1 || requests != 1 {
		t.Fatal("CI contacted provider or credentials")
	}
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte(strings.Repeat("x", 1025)))
	}))
	defer srv2.Close()
	c.CredentialID = ""
	c.Endpoint = srv2.URL
	c.Limits.MaxResponseBytes = 1024
	_, e = (HTTPAdapter{Client: srv2.Client()}).Propose(context.Background(), Request{}, c)
	if e == nil || e.(*contracts.Error).Code != "output_limit" {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(DefaultConfig())
	if strings.Contains(string(raw), "fixture-token") {
		t.Fatal("secret persisted")
	}
}

func (s *testSecrets) Status(context.Context, string) (string, error) { return "available", nil }
