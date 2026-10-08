// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"context"
	"database/sql"
	"github.com/shruggietech/insonic/internal/contracts"
	"math"
	"time"
)

func (s *Store) AdvanceOutboxGeneration(ctx context.Context, owner string, minimum int64, ttl time.Duration) error {
	if !contracts.ValidID(owner) || minimum < 1 || minimum == math.MaxInt64 {
		return contracts.Fail("invalid_request")
	}
	return s.write(ctx, func(tx *sql.Tx, _ int64) error {
		now, e := s.now(ctx, tx)
		if e != nil {
			return e
		}
		until, e := expiry(now, ttl)
		if e != nil {
			return e
		}
		var generation, expires int64
		var current sql.NullString
		if e = s.row(ctx, tx, "SELECT generation,lease_until,owner_id FROM graph_target WHERE workspace_id=? AND id=?", s.workspace, s.GraphTarget()).Scan(&generation, &expires, &current); e != nil {
			return e
		}
		if expires > now && current.String != owner {
			return contracts.Fail("conflict")
		}
		if generation == math.MaxInt64 {
			return contracts.Fail("conflict")
		}
		generation = max(generation+1, minimum)
		_, e = s.exec(ctx, tx, "UPDATE graph_target SET generation=?,owner_id=?,lease_until=? WHERE workspace_id=? AND id=?", generation, owner, until, s.workspace, s.GraphTarget())
		return e
	})
}
