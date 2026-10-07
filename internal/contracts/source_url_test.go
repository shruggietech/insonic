// SPDX-License-Identifier: Apache-2.0
package contracts_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/app"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestSourceQuerySelectorsAndCredentialReferences(t *testing.T) {
	good := "https://example.org/media?id=fixture&page=2&download=true"
	u, e := contracts.SourceURL(good)
	if e != nil || u.String() != good {
		t.Fatal("ordinary selector lost")
	}
	for _, bad := range []string{"https://user:password@example.org/media", "https://example.org/media?token=secret", "https://example.org/media?X-Amz-Signature=secret", "https://example.org/media?api_key=secret", "https://example.org/media#private"} {
		if _, e = contracts.SourceURL(bad); e == nil {
			t.Fatal("authentication locator accepted")
		}
	}
}

func TestSourceAuthenticationAliasesNormalizeEncodedNames(t *testing.T) {
	for _, alias := range []string{
		"jwt", "JWT", "%6a%77%74", "j-w_t", "session", "Session-Token",
		"session%5Ftoken", "auth.session.id", "OAuth_Token", "oauth2-token",
		"ID+TOKEN", "%69d%5Ftoken", "identity_token", "access.token", "authentication-token",
		"__Secure-next-auth.session-token", "PHPSESSID", "JSESSIONID", "ASP.NET_SessionId",
		"connect.sid", "sid", "oauth.code", "SAMLResponse", "saml_assertion",
	} {
		t.Run(alias, func(t *testing.T) {
			if _, e := contracts.SourceURL("https://example.org/media?" + alias + "=fixture-private-input"); e == nil {
				t.Fatal("authentication alias accepted")
			}
		})
	}
	for _, raw := range []string{
		"https://example.org/media?id=fixture&page=2&download=true",
		"https://example.org/media?format=wav&version=1&language=en&session_name=recording",
		"https://example.org/media?%69d=fixture&file_name=original&sort=asc",
	} {
		if u, e := contracts.SourceURL(raw); e != nil || u.String() != raw {
			t.Fatal("ordinary selector changed", e)
		}
	}
}

func TestCredentialURLAliasesRejectedBeforeMediaOrModelWorkPersists(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "Nonsecret source admission fixture")
	if e != nil {
		t.Fatal(e)
	}
	a, e := app.New(w)
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	for _, alias := range []string{"jwt", "session", "session%5Ftoken", "OAuth-Token", "%69d_token", "auth.session.id"} {
		locator := "https://example.org/media?" + alias + "=fixture-private-input"
		manifest := models.Manifest{Kind: "base-model-manifest", Version: contracts.Version, Name: "fixture", ModelVersion: "1", Revision: "fixture-v1", License: "unknown", Capabilities: []string{"transcription"}, Files: []models.File{{Role: "weights", SHA256: strings.Repeat("a", 64), Size: 1, URL: locator}}}
		for _, fixture := range []struct {
			operation string
			data      any
		}{
			{"media.import", library.ImportRequest{Items: []library.Item{{Source: locator}}}},
			{"models.register", models.Request{Manifest: manifest}},
			{"models.acquire", models.Request{Manifest: manifest}},
		} {
			raw, _ := json.Marshal(fixture.data)
			response := a.Dispatch(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: fixture.operation, Data: raw})
			if response.Error == nil || response.Error.Code != "invalid_request" {
				t.Fatal("credential source admitted", alias, fixture.operation, response.Error)
			}
		}
	}
	works, e := a.Catalog.Works(context.Background())
	if e != nil || len(works) != 0 {
		t.Fatal("rejected source left durable work", len(works), e)
	}
	snapshot, e := a.Catalog.Export(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := json.Marshal(snapshot)
	if strings.Contains(string(raw), "fixture-private-input") {
		t.Fatal("rejected authentication input persisted")
	}
}
