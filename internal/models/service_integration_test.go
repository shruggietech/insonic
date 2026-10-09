// SPDX-License-Identifier: Apache-2.0
package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func modelServiceFixture(t *testing.T) (*Service, *catalog.Store) {
	t.Helper()
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "models")
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	a, e := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Close(); db.Close() })
	return NewService(a, db, nil), db
}
func modelWork(t *testing.T, db catalog.Catalog, kind string, m Manifest) catalog.Work {
	t.Helper()
	payload, _ := json.Marshal(Request{Manifest: m})
	w, e := db.EnqueueWork(context.Background(), contracts.ID(), kind, payload)
	if e != nil {
		t.Fatal(e)
	}
	w, e = db.ClaimWork(context.Background(), w.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	return w
}
func localModelFile(role, url string, body []byte) File {
	sum := sha256.Sum256(body)
	return File{Role: role, URL: url, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(body)), LocalHTTP: true}
}
func completedModel(t *testing.T, s *Service, w catalog.Work) string {
	t.Helper()
	result, e := s.Execute(context.Background(), w)
	if e != nil {
		t.Fatal(e)
	}
	id, ok := result.(map[string]any)["model_id"].(string)
	if !ok {
		t.Fatal("missing installed model")
	}
	return id
}
func TestModelAcquisitionVerifiesAllFilesAndMaterializes(t *testing.T) {
	s, db := modelServiceFixture(t)
	ctx := context.Background()
	bodies := map[string][]byte{"/weights": []byte("model fixture"), "/tokenizer": []byte("tokenizer fixture")}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, ok := bodies[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(b)))
		w.Write(b)
	}))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", bodies["/weights"]), localModelFile("tokenizer", server.URL+"/tokenizer", bodies["/tokenizer"])}
	work := modelWork(t, db, "models.acquire", m)
	id := completedModel(t, s, work)
	current, e := db.BaseModel(ctx, id)
	if e != nil || current.State != "available" {
		t.Fatal("model unavailable", e)
	}
	ids, e := catalog.PublicationIDs(current.PublicationIDs)
	if e != nil || len(ids) != 2 {
		t.Fatal("missing required publications", e)
	}
	if e = s.Verify(ctx, id); e != nil {
		t.Fatal(e)
	}
	materialized, e := s.Materialize(ctx, id)
	if e != nil || len(materialized) != 2 {
		t.Fatal("materialize", e)
	}
	for i, path := range materialized {
		b, e := os.ReadFile(path.Path)
		if e != nil || int64(len(b)) != m.Files[i].Size {
			t.Fatal("wrong model bytes", e)
		}
		if e = s.Artifacts.Release(ctx, path.PublicationID, path.Lease.ID); e != nil {
			t.Fatal(e)
		}
	}
	snapshot, e := db.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	dest, e := catalog.OpenSQLite(ctx, filepath.Join(t.TempDir(), "restored.sqlite"), s.Artifacts.Workspace.Config.WorkspaceID)
	if e != nil {
		t.Fatal(e)
	}
	defer dest.Close()
	if e = dest.Restore(ctx, snapshot); e != nil {
		t.Fatal("real model manifest snapshot rejected", e)
	}
	restored, e := dest.BaseModel(ctx, id)
	expectedManifest, _ := json.Marshal(m)
	if e != nil || string(restored.Manifest) != string(expectedManifest) || restored.Digest != m.Digest() {
		t.Fatal("manifest bytes/digest changed across snapshot", e)
	}
	same := completedModel(t, s, work)
	if same != id {
		t.Fatal("same operation changed model identity")
	}
	list, e := db.BaseModels(ctx)
	if e != nil || len(list) != 1 {
		t.Fatal("duplicate install", e)
	}
}
func TestModelPartialDownloadResumesAcrossRestart(t *testing.T) {
	s, db := modelServiceFixture(t)
	ctx := context.Background()
	body := []byte("model fixture")
	var requests, rangeRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		if n == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
			w.Write(body[:5])
			return
		}
		if r.Header.Get("Range") != "bytes=5-" {
			t.Errorf("resume ignored persisted fragment: %q", r.Header.Get("Range"))
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		rangeRequests.Add(1)
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 5-%d/%d", len(body)-1, len(body)))
		w.Header().Set("Content-Length", strconv.Itoa(len(body)-5))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(body[5:])
	}))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	work := modelWork(t, db, "models.acquire", m)
	if _, e := s.Execute(ctx, work); e == nil {
		t.Fatal("incomplete response accepted")
	}
	list, e := db.BaseModels(ctx)
	if e != nil || len(list) != 0 {
		t.Fatal("partial bytes marked available", e)
	}
	if e = db.InterruptOwner(ctx, work.Owner); e != nil {
		t.Fatal(e)
	}
	workspace := s.Artifacts.Workspace
	s.Artifacts.Close()
	db.Close()
	reopened, e := catalog.OpenWorkspace(ctx, workspace, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	a, e := artifact.NewService(ctx, workspace, reopened, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { a.Close(); reopened.Close() })
	next := NewService(a, reopened, nil)
	claim, e := reopened.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	id := completedModel(t, next, claim)
	if e = next.Verify(ctx, id); e != nil || rangeRequests.Load() != 1 {
		t.Fatal("resume failed", e)
	}
}
func TestLateModelDigestFailureLeavesNoAvailableInstall(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.Write([]byte(strings.Repeat("x", len(body))))
			return
		}
		w.Write(body)
	}))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/good", body), localModelFile("config", server.URL+"/bad", body)}
	work := modelWork(t, db, "models.acquire", m)
	if _, e := s.Execute(context.Background(), work); e == nil {
		t.Fatal("late hash failure accepted")
	}
	list, e := db.BaseModels(context.Background())
	if e != nil || len(list) != 0 {
		t.Fatal("partially verified model available", e)
	}
}
func TestCancelledModelClaimCannotPublishInstallation(t *testing.T) {
	s, db := modelServiceFixture(t)
	ctx := context.Background()
	body := []byte("model fixture")
	started, release := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-release; w.Write(body) }))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	work := modelWork(t, db, "models.acquire", m)
	done := make(chan error, 1)
	go func() { _, e := s.Execute(ctx, work); done <- e }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("download did not start")
	}
	if _, e := db.CancelWork(ctx, contracts.ID(), work.ID); e != nil {
		t.Fatal(e)
	}
	close(release)
	if e := <-done; e == nil {
		t.Fatal("cancelled claim admitted model")
	}
	list, e := db.BaseModels(ctx)
	if e != nil || len(list) != 0 {
		t.Fatal("cancelled work published references", e)
	}
}
func TestContextCancellationRetainsResumableBytes(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Write(body[:5])
		w.(http.Flusher).Flush()
		close(started)
		<-r.Context().Done()
	}))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	work := modelWork(t, db, "models.acquire", m)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := s.Execute(ctx, work); done <- e }()
	<-started
	cancel()
	if e := <-done; e == nil {
		t.Fatal("cancelled download succeeded")
	}
	path := filepath.Join(s.Artifacts.Workspace.Control, "model-scratch", work.ID+"-"+m.Files[0].SHA256+".download")
	st, e := os.Stat(path)
	if e != nil || st.Size() > m.Files[0].Size {
		t.Fatal("unbounded cancellation fragment", e)
	}
}

type uncertainModelUpload struct{ artifact.Store }

func (s uncertainModelUpload) PublishImmutable(ctx context.Context, p catalog.Publication, source *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	p, e := s.Store.PublishImmutable(ctx, p, source, save)
	if e == nil {
		e = contracts.Fail("unavailable")
	}
	return p, e
}

func TestModelUncertainUploadRecoveryRemovesCompletedDownload(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.Write(body) }))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	work := modelWork(t, db, "models.acquire", m)
	store := s.Artifacts.Store
	s.Artifacts.Store = uncertainModelUpload{store}
	if _, e := s.Execute(context.Background(), work); e == nil {
		t.Fatal("lost upload response was not injected")
	}
	path := filepath.Join(s.Artifacts.Workspace.Control, "model-scratch", work.ID+"-"+m.Files[0].SHA256+".download")
	if _, e := os.Stat(path); e != nil {
		t.Fatal("recovery download missing", e)
	}
	s.Artifacts.Store = store
	id := completedModel(t, s, work)
	if e := s.Verify(context.Background(), id); e != nil {
		t.Fatal(e)
	}
	if requests.Load() != 1 {
		t.Fatal("reconciled object was downloaded again")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("completed recovery retained duplicate model bytes", e)
	}
}

func TestAvailableModelReplayRechecksStoredBytes(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	work := modelWork(t, db, "models.acquire", m)
	id := completedModel(t, s, work)
	current, e := db.BaseModel(context.Background(), id)
	if e != nil {
		t.Fatal(e)
	}
	ids, e := catalog.PublicationIDs(current.PublicationIDs)
	if e != nil {
		t.Fatal(e)
	}
	p, e := db.Publication(context.Background(), ids[0])
	if e != nil {
		t.Fatal(e)
	}
	storage := s.Artifacts.Workspace.Config.Profiles.Storage.Configuration["root"].(string)
	if !filepath.IsAbs(storage) {
		storage = filepath.Join(s.Artifacts.Workspace.Control, storage)
	}
	if e = os.WriteFile(filepath.Join(storage, filepath.FromSlash(p.Key)), []byte(strings.Repeat("x", len(body))), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Execute(context.Background(), work); e == nil {
		t.Fatal("available replay accepted corrupted model bytes")
	}
	if _, e = s.Materialize(context.Background(), id); e == nil {
		t.Fatal("materialized corrupted model bytes")
	}
}

func TestModelAcquireRetryRecoversCorruptAndMissingBytes(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(fmt.Sprint(missing), func(t *testing.T) {
			s, db := modelServiceFixture(t)
			ctx := context.Background()
			body := []byte("recovery model fixture")
			var downloads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { downloads.Add(1); w.Write(body) }))
			defer server.Close()
			m := fixtureManifest()
			m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
			work := modelWork(t, db, "models.acquire", m)
			id := completedModel(t, s, work)
			current, e := db.BaseModel(ctx, id)
			if e != nil {
				t.Fatal(e)
			}
			ids, _ := catalog.PublicationIDs(current.PublicationIDs)
			p, e := db.Publication(ctx, ids[0])
			if e != nil {
				t.Fatal(e)
			}
			storage := s.Artifacts.Workspace.Config.Profiles.Storage.Configuration["root"].(string)
			if !filepath.IsAbs(storage) {
				storage = filepath.Join(s.Artifacts.Workspace.Control, storage)
			}
			path := filepath.Join(storage, filepath.FromSlash(p.Key))
			if missing {
				e = os.Remove(path)
			} else {
				e = os.WriteFile(path, []byte(strings.Repeat("x", len(body))), 0600)
			}
			if e != nil {
				t.Fatal(e)
			}
			if s.Verify(ctx, id) == nil {
				t.Fatal("corruption accepted")
			}
			if _, e = db.CheckpointWork(ctx, work, "complete", "succeeded", json.RawMessage(`{"state":"available"}`), time.Minute); e != nil {
				t.Fatal(e)
			}
			retry, e := db.RetryWork(ctx, contracts.ID(), work.ID)
			if e != nil {
				t.Fatal(e)
			}
			claim, e := db.ClaimWork(ctx, retry.ID, contracts.ID(), time.Minute)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = s.Execute(ctx, claim); e != nil {
				t.Fatal("recover exact bundle", e)
			}
			if s.Verify(ctx, id) != nil || downloads.Load() != 2 {
				t.Fatal("verified recovery", downloads.Load())
			}
			recovered, e := db.BaseModel(ctx, id)
			if e != nil || recovered.Digest != current.Digest || recovered.Revision <= current.Revision {
				t.Fatal("immutable manifest changed", e)
			}
			newIDs, _ := catalog.PublicationIDs(recovered.PublicationIDs)
			if newIDs[0] == ids[0] {
				t.Fatal("overwrote immutable publication")
			}
			if _, e = s.Execute(ctx, work); e == nil {
				t.Fatal("stale generation republished")
			}
		})
	}
}

func TestModelServiceRejectsUnknownWorkKindEvenAfterInstallation(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(body) }))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/weights", body)}
	work := modelWork(t, db, "models.acquire", m)
	completedModel(t, s, work)
	work.Kind = "media.import"
	if _, e := s.Execute(context.Background(), work); e == nil {
		t.Fatal("wrong service operation accepted")
	}
}

func TestModelRedirectRejectsFragmentBeforeFetchingTarget(t *testing.T) {
	s, db := modelServiceFixture(t)
	body := []byte("model fixture")
	var fetched atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/weights#credential", http.StatusFound)
			return
		}
		fetched.Store(true)
		w.Write(body)
	}))
	defer server.Close()
	m := fixtureManifest()
	m.Files = []File{localModelFile("weights", server.URL+"/redirect", body)}
	work := modelWork(t, db, "models.acquire", m)
	if _, e := s.Execute(context.Background(), work); e == nil {
		t.Fatal("fragment redirect accepted")
	}
	if fetched.Load() {
		t.Fatal("forbidden redirect target fetched")
	}
}
