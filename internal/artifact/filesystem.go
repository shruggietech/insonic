// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"github.com/gofrs/flock"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"io"
	"os"
	"path"
	"path/filepath"
)

type Filesystem struct {
	root    *os.Root
	lock    *flock.Flock
	persist func(*os.Root, string) error
}

func NewFilesystem(location string) (*Filesystem, error) {
	created := false
	if _, e := os.Lstat(location); os.IsNotExist(e) {
		if e = os.MkdirAll(location, 0700); e != nil {
			return nil, contracts.Fail("unavailable")
		}
		created = true
	}
	if e := workspace.SecureDirectory(location, created); e != nil {
		return nil, e
	}
	root, e := os.OpenRoot(location)
	if e != nil {
		return nil, contracts.Fail("unavailable")
	}
	return &Filesystem{root: root, lock: flock.New(filepath.Join(location, ".lifecycle.lock")), persist: persistObject}, nil
}
func (s *Filesystem) Close() error { s.lock.Close(); return s.root.Close() }
func (s *Filesystem) Capabilities() Capabilities {
	return Capabilities{Adapter: "filesystem", ConditionalCreate: true, RangedReads: true, Verification: "sha256-readback"}
}
func (s *Filesystem) Stat(ctx context.Context, p catalog.Publication) (Object, error) {
	if !validKey(p.Key) {
		return Object{}, contracts.Fail("invalid_request")
	}
	if p.State == "pending" {
		if e := s.persist(s.root, p.Key); e != nil {
			return Object{}, redact(ctx, e)
		}
	}
	f, e := s.root.Open(p.Key)
	if e != nil {
		return Object{}, redact(ctx, e)
	}
	defer f.Close()
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		return Object{}, contracts.Fail("unavailable")
	}
	return Object{Size: st.Size()}, nil
}

type sectionCloser struct {
	io.Reader
	io.Closer
}

func (s *Filesystem) OpenRange(ctx context.Context, p catalog.Publication, offset, length int64) (io.ReadCloser, error) {
	if !validKey(p.Key) || !validRange(p, offset, length) {
		return nil, contracts.Fail("invalid_request")
	}
	f, e := s.root.Open(p.Key)
	if e != nil {
		return nil, redact(ctx, e)
	}
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() || st.Size() != p.Size {
		f.Close()
		return nil, contracts.Fail("conflict")
	}
	return sectionCloser{contextReader{ctx, io.NewSectionReader(f, offset, length)}, f}, nil
}
func (s *Filesystem) PublishImmutable(ctx context.Context, p catalog.Publication, source *os.File, save func(catalog.Publication) (catalog.Publication, error)) (catalog.Publication, error) {
	if !validKey(p.Key) {
		return p, contracts.Fail("invalid_request")
	}
	if e := s.lock.Lock(); e != nil {
		return p, redact(ctx, e)
	}
	defer s.lock.Unlock()
	if _, e := s.root.Stat(p.Key); e == nil {
		if e = Verify(ctx, s, p); e != nil {
			return p, e
		}
		return p, nil
	}
	if e := s.root.MkdirAll(path.Dir(p.Key), 0700); e != nil {
		return p, redact(ctx, e)
	}
	stage := path.Dir(p.Key) + "/.stage-" + contracts.ID()
	f, e := s.root.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return p, redact(ctx, e)
	}
	defer s.root.Remove(stage)
	if _, e = source.Seek(0, 0); e == nil {
		_, e = copyContext(ctx, f, source)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return p, redact(ctx, e)
	}
	// Hard link publishes fully flushed bytes without ever replacing an object.
	if e = s.root.Link(stage, p.Key); e != nil {
		return p, redact(ctx, e)
	}
	if e = s.persist(s.root, p.Key); e != nil {
		return p, redact(ctx, e)
	}
	return p, Verify(ctx, s, p)
}
func (s *Filesystem) Abort(ctx context.Context, p catalog.Publication) error {
	if !validKey(p.Key) {
		return contracts.Fail("invalid_request")
	}
	if _, e := s.root.Stat(p.Key); e == nil {
		return contracts.Fail("conflict")
	} else if !os.IsNotExist(e) {
		return redact(ctx, e)
	}
	return nil
}
func (s *Filesystem) DeleteUnreferenced(ctx context.Context, p catalog.Publication) error {
	if p.State != "retiring" || !validKey(p.Key) {
		return contracts.Fail("conflict")
	}
	if e := s.lock.Lock(); e != nil {
		return redact(ctx, e)
	}
	defer s.lock.Unlock()
	e := s.root.Remove(p.Key)
	if os.IsNotExist(e) {
		return nil
	}
	return redact(ctx, e)
}
