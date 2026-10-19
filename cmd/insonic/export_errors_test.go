// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestCatalogExportDestinationFailuresHaveActualCorrections(t *testing.T) {
	root := t.TempDir()
	existing := filepath.Join(root, "existing.json")
	if err := os.WriteFile(existing, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ path, reason string }{{existing, "output_exists"}, {root, "output_directory"}, {filepath.Join(root, "absent", "output.json"), "output_parent_missing"}} {
		err := exportCatalogFile(tc.path, map[string]any{"value": 1})
		var typed *contracts.Error
		if !errors.As(err, &typed) || typed.Reason != tc.reason {
			t.Fatalf("%s: %v", tc.reason, err)
		}
	}
	if raw, _ := os.ReadFile(existing); string(raw) != "original" {
		t.Fatal("overwrote existing destination")
	}
	for _, tc := range []struct {
		err    error
		reason string
	}{{os.ErrPermission, "output_permission"}, {syscall.ENOSPC, "output_space"}, {errors.New("private raw disk failure"), "output_write"}} {
		if got := exportPathError("", tc.err); got.Reason != tc.reason {
			t.Fatal(got)
		}
	}
}

type failedExportFile struct {
	writeErr, syncErr, closeErr error
	closed                      bool
}

func (f *failedExportFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return len(p), nil
}
func (f *failedExportFile) Sync() error  { return f.syncErr }
func (f *failedExportFile) Close() error { f.closed = true; return f.closeErr }

func TestCatalogExportWriteSyncAndCloseFailuresKeepTheirCause(t *testing.T) {
	for _, stage := range []string{"write", "sync", "close"} {
		t.Run(stage, func(t *testing.T) {
			file := &failedExportFile{}
			switch stage {
			case "write":
				file.writeErr = syscall.ENOSPC
			case "sync":
				file.syncErr = syscall.ENOSPC
			case "close":
				file.closeErr = syscall.ENOSPC
			}
			err := writeCatalogSnapshot(file, map[string]any{"value": 1})
			if !file.closed || !errors.Is(err, syscall.ENOSPC) || exportPathError("", err).Reason != "output_space" {
				t.Fatalf("lost %s cause or close: %v", stage, err)
			}
		})
	}
	path := filepath.Join(t.TempDir(), "incomplete.json")
	err := exportCatalogFile(path, map[string]any{"unsupported": make(chan int)})
	var typed *contracts.Error
	if !errors.As(err, &typed) || typed.Reason != "output_write" {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("incomplete owned destination remains: %v", err)
	}
}
