// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func sharedStoreIsolation(t *testing.T, firstStore, secondStore Store) {
	t.Helper()
	a, b := testService(t), testService(t)
	a.Store.Close()
	b.Store.Close()
	a.Store = firstStore
	b.Store = secondStore
	ctx := context.Background()
	id := contracts.ID()
	source := bytesFile(t, []byte("same content and request ID"))
	first, e := a.Publish(ctx, id, source.Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	second, e := b.Publish(ctx, id, source.Name(), "other")
	if e != nil {
		t.Fatal(e)
	}
	if first.Key == second.Key || !strings.Contains(first.Key, a.Workspace.Config.WorkspaceID) || !strings.Contains(second.Key, b.Workspace.Config.WorkspaceID) {
		t.Fatal("shared store escaped workspace key scope")
	}
	if _, e = a.Retire(ctx, id); e != nil {
		t.Fatal(e)
	}
	if e = b.Verify(ctx, id); e != nil {
		t.Fatal("retirement deleted another workspace object", e)
	}
	if _, e = b.Retire(ctx, id); e != nil {
		t.Fatal(e)
	}
}

func TestSharedFilesystemWorkspaceIsolation(t *testing.T) {
	location := t.TempDir() + "/shared"
	a, e := NewFilesystem(location)
	if e != nil {
		t.Fatal(e)
	}
	b, e := NewFilesystem(location)
	if e != nil {
		t.Fatal(e)
	}
	sharedStoreIsolation(t, a, b)
}

func TestPublicationRequiresFinalMetadataFlush(t *testing.T) {
	s := testService(t)
	fs := s.Store.(*Filesystem)
	fs.persist = func(*os.Root, string) error { return errors.New("injected metadata flush failure") }
	id := contracts.ID()
	if _, e := s.Publish(context.Background(), id, bytesFile(t, []byte("durability")).Name(), "other"); e == nil {
		t.Fatal("admitted failed metadata flush")
	}
	if _, e := s.Reconcile(context.Background(), id); e == nil {
		t.Fatal("recovery bypassed failed metadata flush")
	}
	snapshot, e := s.Catalog.Export(context.Background())
	if e != nil || len(snapshot.Records.Artifacts) != 0 {
		t.Fatal("available catalog evidence preceded durability")
	}
	fs.persist = persistObject
	if p, e := s.Reconcile(context.Background(), id); e != nil || p.State != "available" {
		t.Fatalf("flush recovery %+v %v", p, e)
	}
}
