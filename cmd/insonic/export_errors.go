// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"runtime"
	"syscall"

	"github.com/shruggietech/insonic/internal/contracts"
)

func exportPathError(path string, err error) *contracts.Error {
	if path != "" {
		if info, e := os.Stat(path); e == nil && info.IsDir() {
			return contracts.Condition("unavailable", "output_directory")
		}
	}
	switch {
	case errors.Is(err, os.ErrExist):
		return contracts.Condition("conflict", "output_exists")
	case errors.Is(err, os.ErrNotExist):
		return contracts.Condition("unavailable", "output_parent_missing")
	case errors.Is(err, os.ErrPermission):
		return contracts.Condition("unavailable", "output_permission")
	case errors.Is(err, syscall.ENOSPC), runtime.GOOS == "windows" && errors.Is(err, syscall.Errno(112)):
		return contracts.Condition("unavailable", "output_space")
	default:
		return contracts.Condition("unavailable", "output_write")
	}
}

func exportCatalogFile(path string, snapshot any) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return exportPathError(path, err)
	}
	identity, statErr := file.Stat()
	if statErr != nil {
		file.Close()
		return exportPathError(path, statErr)
	}
	err = writeCatalogSnapshot(file, snapshot)
	if err != nil {
		// Remove only this invocation's incomplete file, never a replacement.
		if current, e := os.Lstat(path); e == nil && os.SameFile(identity, current) {
			_ = os.Remove(path)
		}
		return exportPathError(path, err)
	}
	return nil
}

// Always close an owned destination, retaining the earliest write/sync failure.
func writeCatalogSnapshot(file interface {
	io.Writer
	Sync() error
	Close() error
}, snapshot any) error {
	err := json.NewEncoder(file).Encode(snapshot)
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}
