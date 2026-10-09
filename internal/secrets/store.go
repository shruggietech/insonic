// SPDX-License-Identifier: Apache-2.0
// Package secrets provides catalog-independent, workspace-scoped credentials.
package secrets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

const MaxValueBytes = 2048
const maxFileBytes = 1 << 20
const maxCredentials = 256

type Options struct {
	Mode       string
	Passphrase []byte
	Session    map[string][]byte
}
type CredentialStatus struct {
	CredentialID string `json:"credential_id"`
	Backend      string `json:"backend"`
	State        string `json:"state"`
}
type nativeBackend interface {
	available(context.Context) error
	get(context.Context, string) ([]byte, error)
	put(context.Context, string, []byte) error
	remove(context.Context, string) error
}
type Manager struct {
	mu                     sync.Mutex
	mode, workspaceID, dir string
	root                   *os.Root
	native                 nativeBackend
	values                 map[string][]byte
	passphrase, key, salt  []byte
	closed                 bool
}

func privateRoot(w *workspace.Workspace) (*os.Root, string, error) {
	if w == nil || !contracts.ValidID(w.Config.WorkspaceID) || !contracts.ValidID(w.CredentialNamespace()) {
		return nil, "", contracts.Fail("invalid_request")
	}
	dir := filepath.Join(w.Control, "secrets")
	created := false
	if e := os.Mkdir(dir, 0700); e == nil {
		created = true
	} else if !os.IsExist(e) {
		return nil, "", contracts.Fail("unavailable")
	}
	if e := workspace.SecureDirectory(dir, created); e != nil {
		return nil, "", e
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return nil, "", contracts.Fail("unavailable")
	}
	return root, dir, nil
}
func validMode(mode string) bool { return mode == "native" || mode == "vault" || mode == "session" }
func ReadSelection(w *workspace.Workspace) (string, error) {
	root, _, e := privateRoot(w)
	if e != nil {
		return "", e
	}
	defer root.Close()
	raw, e := readFile(root, "selection.json")
	if os.IsNotExist(e) {
		return "native", nil
	}
	if e != nil {
		return "", contracts.Fail("unavailable")
	}
	var cfg struct {
		Version     int    `json:"version"`
		WorkspaceID string `json:"workspace_id"`
		Mode        string `json:"mode"`
	}
	if strict(raw, &cfg) != nil || cfg.Version != 1 || cfg.WorkspaceID != w.CredentialNamespace() || !validMode(cfg.Mode) {
		return "", contracts.Fail("invalid_request")
	}
	return cfg.Mode, nil
}
func Select(w *workspace.Workspace, mode string) error {
	if !validMode(mode) {
		return contracts.Fail("invalid_request")
	}
	root, dir, e := privateRoot(w)
	if e != nil {
		return e
	}
	defer root.Close()
	lock := flock.New(filepath.Join(dir, "credentials.lock"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ok, e := lock.TryLockContext(ctx, 10*time.Millisecond)
	if e != nil || !ok {
		return contracts.Fail("unavailable")
	}
	defer lock.Unlock()
	raw, _ := json.Marshal(struct {
		Version     int    `json:"version"`
		WorkspaceID string `json:"workspace_id"`
		Mode        string `json:"mode"`
	}{1, w.CredentialNamespace(), mode})
	return atomicWrite(root, "selection.json", raw)
}
func Open(w *workspace.Workspace, opts Options) (*Manager, error) {
	if opts.Mode == "" {
		mode, e := ReadSelection(w)
		if e != nil {
			return nil, e
		}
		opts.Mode = mode
	}
	if !validMode(opts.Mode) || len(opts.Passphrase) > MaxValueBytes || (opts.Mode != "vault" && len(opts.Passphrase) > 0) || (opts.Mode != "session" && len(opts.Session) > 0) {
		return nil, contracts.Fail("invalid_request")
	}
	root, dir, e := privateRoot(w)
	if e != nil {
		return nil, e
	}
	m := &Manager{mode: opts.Mode, workspaceID: w.CredentialNamespace(), root: root, dir: dir, values: map[string][]byte{}, passphrase: append([]byte(nil), opts.Passphrase...)}
	if opts.Mode == "native" {
		m.native = newNative("insonic/" + m.workspaceID)
	}
	if len(opts.Session) > maxCredentials {
		m.Close()
		return nil, contracts.Fail("invalid_request")
	}
	for id, value := range opts.Session {
		if !validValue(id, value) {
			m.Close()
			return nil, contracts.Fail("invalid_request")
		}
		m.values[id] = append([]byte(nil), value...)
	}
	if opts.Mode == "vault" && len(opts.Passphrase) > 0 {
		e = m.locked(context.Background(), func() error { values, e := m.readVault(); wipeMap(values); return e })
		if e != nil {
			m.Close()
			return nil, e
		}
	}
	return m, nil
}
func validValue(id string, value []byte) bool {
	return contracts.ValidID(id) && len(value) > 0 && len(value) <= MaxValueBytes
}
func (m *Manager) locked(ctx context.Context, fn func() error) error {
	if ctx.Err() != nil {
		return contracts.Fail("cancelled")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return contracts.Fail("unavailable")
	}
	if m.mode == "session" {
		return fn()
	}
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	lock := flock.New(filepath.Join(m.dir, "credentials.lock"))
	ok, e := lock.TryLockContext(waitCtx, 10*time.Millisecond)
	if e != nil || !ok {
		return contracts.Fail("unavailable")
	}
	defer lock.Unlock()
	return fn()
}
func (m *Manager) Inspect(ctx context.Context, id string) (CredentialStatus, error) {
	result := CredentialStatus{CredentialID: id, Backend: m.mode, State: "missing"}
	if !contracts.ValidID(id) {
		return result, contracts.Fail("invalid_request")
	}
	e := m.locked(ctx, func() error {
		switch m.mode {
		case "session":
			if _, ok := m.values[id]; ok {
				result.State = "configured"
			}
		case "vault":
			// Status validates only the nonsecret envelope. It never decrypts saved values.
			if len(m.passphrase) == 0 {
				result.State = "rejected"
				return nil
			}
			raw, e := readFile(m.root, "vault.json")
			if os.IsNotExist(e) {
				return nil
			}
			if e != nil {
				result.State = "rejected"
				return nil
			}
			envelope, e := m.parseEnvelope(raw)
			if e != nil {
				result.State = "rejected"
				return nil
			}
			for _, candidate := range envelope.IDs {
				if candidate == id {
					result.State = "configured"
				}
			}
		case "native":
			if m.native.available(ctx) != nil {
				result.State = "rejected"
				return nil
			}
			ids, e := m.readIndex()
			if e != nil {
				result.State = "rejected"
				return nil
			}
			if ids[id] {
				result.State = "configured"
			}
		}
		return nil
	})
	return result, e
}
func (m *Manager) Status(ctx context.Context, id string) (string, error) {
	s, e := m.Inspect(ctx, id)
	return s.State, e
}
func (m *Manager) Resolve(ctx context.Context, id string) ([]byte, error) {
	if !contracts.ValidID(id) {
		return nil, contracts.Fail("invalid_request")
	}
	var result []byte
	e := m.locked(ctx, func() error {
		switch m.mode {
		case "session":
			if value, ok := m.values[id]; ok {
				result = append([]byte(nil), value...)
				return nil
			}
		case "vault":
			values, e := m.readVault()
			defer wipeMap(values)
			if e != nil {
				return e
			}
			if value, ok := values[id]; ok {
				result = append([]byte(nil), value...)
				return nil
			}
		case "native":
			ids, e := m.readIndex()
			if e != nil {
				return e
			}
			if !ids[id] {
				return contracts.Fail("unavailable")
			}
			result, e = m.native.get(ctx, id)
			if e != nil {
				return contracts.Fail("unavailable")
			}
			if !validValue(id, result) {
				wipe(result)
				result = nil
				return contracts.Fail("unavailable")
			}
			return nil
		}
		return contracts.Fail("unavailable")
	})
	if e != nil {
		wipe(result)
		return nil, e
	}
	return result, nil
}
func (m *Manager) Add(ctx context.Context, id string, value []byte) error {
	return m.write(ctx, id, value, false, false)
}
func (m *Manager) Replace(ctx context.Context, id string, value []byte) error {
	return m.write(ctx, id, value, true, false)
}
func (m *Manager) Delete(ctx context.Context, id string) error {
	return m.write(ctx, id, nil, false, true)
}
func (m *Manager) write(ctx context.Context, id string, value []byte, replace, remove bool) error {
	if !contracts.ValidID(id) || (!remove && !validValue(id, value)) {
		return contracts.Fail("invalid_request")
	}
	return m.locked(ctx, func() error {
		values := m.values
		var ids map[string]bool
		var e error
		switch m.mode {
		case "vault":
			values, e = m.readVault()
			if e != nil {
				return e
			}
			defer wipeMap(values)
		case "native":
			ids, e = m.readIndex()
			if e != nil {
				return e
			}
		}
		_, exists := values[id]
		count := len(values)
		if m.mode == "native" {
			exists = ids[id]
			count = len(ids)
			if m.native.available(ctx) != nil {
				return contracts.Fail("unavailable")
			}
			if !exists {
				// An interrupted index write can leave an OS credential. Only an
				// explicitly requested mutation may inspect and reconcile it.
				previous, lookup := m.native.get(ctx, id)
				wipe(previous)
				if lookup == nil {
					exists = true
				} else {
					var failure *contracts.Error
					if !errors.As(lookup, &failure) || failure.Code != "not_found" {
						return contracts.Fail("unavailable")
					}
				}
			}
		}
		if !exists && (replace || remove) {
			return contracts.Fail("not_found")
		}
		if exists && !replace && !remove {
			return contracts.Fail("conflict")
		}
		if !remove && (!exists || m.mode == "native" && !ids[id]) && count >= maxCredentials {
			return contracts.Fail("conflict")
		}
		if m.mode == "native" {
			if remove {
				e = m.native.remove(ctx, id)
				var failure *contracts.Error
				if errors.As(e, &failure) && failure.Code == "not_found" {
					e = nil
				}
				delete(ids, id)
			} else {
				e = m.native.put(ctx, id, value)
				ids[id] = true
			}
			if e != nil {
				return contracts.Fail("unavailable")
			}
			return m.saveIndex(ids)
		}
		wipe(values[id])
		if remove {
			delete(values, id)
		} else {
			values[id] = append([]byte(nil), value...)
		}
		if m.mode == "vault" {
			return m.saveVault(values)
		}
		return nil
	})
}
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	wipeMap(m.values)
	wipe(m.passphrase)
	wipe(m.key)
	wipe(m.salt)
	m.passphrase = nil
	m.key = nil
	m.salt = nil
	return m.root.Close()
}
func wipe(value []byte) { clear(value) }
func wipeMap(values map[string][]byte) {
	for id, value := range values {
		wipe(value)
		delete(values, id)
	}
}
func strict(raw []byte, target any) error {
	validation := json.NewDecoder(bytes.NewReader(raw))
	validation.UseNumber()
	if uniqueJSON(validation, 0) != nil {
		return contracts.Fail("invalid_request")
	}
	if _, e := validation.Token(); e != io.EOF {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(target) != nil {
		return contracts.Fail("invalid_request")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}

// Reject duplicate object keys and excessive nesting before typed decoding.
func uniqueJSON(d *json.Decoder, depth int) error {
	if depth > 64 {
		return contracts.Fail("invalid_request")
	}
	token, e := d.Token()
	if e != nil {
		return e
	}
	delimiter, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	if delimiter != '{' && delimiter != '[' {
		return contracts.Fail("invalid_request")
	}
	seen := map[string]bool{}
	for d.More() {
		if delimiter == '{' {
			key, e := d.Token()
			if e != nil {
				return e
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return contracts.Fail("invalid_request")
			}
			seen[name] = true
		}
		if e := uniqueJSON(d, depth+1); e != nil {
			return e
		}
	}
	end, e := d.Token()
	if e != nil {
		return e
	}
	if delimiter == '{' && end != json.Delim('}') || delimiter == '[' && end != json.Delim(']') {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func readFile(root *os.Root, name string) ([]byte, error) {
	info, e := root.Lstat(name)
	if e != nil {
		return nil, e
	}
	if !info.Mode().IsRegular() || info.Size() > maxFileBytes {
		return nil, contracts.Fail("unavailable")
	}
	f, e := root.Open(name)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	data, e := io.ReadAll(io.LimitReader(f, maxFileBytes+1))
	if len(data) > maxFileBytes {
		return nil, contracts.Fail("unavailable")
	}
	return data, e
}
func atomicWrite(root *os.Root, name string, data []byte) error {
	tmp := ".write-" + contracts.ID()
	f, e := root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer root.Remove(tmp)
	_, e = f.Write(append(data, '\n'))
	if e == nil {
		e = f.Sync()
	}
	closed := f.Close()
	if e == nil {
		e = closed
	}
	if e == nil {
		e = root.Rename(tmp, name)
	}
	if e == nil {
		e = persist(root, name)
	}
	if e != nil {
		return contracts.Fail("unavailable")
	}
	return nil
}

type nativeIndex struct {
	Version     int             `json:"version"`
	WorkspaceID string          `json:"workspace_id"`
	IDs         map[string]bool `json:"ids"`
}

func (m *Manager) readIndex() (map[string]bool, error) {
	raw, e := readFile(m.root, "native-index.json")
	if os.IsNotExist(e) {
		return map[string]bool{}, nil
	}
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	var idx nativeIndex
	if strict(raw, &idx) != nil || idx.Version != 1 || idx.WorkspaceID != m.workspaceID || len(idx.IDs) > maxCredentials {
		return nil, contracts.Fail("unavailable")
	}
	if idx.IDs == nil {
		idx.IDs = map[string]bool{}
	}
	for id, exists := range idx.IDs {
		if !contracts.ValidID(id) || !exists {
			return nil, contracts.Fail("unavailable")
		}
	}
	return idx.IDs, nil
}
func (m *Manager) saveIndex(ids map[string]bool) error {
	raw, e := json.Marshal(nativeIndex{1, m.workspaceID, ids})
	if e != nil {
		return contracts.Fail("unavailable")
	}
	return atomicWrite(m.root, "native-index.json", raw)
}
