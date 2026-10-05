//go:build native_ladybug

package qualification

import (
	"database/sql"
	lbug "github.com/LadybugDB/go-ladybug"
	_ "github.com/mattn/go-sqlite3"
	"path/filepath"
	"testing"
)

func TestEmbeddedNativePair(t *testing.T) {
	dir := t.TempDir()
	sqlDB, err := sql.Open("sqlite3", filepath.Join(dir, "catalog.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if _, err := sqlDB.Exec("PRAGMA foreign_keys=ON; CREATE TABLE evidence (id INTEGER PRIMARY KEY, value TEXT NOT NULL); INSERT INTO evidence VALUES (1,'source')"); err != nil {
		t.Fatal(err)
	}
	tx, err := sqlDB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	tx.Exec("INSERT INTO evidence VALUES (2,'rollback')")
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := sqlDB.QueryRow("SELECT count(*) FROM evidence").Scan(&count); err != nil || count != 1 {
		t.Fatalf("catalog roundtrip %d %v", count, err)
	}
	config := lbug.DefaultSystemConfig()
	config.BufferPoolSize = 64 << 20
	config.MaxNumThreads = 2
	db, err := lbug.OpenDatabase(filepath.Join(dir, "graph"), config)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn, err := lbug.OpenConnection(db)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for _, query := range []string{"CREATE NODE TABLE Evidence(id INT64, value STRING, PRIMARY KEY(id))", "CREATE (:Evidence {id: 1, value: 'source'})", "BEGIN TRANSACTION", "CREATE (:Evidence {id: 2, value: 'rollback'})", "ROLLBACK"} {
		result, err := conn.Query(query)
		if err != nil {
			t.Fatal(err)
		}
		result.Close()
	}
	result, err := conn.Query("MATCH (e:Evidence) RETURN count(e)")
	if err != nil {
		t.Fatal(err)
	}
	defer result.Close()
	tuple, err := result.Next()
	if err != nil {
		t.Fatal(err)
	}
	defer tuple.Close()
	value, err := tuple.GetValue(0)
	if err != nil || value != int64(1) {
		t.Fatalf("native graph rollback: %v %v", value, err)
	}
	t.Log("real SQLite and pinned Ladybug native transaction/write/read/rollback passed")
}
