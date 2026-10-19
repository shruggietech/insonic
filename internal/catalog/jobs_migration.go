// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

// finishMigration applies the duration correction to every recognized older
// catalog, then advances both markers only after all copied data is verified.
func (s *Store) finishMigration(ctx context.Context, tx *sql.Tx) error {
	if err := s.migrateJobDuration(ctx, tx); err != nil {
		return sanitize(err)
	}
	if _, err := s.exec(ctx, tx, "UPDATE catalog_schema SET version=?,digest=? WHERE singleton=1", SchemaVersion, migrationDigest()); err != nil {
		return sanitize(err)
	}
	if _, err := s.exec(ctx, tx, "UPDATE workspace SET schema_version=?", SchemaVersion); err != nil {
		return sanitize(err)
	}
	return sanitize(tx.Commit())
}

func (s *Store) migrateJobDuration(ctx context.Context, tx *sql.Tx) error {
	if s.backend == "postgresql" {
		// The constraint name is PostgreSQL's name for the published inline CHECK.
		// Altering it preserves the job table and every referencing key/index.
		for _, q := range []string{"ALTER TABLE job DROP CONSTRAINT job_duration_ms_check", "ALTER TABLE job ADD CONSTRAINT job_duration_ms_check CHECK(duration_ms BETWEEN 1 AND 600000)"} {
			if _, err := tx.ExecContext(ctx, q); err != nil {
				return err
			}
		}
		return nil
	}
	// migrate already disabled foreign keys on this connection before BEGIN.
	// Build the replacement first: renaming the old table would rewrite child
	// references to the temporary name, including retained training lineage.
	ddl := strings.Replace(jobDDL, "CREATE TABLE IF NOT EXISTS job (", "CREATE TABLE job_duration_new (", 1)
	columns := "workspace_id,id,operation_id,duration_ms,generation,state"
	for _, q := range []string{ddl, "INSERT INTO job_duration_new(" + columns + ") SELECT " + columns + " FROM job", "DROP TABLE job", "ALTER TABLE job_duration_new RENAME TO job"} {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	rows, err := tx.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		return err
	}
	defer rows.Close()
	if rows.Next() {
		return contracts.Fail("invalid_request")
	}
	return rows.Err()
}
