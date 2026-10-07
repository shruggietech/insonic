// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/gofrs/flock"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type Service struct {
	Workspace          *workspace.Workspace
	Catalog            catalog.Catalog
	Store              Store
	Owner              string
	Grace              time.Duration
	TTL                time.Duration
	MaxMaterialization int64
	scratch            *os.Root
	scratchPath        string
	mu                 sync.Mutex
	active             map[string]transfer
}
type transfer struct {
	cancel context.CancelFunc
	done   chan struct{}
}
type Materialization struct {
	PublicationID string                       `json:"publication_id"`
	Path          string                       `json:"path"`
	Lease         catalog.MaterializationLease `json:"lease"`
}

func NewService(ctx context.Context, w *workspace.Workspace, db catalog.Catalog, secrets contracts.SecretProvider, owner string) (*Service, error) {
	profile := w.Config.Profiles.Storage
	var store Store
	var e error
	raw, e := json.Marshal(profile.Configuration)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	switch profile.Adapter {
	case "filesystem":
		var c struct {
			Root string `json:"root"`
		}
		if json.Unmarshal(raw, &c) != nil || c.Root == "" {
			return nil, contracts.Fail("invalid_request")
		}
		location := c.Root
		if !filepath.IsAbs(location) {
			location = filepath.Join(w.Control, location)
		}
		store, e = NewFilesystem(location)
	case "s3":
		var c S3Config
		if json.Unmarshal(raw, &c) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		store, e = NewS3(ctx, c, secrets, nil)
	default:
		return nil, contracts.Fail("unavailable")
	}
	if e != nil {
		return nil, e
	}
	dir := filepath.Join(w.Control, "artifact-scratch")
	created := false
	if e = os.Mkdir(dir, 0700); e == nil {
		created = true
	} else if !os.IsExist(e) {
		store.Close()
		return nil, contracts.Fail("unavailable")
	}
	if e = workspace.SecureDirectory(dir, created); e != nil {
		store.Close()
		return nil, e
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		store.Close()
		return nil, contracts.Fail("unavailable")
	}
	return &Service{Workspace: w, Catalog: db, Store: store, Owner: owner, Grace: time.Minute, TTL: 30 * time.Second, MaxMaterialization: 10 << 30, scratch: root, scratchPath: dir, active: map[string]transfer{}}, nil
}
func (s *Service) Close() error { s.scratch.Close(); return s.Store.Close() }
func (s *Service) checkProfile(p catalog.Publication) error {
	profile := s.Workspace.Config.Profiles.Storage
	if p.ProfileID != profile.ID || p.ProfileRevision != int64(profile.Revision) {
		return contracts.Fail("conflict")
	}
	return nil
}
func (s *Service) selected(ctx context.Context, id string) (catalog.Publication, error) {
	p, e := s.Catalog.Publication(ctx, id)
	if e == nil {
		e = s.checkProfile(p)
	}
	return p, e
}
func (s *Service) Stage(ctx context.Context, source string) (*os.File, string, int64, error) {
	if !filepath.IsAbs(source) {
		return nil, "", 0, contracts.Fail("invalid_request")
	}
	input, e := openSource(source)
	if e != nil {
		return nil, "", 0, contracts.Fail("unavailable")
	}
	defer input.Close()
	name := "stage-" + contracts.ID()
	f, e := s.scratch.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e != nil {
		return nil, "", 0, redact(ctx, e)
	}
	h := sha256.New()
	n, e := copyContext(ctx, io.MultiWriter(f, h), input)
	if e == nil {
		e = f.Sync()
	}
	if e != nil {
		f.Close()
		s.scratch.Remove(name)
		return nil, "", 0, redact(ctx, e)
	}
	f.Seek(0, 0)
	return f, hex.EncodeToString(h.Sum(nil)), n, nil
}

func openSource(source string) (*os.File, error) {
	f, e := openNoFollow(source)
	if e != nil {
		return nil, e
	}
	opened, e := f.Stat()
	if e != nil || !opened.Mode().IsRegular() {
		f.Close()
		return nil, contracts.Fail("conflict")
	}
	return f, nil
}
func (s *Service) Publish(ctx context.Context, id, source, kind string) (catalog.Publication, error) {
	if !contracts.ValidID(id) || kind == "" || len(kind) > 128 {
		return catalog.Publication{}, contracts.Fail("invalid_request")
	}
	stage, digest, size, e := s.Stage(ctx, source)
	if e != nil {
		return catalog.Publication{}, e
	}
	defer stage.Close()
	defer s.scratch.Remove(filepath.Base(stage.Name()))
	// Local per-operation locking serializes one runtime's callers; the catalog
	// claim fences owners on other hosts and survives process crashes.
	lock := flock.New(filepath.Join(s.scratchPath, id+".lock"))
	if e = lock.Lock(); e != nil {
		return catalog.Publication{}, contracts.Fail("unavailable")
	}
	defer lock.Unlock()
	defer lock.Close()
	profile := s.Workspace.Config.Profiles.Storage
	p := catalog.Publication{ID: id, ArtifactID: id, LocationID: id, ProfileID: profile.ID, ProfileRevision: profile.Revision, Digest: digest, Size: size, Kind: kind, Key: "objects/" + s.Workspace.Config.WorkspaceID + "/" + profile.ID + "/" + strconv.FormatInt(profile.Revision, 10) + "/sha256/" + digest[:2] + "/" + digest + "/" + id, Owner: s.Owner}
	p, e = s.Catalog.BeginPublication(ctx, p, s.TTL)
	if e != nil {
		return p, e
	}
	if p.State != "pending" {
		return p, nil
	}
	durable := id + ".source"
	recovering := false
	if old, e := s.scratch.Open(durable); e == nil {
		old.Close()
		recovering = true
	} else if os.IsNotExist(e) {
		if e = s.scratch.Link(filepath.Base(stage.Name()), durable); e != nil {
			return p, contracts.Fail("unavailable")
		}
	} else {
		return p, contracts.Fail("unavailable")
	}
	sourceFile, e := s.scratch.Open(durable)
	if e != nil {
		return p, contracts.Fail("unavailable")
	}
	defer sourceFile.Close()
	p, e = s.upload(ctx, p, sourceFile, recovering)
	if e == nil {
		s.scratch.Remove(durable)
	}
	return p, e
}

// Renewal and journal checkpoints share the same mutex so a renewal can never
// replace a newer multipart receipt with its earlier in-memory copy.
func (s *Service) upload(ctx context.Context, p catalog.Publication, source *os.File, recovering bool) (catalog.Publication, error) {
	work, cancel := context.WithCancel(ctx)
	defer cancel()
	s.mu.Lock()
	if _, exists := s.active[p.ID]; exists {
		s.mu.Unlock()
		return p, contracts.Fail("conflict")
	}
	transferDone := make(chan struct{})
	s.active[p.ID] = transfer{cancel, transferDone}
	s.mu.Unlock()
	defer func() { s.mu.Lock(); delete(s.active, p.ID); close(transferDone); s.mu.Unlock() }()
	var mu sync.Mutex
	current := p
	var renewErr error
	done := make(chan struct{})
	joined := make(chan struct{})
	go func() {
		defer close(joined)
		ticker := time.NewTicker(s.TTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-work.Done():
				return
			case <-ticker.C:
				mu.Lock()
				var next catalog.Publication
				next, renewErr = s.Catalog.SavePublication(work, current, s.TTL)
				if renewErr == nil {
					current = next
				} else {
					cancel()
				}
				mu.Unlock()
			}
		}
	}()
	save := func(next catalog.Publication) (catalog.Publication, error) {
		mu.Lock()
		defer mu.Unlock()
		if renewErr != nil {
			return next, renewErr
		}
		next.LeaseUntil = current.LeaseUntil
		out, e := s.Catalog.SavePublication(work, next, s.TTL)
		if e == nil {
			current = out
		}
		return out, e
	}
	result := p
	var e error
	verified := false
	if recovering {
		if st, err := s.Store.Stat(work, p); err == nil {
			result.Version = st.Version
			verified = Verify(work, s.Store, result) == nil
		}
		if !verified {
			if source == nil {
				e = contracts.Fail("unavailable")
			} else {
				h := sha256.New()
				var n int64
				n, e = copyContext(work, h, source)
				if e == nil && (n != p.Size || hex.EncodeToString(h.Sum(nil)) != p.Digest) {
					e = contracts.Fail("conflict")
				}
			}
		}
	}
	if e == nil && !verified {
		result, e = s.Store.PublishImmutable(work, p, source, save)
	}
	if e == nil {
		e = Verify(work, s.Store, result)
	}
	close(done)
	<-joined
	if e != nil {
		return result, e
	}
	mu.Lock()
	e = renewErr
	mu.Unlock()
	if e != nil {
		return result, e
	}
	// Verify after upload independently; no adapter/provider checksum is trusted.
	result.Verification = "sha256-readback"
	return s.Catalog.AdmitPublication(work, result)
}
func (s *Service) Reconcile(ctx context.Context, id string) (catalog.Publication, error) {
	p, e := s.selected(ctx, id)
	if e != nil || p.State != "pending" {
		return p, e
	}
	p.Owner = s.Owner
	p, e = s.Catalog.BeginPublication(ctx, p, s.TTL)
	if e != nil {
		return p, e
	}
	source, e := s.scratch.Open(id + ".source")
	if e != nil && !os.IsNotExist(e) {
		return p, contracts.Fail("unavailable")
	}
	if source != nil {
		defer source.Close()
	}
	p, e = s.upload(ctx, p, source, true)
	if e == nil {
		s.scratch.Remove(id + ".source")
	}
	return p, e
}
func (s *Service) Abort(ctx context.Context, id string) (catalog.Publication, error) {
	s.mu.Lock()
	active, exists := s.active[id]
	if exists {
		active.cancel()
	}
	s.mu.Unlock()
	if exists {
		select {
		case <-active.done:
		case <-ctx.Done():
			return catalog.Publication{}, contracts.Fail("cancelled")
		}
	}
	p, e := s.selected(ctx, id)
	if e != nil {
		return p, e
	}
	if p.State == "aborted" {
		return p, nil
	}
	if p.State != "pending" {
		return p, contracts.Fail("conflict")
	}
	p.Owner = s.Owner
	p, e = s.Catalog.BeginPublication(ctx, p, s.TTL)
	if e != nil {
		return p, e
	}
	cleanup, stop := context.WithTimeout(ctx, 20*time.Second)
	defer stop()
	if e = s.Store.Abort(cleanup, p); e != nil {
		return p, e
	}
	p, e = s.Catalog.AbortPublication(ctx, p)
	if e == nil {
		s.scratch.Remove(id + ".source")
	}
	return p, e
}
func (s *Service) Verify(ctx context.Context, id string) error {
	p, e := s.selected(ctx, id)
	if e != nil {
		return e
	}
	if p.State != "available" {
		return contracts.Fail("unavailable")
	}
	return Verify(ctx, s.Store, p)
}
func (s *Service) Materialize(ctx context.Context, id string) (Materialization, error) {
	return s.MaterializeBound(ctx, id, s.MaxMaterialization)
}
func (s *Service) MaterializeBound(ctx context.Context, id string, maxBytes int64) (Materialization, error) {
	p, e := s.selected(ctx, id)
	if e != nil {
		return Materialization{}, e
	}
	if maxBytes < 0 || p.Size > maxBytes {
		return Materialization{}, contracts.Fail("output_limit")
	}
	leaseID := contracts.ID()
	lease, e := s.Catalog.ArtifactLease(ctx, id, leaseID, leaseID, s.TTL, false)
	if e != nil {
		return Materialization{}, e
	}
	work, finish := s.maintain(ctx, func(current context.Context) error {
		_, e := s.Catalog.RenewArtifactLease(current, id, leaseID, leaseID, s.TTL)
		return e
	})
	defer finish()
	name := leaseID + ".cache"
	success := false
	defer func() {
		if !success {
			cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			s.Catalog.ArtifactLease(cleanup, id, leaseID, leaseID, 0, true)
			if e := s.scratch.Remove(name); e == nil || os.IsNotExist(e) {
				s.Catalog.ForgetMaterialization(cleanup, id, leaseID)
			}
		}
	}()
	f, e := s.scratch.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return Materialization{}, redact(ctx, e)
	}
	body, e := s.Store.OpenRange(work, p, 0, p.Size)
	if e != nil {
		f.Close()
		return Materialization{}, e
	}
	h := sha256.New()
	n, e := copyContext(work, io.MultiWriter(f, h), body)
	body.Close()
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return Materialization{}, redact(ctx, e)
	}
	if n != p.Size || hex.EncodeToString(h.Sum(nil)) != p.Digest {
		return Materialization{}, contracts.Fail("conflict")
	}
	if e = finish(); e != nil {
		return Materialization{}, e
	}
	lease, e = s.Catalog.RenewArtifactLease(ctx, id, leaseID, leaseID, s.TTL)
	if e != nil {
		return Materialization{}, e
	}
	success = true
	return Materialization{id, filepath.Join(s.scratchPath, name), lease}, nil
}
func (s *Service) Renew(ctx context.Context, id, lease string) (catalog.MaterializationLease, error) {
	if _, e := s.selected(ctx, id); e != nil {
		return catalog.MaterializationLease{}, e
	}
	return s.Catalog.RenewArtifactLease(ctx, id, lease, lease, s.TTL)
}
func (s *Service) Release(ctx context.Context, id, lease string) error {
	if _, e := s.selected(ctx, id); e != nil {
		return e
	}
	_, e := s.Catalog.ArtifactLease(ctx, id, lease, lease, 0, true)
	if e == nil {
		e = s.scratch.Remove(lease + ".cache")
		if os.IsNotExist(e) {
			e = nil
		}
		if e == nil {
			e = s.Catalog.ForgetMaterialization(ctx, id, lease)
		}
	}
	return redact(ctx, e)
}
func (s *Service) Prune(ctx context.Context, id string) ([]string, error) {
	if _, e := s.selected(ctx, id); e != nil {
		return nil, e
	}
	ids, e := s.Catalog.ExpireMaterializations(ctx, id)
	if e != nil {
		return nil, e
	}
	for _, lease := range ids {
		if e = s.scratch.Remove(lease + ".cache"); e != nil && !os.IsNotExist(e) {
			return ids, redact(ctx, e)
		}
		if e = s.Catalog.ForgetMaterialization(ctx, id, lease); e != nil {
			return ids, e
		}
	}
	return ids, nil
}
func (s *Service) Retire(ctx context.Context, id string) (catalog.Publication, error) {
	p, e := s.selected(ctx, id)
	if e != nil {
		return p, e
	}
	p, e = s.Catalog.ClaimRetirement(ctx, id, s.Owner, s.Grace, s.TTL)
	if e != nil || p.State == "retired" {
		return p, e
	}
	if e = s.Store.DeleteUnreferenced(ctx, p); e != nil {
		return p, e
	}
	return s.Catalog.FinishRetirement(ctx, p)
}
