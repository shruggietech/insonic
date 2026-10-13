// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArcadePreflightNoProvisioningOrSecretEcho(t *testing.T) {
	paths := []string{}
	mode := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		user, password, ok := r.BasicAuth()
		if !ok || user != "fixture" || password != "not-for-output" {
			t.Error("missing protected auth")
		}
		if mode == 1 {
			http.Error(w, "not-for-output", 401)
			return
		}
		if mode == 2 {
			http.Redirect(w, r, "/should-not-follow", 302)
			return
		}
		w.Write([]byte("{\"result\":true}"))
	}))
	defer server.Close()
	id := contracts.ID()
	raw, _ := json.Marshal(map[string]any{"display_name": "Fixture", "profiles": map[string]any{"graph": map[string]any{"adapter_id": "arcadedb", "configuration": map[string]any{"endpoint": server.URL, "database": "fixture", "credential_id": id}}}})
	c, e := Build(raw)
	if e != nil {
		t.Fatal(e)
	}
	input := &credentialcmd.Input{Session: map[string][]byte{id: []byte("{\"username\":\"fixture\",\"password\":\"not-for-output\"}")}}
	defer input.Close()
	for mode = 0; mode < 3; mode++ {
		report, e := Preflight(context.Background(), t.TempDir(), c, Transient{Input: input})
		if e != nil {
			t.Fatal(e)
		}
		out, _ := json.Marshal(report)
		if strings.Contains(string(out), "not-for-output") {
			t.Fatal("leaked credential")
		}
		if mode == 0 && report.Roles[2].Dimensions["authentication"] != "database-access-observed" {
			t.Fatal(report)
		}
		if mode > 0 && report.Roles[2].Dimensions["reachability"] != "failed" {
			t.Fatal(report)
		}
	}
	for _, p := range paths {
		if p != "GET /api/v1/exists/fixture" {
			t.Fatal("unexpected request", p)
		}
	}
}
