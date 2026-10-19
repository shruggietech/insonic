// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestPublishedV9DurationMigration(t *testing.T) {
	durationMigrationSuite(t, localStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}

func durationMigrationSuite(t *testing.T, s *Store, open func(string) *Store) {
	t.Helper()
	ctx := context.Background()
	if historicalV9Digest() != historicalV9ExpectedDigest {
		t.Fatal("published schema-9 DDL changed")
	}
	op, owner := contracts.ID(), contracts.ID()
	first, err := s.StartJob(ctx, op, owner, 60000, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CancelJob(ctx, contracts.ID(), first.JobID); err != nil {
		t.Fatal(err)
	}
	before, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Reconstruct the actual published CHECK with populated child rows. Merely
	// changing the schema marker would not exercise the required SQL migration.
	conn, err := s.db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s.backend == "sqlite" {
		if _, err = conn.ExecContext(ctx, "PRAGMA foreign_keys=OFF"); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.backend == "postgresql" {
		for _, q := range []string{"ALTER TABLE job DROP CONSTRAINT job_duration_ms_check", "ALTER TABLE job ADD CONSTRAINT job_duration_ms_check CHECK(duration_ms BETWEEN 1 AND 60000)"} {
			if _, err = tx.ExecContext(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
	} else {
		var ddl string
		for _, q := range historicalV9DDL {
			if strings.HasPrefix(q, "CREATE TABLE IF NOT EXISTS job (") {
				ddl = strings.Replace(q, "job (", "job_v9 (", 1)
			}
		}
		for _, q := range []string{ddl, "INSERT INTO job_v9 SELECT * FROM job", "DROP TABLE job", "ALTER TABLE job_v9 RENAME TO job"} {
			if _, err = tx.ExecContext(ctx, q); err != nil {
				t.Fatal(err)
			}
		}
	}
	if _, err = s.exec(ctx, tx, "UPDATE catalog_schema SET version=9,digest=?", historicalV9Digest()); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(ctx, "UPDATE workspace SET schema_version=9"); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if s.backend == "sqlite" {
		if _, err = conn.ExecContext(ctx, "PRAGMA foreign_keys=ON"); err != nil {
			t.Fatal(err)
		}
	}
	conn.Close()
	// Reopen the physical SQLite file, exercising persisted migration rather
	// than relabelling a newly initialized in-memory schema.
	if s.backend == "sqlite" {
		var seq int
		var name, path string
		if err = s.db.QueryRowContext(ctx, "PRAGMA database_list").Scan(&seq, &name, &path); err != nil {
			t.Fatal(err)
		}
		if err = s.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, openErr := OpenSQLite(ctx, path, s.workspace)
		if openErr != nil {
			t.Fatal(openErr)
		}
		*s = *reopened
	} else if err = s.migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if SchemaVersion != 10 {
		t.Fatal("duration correction requires forward migration")
	}
	if err = s.checkVersion(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before.Revision != after.Revision || !reflect.DeepEqual(before.State, after.State) || !reflect.DeepEqual(before.Records, after.Records) {
		t.Fatal("migration changed accepted data or receipts")
	}
	replay, err := s.StartJob(ctx, op, contracts.ID(), 60000, time.Second)
	if err != nil || replay != first {
		t.Fatalf("published receipt changed: %+v %v", replay, err)
	}
	// Historical exports keep their own schema/digest and restore unchanged.
	before.CatalogSchema = 9
	before.Digest, _ = before.digest()
	if err = open(s.workspace).Restore(ctx, before); err != nil {
		t.Fatal("schema-9 snapshot", err)
	}
	durationBoundarySuite(t, s, open)
}

func TestSQLiteDurationBoundaries(t *testing.T) {
	durationBoundarySuite(t, localStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}
func durationBoundarySuite(t *testing.T, s *Store, open func(string) *Store) {
	t.Helper()
	ctx := context.Background()
	for _, duration := range []int{1, 60000, 60001, 600000} {
		op, owner := contracts.ID(), contracts.ID()
		job, err := s.StartJob(ctx, op, owner, duration, time.Second)
		if err != nil {
			t.Fatalf("duration %d: %v", duration, err)
		}
		replay, err := s.ReconcileJob(ctx, op, "start", "", duration)
		if err != nil || replay.AttemptID != job.AttemptID || replay.DurationMS != duration {
			t.Fatalf("duration replay: %+v %v", replay, err)
		}
		if _, err = s.CancelJob(ctx, contracts.ID(), job.JobID); err != nil {
			t.Fatal(err)
		}
		retry, err := s.RetryJob(ctx, contracts.ID(), job.JobID, owner, time.Second)
		if err != nil || retry.DurationMS != duration {
			t.Fatalf("duration retry: %+v %v", retry, err)
		}
		if _, err = s.CancelJob(ctx, contracts.ID(), job.JobID); err != nil {
			t.Fatal(err)
		}
	}
	for _, duration := range []int{0, -1, 600001} {
		if _, err := s.StartJob(ctx, contracts.ID(), contracts.ID(), duration, time.Second); err == nil {
			t.Fatal("invalid duration accepted", duration)
		}
		if _, err := s.ReconcileJob(ctx, contracts.ID(), "start", "", duration); err == nil {
			t.Fatal("invalid duration reconciled", duration)
		}
		// Assert the database remains an independent integrity boundary.
		_, err := s.db.ExecContext(ctx, s.query("INSERT INTO job(workspace_id,id,operation_id,duration_ms,generation,state) VALUES(?,?,?,?,1,'cancelled')"), s.workspace, contracts.ID(), contracts.ID(), duration)
		if err == nil {
			t.Fatal("SQL accepted invalid duration", duration)
		}
	}
	snap, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(snap)
	decoded, err := ReadSnapshot(raw)
	if err != nil {
		t.Fatal(err)
	}
	restored := open(s.workspace)
	if err = restored.Restore(ctx, decoded); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = restored.db.QueryRowContext(ctx, "SELECT count(*) FROM job WHERE duration_ms=600000").Scan(&count); err != nil || count != 1 {
		t.Fatal("long duration lost on restore", count, err)
	}
}
