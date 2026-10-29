// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

type failingStageWriter struct {
	output  io.Writer
	failure error
}

func (f failingStageWriter) Write(p []byte) (int, error) {
	n := len(p) / 2
	if n == 0 {
		return 0, f.failure
	}
	written, e := f.output.Write(p[:n])
	if e != nil {
		return written, e
	}
	return written, f.failure
}

func TestStorageCapacityFailureIsActionable(t *testing.T) {
	for _, e := range []error{syscall.ENOSPC, &os.PathError{Op: "write", Path: "private-owner-secret", Err: syscall.ENOSPC}, fmt.Errorf("private-owner-secret: %w", syscall.ENOSPC)} {
		got := contracts.SafeError(redact(context.Background(), e))
		if got.Code != "insufficient_space" || strings.Contains(got.Message, "private-owner-secret") {
			t.Fatalf("capacity cause lost or leaked: %+v", got)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got := contracts.SafeError(redact(cancelled, syscall.ENOSPC)); got.Code != "cancelled" {
		t.Fatal(got)
	}
	original := contracts.AssetFailure("asset_hash", "artifact", "object")
	if got := contracts.SafeError(redact(context.Background(), original)); got.Reason != "asset_hash" {
		t.Fatal(got)
	}
}

func TestStablePartialWritesNeverClaimCompleteSnapshot(t *testing.T) {
	data := bytes.Repeat([]byte("bounded original bytes"), 100)
	for _, failure := range []error{nil, syscall.ENOSPC} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			source := bytesFile(t, data)
			output := bytesFile(t, nil)
			digest, size, e := copyStableBound(context.Background(), source.Name(), source, failingStageWriter{output, failure}, int64(len(data)))
			if e == nil || digest != "" || size != 0 {
				t.Fatalf("partial stage declared complete: %s %d %v", digest, size, e)
			}
			expected := "unavailable"
			if failure != nil {
				expected = "insufficient_space"
			}
			if got := contracts.SafeError(e); got.Code != expected {
				t.Fatalf("wrong partial-write cause: %+v", got)
			}
			st, e := output.Stat()
			if e != nil || st.Size() <= 0 || st.Size() >= int64(len(data)) {
				t.Fatal("partial write boundary not reached", st, e)
			}
			original, e := os.ReadFile(source.Name())
			if e != nil || !bytes.Equal(original, data) {
				t.Fatal("owner source changed", e)
			}
		})
	}
}

type stageFinalizationFault struct {
	*os.File
	syncErr, seekErr, closeErr error
	closed                     bool
}

func (s *stageFinalizationFault) Sync() error {
	if s.syncErr != nil {
		return s.syncErr
	}
	return s.File.Sync()
}
func (s *stageFinalizationFault) Seek(offset int64, whence int) (int64, error) {
	if s.seekErr != nil {
		return 0, s.seekErr
	}
	return s.File.Seek(offset, whence)
}
func (s *stageFinalizationFault) Close() error {
	s.closed = true
	e := s.File.Close()
	if s.closeErr != nil {
		return s.closeErr
	}
	return e
}

func TestStageFinalizationRequiresFlushRewindAndObservedCleanup(t *testing.T) {
	for _, failure := range []string{"flush", "rewind", "cleanup", "ready"} {
		t.Run(failure, func(t *testing.T) {
			data := []byte("original small snapshot")
			source := bytesFile(t, data)
			output := bytesFile(t, nil)
			stage := &stageFinalizationFault{File: output}
			if failure == "flush" {
				stage.syncErr = syscall.ENOSPC
			}
			if failure == "rewind" {
				stage.seekErr = os.ErrInvalid
			}
			cleanupCause := fmt.Errorf("private-cleanup-path: %w", os.ErrPermission)
			if failure == "cleanup" {
				stage.syncErr = syscall.ENOSPC
				stage.closeErr = cleanupCause
			}
			removed := false
			discard := func() error {
				removed = true
				if failure == "cleanup" {
					return cleanupCause
				}
				return os.Remove(output.Name())
			}
			digest, size, e := prepareStableStage(context.Background(), source.Name(), source, stage, 1024, discard)
			if failure == "ready" {
				if e != nil || digest == "" || size != int64(len(data)) || removed || stage.closed {
					t.Fatal("healthy snapshot not retained", e)
				}
				got, e := io.ReadAll(output)
				if e != nil || !bytes.Equal(got, data) {
					t.Fatal("ready stage not rewound", e)
				}
				output.Close()
				os.Remove(output.Name())
				return
			}
			if e == nil || digest != "" || size != 0 || !removed || !stage.closed {
				t.Fatal("failed stage readiness/cleanup lost", e)
			}
			if failure == "cleanup" {
				if !errors.Is(e, os.ErrPermission) {
					t.Fatal("cleanup cause discarded", e)
				}
				if _, err := os.Stat(output.Name()); err != nil {
					t.Fatal("fixture did not retain failed cleanup bytes", err)
				}
				if strings.Contains(contracts.SafeError(redact(context.Background(), e)).Message, "private-cleanup-path") {
					t.Fatal("cleanup path leaked")
				}
				os.Remove(output.Name())
			} else if _, err := os.Stat(output.Name()); !os.IsNotExist(err) {
				t.Fatal("failed incomplete stage remains", err)
			}
			original, err := os.ReadFile(source.Name())
			if err != nil || !bytes.Equal(original, data) {
				t.Fatal("source changed", err)
			}
		})
	}
}

func TestStorageCapacityPersistenceKeepsUncertainRecoveryAuthority(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	fs := s.Store.(*routedStore).defaultStore.(*Filesystem)
	fs.persist = func(*os.Root, string) error {
		return &os.PathError{Op: "flush", Path: "private-storage-path", Err: syscall.ENOSPC}
	}
	source := bytesFile(t, []byte("completed durable bytes"))
	id := contracts.ID()
	if _, e := s.Publish(ctx, id, source.Name(), "other"); contracts.SafeError(e).Code != "insufficient_space" {
		t.Fatal("flush capacity cause lost", e)
	}
	pending, e := s.Catalog.Publication(ctx, id)
	if e != nil || pending.State != "pending" {
		t.Fatal("uncertain journal lost", e)
	}
	object, e := fs.root.Open(pending.Key)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := io.ReadAll(object)
	object.Close()
	if e != nil || string(raw) != "completed durable bytes" {
		t.Fatal("completed object changed", e)
	}
	if _, e = s.Reconcile(ctx, id); contracts.SafeError(e).Code != "insufficient_space" {
		t.Fatal("unverified persistence accepted", e)
	}
	snapshot, e := s.Catalog.Export(ctx)
	if e != nil || len(snapshot.Records.Artifacts) != 0 {
		t.Fatal("unverified bytes admitted", e)
	}
	if _, e = os.Stat(filepath.Join(s.scratchPath, id+".source")); e != nil {
		t.Fatal("recovery source lost", e)
	}
	fs.persist = persistObject
	if p, e := s.Reconcile(ctx, id); e != nil || p.State != "available" {
		t.Fatal("cleared flush did not reconcile", p, e)
	}
	if _, e = os.Stat(filepath.Join(s.scratchPath, id+".source")); !os.IsNotExist(e) {
		t.Fatal("accepted recovery source remains", e)
	}
	if e = s.Verify(ctx, id); e != nil {
		t.Fatal(e)
	}
}

func TestFilesystemPublicationCleanupCannotClaimSuccess(t *testing.T) {
	fs, e := NewFilesystem(t.TempDir() + "/objects")
	if e != nil {
		t.Fatal(e)
	}
	defer fs.Close()
	source := bytesFile(t, []byte("fully published bytes"))
	sum := sha256.Sum256([]byte("fully published bytes"))
	p := catalog.Publication{ID: contracts.ID(), Key: "objects/" + contracts.ID(), Digest: hex.EncodeToString(sum[:]), Size: 21, State: "pending"}
	removed := false
	_, e = fs.publishImmutable(context.Background(), p, source, func(string) error { removed = true; return os.ErrPermission })
	if e == nil || !removed || contracts.SafeError(e).Code != "unavailable" {
		t.Fatal("failed stage cleanup reported success", e)
	}
	if e = Verify(context.Background(), fs, p); e != nil {
		t.Fatal("cleanup failure changed completed object", e)
	}
	directory, e := fs.root.Open("objects")
	if e != nil {
		t.Fatal(e)
	}
	entries, e := directory.ReadDir(-1)
	directory.Close()
	if e != nil {
		t.Fatal(e)
	}
	staged := 0
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".stage-") {
			staged++
			if e = fs.root.Remove("objects/" + entry.Name()); e != nil {
				t.Fatal(e)
			}
		}
	}
	if staged != 1 {
		t.Fatal("failed cleanup residue was not observed", staged)
	}
}
