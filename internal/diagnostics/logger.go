// SPDX-License-Identifier: Apache-2.0
// Package diagnostics writes only generated operator context, never arguments,
// input documents, tool output, credentials, endpoints or environment values.
package diagnostics

import (
	"encoding/json"
	"github.com/gofrs/flock"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const MaxBytes = 1 << 20

type Event struct {
	Time        string `json:"time"`
	Operation   string `json:"operation,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Code        string `json:"code,omitempty"`
	Reason      string `json:"reason,omitempty"`
	State       string `json:"state,omitempty"`
	DurationMS  int64  `json:"duration_ms,omitempty"`
}
type Logger struct {
	mu    sync.Mutex
	root  *os.Root
	name  string
	file  *os.File
	lock  *flock.Flock
	level int
}

var operationPattern = regexp.MustCompile("^[a-z][a-z.-]{0,63}$")
var levels = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}

func logError() error {
	return &contracts.Error{Code: "unavailable", Message: "The elected diagnostic log could not be opened, secured or written. Check its directory and access permissions; use a separate file for concurrent commands."}
}
func Open(path, level string) (*Logger, error) {
	if path == "" {
		return nil, nil
	}
	threshold, ok := levels[level]
	if !ok {
		return nil, contracts.Fail("invalid_request")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, logError()
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return nil, logError()
	}
	log := &Logger{root: root, name: filepath.Base(absolute), level: threshold}
	fail := func() (*Logger, error) { log.Close(); return nil, logError() }
	// Refuse links before opening; OpenRoot also confines resolution to this directory.
	for _, suffix := range []string{"", ".1", ".2", ".lock"} {
		st, e := root.Lstat(log.name + suffix)
		if e != nil && !os.IsNotExist(e) || e == nil && (!st.Mode().IsRegular() || st.Size() > MaxBytes || suffix == ".lock" && st.Size() != 0) {
			return fail()
		}
	}
	lockFile, created, err := openFile(root, log.name+".lock", os.O_RDWR)
	if err != nil {
		return fail()
	}
	err = privateFile(lockFile, created)
	closeErr := lockFile.Close()
	if err != nil || closeErr != nil {
		return fail()
	}
	log.lock = flock.New(absolute + ".lock")
	locked, err := log.lock.TryLock()
	if err != nil || !locked {
		return fail()
	}
	for _, suffix := range []string{".1", ".2"} {
		file, e := root.OpenFile(log.name+suffix, os.O_RDWR, 0600)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return fail()
		}
		e = privateFile(file, false)
		closeErr := file.Close()
		if e != nil || closeErr != nil {
			return fail()
		}
	}
	if err = log.open(); err != nil {
		return fail()
	}
	return log, nil
}
func (l *Logger) open() error {
	f, created, err := openFile(l.root, l.name, os.O_APPEND|os.O_WRONLY)
	if err != nil {
		return err
	}
	if err = privateFile(f, created); err != nil {
		f.Close()
		return err
	}
	l.file = f
	return nil
}
func (l *Logger) Write(level string, event Event) error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	priority, ok := levels[level]
	if !ok || priority < l.level {
		return nil
	}
	if l.file == nil {
		return logError()
	}
	event.Time = time.Now().UTC().Format(time.RFC3339Nano)
	if !operationPattern.MatchString(event.Operation) {
		event.Operation = ""
	}
	if !contracts.ValidID(event.RequestID) {
		event.RequestID = ""
	}
	if !contracts.ValidID(event.WorkspaceID) {
		event.WorkspaceID = ""
	}
	if event.Code != "" {
		event.Code = contracts.Fail(event.Code).Code
	}
	if !contracts.ValidOperatorReason(event.Reason) {
		event.Reason = ""
	}
	switch event.State {
	case "pending", "running", "succeeded", "failed", "cancelled", "interrupted":
	default:
		event.State = ""
	}
	if event.DurationMS < 0 {
		event.DurationMS = 0
	}
	raw, err := json.Marshal(struct {
		Level string `json:"level"`
		Event
	}{level, event})
	if err != nil {
		return logError()
	}
	raw = append(raw, '\n')
	stat, err := l.file.Stat()
	if err != nil {
		return logError()
	}
	if stat.Size()+int64(len(raw)) > MaxBytes {
		if err = l.file.Close(); err != nil {
			return logError()
		}
		l.file = nil
		if err = l.root.Remove(l.name + ".2"); err != nil && !os.IsNotExist(err) {
			return logError()
		}
		if err = l.root.Rename(l.name+".1", l.name+".2"); err != nil && !os.IsNotExist(err) {
			return logError()
		}
		if err = l.root.Rename(l.name, l.name+".1"); err != nil {
			return logError()
		}
		if err = l.open(); err != nil {
			return logError()
		}
	}
	if _, err = l.file.Write(raw); err != nil {
		return logError()
	}
	return nil
}
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	var err error
	if l.file != nil {
		err = l.file.Close()
		l.file = nil
	}
	if l.lock != nil {
		if e := l.lock.Unlock(); err == nil {
			err = e
		}
		l.lock = nil
	}
	if l.root != nil {
		if e := l.root.Close(); err == nil {
			err = e
		}
		l.root = nil
	}
	return err
}

func openFile(root *os.Root, name string, flags int) (*os.File, bool, error) {
	file, err := root.OpenFile(name, flags|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		file, err = root.OpenFile(name, flags, 0600)
		return file, false, err
	}
	return file, err == nil, err
}
