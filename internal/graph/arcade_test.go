//go:build integration

package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/qualificationtiming"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestRealArcadeOrderedReferences(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_ARCADE")
	if endpoint == "" {
		t.Skip("explicit fixture endpoint required")
	}
	db := fmt.Sprintf("insonicgraph%d", time.Now().UnixNano())
	client := &http.Client{Timeout: qualificationtiming.Duration("request_seconds")}
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
	allowArcadeFixtureRequests(a)
	qualifyAdapter(t, a)
	_ = context.Background()
}

// Functional fixture headroom is separate from the production transport contract.
func allowArcadeFixtureRequests(a Adapter) {
	a.(*adapter).engine.(*arcadeEngine).client.Timeout = qualificationtiming.Duration("request_seconds")
}

type arcadeFixtureTransport func(*http.Request) (*http.Response, error)

func (f arcadeFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestArcadeFixtureAllowanceKeepsProductionAndCallerBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		fixture bool
		parent  time.Duration
		want    string
		elapsed time.Duration
	}{
		{"production cutoff", false, 0, "unavailable", 20 * time.Second},
		{"fixture completion", true, 0, "", 21 * time.Second},
		{"earlier caller", true, time.Second, "cancelled", time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"version":"fixture"}`)
			}))
			defer server.Close()
			value, err := OpenArcade(ArcadeConfig{Endpoint: server.URL, Database: "fixture"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer value.Close()
			client := value.(*adapter).engine.(*arcadeEngine).client
			if client.Timeout != 20*time.Second {
				t.Fatal("production timeout changed")
			}
			if tc.fixture {
				allowArcadeFixtureRequests(value)
			}
			client.Transport = arcadeFixtureTransport(func(req *http.Request) (*http.Response, error) {
				select {
				case <-time.After(21 * time.Second):
					return &http.Response{StatusCode: 200, Header: http.Header{"Arcadedb-Session-Id": []string{"fixture"}},
						Body: io.NopCloser(strings.NewReader(`{"result":[]}`))}, nil
				case <-req.Context().Done():
					return nil, req.Context().Err()
				}
			})
			synctest.Test(t, func(t *testing.T) {
				ctx := context.Background()
				if tc.parent > 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, tc.parent)
					defer cancel()
				}
				started := time.Now()
				_, err := value.(*adapter).engine.Begin(ctx, false)
				if tc.want == "" && err != nil || tc.want != "" && (err == nil || contracts.SafeError(err).Code != tc.want) {
					t.Fatalf("request outcome: %v", err)
				}
				if time.Since(started) != tc.elapsed {
					t.Fatal("original boundary changed", time.Since(started))
				}
			})
		})
	}
}
