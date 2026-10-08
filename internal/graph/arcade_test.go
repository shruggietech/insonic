//go:build integration

package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestRealArcadeOrderedReferences(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_ARCADE")
	if endpoint == "" {
		t.Skip("explicit fixture endpoint required")
	}
	db := fmt.Sprintf("insonicgraph%d", time.Now().UnixNano())
	client := &http.Client{Timeout: 30 * time.Second}
	call := func(command string) {
		raw, _ := json.Marshal(map[string]any{"command": command})
		r, _ := http.NewRequest("POST", endpoint+"/api/v1/server", bytes.NewReader(raw))
		r.SetBasicAuth("root", "fixture-only-password")
		r.Header.Set("Content-Type", "application/json")
		res, e := client.Do(r)
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
	id := contracts.ID()
	a, e := OpenArcade(ArcadeConfig{Endpoint: endpoint, Database: db, CredentialID: id}, catalog.SessionSecrets{id: json.RawMessage(`{"username":"root","password":"fixture-only-password"}`)})
	if e != nil {
		t.Fatal(e)
	}
	qualifyAdapter(t, a)
	_ = context.Background()
}
