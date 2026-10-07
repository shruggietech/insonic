// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"os"
	"testing"
	"time"
)

func TestServiceJourney(t *testing.T) {
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "test")
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	s, e := NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	s.Grace = 0
	source := bytesFile(t, []byte("source bytes 音声"))
	op := contracts.ID()
	p, e := s.Publish(ctx, op, source.Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.Publish(ctx, op, source.Name(), "other")
	if e != nil || again.ID != p.ID {
		t.Fatalf("replay %+v %v", again, e)
	}
	if e = s.Verify(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	m, e := s.Materialize(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(m.Path)
	if e != nil || string(b) != "source bytes 音声" {
		t.Fatalf("materialize %q %v", b, e)
	}
	if _, e = s.Retire(ctx, p.ID); e == nil {
		t.Fatal("live lease retired")
	}
	if e = s.Release(ctx, p.ID, m.Lease.ID); e != nil {
		t.Fatal(e)
	}
	p, e = s.Retire(ctx, p.ID)
	if e != nil || p.State != "retired" {
		t.Fatalf("retire %+v %v", p, e)
	}
	if e = s.Verify(ctx, p.ID); e == nil {
		t.Fatal("retired usable")
	}
	time.Sleep(time.Millisecond)
}
