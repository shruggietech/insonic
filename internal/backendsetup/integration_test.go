//go:build integration

// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/credentialcmd"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestPostgresInitialCredentialBootstrapAndReadOnlyPreflight(t *testing.T) {
	endpoint := os.Getenv("INSONIC_FIXTURE_POSTGRES")
	if endpoint == "" {
		t.Skip("explicit PostgreSQL fixture required")
	}
	u, e := url.Parse(endpoint)
	if e != nil {
		t.Fatal(e)
	}
	port := 5432
	if u.Port() != "" {
		port, _ = strconv.Atoi(u.Port())
	}
	password, _ := u.User.Password()
	schema := "s022_" + strings.ReplaceAll(contracts.ID(), "-", "")
	id := contracts.ID()
	raw, _ := json.Marshal(map[string]any{"schema_version": contracts.Version, "kind": "backend-plan", "display_name": "Fixture", "profiles": map[string]any{"catalog": map[string]any{"adapter_id": "postgresql", "configuration": map[string]any{"host": u.Hostname(), "port": port, "database": strings.TrimPrefix(u.Path, "/"), "schema": schema, "tls_mode": "local", "credential_id": id}}}})
	c, e := Build(raw)
	if e != nil {
		t.Fatal(e)
	}
	secret, _ := json.Marshal(map[string]string{"username": u.User.Username(), "password": password})
	input := &credentialcmd.Input{Session: map[string][]byte{id: secret}}
	defer input.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	root := t.TempDir()
	report, e := Preflight(ctx, root, c, Transient{Input: input})
	if e != nil || report.Roles[1].Dimensions["compatibility"] != "absent" {
		t.Fatal(report, e)
	}
	conn, e := pgx.Connect(ctx, endpoint)
	if e != nil {
		t.Fatal(e)
	}
	defer conn.Close(context.Background())
	defer conn.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
	var exists bool
	if e = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)", schema).Scan(&exists); e != nil || exists {
		t.Fatal("probe provisioned schema", e)
	}
	w, e := (Provisioner{GraphFactory: fixtureFactory}).ProvisionMode(ctx, root, c, input, "session")
	if e != nil || w.Config.Profiles.Catalog.Configuration["credential_id"] != id {
		t.Fatal(w, e)
	}
	report, e = Preflight(ctx, root, c, Transient{Input: input})
	if e != nil || report.Roles[1].Dimensions["compatibility"] != "passed" {
		t.Fatal(report, e)
	}
	var before int64
	if e = conn.QueryRow(ctx, "SELECT revision FROM "+pgx.Identifier{schema, "workspace"}.Sanitize()+" WHERE id=$1", c.WorkspaceID).Scan(&before); e != nil {
		t.Fatal(e)
	}
	if _, e = Preflight(ctx, root, c, Transient{Input: input}); e != nil {
		t.Fatal(e)
	}
	var after int64
	conn.QueryRow(ctx, "SELECT revision FROM "+pgx.Identifier{schema, "workspace"}.Sanitize()+" WHERE id=$1", c.WorkspaceID).Scan(&after)
	if before != after {
		t.Fatal("preflight changed catalog revision")
	}
}
