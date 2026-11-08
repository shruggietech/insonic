package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"path/filepath"
	"testing"
)

func TestPortableDDL(t *testing.T) {
	db, e := sql.Open("sqlite3", ":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, q := range migrationStatements() {
		if _, e = db.ExecContext(context.Background(), q); e != nil {
			t.Fatalf("DDL %s: %v", q, e)
		}
	}
}
func TestNewerMigrationFailsWithoutChanges(t *testing.T) {
	newerMigrationSuite(t, localStore)
}
func newerMigrationSuite(t *testing.T, create func(*testing.T, string) *Store) {
	s := create(t, contracts.ID())
	if _, err := s.PutNamedSetting(context.Background(), contracts.ID(), "accepted", 0, json.RawMessage(`{"text":"retained"}`)); err != nil {
		t.Fatal(err)
	}
	before, revision, err := s.NamedSetting(context.Background(), "accepted")
	if err != nil {
		t.Fatal(err)
	}
	if _, e := s.db.Exec("UPDATE catalog_schema SET version=99"); e != nil {
		t.Fatal(e)
	}
	for _, open := range []func(context.Context) error{s.checkVersion, s.migrate} {
		e := contracts.SafeError(open(context.Background()))
		if e.Code != "incompatible_version" || e.Reason != "catalog_schema_incompatible" {
			t.Fatal("newer catalog classification lost", e)
		}
		after, gotRevision, err := s.NamedSetting(context.Background(), "accepted")
		if err != nil || string(after) != string(before) || gotRevision != revision {
			t.Fatal("accepted authority changed", err)
		}
	}
	var version int
	if e := s.db.QueryRow("SELECT version FROM catalog_schema WHERE singleton=1").Scan(&version); e != nil || version != 99 {
		t.Fatal("version overwritten", e)
	}
}
func TestSQLiteFileURL(t *testing.T) {
	p := filepath.Join(t.TempDir(), "test.sqlite")
	u := sqliteURL(p)
	u.RawQuery = "_foreign_keys=on&_txlock=immediate&_journal_mode=WAL&_synchronous=FULL"
	db, e := sql.Open("sqlite3", u.String())
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.Ping(); e != nil {
		t.Fatal(u.String(), e)
	}
}

func TestHistoricalV3Frozen(t *testing.T) {
	if historicalV3Digest() != historicalV3ExpectedDigest {
		t.Fatal("historical schema3 identity changed")
	}
}
