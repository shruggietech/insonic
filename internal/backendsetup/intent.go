// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"os"
	"path/filepath"
)

// retainIntent runs under canonical workspace ownership, before secret or backend mutation.
// It intentionally survives failure and success as the exact owned bootstrap receipt.
func retainIntent(w *workspace.Workspace, mode string) error {
	file := filepath.Join(w.Control, "provision-intent.json")
	expected, _ := json.Marshal(struct {
		Version string           `json:"schema_version"`
		Mode    string           `json:"credential_backend"`
		Config  workspace.Config `json:"configuration"`
	}{contracts.Version, mode, w.Config})
	if st, e := os.Lstat(file); e == nil {
		if !st.Mode().IsRegular() || st.Size() > 2<<20 {
			return contracts.Fail("conflict")
		}
		f, e := os.Open(file)
		if e != nil {
			return contracts.Fail("unavailable")
		}
		raw, e := io.ReadAll(io.LimitReader(f, (2<<20)+1))
		f.Close()
		if e != nil || !bytes.Equal(bytes.TrimSpace(raw), expected) {
			return contracts.Condition("conflict", "backend_pending_plan")
		}
		return nil
	} else if !os.IsNotExist(e) {
		return contracts.Fail("unavailable")
	}
	tmp, e := os.CreateTemp(w.Control, ".provision-intent-*")
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer os.Remove(tmp.Name())
	if e = tmp.Chmod(0600); e == nil {
		_, e = tmp.Write(append(expected, 10))
	}
	if e == nil {
		e = tmp.Sync()
	}
	closeErr := tmp.Close()
	if e == nil {
		e = closeErr
	}
	if e != nil {
		return contracts.Fail("unavailable")
	}
	if e = os.Link(tmp.Name(), file); e != nil {
		return contracts.Fail("conflict")
	}
	return nil
}
