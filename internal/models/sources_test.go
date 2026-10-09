// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type sourceTransportFunc func(*http.Request) (*http.Response, error)

func (f sourceTransportFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func TestCatalogContentCannotAuthorizeLoopbackModelRoutes(t *testing.T) {
	cases := []struct {
		name, sourceURL, fileURL        string
		sourceLocal, fileLocal, allowed bool
	}{
		{"remote-http-opt-in", "https://catalog.example.test/models", "http://127.0.0.1:9999/weights", false, true, false},
		{"remote-source-flag", "https://catalog.example.test/models", "http://localhost:9999/weights", true, true, false},
		{"remote-https-loopback", "https://catalog.example.test/models", "https://127.0.0.1:9999/weights", false, false, false},
		{"remote-https-localhost", "https://catalog.example.test/models", "https://LOCALHOST.:9999/weights", false, false, false},
		{"remote-ipv6-loopback", "https://catalog.example.test/models", "https://[::1]:9999/weights", false, false, false},
		{"remote-ipv4-mapped-loopback", "https://catalog.example.test/models", "https://[::ffff:127.0.0.1]:9999/weights", false, false, false},
		{"remote-localhost-subdomain", "https://catalog.example.test/models", "https://models.localhost:9999/weights", false, false, false},
		{"loopback-without-owner-opt-in", "https://localhost:8443/models", "https://localhost:9999/weights", false, false, false},
		{"owner-http-loopback", "http://127.0.0.1:8888/models", "http://127.0.0.1:9999/weights", true, true, true},
		{"owner-https-loopback", "https://localhost:8443/models", "https://[::1]:9999/weights", true, false, true},
		{"ordinary-https-bundle", "https://catalog.example.test/models", "https://files.example.test/weights", false, false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, db := modelServiceFixture(t)
			ctx := context.Background()
			m := fixtureManifest()
			m.Files[0].URL = tc.fileURL
			m.Files[0].LocalHTTP = tc.fileLocal
			// The file flag remains valid in an owner-supplied direct manifest; it is
			// remote metadata's authority to grant that route which is restricted.
			if e := m.Validate(); e != nil {
				t.Fatal("direct manifest policy changed", e)
			}
			source, e := db.PutModelSource(ctx, contracts.ID(), 0, catalog.ModelSource{Name: "routes", State: "active", URL: tc.sourceURL, LocalHTTP: tc.sourceLocal})
			if e != nil {
				t.Fatal(e)
			}
			encoded, _ := json.Marshal(SourceCatalog{Kind: "model-catalog", Version: contracts.Version, Entries: []CatalogEntry{{Selector: "pinned", Manifest: m}}})
			var requests int
			resolver := NewResolver(db, nil)
			resolver.Client.Transport = sourceTransportFunc(func(req *http.Request) (*http.Response, error) {
				requests++
				if req.URL.String() != tc.sourceURL {
					t.Fatal("file route contacted during metadata inspection")
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(string(encoded))), Request: req}, nil
			})
			_, e = resolver.Discover(ctx, source.ID)
			if (e == nil) != tc.allowed {
				t.Fatalf("discovery route authority: allowed=%v, error=%v", tc.allowed, e)
			}
			_, e = resolver.ResolveSource(ctx, source.ID, "pinned", "transcription")
			if (e == nil) != tc.allowed {
				t.Fatalf("election route authority: allowed=%v, error=%v", tc.allowed, e)
			}
			if requests != 2 {
				t.Fatal("unexpected transport effects", requests)
			}
		})
	}
}

func TestSourceSelectorPinsCompleteManifest(t *testing.T) {
	_, db := modelServiceFixture(t)
	ctx := context.Background()
	m := fixtureManifest()
	m.Files = nil
	for _, role := range []string{"config.json", "model.bin", "tokenizer.json", "vocabulary.txt"} {
		m.Files = append(m.Files, File{Role: role, SHA256: strings.Repeat("a", 64), Size: 1, URL: "https://example.org/" + role})
	}
	doc := SourceCatalog{Kind: "model-catalog", Version: contracts.Version, Entries: []CatalogEntry{{Selector: "latest", Manifest: m}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(doc) }))
	defer server.Close()
	source, e := db.PutModelSource(ctx, contracts.ID(), 0, catalog.ModelSource{Name: "fixture", State: "active", URL: server.URL, LocalHTTP: true})
	if e != nil {
		t.Fatal(e)
	}
	resolver := NewResolver(db, nil)
	old, e := resolver.Resolve(ctx, "source:fixture/latest", "transcription")
	if e != nil || !old.Compatible || old.SourceRevision != source.Revision {
		t.Fatal("resolve", old, e)
	}
	doc.Entries[0].Manifest.ModelVersion = "2"
	doc.Entries[0].Manifest.Revision = "new-pinned"
	current, e := resolver.Resolve(ctx, "source:fixture/latest", "transcription")
	if e != nil || current.Digest == old.Digest || current.Target.ID == old.Target.ID {
		t.Fatal("mutable source", e)
	}
	if old.Manifest.ModelVersion != m.ModelVersion || old.Digest != m.Digest() {
		t.Fatal("frozen selection mutated")
	}
	doc.Entries[0].Manifest.Capabilities = []string{"voice-matching"}
	items, e := resolver.Discover(ctx, source.ID)
	if e != nil || items[0].Compatible || len(items[0].Diagnostics) == 0 {
		t.Fatal("future consumer advertised executable", e)
	}
	doc.Entries = append(doc.Entries, doc.Entries[0])
	if _, e = resolver.Discover(ctx, source.ID); e == nil {
		t.Fatal("ambiguous selector accepted")
	}
	doc.Entries = []CatalogEntry{{Selector: "bad", Manifest: m}}
	doc.Entries[0].Manifest.Files[0].URL = "https://example.org/weights?token=secret"
	if _, e = resolver.Discover(ctx, source.ID); e == nil {
		t.Fatal("credential URL accepted")
	}
}
func TestModelAliasResolutionAndRetargetIsolation(t *testing.T) {
	service, db := modelServiceFixture(t)
	ctx := context.Background()
	m := fixtureManifest()
	claim := modelWork(t, db, "models.register", m)
	if _, e := service.Execute(ctx, claim); e != nil {
		t.Fatal(e)
	}
	target, _ := json.Marshal(catalog.ModelTarget{Kind: "base", ID: InstallationID(m), Operation: "transcription"})
	alias, e := db.PutModelAlias(ctx, contracts.ID(), 0, catalog.ModelAlias{Name: "tiny", State: "active", Target: target})
	if e != nil {
		t.Fatal(e)
	}
	resolver := NewResolver(db, nil)
	out, e := resolver.Resolve(ctx, "tiny", "transcription")
	if e != nil || out.Target.ID != InstallationID(m) || out.Compatible || out.State != "registered" {
		t.Fatal("exact incompatible inspect", out, e)
	}
	if _, e = resolver.Resolve(ctx, "tiny", "diarization"); e == nil {
		t.Fatal("wrong operation")
	}
	second := m
	second.ModelVersion = "2"
	if _, e = service.Execute(ctx, modelWork(t, db, "models.register", second)); e != nil {
		t.Fatal(e)
	}
	alias.Target, _ = json.Marshal(catalog.ModelTarget{Kind: "base", ID: InstallationID(second), Operation: "transcription"})
	if _, e = db.PutModelAlias(ctx, contracts.ID(), alias.Revision, alias); e != nil {
		t.Fatal(e)
	}
	if out.Target.ID != InstallationID(m) || out.Manifest.ModelVersion != m.ModelVersion {
		t.Fatal("retarget mutated election")
	}
}
func TestSourceBoundAndSameOriginRedirect(t *testing.T) {
	_, db := modelServiceFixture(t)
	var targetCalls int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { targetCalls++ }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oversize" {
			w.Write([]byte(strings.Repeat("x", (4<<20)+1)))
			return
		}
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()
	ctx := context.Background()
	v, e := db.PutModelSource(ctx, contracts.ID(), 0, catalog.ModelSource{Name: "remote", State: "active", URL: source.URL, LocalHTTP: true})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = NewResolver(db, nil).Discover(ctx, v.ID); e == nil || targetCalls != 0 {
		t.Fatal("cross origin followed")
	}
	v.URL = source.URL + "/oversize"
	v, e = db.PutModelSource(ctx, contracts.ID(), v.Revision, v)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = NewResolver(db, nil).Discover(ctx, v.ID); e == nil {
		t.Fatal("unbounded catalog")
	}
}
func TestSourceDiscoveryConsidersAllDeclaredOperations(t *testing.T) {
	_, db := modelServiceFixture(t)
	ctx := context.Background()
	m := fixtureManifest()
	m.Capabilities = []string{"transcription", "diarization"}
	m.Compatibility = &Compatibility{Adapter: "pyannote", ContractVersion: "1", Architecture: "pyannote", RuntimeFormat: "torch"}
	m.Files = nil
	for _, role := range []string{"config.yaml", "embedding/pytorch_model.bin", "segmentation/pytorch_model.bin", "plda/plda.npz", "plda/xvec_transform.npz"} {
		m.Files = append(m.Files, File{Role: role, SHA256: strings.Repeat("a", 64), Size: 1, URL: "https://example.org/" + role})
	}
	doc := SourceCatalog{Kind: "model-catalog", Version: contracts.Version, Entries: []CatalogEntry{{Selector: "combined", Manifest: m}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(doc) }))
	defer server.Close()
	source, e := db.PutModelSource(ctx, contracts.ID(), 0, catalog.ModelSource{Name: "multiple", State: "active", URL: server.URL, LocalHTTP: true})
	if e != nil {
		t.Fatal(e)
	}
	resolver := NewResolver(db, nil)
	found, e := resolver.Discover(ctx, source.ID)
	if e != nil || len(found) != 1 || !found[0].Compatible {
		t.Fatal("ignored compatible later capability", e)
	}
	selected, e := resolver.Resolve(ctx, "source:multiple/combined", "transcription")
	if e != nil || selected.Compatible {
		t.Fatal("discovery granted wrong task execution", e)
	}
}
