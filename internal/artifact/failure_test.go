// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testService(t *testing.T) *Service {
	t.Helper()
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "fixture")
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	s, e := NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	s.Grace = 0
	t.Cleanup(func() { s.Close(); db.Close() })
	return s
}

type lostReply struct{ Store }

func (s lostReply) PublishImmutable(ctx context.Context, p catalog.Publication, f *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	p, e := s.Store.PublishImmutable(ctx, p, f, save)
	if e == nil {
		e = contracts.Fail("unavailable")
	}
	return p, e
}
func TestUnknownPublicationReconciles(t *testing.T) {
	ctx := context.Background()
	s := testService(t)
	store := s.Store
	s.Store = lostReply{store}
	source := bytesFile(t, []byte("verified but response lost"))
	id := contracts.ID()
	if _, e := s.Publish(ctx, id, source.Name(), "other"); e == nil {
		t.Fatal("fault not injected")
	}
	p, e := s.Catalog.Publication(ctx, id)
	if e != nil || p.State != "pending" {
		t.Fatalf("unknown admitted: %+v %v", p, e)
	}
	snapshot, e := s.Catalog.Export(ctx)
	if e != nil || len(snapshot.Records.Artifacts) != 0 {
		t.Fatal("bytes not verified before references")
	}
	s.Store = store
	if _, e = s.Abort(ctx, id); e == nil {
		t.Fatal("unknown completion abandoned")
	}
	if p, e = s.Catalog.Publication(ctx, id); e != nil || p.State != "pending" {
		t.Fatal("completed bytes lost their recovery journal")
	}
	p, e = s.Reconcile(ctx, id)
	if e != nil || p.State != "available" {
		t.Fatalf("reconcile %+v %v", p, e)
	}
	changed := bytesFile(t, []byte("different"))
	if _, e = s.Publish(ctx, id, changed.Name(), "other"); e == nil {
		t.Fatal("changed replay accepted")
	}
}
func TestCancelledAndLargeStreaming(t *testing.T) {
	s := testService(t)
	source := bytesFile(t, nil)
	if e := source.Truncate(65 << 20); e != nil {
		t.Fatal(e)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	id := contracts.ID()
	if _, e := s.Publish(cancelled, id, source.Name(), "other"); e == nil {
		t.Fatal("cancelled publication accepted")
	}
	p, e := s.Publish(context.Background(), contracts.ID(), source.Name(), "other")
	if e != nil || p.Size != 65<<20 {
		t.Fatalf("large %+v %v", p, e)
	}
	s.MaxMaterialization = 64 << 20
	if _, e = s.Materialize(context.Background(), p.ID); e == nil {
		t.Fatal("cache bound ignored")
	}
	if _, e = s.Retire(context.Background(), p.ID); e != nil {
		t.Fatal(e)
	}
}

type slowStore struct {
	Store
	delay time.Duration
}
type slowBody struct {
	io.ReadCloser
	delay time.Duration
}

func (s slowBody) Read(b []byte) (int, error) { time.Sleep(s.delay); return s.ReadCloser.Read(b) }
func (s slowStore) OpenRange(ctx context.Context, p catalog.Publication, o, n int64) (io.ReadCloser, error) {
	body, e := s.Store.OpenRange(ctx, p, o, n)
	if e != nil {
		return nil, e
	}
	return slowBody{body, s.delay}, nil
}
func TestMaterializationRenewsDuringRead(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	source := bytesFile(t, []byte("slow read"))
	p, e := s.Publish(ctx, contracts.ID(), source.Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	s.TTL = 3 * time.Second
	s.Store = slowStore{s.Store, 1600 * time.Millisecond}
	m, e := s.Materialize(ctx, p.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Retire(ctx, p.ID); e == nil {
		t.Fatal("renewed cache lease retired")
	}
	if e = s.Release(ctx, p.ID, m.Lease.ID); e != nil {
		t.Fatal(e)
	}
}

func TestReconciliationRenewsDuringRead(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	store := s.Store
	s.Store = lostReply{store}
	source := bytesFile(t, []byte("slow recovery read"))
	id := contracts.ID()
	if _, e := s.Publish(ctx, id, source.Name(), "other"); e == nil {
		t.Fatal("fault not injected")
	}
	s.TTL = 3 * time.Second
	s.Store = slowStore{store, 1600 * time.Millisecond}
	if p, e := s.Reconcile(ctx, id); e != nil || p.State != "available" {
		t.Fatalf("recovery %+v %v", p, e)
	}
	if _, e := os.Stat(filepath.Join(s.scratchPath, id+".source")); !os.IsNotExist(e) {
		t.Fatal("recovery source retained after admission")
	}
}

type blockedStore struct {
	Store
	started chan struct{}
	stopped chan struct{}
}

func (s blockedStore) PublishImmutable(ctx context.Context, p catalog.Publication, source *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	close(s.started)
	<-ctx.Done()
	close(s.stopped)
	return p, contracts.Fail("cancelled")
}
func (s blockedStore) Abort(ctx context.Context, p catalog.Publication) error {
	select {
	case <-s.stopped:
		return nil
	default:
		return contracts.Fail("conflict")
	}
}
func TestAbortWaitsForActivePublication(t *testing.T) {
	s := testService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	blocked := blockedStore{s.Store, make(chan struct{}), make(chan struct{})}
	s.Store = blocked
	source := bytesFile(t, []byte("active publication"))
	id := contracts.ID()
	done := make(chan error, 1)
	go func() { _, e := s.Publish(ctx, id, source.Name(), "other"); done <- e }()
	select {
	case <-blocked.started:
	case <-ctx.Done():
		t.Fatal("publisher did not start")
	}
	if p, e := s.Abort(ctx, id); e != nil || p.State != "aborted" {
		t.Fatalf("abort %+v %v", p, e)
	}
	if e := <-done; e == nil {
		t.Fatal("cancelled publisher succeeded")
	}
}
