// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

func OpenStore(ctx context.Context, p workspace.Profile, control string, secrets contracts.SecretProvider) (Store, error) {
	raw, e := json.Marshal(p.Configuration)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	switch p.Adapter {
	case "filesystem":
		var c struct {
			Root string `json:"root"`
		}
		if json.Unmarshal(raw, &c) != nil || c.Root == "" {
			return nil, contracts.Fail("invalid_request")
		}
		if !filepath.IsAbs(c.Root) {
			c.Root = filepath.Join(control, c.Root)
		}
		return NewFilesystem(c.Root)
	case "s3":
		var c S3Config
		if json.Unmarshal(raw, &c) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return NewS3(ctx, c, secrets, nil)
	}
	return nil, contracts.Fail("unsupported_capability")
}
func WorkspaceProfile(p catalog.Profile) (workspace.Profile, error) {
	out := workspace.Profile{ID: p.ID, Revision: p.Revision, Adapter: p.Adapter, Version: p.Version}
	if p.Role != "storage" || json.Unmarshal(p.Configuration, &out.Configuration) != nil {
		return out, contracts.Fail("invalid_request")
	}
	if p.ExpectedBackendVersion != nil {
		out.ExpectedBackendVersion = *p.ExpectedBackendVersion
	}
	return out, nil
}

type routedStore struct {
	mu           sync.Mutex
	catalog      catalog.Catalog
	control      string
	secrets      contracts.SecretProvider
	defaultStore Store
	stores       map[string]Store
}

func profileKey(id string, revision int64) string { return id + ":" + strconv.FormatInt(revision, 10) }
func (r *routedStore) selected(ctx context.Context, p catalog.Publication) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := profileKey(p.ProfileID, p.ProfileRevision)
	if store := r.stores[key]; store != nil {
		return store, nil
	}
	record, e := r.catalog.Profile(ctx, p.ProfileID, p.ProfileRevision)
	if e != nil {
		return nil, e
	}
	profile, e := WorkspaceProfile(record)
	if e != nil {
		return nil, e
	}
	store, e := OpenStore(ctx, profile, r.control, r.secrets)
	if e == nil {
		r.stores[key] = store
	}
	return store, e
}
func (r *routedStore) Capabilities() Capabilities { return r.defaultStore.Capabilities() }
func (r *routedStore) Stat(ctx context.Context, p catalog.Publication) (Object, error) {
	s, e := r.selected(ctx, p)
	if e != nil {
		return Object{}, e
	}
	return s.Stat(ctx, p)
}
func (r *routedStore) OpenRange(ctx context.Context, p catalog.Publication, o, n int64) (io.ReadCloser, error) {
	s, e := r.selected(ctx, p)
	if e != nil {
		return nil, e
	}
	return s.OpenRange(ctx, p, o, n)
}
func (r *routedStore) PublishImmutable(ctx context.Context, p catalog.Publication, f *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	s, e := r.selected(ctx, p)
	if e != nil {
		return p, e
	}
	return s.PublishImmutable(ctx, p, f, save)
}
func (r *routedStore) Abort(ctx context.Context, p catalog.Publication) error {
	s, e := r.selected(ctx, p)
	if e != nil {
		return e
	}
	return s.Abort(ctx, p)
}
func (r *routedStore) DeleteUnreferenced(ctx context.Context, p catalog.Publication) error {
	s, e := r.selected(ctx, p)
	if e != nil {
		return e
	}
	return s.DeleteUnreferenced(ctx, p)
}
func (r *routedStore) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	var first error
	for _, s := range r.stores {
		if e := s.Close(); e != nil && first == nil {
			first = e
		}
	}
	return first
}
func (s *Service) Destination(ctx context.Context, id string, revision int64) error {
	if r, ok := s.Store.(*routedStore); ok {
		_, e := r.selected(ctx, catalog.Publication{ProfileID: id, ProfileRevision: revision})
		return e
	}
	if id == s.Workspace.Config.Profiles.Storage.ID && revision == s.Workspace.Config.Profiles.Storage.Revision {
		return nil
	}
	return contracts.Fail("unsupported_capability")
}
