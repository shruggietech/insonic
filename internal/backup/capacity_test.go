// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestBackupWrappedCapacityAndContractCauses(t *testing.T) {
	for _, cause := range []error{syscall.ENOSPC, &os.PathError{Op: "write", Path: "private-owner-secret", Err: syscall.ENOSPC}, fmt.Errorf("private-owner-secret: %w", syscall.ENOSPC)} {
		got := contracts.SafeError(fail(cause))
		if got.Code != "insufficient_space" || strings.Contains(got.Message, "private-owner-secret") {
			t.Fatalf("capacity cause lost or leaked: %+v", got)
		}
	}
	for _, code := range []string{"conflict", "cancelled", "checksum_mismatch", "unsupported_capability"} {
		if got := contracts.SafeError(fail(fmt.Errorf("private-owner-secret: %w", contracts.Fail(code)))); got.Code != code {
			t.Fatalf("wrapped contract changed: %+v", got)
		}
	}
}

func TestBackupCleanupRetainsPrimaryAndRedactsSecondary(t *testing.T) {
	secondary := fmt.Errorf("private-cleanup-path: %w", os.ErrPermission)
	e := fail(cleanupError(syscall.ENOSPC, secondary))
	got := contracts.SafeError(e)
	if got.Code != "insufficient_space" || got.Reason != "backup_cleanup" || !errors.Is(e, syscall.ENOSPC) || !errors.Is(e, os.ErrPermission) || strings.Contains(e.Error(), "private-") {
		t.Fatalf("primary/secondary authority lost or leaked: %+v %v", got, e)
	}
}

func TestBackupCreationFailureBoundariesAndExactRetry(t *testing.T) {
	for _, boundary := range []string{"object-open", "object-write", "catalog-short", "manifest-flush", "object-close", "publish", "parent-durability", "cleanup"} {
		t.Run(boundary, func(t *testing.T) {
			ctx := context.Background()
			w, db, artifacts := fixture(t)
			p := publish(t, artifacts, "immutable accepted source", "canonical-audio")
			parent := t.TempDir()
			options := CreateOptions{ID: contracts.ID(), Directory: filepath.Join(parent, "bundle")}
			ops := realFiles()
			observed := false
			switch boundary {
			case "object-open":
				ops.openFile = func(r *os.Root, key string) (*os.File, error) {
					if strings.HasPrefix(key, "objects/") {
						observed = true
						return nil, syscall.ENOSPC
					}
					return realFiles().openFile(r, key)
				}
			case "object-write", "catalog-short", "cleanup":
				ops.write = func(f *os.File, raw []byte) (int, error) {
					if boundary == "catalog-short" && filepath.Base(f.Name()) == "catalog.json" {
						observed = true
						return f.Write(raw[:len(raw)/2])
					}
					if boundary != "catalog-short" && strings.Contains(f.Name(), "objects") {
						observed = true
						n, _ := f.Write(raw[:len(raw)/2])
						return n, syscall.ENOSPC
					}
					return f.Write(raw)
				}
			case "manifest-flush":
				ops.syncFile = func(f *os.File) error {
					if filepath.Base(f.Name()) == "manifest.json" {
						observed = true
						return syscall.ENOSPC
					}
					return f.Sync()
				}
			case "object-close":
				ops.closeFile = func(f *os.File) error {
					e := f.Close()
					if strings.Contains(f.Name(), "objects") {
						observed = true
						return syscall.ENOSPC
					}
					return e
				}
			case "publish":
				ops.publish = func(string, string) error { observed = true; return syscall.ENOSPC }
			case "parent-durability":
				ops.syncDir = func(dir string) error {
					if dir == parent {
						observed = true
						return syscall.ENOSPC
					}
					return realFiles().syncDir(dir)
				}
			}
			var residual string
			if boundary == "cleanup" {
				ops.removeAll = func(path string) error { residual = path; return os.ErrPermission }
			}
			_, e := createWithIO(ctx, w, db, nil, options, ops)
			want := "insufficient_space"
			if boundary == "catalog-short" {
				want = "unavailable"
			}
			if !observed || e == nil || contracts.SafeError(e).Code != want {
				t.Fatalf("boundary not exercised/classified: %v %+v", observed, e)
			}
			if boundary == "cleanup" {
				if contracts.SafeError(e).Reason != "backup_cleanup" || residual == "" {
					t.Fatal("cleanup unobserved", e)
				}
				if e = os.RemoveAll(residual); e != nil {
					t.Fatal(e)
				}
			}
			if boundary != "parent-durability" {
				if _, e = os.Stat(options.Directory); !os.IsNotExist(e) {
					t.Fatal("incomplete backup advertised", e)
				}
			}
			retained, e := db.Publication(ctx, p.ID)
			if e != nil || !retained.References[options.ID] || retained.Digest != p.Digest {
				t.Fatal("capture authority lost", e)
			}
			if e = artifact.Verify(ctx, artifacts.Store, p); e != nil {
				t.Fatal("source changed", e)
			}
			if boundary == "parent-durability" {
				if _, e = createWithIO(ctx, w, db, nil, options, ops); contracts.SafeError(e).Code != "insufficient_space" {
					t.Fatal("retry skipped durability", e)
				}
			}
			m, e := Create(ctx, w, db, nil, options)
			if e != nil || m.ID != options.ID {
				t.Fatal("identical retry", e)
			}
			if _, e = Verify(ctx, options.Directory, nil); e != nil {
				t.Fatal("retry verification", e)
			}
			retained, e = db.Publication(ctx, p.ID)
			if e != nil || retained.References[options.ID] {
				t.Fatal("completed capture pins not reconciled", e)
			}
			if files, e := filepath.Glob(filepath.Join(parent, ".insonic-backup-*")); e != nil || len(files) != 0 {
				t.Fatal("owned staging leaked", files, e)
			}
		})
	}
}

type capacityDestination struct {
	artifact.Store
	observed *bool
}

func (s capacityDestination) PublishImmutable(context.Context, catalog.Publication, *os.File, func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	*s.observed = true
	return catalog.Publication{}, syscall.ENOSPC
}

func TestRestoreCapacityFailureBoundariesAndExactRetry(t *testing.T) {
	for _, boundary := range []string{"publication", "progress-open", "progress-short", "progress-flush", "progress-replace", "activation-open", "activation-write", "activation-flush", "activation-close", "activation-replace", "progress-cleanup"} {
		t.Run(boundary, func(t *testing.T) {
			ctx := context.Background()
			w, db, artifacts := fixture(t)
			p := publish(t, artifacts, "accepted restore bytes", "canonical-audio")
			bundle := filepath.Join(t.TempDir(), "bundle")
			m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: bundle})
			if e != nil {
				t.Fatal(e)
			}
			target, e := workspace.Init(t.TempDir(), "Capacity target")
			if e != nil {
				t.Fatal(e)
			}
			marker := filepath.Join(target.Root, ".insonic/workspace.json")
			original, e := os.ReadFile(marker)
			if e != nil {
				t.Fatal(e)
			}
			ops := realFiles()
			observed := false
			matches := func(name string) bool {
				return strings.Contains(filepath.Base(name), ".progress-") && strings.HasPrefix(boundary, "progress-") || strings.Contains(filepath.Base(name), ".restore-workspace-") && strings.HasPrefix(boundary, "activation-")
			}
			if boundary == "publication" {
				ops.storage = func(c context.Context, p workspace.Profile, control string, provider contracts.SecretProvider) (artifact.Store, error) {
					s, e := openStorage(c, p, control, provider)
					if e != nil {
						return nil, e
					}
					return capacityDestination{s, &observed}, nil
				}
			}
			ops.createTemp = func(dir, pattern string) (*os.File, error) {
				if strings.HasSuffix(boundary, "-open") && matches(pattern) {
					observed = true
					return nil, syscall.ENOSPC
				}
				return os.CreateTemp(dir, pattern)
			}
			ops.write = func(f *os.File, raw []byte) (int, error) {
				if matches(f.Name()) && (strings.HasSuffix(boundary, "-short") || strings.HasSuffix(boundary, "-write")) {
					observed = true
					n, e := f.Write(raw[:len(raw)/2])
					if strings.HasSuffix(boundary, "-write") {
						e = syscall.ENOSPC
					}
					return n, e
				}
				return f.Write(raw)
			}
			ops.syncFile = func(f *os.File) error {
				if strings.HasSuffix(boundary, "-flush") && matches(f.Name()) {
					observed = true
					return syscall.ENOSPC
				}
				return f.Sync()
			}
			ops.closeFile = func(f *os.File) error {
				e := f.Close()
				if strings.HasSuffix(boundary, "-close") && matches(f.Name()) {
					observed = true
					return syscall.ENOSPC
				}
				return e
			}
			ops.replace = func(from, to string) error {
				if strings.HasSuffix(boundary, "-replace") && matches(from) {
					observed = true
					return syscall.ENOSPC
				}
				return realFiles().replace(from, to)
			}
			ops.remove = func(path string) error {
				if boundary == "progress-cleanup" && filepath.Base(path) == p.ID+".json" {
					observed = true
					return os.ErrPermission
				}
				return os.Remove(path)
			}
			_, e = restoreWithIO(ctx, target, nil, bundle, ops)
			want := "insufficient_space"
			if boundary == "progress-short" || boundary == "progress-cleanup" {
				want = "unavailable"
			}
			if !observed || e == nil || contracts.SafeError(e).Code != want {
				t.Fatalf("boundary not exercised/classified: %v %v", observed, e)
			}
			if boundary != "progress-cleanup" {
				current, e := os.ReadFile(marker)
				if e != nil || string(current) != string(original) {
					t.Fatal("incomplete restore activated", e)
				}
			} else if contracts.SafeError(e).Reason != "backup_cleanup" {
				t.Fatal("progress cleanup unobserved", e)
			}
			if strings.HasPrefix(boundary, "activation-") {
				passive := *target
				passive.Config = target.Config
				passive.Config.WorkspaceID = w.Config.WorkspaceID
				passive.Config.CredentialNamespaceID = target.CredentialNamespace()
				store, e := catalog.OpenWorkspace(ctx, &passive, nil, true)
				if e != nil {
					t.Fatal(e)
				}
				_, e = store.PutNamedSetting(ctx, contracts.ID(), "must-stay-fenced", 0, []byte(`{}`))
				store.Close()
				if contracts.SafeError(e).Code != "conflict" {
					t.Fatal("pending activation writable", e)
				}
			}
			if _, e = RestoreWorkspace(ctx, target, nil, bundle); e != nil {
				t.Fatal("identical restore retry", e)
			}
			verified, e := Verify(ctx, bundle, nil)
			if e != nil || verified.Digest != m.Digest {
				t.Fatal("bundle changed", e)
			}
			store, e := catalog.OpenWorkspace(ctx, target, nil, false)
			if e != nil {
				t.Fatal(e)
			}
			defer store.Close()
			restored, e := store.Publication(ctx, p.ID)
			if e != nil || restored.Digest != p.Digest {
				t.Fatal("accepted identity changed", e)
			}
			destination, e := openStorage(ctx, target.Config.Profiles.Storage, target.Control, nil)
			if e != nil {
				t.Fatal(e)
			}
			defer destination.Close()
			if e = artifact.Verify(ctx, destination, restored); e != nil {
				t.Fatal("restored bytes differ", e)
			}
			if _, e = os.Stat(filepath.Join(target.Control, "backup-restore-progress", m.ID)); !os.IsNotExist(e) {
				t.Fatal("progress leaked", e)
			}
		})
	}
}

type shortBackupWriter struct{}

func (shortBackupWriter) Write(p []byte) (int, error) { return len(p) / 2, nil }

func TestBackupShortStreamNeverVerifies(t *testing.T) {
	if e := hashStream(context.Background(), shortBackupWriter{}, strings.NewReader("original"), 8, strings.Repeat("0", 64)); e == nil || contracts.SafeError(e).Code != "unavailable" {
		t.Fatalf("short stream accepted or mislabeled: %v", e)
	}
	if contracts.SafeError(fail(io.ErrShortWrite)).Code != "unavailable" {
		t.Fatal("short write must remain an I/O failure")
	}
}
