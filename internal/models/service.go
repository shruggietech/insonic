// SPDX-License-Identifier: Apache-2.0
package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Request struct {
	Manifest       Manifest           `json:"manifest"`
	LocalDirectory string             `json:"local_directory,omitempty"`
	Destination    *workspace.Profile `json:"destination,omitempty"`
}
type Service struct {
	Artifacts *artifact.Service
	Catalog   catalog.Catalog
	Secrets   contracts.SecretProvider
	Client    *http.Client
}

func NewService(a *artifact.Service, c catalog.Catalog, p contracts.SecretProvider) *Service {
	return &Service{a, c, p, &http.Client{Timeout: 10 * time.Minute, CheckRedirect: boundedRedirect}}
}
func (s *Service) Execute(ctx context.Context, work catalog.Work) (any, error) {
	if work.Kind != "models.register" && work.Kind != "models.acquire" {
		return nil, contracts.Fail("invalid_request")
	}
	var req Request
	if catalog.ValidateJSON(work.Payload) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(work.Payload))
	d.DisallowUnknownFields()
	if d.Decode(&req) != nil || d.Decode(new(any)) != io.EOF || req.Manifest.Validate() != nil || req.LocalDirectory != "" && !filepath.IsAbs(req.LocalDirectory) {
		return nil, contracts.Fail("invalid_request")
	}
	m := req.Manifest
	// Replays may inspect an accepted installation, but stale/cancelled claims
	// cannot perform scratch cleanup or new recovery effects.
	if _, e := s.Catalog.RenewWork(ctx, work, 30*time.Second); e != nil {
		return nil, e
	}
	digest := m.Digest()
	id := InstallationID(m)
	raw, _ := json.Marshal(m)
	install := catalog.BaseModelInstall{ID: id, Name: m.Name, Version: m.ModelVersion, Digest: digest, Manifest: raw, PublicationIDs: json.RawMessage(`[]`), State: "registered"}
	if current, e := s.Catalog.BaseModel(ctx, id); e == nil {
		if current.Digest != digest {
			return nil, contracts.Fail("conflict")
		}
		install.Revision = current.Revision
		if current.State == "available" {
			verifyErr := s.Verify(ctx, id)
			if verifyErr != nil && work.Kind == "models.register" {
				return nil, verifyErr
			}
			if verifyErr == nil {
				for _, f := range m.Files {
					if e = s.removeDownload(work.ID, f); e != nil {
						return nil, e
					}
				}
				return map[string]any{"model_id": id, "state": "available"}, nil
			}
		}
	}
	if work.Kind == "models.acquire" {
		var total, complete int64
		for _, f := range m.Files {
			total += f.Size
		}
		ids := []string{}
		var currentIDs []string
		if old, e := s.Catalog.BaseModel(ctx, id); e == nil {
			currentIDs, _ = catalog.PublicationIDs(old.PublicationIDs)
		}
		for i, f := range m.Files {
			if ctx.Err() != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					return nil, contracts.Fail("acquisition_timeout")
				}
				return nil, contracts.Fail("cancelled")
			}
			pubID := StableID("model-file:" + id + ":" + f.Role + ":" + f.SHA256)
			if len(currentIDs) == len(m.Files) {
				pubID = currentIDs[i]
			}
			p, e := s.Catalog.Publication(ctx, pubID)
			if e == nil && p.State != "pending" {
				if p.Digest != f.SHA256 || p.Size != f.Size {
					return nil, contracts.Fail("conflict")
				}
				if p.State != "available" || s.Artifacts.Verify(ctx, pubID) != nil {
					pubID = StableID("model-file-recovery:" + id + ":" + f.Role + ":" + f.SHA256 + ":" + work.ID + ":" + strconv.FormatInt(work.Generation, 10))
					p, e = s.Catalog.Publication(ctx, pubID)
				}
			}
			if e != nil || p.State != "available" {
				if e == nil && p.State == "pending" {
					p, e = s.Artifacts.Reconcile(ctx, pubID)
				}
				if e != nil || p.State != "available" {
					var source string
					if req.LocalDirectory != "" {
						if !ValidRole(f.Role) {
							return nil, contracts.Fail("invalid_request")
						}
						source = filepath.Join(req.LocalDirectory, filepath.FromSlash(f.Role))
						var local *os.File
						// Cache snapshots commonly use links into a content-addressed
						// blob directory. Elect the resolved regular file for this read;
						// publication still verifies its complete manifest identity.
						source, e = filepath.EvalSymlinks(source)
						if e == nil {
							local, e = os.Open(source)
						}
						if e == nil {
							e = verifyFile(local, f)
							local.Close()
						}
					} else {
						source, e = s.downloadProgress(ctx, work.ID, f, func(n int64) error {
							raw, _ := json.Marshal(map[string]any{"model_id": id, "files_verified": i, "current_role": f.Role, "downloaded_bytes": complete + n, "total_bytes": total})
							_, e := s.Catalog.CheckpointWork(ctx, work, "downloading", "running", raw, 5*time.Second)
							return e
						})
					}
					if e != nil {
						return nil, e
					}
					if req.Destination != nil {
						p, e = s.Artifacts.PublishTo(ctx, *req.Destination, pubID, source, "base-model")
					} else {
						p, e = s.Artifacts.Publish(ctx, pubID, source, "base-model")
					}
					if e != nil {
						return nil, e
					}
					if p.State != "available" || p.Digest != f.SHA256 || p.Size != f.Size {
						return nil, contracts.Fail("conflict")
					}
				}
			}
			if p.State != "available" || p.Digest != f.SHA256 || p.Size != f.Size {
				return nil, contracts.Fail("conflict")
			}
			if e = s.removeDownload(work.ID, f); e != nil {
				return nil, e
			}
			ids = append(ids, pubID)
			complete += f.Size
			progress, _ := json.Marshal(map[string]any{"model_id": id, "files_verified": i + 1, "downloaded_bytes": complete, "total_bytes": total, "current_role": f.Role})
			if _, e = s.Catalog.CheckpointWork(ctx, work, "verified-files", "running", progress, 5*time.Second); e != nil {
				return nil, e
			}
		}
		install.PublicationIDs, _ = json.Marshal(ids)
		install.State = "available"
	}
	got, e := s.Catalog.CommitBaseModel(ctx, work, install)
	if e != nil {
		return nil, e
	}
	return map[string]any{"model_id": got.ID, "state": got.State, "manifest_digest": got.Digest}, nil
}

// A verified publication owns the model bytes. Remove its temporary source on
// fresh uploads and reconciliation alike, including replay after admission.
func (s *Service) removeDownload(workID string, f File) error {
	if !contracts.ValidID(workID) || !digestPattern.MatchString(f.SHA256) {
		return contracts.Fail("invalid_request")
	}
	root, e := os.OpenRoot(filepath.Join(s.Artifacts.Workspace.Control, "model-scratch"))
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer root.Close()
	if e = root.Remove(workID + "-" + f.SHA256 + ".download"); e != nil && !os.IsNotExist(e) {
		return contracts.Fail("unavailable")
	}
	return nil
}

func (s *Service) download(ctx context.Context, workID string, f File) (string, error) {
	return s.downloadProgress(ctx, workID, f, nil)
}

// DownloadAsset shares bounded resumable transfer with model acquisition.
// The caller's durable work owns progress and election; no model is registered.
func (s *Service) DownloadAsset(ctx context.Context, workID string, f File, progress func(int64) error) (string, error) {
	return s.downloadProgress(ctx, workID, f, progress)
}

func (s *Service) downloadProgress(ctx context.Context, workID string, f File, progress func(int64) error) (string, error) {
	if !contracts.ValidID(workID) || !digestPattern.MatchString(f.SHA256) {
		return "", contracts.Fail("invalid_request")
	}
	u, e := sourceURL(f.URL, f.LocalHTTP)
	if e != nil {
		return "", e
	}
	dir := filepath.Join(s.Artifacts.Workspace.Control, "model-scratch")
	created := false
	if e = os.Mkdir(dir, 0700); e == nil {
		created = true
	} else if !os.IsExist(e) {
		return "", contracts.Fail("unavailable")
	}
	if workspace.SecureDirectory(dir, created) != nil {
		return "", contracts.Fail("unavailable")
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return "", contracts.Fail("unavailable")
	}
	defer root.Close()
	name := workID + "-" + f.SHA256 + ".download"
	if st, e := root.Lstat(name); e == nil && !st.Mode().IsRegular() {
		return "", contracts.Fail("conflict")
	}
	file, e := root.OpenFile(name, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		return "", contracts.Fail("unavailable")
	}
	defer file.Close()
	st, e := file.Stat()
	if e != nil || st.Size() > f.Size {
		return "", contracts.Fail("conflict")
	}
	offset := st.Size()
	if offset == f.Size && verifyFile(file, f) == nil {
		if progress != nil {
			if e = progress(f.Size); e != nil {
				return "", e
			}
		}
		return filepath.Join(dir, name), nil
	}
	if offset == f.Size {
		if file.Truncate(0) != nil {
			return "", contracts.Fail("unavailable")
		}
		offset = 0
	}
	req, e := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if e != nil {
		return "", contracts.Fail("invalid_request")
	}
	req.Header.Set("Accept-Encoding", "identity")
	if offset > 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(offset, 10)+"-")
	}
	if f.CredentialID != "" {
		if s.Secrets == nil {
			return "", contracts.Fail("unavailable")
		}
		raw, e := s.Secrets.Resolve(ctx, f.CredentialID)
		if e != nil {
			return "", contracts.Fail("unavailable")
		}
		token := string(raw)
		if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
			var auth struct {
				Token string `json:"token"`
			}
			dec := json.NewDecoder(bytes.NewReader(raw))
			dec.DisallowUnknownFields()
			if catalog.ValidateJSON(raw) != nil || dec.Decode(&auth) != nil || dec.Decode(new(any)) != io.EOF {
				clear(raw)
				return "", contracts.Fail("invalid_request")
			}
			token = auth.Token
		}
		clear(raw)
		if token == "" || strings.ContainsAny(token, "\r\n") {
			return "", contracts.Fail("invalid_request")
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, e := s.Client.Do(req)
	if e != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return "", contracts.Fail("cancelled")
		}
		var timeout interface{ Timeout() bool }
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(e, &timeout) && timeout.Timeout() {
			return "", contracts.Fail("acquisition_timeout")
		}
		return "", contracts.Fail("unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		offset = 0
		if file.Truncate(0) != nil {
			return "", contracts.Fail("unavailable")
		}
	} else if response.StatusCode == http.StatusPartialContent && offset > 0 {
		expected := fmt.Sprintf("bytes %d-%d/%d", offset, f.Size-1, f.Size)
		if response.Header.Get("Content-Range") != expected {
			return "", contracts.Fail("conflict")
		}
	} else {
		return "", contracts.Fail("unavailable")
	}
	if response.Header.Get("Content-Encoding") != "" && response.Header.Get("Content-Encoding") != "identity" {
		return "", contracts.Fail("conflict")
	}
	if _, e = file.Seek(offset, io.SeekStart); e != nil {
		return "", contracts.Fail("unavailable")
	}
	buffer := make([]byte, 128<<10)
	remaining := f.Size - offset
	lastProgress := time.Time{}
	for {
		if ctx.Err() != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return "", contracts.Fail("acquisition_timeout")
			}
			return "", contracts.Fail("cancelled")
		}
		n, err := response.Body.Read(buffer)
		if int64(n) > remaining {
			return "", contracts.Fail("input_limit")
		}
		if n > 0 {
			if _, e = file.Write(buffer[:n]); e != nil {
				return "", contracts.Fail("unavailable")
			}
			remaining -= int64(n)
			if progress != nil && time.Since(lastProgress) >= 250*time.Millisecond {
				if e = progress(f.Size - remaining); e != nil {
					return "", e
				}
				lastProgress = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			file.Sync()
			var timeout interface{ Timeout() bool }
			if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.As(err, &timeout) && timeout.Timeout() {
				return "", contracts.Fail("acquisition_timeout")
			}
			if errors.Is(ctx.Err(), context.Canceled) {
				return "", contracts.Fail("cancelled")
			}
			return "", contracts.Fail("unavailable")
		}
	}
	if remaining != 0 || file.Sync() != nil {
		return "", contracts.Fail("unavailable")
	}
	if e = verifyFile(file, f); e != nil {
		return "", e
	}
	if progress != nil {
		if e = progress(f.Size); e != nil {
			return "", e
		}
	}
	return filepath.Join(dir, name), nil
}
func verifyFile(file *os.File, f File) error {
	if _, e := file.Seek(0, io.SeekStart); e != nil {
		return contracts.Fail("unavailable")
	}
	h := sha256.New()
	n, e := io.Copy(h, file)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	if n != f.Size || hex.EncodeToString(h.Sum(nil)) != f.SHA256 {
		return contracts.Fail("checksum_mismatch")
	}
	return nil
}

// VerifyLocal checks every declared role without modifying the selected tree.
func VerifyLocal(ctx context.Context, directory string, m Manifest) error {
	if !filepath.IsAbs(directory) || m.Validate() != nil {
		return contracts.Fail("invalid_request")
	}
	for _, expected := range m.Files {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		if !ValidRole(expected.Role) {
			return contracts.Fail("invalid_request")
		}
		file, e := os.Open(filepath.Join(directory, filepath.FromSlash(expected.Role)))
		if e != nil {
			return contracts.Fail("unavailable")
		}
		st, e := file.Stat()
		if e == nil && st.Mode().IsRegular() {
			e = verifyFile(file, expected)
		} else {
			e = contracts.Fail("invalid_request")
		}
		file.Close()
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *Service) Verify(ctx context.Context, id string) error {
	m, e := s.Catalog.BaseModel(ctx, id)
	if e != nil {
		return e
	}
	if m.State != "available" {
		return contracts.Condition("unavailable", "model_not_acquired")
	}
	var req Manifest
	if DecodeManifest(m.Manifest, &req) != nil {
		return contracts.Condition("conflict", "model_damaged")
	}
	var ids []string
	if json.Unmarshal(m.PublicationIDs, &ids) != nil || len(ids) != len(req.Files) {
		return contracts.Condition("conflict", "model_damaged")
	}
	for i, p := range ids {
		pub, e := s.Catalog.Publication(ctx, p)
		if e != nil {
			return contracts.Condition("unavailable", "model_storage_unavailable")
		}
		if pub.Digest != req.Files[i].SHA256 || pub.Size != req.Files[i].Size {
			return contracts.Condition("conflict", "model_damaged")
		}
		if e = s.Artifacts.Verify(ctx, p); e != nil {
			var typed *contracts.Error
			if errors.As(e, &typed) && (typed.Code == "checksum_mismatch" || typed.Code == "conflict") {
				return contracts.Condition(typed.Code, "model_damaged")
			}
			return contracts.Condition("unavailable", "model_storage_unavailable")
		}
	}
	return nil
}
func (s *Service) Materialize(ctx context.Context, id string) ([]artifact.Materialization, error) {
	if e := s.Verify(ctx, id); e != nil {
		return nil, e
	}
	m, e := s.Catalog.BaseModel(ctx, id)
	if e != nil {
		return nil, e
	}
	var ids []string
	json.Unmarshal(m.PublicationIDs, &ids)
	out := []artifact.Materialization{}
	for _, id := range ids {
		p, e := s.Artifacts.Materialize(ctx, id)
		if e != nil {
			for _, old := range out {
				s.Artifacts.Release(context.Background(), old.PublicationID, old.Lease.ID)
			}
			return nil, e
		}
		out = append(out, p)
	}
	return out, nil
}
