package catalog

import (
	"context"
	"database/sql"
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
	s := localStore(t, contracts.ID())
	if _, e := s.db.Exec("UPDATE catalog_schema SET version=99"); e != nil {
		t.Fatal(e)
	}
	if e := s.checkVersion(context.Background()); e == nil {
		t.Fatal("newer catalog opened")
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
