// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"testing"
	"time"
)

func TestExpiredCacheCleanupFencesRenewal(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	source := bytesFile(t, []byte("cached bytes"))
	p, e := s.Publish(ctx, contracts.ID(), source.Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	m, e := s.Materialize(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if ids, e := s.Prune(ctx, p.ID); e != nil || len(ids) != 0 {
		t.Fatal("pruned live cache")
	}
	// Complete the real cache path under a normal lease before deliberately
	// shortening its authority. Disk/runner speed must not race setup itself.
	if _, e = s.Catalog.ArtifactLease(ctx, p.ID, m.Lease.ID, m.Lease.Owner, time.Millisecond, false); e != nil {
		t.Fatal(e)
	}
	time.Sleep(5 * time.Millisecond)
	ids, e := s.Prune(ctx, p.ID)
	if e != nil || len(ids) != 1 {
		t.Fatalf("prune %v %v", ids, e)
	}
	if _, e = os.Stat(m.Path); !os.IsNotExist(e) {
		t.Fatal("cache remains")
	}
	if _, e = s.Renew(ctx, p.ID, m.Lease.ID); e == nil {
		t.Fatal("stale cache revived")
	}
}
