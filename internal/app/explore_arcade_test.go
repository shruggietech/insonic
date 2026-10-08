//go:build integration

// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/graph"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestExploreRealArcadeCurrentQueries(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_ARCADE")
	if endpoint == "" {
		t.Skip("explicit fixture required")
	}
	db := fmt.Sprintf("insonicexplore%d", time.Now().UnixNano())
	client := &http.Client{Timeout: 30 * time.Second}
	call := func(command string) {
		raw, _ := json.Marshal(map[string]any{"command": command})
		req, _ := http.NewRequest("POST", endpoint+"/api/v1/server", bytes.NewReader(raw))
		req.SetBasicAuth("root", "fixture-only-password")
		req.Header.Set("Content-Type", "application/json")
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatal(res.StatusCode)
		}
	}
	call("create database " + db)
	defer call("drop database " + db)
	a := configuredApp(t)
	id := contracts.ID()
	g, e := graph.OpenArcade(graph.ArcadeConfig{Endpoint: endpoint, Database: db, CredentialID: id}, catalog.SessionSecrets{id: json.RawMessage(`{"username":"root","password":"fixture-only-password"}`)})
	if e != nil {
		t.Fatal(e)
	}
	defer a.Close()
	exploreBackendSuite(t, a, g)
}
