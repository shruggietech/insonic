// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"testing"
)

func TestNewPostgresConnectionResolvesReplacedCredentials(t *testing.T) {
	raw, _ := json.Marshal(Credentials{Username: "fixture-first", Password: "fixture-one"})
	provider := SessionSecrets{"fixture": raw}
	hook := referencedPostgresAuthentication(provider, "fixture")
	cfg := &pgx.ConnConfig{}
	if e := hook(context.Background(), cfg); e != nil || cfg.User != "fixture-first" || cfg.Password != "fixture-one" {
		t.Fatal("initial reference unusable")
	}
	raw, _ = json.Marshal(Credentials{Username: "fixture-replaced", Password: "fixture-two"})
	provider["fixture"] = raw
	if e := hook(context.Background(), cfg); e != nil || cfg.User != "fixture-replaced" || cfg.Password != "fixture-two" {
		t.Fatal("new connection retained old authentication")
	}
	delete(provider, "fixture")
	if e := hook(context.Background(), cfg); e == nil {
		t.Fatal("deleted reference silently reused")
	}
}
