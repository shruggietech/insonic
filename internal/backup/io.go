// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

// Per-call operations keep fault tests isolated. Public entry points always use
// realFiles; there is no product configuration or environment injection path.
type fileOperations struct {
	openFile   func(*os.Root, string) (*os.File, error)
	createTemp func(string, string) (*os.File, error)
	write      func(*os.File, []byte) (int, error)
	syncFile   func(*os.File) error
	closeFile  func(*os.File) error
	removeFile func(*os.Root, string) error
	remove     func(string) error
	removeAll  func(string) error
	syncDir    func(string) error
	publish    func(string, string) error
	replace    func(string, string) error
	storage    func(context.Context, workspace.Profile, string, contracts.SecretProvider) (artifact.Store, error)
}

// Keep native causes available to errors.Is/As without exposing their text.
type ioFailure struct {
	public *contracts.Error
	cause  error
}

func (e *ioFailure) Error() string   { return e.public.Error() }
func (e *ioFailure) Unwrap() []error { return []error{e.public, e.cause} }

func realFiles() fileOperations {
	return fileOperations{
		openFile: func(r *os.Root, key string) (*os.File, error) {
			return r.OpenFile(key, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		},
		createTemp: os.CreateTemp, write: (*os.File).Write, syncFile: (*os.File).Sync, closeFile: (*os.File).Close,
		removeFile: (*os.Root).Remove, remove: os.Remove, removeAll: os.RemoveAll,
		syncDir: syncDirectory, publish: publishDirectory, replace: replaceConfiguration, storage: openStorage,
	}
}

func cleanupError(primary, cleanup error) error {
	if cleanup == nil || os.IsNotExist(cleanup) {
		return primary
	}
	cause := primary
	if cause == nil {
		cause = cleanup
	}
	return &ioFailure{contracts.Condition(contracts.SafeError(fail(cause)).Code, "backup_cleanup"), errors.Join(primary, cleanup)}
}

func writeAndClose(ops fileOperations, f *os.File, raw []byte) error {
	n, e := ops.write(f, raw)
	if e == nil && n != len(raw) {
		e = io.ErrShortWrite
	}
	if e == nil {
		e = ops.syncFile(f)
	}
	closed := ops.closeFile(f)
	if e != nil && closed != nil {
		return cleanupError(e, closed)
	}
	if e == nil {
		e = closed
	}
	return fail(e)
}

func writePrivateWithIO(ops fileOperations, root *os.Root, key string, raw []byte) error {
	f, e := ops.openFile(root, key)
	if e != nil {
		if os.IsExist(e) {
			return contracts.Fail("conflict")
		}
		return fail(e)
	}
	e = writeAndClose(ops, f, raw)
	if e != nil {
		return cleanupError(e, ops.removeFile(root, key))
	}
	return nil
}

func replacePrivate(ops fileOperations, directory, pattern, destination string, raw []byte) (result error) {
	f, e := ops.createTemp(directory, pattern)
	if e != nil {
		return fail(e)
	}
	name := f.Name()
	defer func() { result = cleanupError(result, ops.remove(name)) }()
	if e = writeAndClose(ops, f, raw); e != nil {
		return e
	}
	if e = ops.replace(name, destination); e != nil {
		return fail(e)
	}
	return fail(ops.syncDir(directory))
}

type outputWriter struct {
	ops  fileOperations
	file *os.File
}

func (w outputWriter) Write(p []byte) (int, error) { return w.ops.write(w.file, p) }

func bundleDurability(ops fileOperations, directory string) error {
	for _, path := range []string{filepath.Join(directory, "objects"), directory, filepath.Dir(directory)} {
		if e := ops.syncDir(path); e != nil {
			return fail(e)
		}
	}
	return nil
}
