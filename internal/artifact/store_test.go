// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"testing"
)

func exerciseStore(t *testing.T, s Store) {
	t.Helper()
	ctx := context.Background()
	for _, data := range [][]byte{{}, []byte("音声\x00data"), bytes.Repeat([]byte("multipart"), 2<<20)} {
		h := sha256.Sum256(data)
		p := catalog.Publication{ID: contracts.ID(), Key: "objects/" + contracts.ID(), Digest: hex.EncodeToString(h[:]), Size: int64(len(data)), State: "pending"}
		f, e := os.CreateTemp(t.TempDir(), "stage")
		if e != nil {
			t.Fatal(e)
		}
		f.Write(data)
		f.Sync()
		f.Seek(0, 0)
		p, e = s.PublishImmutable(ctx, p, f, func(next catalog.Publication) (catalog.Publication, error) { return next, nil })
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
		if e = Verify(ctx, s, p); e != nil {
			t.Fatal(e)
		}
		if e = s.Abort(ctx, p); e == nil {
			t.Fatal("existing final object abandoned by abort")
		}
		if p.Size > 3 {
			r, e := s.OpenRange(ctx, p, 1, 3)
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(r)
			r.Close()
			if e != nil || !bytes.Equal(b, data[1:4]) {
				t.Fatalf("range %q %v", b, e)
			}
		}
		if _, e = s.OpenRange(ctx, p, -1, 1); e == nil {
			t.Fatal("invalid range")
		}
		changed := p
		changed.Size = 7
		ch := sha256.Sum256([]byte("changed"))
		changed.Digest = hex.EncodeToString(ch[:])
		if _, e = s.PublishImmutable(ctx, changed, bytesFile(t, []byte("changed")), func(next catalog.Publication) (catalog.Publication, error) { return next, nil }); e == nil {
			t.Fatal("overwrite allowed")
		}
		p.State = "retiring"
		if e = s.DeleteUnreferenced(ctx, p); e != nil {
			t.Fatal(e)
		}
		if _, e = s.Stat(ctx, p); e == nil {
			t.Fatal("object remains")
		}
	}
}
func bytesFile(t *testing.T, b []byte) *os.File {
	t.Helper()
	f, e := os.CreateTemp(t.TempDir(), "source")
	if e != nil {
		t.Fatal(e)
	}
	f.Write(b)
	f.Seek(0, 0)
	t.Cleanup(func() { f.Close() })
	return f
}
func TestFilesystemContract(t *testing.T) {
	s, e := NewFilesystem(t.TempDir() + "/objects")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	exerciseStore(t, s)
}
func TestFilesystemConfinement(t *testing.T) {
	s, e := NewFilesystem(t.TempDir() + "/objects")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for _, key := range []string{"../escape", "/abs", "a\\b", "CON", "a/../b", "a:stream", "a./b"} {
		if _, e = s.Stat(context.Background(), catalog.Publication{Key: key}); e == nil {
			t.Fatalf("key admitted %s", key)
		}
	}
}
