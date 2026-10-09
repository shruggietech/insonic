// SPDX-License-Identifier: Apache-2.0
// Package backup transfers validated current workspace authority and immutable
// artifact bytes without starting processing workers or copying caches.
package backup

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
)

type CreateOptions struct {
	ID        string
	Directory string
	Mode      string
}
type Object struct {
	Publication catalog.Publication `json:"publication"`
	File        string              `json:"file,omitempty"`
}
type Manifest struct {
	Kind                  string              `json:"kind"`
	Version               string              `json:"schema_version"`
	ID                    string              `json:"id"`
	Mode                  string              `json:"mode"`
	WorkspaceID           string              `json:"workspace_id"`
	Revision              int64               `json:"catalog_revision"`
	CatalogDigest         string              `json:"catalog_digest"`
	SourceCatalogDigest   string              `json:"source_catalog_digest"`
	SourceCatalogRevision int64               `json:"source_catalog_revision"`
	CatalogSHA256         string              `json:"catalog_sha256"`
	Config                workspace.Config    `json:"workspace"`
	SourceProfiles        []workspace.Profile `json:"source_profiles"`
	Artifacts             []Object            `json:"artifacts"`
	GraphDisposition      string              `json:"graph_disposition"`
	Dependency            string              `json:"dependency"`
	Digest                string              `json:"sha256"`
}

func digest(v any) (string, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return "", contracts.Fail("invalid_request")
	}
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:]), nil
}
func manifestDigest(m Manifest) (string, error) { m.Digest = ""; return digest(m) }
func fail(e error) error {
	if e == nil {
		return nil
	}
	if _, ok := e.(*contracts.Error); ok {
		return e
	}
	return contracts.Fail("unavailable")
}

func strict(raw []byte, v any) error {
	if catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	d.DisallowUnknownFields()
	if d.Decode(v) != nil || d.Decode(new(any)) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}
func regular(root *os.Root, key string) (*os.File, error) {
	info, e := root.Lstat(key)
	if e != nil {
		return nil, fail(e)
	}
	if !info.Mode().IsRegular() {
		return nil, contracts.Fail("invalid_request")
	}
	f, e := openRegular(root, key)
	if e != nil {
		return nil, fail(e)
	}
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		f.Close()
		return nil, contracts.Fail("conflict")
	}
	return f, nil
}
func hashStream(ctx context.Context, w io.Writer, r io.Reader, size int64, expected string) error {
	h := sha256.New()
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if ctx.Err() != nil {
			return contracts.Fail("cancelled")
		}
		n, e := r.Read(buffer)
		if n > 0 {
			total += int64(n)
			if total > size {
				return contracts.Fail("conflict")
			}
			if _, we := io.MultiWriter(w, h).Write(buffer[:n]); we != nil {
				return fail(we)
			}
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return fail(e)
		}
	}
	if total != size || hex.EncodeToString(h.Sum(nil)) != expected {
		return contracts.Fail("conflict")
	}
	return nil
}
func openStorage(ctx context.Context, p workspace.Profile, control string, provider contracts.SecretProvider) (artifact.Store, error) {
	raw, e := json.Marshal(p.Configuration)
	if e != nil {
		return nil, contracts.Fail("invalid_request")
	}
	switch p.Adapter {
	case "filesystem":
		var c struct {
			Root string `json:"root"`
		}
		if strict(raw, &c) != nil || c.Root == "" {
			return nil, contracts.Fail("invalid_request")
		}
		if !filepath.IsAbs(c.Root) {
			c.Root = filepath.Join(control, c.Root)
		}
		return artifact.NewFilesystem(c.Root)
	case "s3":
		var c artifact.S3Config
		if strict(raw, &c) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return artifact.NewS3(ctx, c, provider, nil)
	}
	return nil, contracts.Fail("unsupported_capability")
}

// transfer renews independently while bytes move. Loss of catalog ownership
// cancels work; every final publication rechecks the fenced generation.
func transfer(ctx context.Context, store *catalog.Store, owner string) (context.Context, func(), error) {
	lease, e := store.BeginTransfer(ctx, owner, 30*time.Second)
	if e != nil {
		return nil, nil, e
	}
	work, cancel := context.WithCancel(catalog.WithTransfer(ctx, lease))
	done := make(chan struct{})
	stop := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-work.Done():
				return
			case <-ticker.C:
				if store.RenewTransfer(work, lease, 30*time.Second) != nil {
					cancel()
					return
				}
			}
		}
	}()
	finish := func() {
		close(stop)
		<-done
		end, halt := context.WithTimeout(catalog.WithTransfer(context.Background(), lease), 5*time.Second)
		defer halt()
		store.EndTransfer(end, lease)
		cancel()
	}
	return work, finish, nil
}

func writePrivate(root *os.Root, key string, raw []byte) error {
	f, e := root.OpenFile(key, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		return contracts.Fail("conflict")
	}
	_, e = f.Write(raw)
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		root.Remove(key)
	}
	return fail(e)
}

func Create(ctx context.Context, w *workspace.Workspace, store *catalog.Store, provider contracts.SecretProvider, o CreateOptions) (Manifest, error) {
	var m Manifest
	if o.Mode == "" {
		o.Mode = "self-contained"
	}
	if o.Mode != "self-contained" && o.Mode != "reference-only" || !contracts.ValidID(o.ID) || o.Directory == "" {
		return m, contracts.Fail("invalid_request")
	}
	// A completed same-identity retry is observational and cannot overwrite.
	if _, e := os.Lstat(o.Directory); e == nil {
		old, e := Inspect(ctx, o.Directory)
		if e == nil && old.ID == o.ID && old.WorkspaceID == w.Config.WorkspaceID && old.Mode == o.Mode {
			if old.Mode == "self-contained" {
				if e = Release(ctx, store, o.Directory); e != nil {
					return m, e
				}
			}
			return old, nil
		}
		return m, contracts.Fail("conflict")
	}
	ctx, finish, e := transfer(ctx, store, contracts.ID())
	if e != nil {
		return m, e
	}
	defer finish()
	snap, e := store.Export(ctx)
	if e != nil {
		return m, e
	}
	pubs, e := catalog.BackupPublications(snap)
	if e != nil {
		return m, e
	}
	// Pin both modes during capture. Self-contained pins release only after the
	// completed private bundle is durable; interruption retains recoverable pins.
	if e = store.PinBackup(ctx, o.ID, pubs); e != nil {
		return m, e
	}
	snap, e = store.Export(ctx)
	if e != nil {
		return m, e
	}
	originalRevision, originalDigest := snap.Revision, snap.Digest
	snap, e = catalog.PortableSnapshot(ctx, snap)
	if e != nil {
		return m, e
	}
	pubs, e = catalog.BackupPublications(snap)
	if e != nil {
		return m, e
	}
	m = Manifest{Kind: "workspace-backup", Version: contracts.Version, ID: o.ID, Mode: o.Mode, WorkspaceID: snap.WorkspaceID, Revision: snap.Revision, CatalogDigest: snap.Digest, Config: w.Config, Artifacts: []Object{}, SourceProfiles: []workspace.Profile{}, GraphDisposition: "accepted-evidence-rebuild", Dependency: "none"}
	m.SourceCatalogDigest = originalDigest
	m.SourceCatalogRevision = originalRevision
	m.Config.ControlDirectory = ".insonic"
	m.Config.CredentialNamespaceID = ""
	profiles := map[string]workspace.Profile{}
	for _, p := range snap.Records.Profiles {
		if p.Role != "storage" {
			continue
		}
		var configuration map[string]any
		if json.Unmarshal(p.Configuration, &configuration) != nil {
			return m, contracts.Fail("invalid_request")
		}
		profile := workspace.Profile{ID: p.ID, Revision: p.Revision, Adapter: p.Adapter, Version: p.Version, Configuration: configuration}
		if p.ExpectedBackendVersion != nil {
			profile.ExpectedBackendVersion = *p.ExpectedBackendVersion
		}
		if p.Adapter == "filesystem" {
			root, ok := configuration["root"].(string)
			if !ok {
				return m, contracts.Fail("invalid_request")
			}
			if !filepath.IsAbs(root) {
				configuration["root"] = filepath.Join(w.Control, root)
			}
		}
		profiles[p.ID+":"+strconv.FormatInt(p.Revision, 10)] = profile
	}
	used := map[string]bool{}
	for _, p := range pubs {
		key := p.ProfileID + ":" + strconv.FormatInt(p.ProfileRevision, 10)
		profile, ok := profiles[key]
		if !ok {
			return m, contracts.Fail("invalid_request")
		}
		if !used[key] {
			m.SourceProfiles = append(m.SourceProfiles, profile)
			used[key] = true
		}
		object := Object{Publication: p}
		if o.Mode == "self-contained" {
			object.File = "objects/" + p.Digest
		}
		m.Artifacts = append(m.Artifacts, object)
	}
	sort.Slice(m.SourceProfiles, func(i, j int) bool {
		return m.SourceProfiles[i].ID+":"+strconv.FormatInt(m.SourceProfiles[i].Revision, 10) < m.SourceProfiles[j].ID+":"+strconv.FormatInt(m.SourceProfiles[j].Revision, 10)
	})
	if o.Mode == "reference-only" {
		m.Dependency = "exact-source-store-objects-and-provider-retention"
	}
	parent := filepath.Dir(o.Directory)
	stage, e := os.MkdirTemp(parent, ".insonic-backup-*")
	if e != nil {
		return m, fail(e)
	}
	defer os.RemoveAll(stage)
	if e = workspace.SecureDirectory(stage, true); e != nil {
		return m, e
	}
	root, e := os.OpenRoot(stage)
	if e != nil {
		return m, fail(e)
	}
	defer root.Close()
	if e = root.Mkdir("objects", 0700); e != nil {
		return m, fail(e)
	}
	stores := map[string]artifact.Store{}
	defer func() {
		for _, s := range stores {
			s.Close()
		}
	}()
	copied := map[string]int64{}
	for _, object := range m.Artifacts {
		p := object.Publication
		key := p.ProfileID + ":" + strconv.FormatInt(p.ProfileRevision, 10)
		source := stores[key]
		if source == nil {
			source, e = openStorage(ctx, profiles[key], "", provider)
			if e != nil {
				return m, e
			}
			stores[key] = source
		}
		if o.Mode == "reference-only" {
			if e = artifact.Verify(ctx, source, p); e != nil {
				return m, e
			}
			continue
		}
		if size, ok := copied[p.Digest]; ok {
			if size != p.Size {
				return m, contracts.Fail("conflict")
			}
			continue
		}
		body, e := source.OpenRange(ctx, p, 0, p.Size)
		if e != nil {
			return m, e
		}
		f, e := root.OpenFile(object.File, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			body.Close()
			return m, fail(e)
		}
		e = hashStream(ctx, f, body, p.Size, p.Digest)
		body.Close()
		if e == nil {
			e = f.Sync()
		}
		ce := f.Close()
		if e == nil {
			e = ce
		}
		if e != nil {
			return m, fail(e)
		}
		copied[p.Digest] = p.Size
	}
	raw, e := json.Marshal(snap)
	if e != nil {
		return m, contracts.Fail("invalid_request")
	}
	h := sha256.Sum256(raw)
	m.CatalogSHA256 = hex.EncodeToString(h[:])
	if e = writePrivate(root, "catalog.json", raw); e != nil {
		return m, e
	}
	m.Digest, e = manifestDigest(m)
	if e != nil {
		return m, e
	}
	raw, e = json.Marshal(m)
	if e != nil {
		return m, contracts.Fail("invalid_request")
	}
	if e = writePrivate(root, "manifest.json", raw); e != nil {
		return m, e
	}
	// Test the staged completed representation before its public directory exists.
	if _, e = Inspect(ctx, stage); e != nil {
		return m, e
	}
	if _, e = store.Export(ctx); e != nil {
		return m, e
	}
	if ctx.Err() != nil {
		return m, contracts.Fail("cancelled")
	}
	root.Close()
	if e = syncDirectory(filepath.Join(stage, "objects")); e != nil {
		return m, fail(e)
	}
	if e = syncDirectory(stage); e != nil {
		return m, fail(e)
	}
	if e = publishDirectory(stage, o.Directory); e != nil {
		return m, contracts.Fail("conflict")
	}
	if e = syncDirectory(parent); e != nil {
		return m, fail(e)
	}
	if o.Mode == "self-contained" {
		if e = store.ReleaseBackup(ctx, o.ID, pubs); e != nil {
			return m, e
		}
	}
	return m, nil
}

func readBundle(ctx context.Context, directory string) (Manifest, catalog.Snapshot, error) {
	var m Manifest
	var snap catalog.Snapshot
	st, e := os.Lstat(directory)
	if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return m, snap, contracts.Fail("invalid_request")
	}
	root, e := os.OpenRoot(directory)
	if e != nil {
		return m, snap, fail(e)
	}
	defer root.Close()
	f, e := regular(root, "manifest.json")
	if e != nil {
		return m, snap, e
	}
	raw, e := io.ReadAll(f)
	f.Close()
	if e != nil || strict(raw, &m) != nil {
		return m, snap, contracts.Fail("invalid_request")
	}
	d, e := manifestDigest(m)
	if e != nil || d != m.Digest || m.Kind != "workspace-backup" || m.Version != contracts.Version || !contracts.ValidID(m.ID) || !contracts.ValidID(m.WorkspaceID) || m.Config.WorkspaceID != m.WorkspaceID || m.Config.Version != m.Version || m.GraphDisposition != "accepted-evidence-rebuild" || m.Mode != "self-contained" && m.Mode != "reference-only" {
		return m, snap, contracts.Fail("invalid_request")
	}
	configJSON, e := json.Marshal(m.Config)
	if e != nil || schemas.ValidateWorkspace(configJSON) != nil || m.Config.CredentialNamespaceID != "" {
		return m, snap, contracts.Fail("invalid_request")
	}
	f, e = regular(root, "catalog.json")
	if e != nil {
		return m, snap, e
	}
	h := sha256.New()
	snap, e = catalog.ReadSnapshotReader(io.TeeReader(f, h))
	f.Close()
	if e != nil {
		return m, snap, e
	}
	if hex.EncodeToString(h.Sum(nil)) != m.CatalogSHA256 || snap.Digest != m.CatalogDigest || snap.WorkspaceID != m.WorkspaceID || snap.Revision != m.Revision {
		return m, snap, contracts.Fail("conflict")
	}
	if m.SourceCatalogRevision < 0 || m.SourceCatalogRevision > m.Revision || len(m.SourceCatalogDigest) != 64 {
		return m, snap, contracts.Fail("invalid_request")
	}
	if _, e = hex.DecodeString(m.SourceCatalogDigest); e != nil {
		return m, snap, contracts.Fail("invalid_request")
	}
	if m.SourceCatalogDigest != m.CatalogDigest || m.SourceCatalogRevision != m.Revision {
		proofFound := false
		want := sha256.Sum256([]byte("restore:" + m.SourceCatalogDigest))
		for _, table := range snap.State {
			if table.Name != "operation_receipt" {
				continue
			}
			for _, row := range table.Rows {
				var receiptDigest, result string
				var revision int64
				if len(row) != 4 || strict(row[1], &receiptDigest) != nil || strict(row[2], &revision) != nil || strict(row[3], &result) != nil {
					return m, snap, contracts.Fail("invalid_request")
				}
				var proof struct {
					Revision *int64 `json:"restored_revision"`
				}
				if json.Unmarshal([]byte(result), &proof) != nil {
					return m, snap, contracts.Fail("invalid_request")
				}
				if receiptDigest == hex.EncodeToString(want[:]) && proof.Revision != nil && *proof.Revision == m.SourceCatalogRevision && revision == m.SourceCatalogRevision+1 {
					proofFound = true
				}
			}
		}
		if !proofFound {
			return m, snap, contracts.Fail("conflict")
		}
	}
	if e = catalog.ValidateSnapshot(ctx, snap); e != nil {
		return m, snap, e
	}
	expected, e := catalog.BackupPublications(snap)
	if e != nil {
		return m, snap, e
	}
	if len(expected) != len(m.Artifacts) {
		return m, snap, contracts.Fail("invalid_request")
	}
	profiles := map[string]workspace.Profile{}
	sourceProfiles := map[string]catalog.Profile{}
	for _, p := range snap.Records.Profiles {
		if p.Role == "storage" {
			sourceProfiles[p.ID+":"+strconv.FormatInt(p.Revision, 10)] = p
		}
	}
	for _, p := range m.SourceProfiles {
		key := p.ID + ":" + strconv.FormatInt(p.Revision, 10)
		if _, ok := profiles[key]; ok || !contracts.ValidID(p.ID) || p.Revision < 1 || p.Version != contracts.Version {
			return m, snap, contracts.Fail("invalid_request")
		}
		original, ok := sourceProfiles[key]
		if !ok || p.Adapter != original.Adapter || p.Version != original.Version {
			return m, snap, contracts.Fail("invalid_request")
		}
		var configuration map[string]any
		if json.Unmarshal(original.Configuration, &configuration) != nil {
			return m, snap, contracts.Fail("invalid_request")
		}
		comparison := map[string]any{}
		for k, v := range p.Configuration {
			comparison[k] = v
		}
		if p.Adapter == "filesystem" {
			rootValue, ok := comparison["root"].(string)
			if !ok || !filepath.IsAbs(rootValue) {
				return m, snap, contracts.Fail("invalid_request")
			}
			originalRoot, ok := configuration["root"].(string)
			if !ok || filepath.IsAbs(originalRoot) && originalRoot != rootValue {
				return m, snap, contracts.Fail("conflict")
			}
			delete(configuration, "root")
			delete(comparison, "root")
		}
		oldJSON, _ := json.Marshal(configuration)
		newJSON, _ := json.Marshal(comparison)
		if !bytes.Equal(oldJSON, newJSON) {
			return m, snap, contracts.Fail("conflict")
		}
		profiles[key] = p
	}
	for i, object := range m.Artifacts {
		p := object.Publication
		want := expected[i]
		publicationDigest, e := digest(p)
		if e != nil {
			return m, snap, e
		}
		expectedDigest, e := digest(want)
		if e != nil {
			return m, snap, e
		}
		if publicationDigest != expectedDigest || p.State != "available" || !p.References[m.ID] {
			return m, snap, contracts.Fail("conflict")
		}
		key := p.ProfileID + ":" + strconv.FormatInt(p.ProfileRevision, 10)
		if _, ok := profiles[key]; !ok {
			return m, snap, contracts.Fail("invalid_request")
		}
		if m.Mode == "reference-only" {
			if object.File != "" || m.Dependency != "exact-source-store-objects-and-provider-retention" {
				return m, snap, contracts.Fail("invalid_request")
			}
			continue
		}
		if object.File != "objects/"+p.Digest || m.Dependency != "none" {
			return m, snap, contracts.Fail("invalid_request")
		}
		f, e := regular(root, object.File)
		if e != nil {
			return m, snap, e
		}
		e = hashStream(ctx, io.Discard, f, p.Size, p.Digest)
		f.Close()
		if e != nil {
			return m, snap, e
		}
	}
	return m, snap, nil
}
func Inspect(ctx context.Context, directory string) (Manifest, error) {
	m, _, e := readBundle(ctx, directory)
	return m, e
}
func Verify(ctx context.Context, directory string, provider contracts.SecretProvider) (Manifest, error) {
	m, e := Inspect(ctx, directory)
	if e != nil || m.Mode != "reference-only" {
		return m, e
	}
	stores := map[string]artifact.Store{}
	defer func() {
		for _, s := range stores {
			s.Close()
		}
	}()
	profiles := map[string]workspace.Profile{}
	for _, p := range m.SourceProfiles {
		profiles[p.ID+":"+strconv.FormatInt(p.Revision, 10)] = p
	}
	for _, object := range m.Artifacts {
		p := object.Publication
		key := p.ProfileID + ":" + strconv.FormatInt(p.ProfileRevision, 10)
		s := stores[key]
		if s == nil {
			s, e = openStorage(ctx, profiles[key], "", provider)
			if e != nil {
				return m, e
			}
			stores[key] = s
		}
		if e = artifact.Verify(ctx, s, p); e != nil {
			return m, e
		}
	}
	return m, nil
}
func Release(ctx context.Context, store *catalog.Store, directory string) error {
	m, e := Inspect(ctx, directory)
	if e != nil {
		return e
	}
	status, e := store.Status(ctx)
	if e != nil {
		return e
	}
	if status["workspace_id"] != m.WorkspaceID {
		return contracts.Fail("workspace_mismatch")
	}
	publications := []catalog.Publication{}
	for _, o := range m.Artifacts {
		publications = append(publications, o.Publication)
	}
	return store.ReleaseBackup(ctx, m.ID, publications)
}

func ReleaseID(ctx context.Context, store *catalog.Store, id string) error {
	return store.ReleaseBackupID(ctx, id)
}

func RestoreWorkspace(ctx context.Context, w *workspace.Workspace, provider contracts.SecretProvider, directory string) (Manifest, error) {
	m, snap, e := readBundle(ctx, directory)
	if e != nil {
		return m, e
	}
	if m.Mode == "reference-only" {
		if _, e = Verify(ctx, directory, provider); e != nil {
			return m, e
		}
	}
	original, e := catalog.OpenWorkspace(ctx, w, provider, true)
	if e != nil {
		return m, e
	}
	defer original.Close()
	emptyErr := original.Empty(ctx)
	if emptyErr == nil {
		locked, finish, e := transfer(ctx, original, contracts.ID())
		if e != nil {
			return m, e
		}
		defer finish()
		ctx = catalog.WithoutTransfer(locked)
		if w.Config.WorkspaceID == m.WorkspaceID {
			ctx = locked
		}
	}
	selected := *w
	selected.Config = w.Config
	selected.Config.WorkspaceID = m.WorkspaceID
	selected.Config.CredentialNamespaceID = w.CredentialNamespace()
	store, e := catalog.OpenWorkspace(ctx, &selected, provider, true)
	if e != nil {
		return m, e
	}
	defer store.Close()
	restored, e := store.RestoredPortableWorkspace(ctx, snap.Digest, &selected)
	if e != nil {
		return m, e
	}
	if !restored && emptyErr != nil {
		return m, emptyErr
	}
	if restored && w.Config.WorkspaceID != m.WorkspaceID && emptyErr != nil {
		return m, emptyErr
	}
	var activation catalog.Lease
	progressDirectory, e := restoreProgressDirectory(&selected, m.ID)
	if e != nil {
		return m, e
	}
	destinationDigest, e := catalog.PortableDestinationDigest(&selected)
	if e != nil {
		return m, e
	}
	if !restored {
		if e = store.Empty(ctx); e != nil {
			return m, e
		}
		destination, e := openStorage(ctx, selected.Config.Profiles.Storage, selected.Control, provider)
		if e != nil {
			return m, e
		}
		defer destination.Close()
		root, e := os.OpenRoot(directory)
		if e != nil {
			return m, fail(e)
		}
		defer root.Close()
		stores := map[string]artifact.Store{}
		defer func() {
			for _, s := range stores {
				s.Close()
			}
		}()
		profiles := map[string]workspace.Profile{}
		for _, p := range m.SourceProfiles {
			profiles[p.ID+":"+strconv.FormatInt(p.Revision, 10)] = p
		}
		relocated := []catalog.Publication{}
		for _, object := range m.Artifacts {
			p := object.Publication
			var source *os.File
			if m.Mode == "self-contained" {
				source, e = regular(root, object.File)
				if e != nil {
					return m, e
				}
			} else {
				key := p.ProfileID + ":" + strconv.FormatInt(p.ProfileRevision, 10)
				s := stores[key]
				if s == nil {
					s, e = openStorage(ctx, profiles[key], "", provider)
					if e != nil {
						return m, e
					}
					stores[key] = s
				}
				body, e := s.OpenRange(ctx, p, 0, p.Size)
				if e != nil {
					return m, e
				}
				source, e = os.CreateTemp(selected.Control, ".restore-object-*")
				if e != nil {
					body.Close()
					return m, fail(e)
				}
				e = hashStream(ctx, source, body, p.Size, p.Digest)
				body.Close()
				if e == nil {
					_, e = source.Seek(0, io.SeekStart)
				}
				if e != nil {
					source.Close()
					os.Remove(source.Name())
					return m, e
				}
			}
			target := p
			target.References = map[string]bool{}
			for id, v := range p.References {
				if id != m.ID {
					target.References[id] = v
				}
			}
			target.ProfileID = selected.Config.Profiles.Storage.ID
			target.ProfileRevision = selected.Config.Profiles.Storage.Revision
			target.Key = "objects/" + m.WorkspaceID + "/" + target.ProfileID + "/" + strconv.FormatInt(target.ProfileRevision, 10) + "/restored/" + m.ID + "/" + p.ID
			target.Version = ""
			target.UploadID = ""
			target.Parts = nil
			target.PartSize = 0
			target.CompletionRequested = false
			target.State = "pending"
			target, e = loadRestoreProgress(progressDirectory, m.Digest, destinationDigest, target)
			if e == nil {
				target, e = destination.PublishImmutable(ctx, target, source, func(p catalog.Publication) (catalog.Publication, error) {
					return saveRestoreProgress(progressDirectory, m.Digest, destinationDigest, p)
				})
			}
			source.Close()
			if m.Mode == "reference-only" {
				os.Remove(source.Name())
			}
			if e != nil {
				return m, e
			}
			target.State = "available"
			if e = artifact.Verify(ctx, destination, target); e != nil {
				return m, e
			}
			if _, e = saveRestoreProgress(progressDirectory, m.Digest, destinationDigest, target); e != nil {
				return m, e
			}
			relocated = append(relocated, target)
		}
		lease, restoreErr := store.RestorePortableFenced(ctx, snap, &selected, relocated)
		if restoreErr != nil {
			return m, restoreErr
		}
		activation = lease
		if e = store.RenewTransfer(ctx, lease, 30*time.Second); e != nil {
			return m, e
		}
	} else {
		activation, e = store.BeginPortableActivation(ctx, snap.Digest, &selected)
		if e != nil {
			return m, e
		}
	}
	defer func() {
		end, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		store.EndTransfer(end, activation)
	}()
	// The selected configuration is the activation marker. Before this atomic
	// replacement the restored catalog scope remains passive and retryable.
	raw, e := json.MarshalIndent(selected.Config, "", "  ")
	if e != nil {
		return m, contracts.Fail("invalid_request")
	}
	configurationDirectory := filepath.Join(w.Root, ".insonic")
	f, e := os.CreateTemp(configurationDirectory, ".restore-workspace-*")
	if e != nil {
		return m, fail(e)
	}
	name := f.Name()
	defer os.Remove(name)
	_, e = f.Write(append(raw, '\n'))
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return m, fail(e)
	}
	if e = replaceConfiguration(name, filepath.Join(configurationDirectory, "workspace.json")); e != nil {
		return m, fail(e)
	}
	if e = syncDirectory(configurationDirectory); e != nil {
		return m, fail(e)
	}
	if e = store.CompletePortableActivation(ctx, activation, snap.Digest, &selected); e != nil {
		return m, e
	}
	w.Config = selected.Config
	for _, object := range m.Artifacts {
		os.Remove(filepath.Join(progressDirectory, object.Publication.ID+".json"))
	}
	os.Remove(progressDirectory)
	return m, nil
}
