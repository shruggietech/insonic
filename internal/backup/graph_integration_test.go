//go:build integration

// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestRestoredArcadeAcceptedEvidence(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_ARCADE")
	if endpoint == "" {
		t.Fatal("ArcadeDB fixture required")
	}
	database := "backup" + strings.ReplaceAll(contracts.ID(), "-", "")
	client := &http.Client{Timeout: qualificationtiming.Duration("request_seconds")}
	call := func(command string) {
		raw, _ := json.Marshal(map[string]any{"command": command})
		req, _ := http.NewRequest("POST", endpoint+"/api/v1/server", bytes.NewReader(raw))
		req.SetBasicAuth("root", "fixture-only-password")
		req.Header.Set("Content-Type", "application/json")
		r, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != http.StatusOK {
			t.Fatal(r.StatusCode)
		}
	}
	call("create database " + database)
	defer call("drop database " + database)
	id := contracts.ID()
	provider := catalog.SessionSecrets{id: json.RawMessage(`{"username":"root","password":"fixture-only-password"}`)}
	g, e := graph.OpenArcade(graph.ArcadeConfig{Endpoint: endpoint, Database: database, CredentialID: id}, provider)
	if e != nil {
		t.Fatal(e)
	}
	restoredGraphSuite(t, g)
}
