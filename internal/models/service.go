// SPDX-License-Identifier: Apache-2.0
package models

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	Manifest Manifest `json:"manifest"`
}
type Service struct {
	Artifacts *artifact.Service
	Catalog   catalog.Catalog
	Secrets   contracts.SecretProvider
	Client    *http.Client
}

func NewService(a *artifact.Service, c catalog.Catalog, p contracts.SecretProvider) *Service {
	return &Service{a, c, p, &http.Client{Timeout: 10 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		_, invalid := contracts.SourceURL(req.URL.String())
		if len(via) > 4 || req.URL.Scheme != via[0].URL.Scheme || req.URL.Host != via[0].URL.Host || invalid != nil {
			return contracts.Fail("unavailable")
		}
		return nil
	}}}
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
	if d.Decode(&req) != nil || d.Decode(new(any)) != io.EOF || req.Manifest.Validate() != nil {
		return nil, contracts.Fail("invalid_request")
	}
	m := req.Manifest
	digest := m.Digest()
	logicalVersion, _ := json.Marshal([]string{m.Name, m.ModelVersion})
	id := StableID("model:" + string(logicalVersion))
	raw, _ := json.Marshal(m)
	install := catalog.BaseModelInstall{ID: id, Name: m.Name, Version: m.ModelVersion, Digest: digest, Manifest: raw, PublicationIDs: json.RawMessage(`[]`), State: "registered"}
	if current, e := s.Catalog.BaseModel(ctx, id); e == nil {
		if current.Digest != digest {
			return nil, contracts.Fail("conflict")
		}
		install.Revision = current.Revision
		if current.State == "available" {
			if e = s.Verify(ctx, id); e != nil {
				return nil, e
			}
			for _, f := range m.Files {
				if e = s.removeDownload(work.ID, f); e != nil {
					return nil, e
				}
			}
			return map[string]any{"model_id": id, "state": "available"}, nil
		}
	}
	if work.Kind == "models.acquire" {
		ids := []string{}
		for i, f := range m.Files {
			if ctx.Err() != nil {
				return nil, contracts.Fail("cancelled")
			}
			pubID := StableID("model-file:" + id + ":" + f.Role + ":" + f.SHA256)
			p, e := s.Catalog.Publication(ctx, pubID)
			if e == nil && p.State == "available" {
				if p.Digest != f.SHA256 || p.Size != f.Size || s.Artifacts.Verify(ctx, pubID) != nil {
					return nil, contracts.Fail("conflict")
				}
			} else {
				if e == nil && p.State == "pending" {
					p, e = s.Artifacts.Reconcile(ctx, pubID)
				}
				if e != nil || p.State != "available" {
					source, e := s.download(ctx, work.ID, f)
					if e != nil {
						return nil, e
					}
					p, e = s.Artifacts.Publish(ctx, pubID, source, "base-model")
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
			progress, _ := json.Marshal(map[string]any{"model_id": id, "files_verified": i + 1})
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
		if ctx.Err() != nil {
			return "", contracts.Fail("cancelled")
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
	for {
		if ctx.Err() != nil {
			return "", contracts.Fail("cancelled")
		}
		n, err := response.Body.Read(buffer)
		if int64(n) > remaining {
			return "", contracts.Fail("conflict")
		}
		if n > 0 {
			if _, e = file.Write(buffer[:n]); e != nil {
				return "", contracts.Fail("unavailable")
			}
			remaining -= int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			file.Sync()
			return "", contracts.Fail("unavailable")
		}
	}
	if remaining != 0 || file.Sync() != nil {
		return "", contracts.Fail("unavailable")
	}
	if verifyFile(file, f) != nil {
		return "", contracts.Fail("conflict")
	}
	return filepath.Join(dir, name), nil
}
func verifyFile(file *os.File, f File) error {
	if _, e := file.Seek(0, io.SeekStart); e != nil {
		return e
	}
	h := sha256.New()
	n, e := io.Copy(h, file)
	if e != nil || n != f.Size || hex.EncodeToString(h.Sum(nil)) != f.SHA256 {
		return contracts.Fail("conflict")
	}
	return nil
}
func (s *Service) Verify(ctx context.Context, id string) error {
	m, e := s.Catalog.BaseModel(ctx, id)
	if e != nil {
		return e
	}
	if m.State != "available" {
		return contracts.Fail("unavailable")
	}
	var req Manifest
	if DecodeManifest(m.Manifest, &req) != nil {
		return contracts.Fail("invalid_request")
	}
	var ids []string
	if json.Unmarshal(m.PublicationIDs, &ids) != nil || len(ids) != len(req.Files) {
		return contracts.Fail("conflict")
	}
	for i, p := range ids {
		pub, e := s.Catalog.Publication(ctx, p)
		if e != nil || pub.Digest != req.Files[i].SHA256 || pub.Size != req.Files[i].Size {
			return contracts.Fail("conflict")
		}
		if e = s.Artifacts.Verify(ctx, p); e != nil {
			return e
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
