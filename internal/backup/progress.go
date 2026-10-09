// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

// Destination progress belongs to the independently configured target, never
// the portable bundle or accepted catalog. A flushed multipart journal allows
// an interrupted process to reuse its exact upload and immutable destination.
type restoreProgress struct {
	Kind        string              `json:"kind"`
	Bundle      string              `json:"bundle_digest"`
	Destination string              `json:"destination_digest"`
	Publication catalog.Publication `json:"publication"`
}

func restoreProgressDirectory(w *workspace.Workspace, id string) (string, error) {
	parent := filepath.Join(w.Control, "backup-restore-progress")
	directory := filepath.Join(parent, id)
	for _, current := range []string{parent, directory} {
		created := false
		if e := os.Mkdir(current, 0700); e == nil {
			created = true
		} else if !os.IsExist(e) {
			return "", fail(e)
		}
		if e := workspace.SecureDirectory(current, created); e != nil {
			return "", e
		}
	}
	return directory, nil
}

func loadRestoreProgress(directory, bundle, destination string, expected catalog.Publication) (catalog.Publication, error) {
	root, e := os.OpenRoot(directory)
	if e != nil {
		return expected, fail(e)
	}
	defer root.Close()
	key := expected.ID + ".json"
	if _, e = root.Lstat(key); os.IsNotExist(e) {
		return expected, nil
	}
	f, e := regular(root, key)
	if e != nil {
		return expected, e
	}
	defer f.Close()
	var progress restoreProgress
	raw, e := io.ReadAll(f)
	if e != nil {
		return expected, fail(e)
	}
	if strict(raw, &progress) != nil {
		return expected, contracts.Fail("invalid_request")
	}
	p := progress.Publication
	immutable := p
	immutable.Version = expected.Version
	immutable.UploadID = expected.UploadID
	immutable.Parts = expected.Parts
	immutable.PartSize = expected.PartSize
	immutable.CompletionRequested = expected.CompletionRequested
	immutable.State = expected.State
	identity, _ := digest(immutable)
	want, _ := digest(expected)
	if progress.Kind != "backup-restore-progress" || progress.Bundle != bundle || progress.Destination != destination || identity != want || (p.State != "pending" && p.State != "available") {
		return expected, contracts.Fail("conflict")
	}
	if len(p.Parts) > 10000 || p.PartSize < 0 || p.UploadID == "" && (p.PartSize != 0 || len(p.Parts) != 0) {
		return expected, contracts.Fail("invalid_request")
	}
	if p.UploadID != "" {
		size := int64(8 << 20)
		if p.Size > 0 {
			size = max(size, (p.Size-1)/10000+1)
		}
		if p.PartSize != size {
			return expected, contracts.Fail("invalid_request")
		}
		for i, part := range p.Parts {
			remaining := p.Size - int64(i)*size
			if remaining <= 0 || part.Number != i+1 || part.Size != min(size, remaining) || part.ETag == "" {
				return expected, contracts.Fail("invalid_request")
			}
		}
	}
	return p, nil
}

func saveRestoreProgress(directory, bundle, destination string, p catalog.Publication) (catalog.Publication, error) {
	raw, e := json.Marshal(restoreProgress{Kind: "backup-restore-progress", Bundle: bundle, Destination: destination, Publication: p})
	if e != nil {
		return p, contracts.Fail("invalid_request")
	}
	f, e := os.CreateTemp(directory, ".progress-*")
	if e != nil {
		return p, fail(e)
	}
	name := f.Name()
	defer os.Remove(name)
	_, e = f.Write(raw)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return p, fail(e)
	}
	if e = replaceConfiguration(name, filepath.Join(directory, p.ID+".json")); e != nil {
		return p, fail(e)
	}
	if e = syncDirectory(directory); e != nil {
		return p, fail(e)
	}
	return p, nil
}
