//go:build integration

// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestPostgreSQLTransferFence(t *testing.T) { transferSuite(t, postgresStore(t, contracts.ID())) }
func TestPostgreSQLPortableAuthority(t *testing.T) {
	portableAuthoritySuite(t, postgresStore(t, contracts.ID()), func(id string) *Store { return localStore(t, id) })
}
func TestSQLiteToPostgreSQLPortableAuthority(t *testing.T) {
	portableAuthoritySuite(t, localStore(t, contracts.ID()), func(id string) *Store { return postgresStore(t, id) })
}
