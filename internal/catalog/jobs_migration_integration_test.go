//go:build integration

// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestPostgreSQLPublishedV9DurationMigration(t *testing.T) {
	durationMigrationSuite(t, postgresStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}
func TestSQLiteLongDurationPostgreSQLRestore(t *testing.T) {
	durationBoundarySuite(t, localStore(t, contracts.ID()), func(id string) *Store { return postgresStore(t, id) })
}
