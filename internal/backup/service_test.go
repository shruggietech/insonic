// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func fixture(t *testing.T) (*workspace.Workspace, *catalog.Store, *artifact.Service) {
	t.Helper()
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "Portable fixture")
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, true)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	s, e := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close(); db.Close() })
	return w, db, s
}

func TestRestoreActivationFailureRemainsFencedAndRecovers(t *testing.T) {
	ctx := context.Background()
	w, db, _ := fixture(t)
	dir := filepath.Join(t.TempDir(), "bundle")
	if _, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir}); e != nil {
		t.Fatal(e)
	}
	target, e := workspace.Init(t.TempDir(), "Activation target")
	if e != nil {
		t.Fatal(e)
	}
	marker := filepath.Join(target.Root, ".insonic", "workspace.json")
	original, e := os.ReadFile(marker)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(marker); e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(marker, 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(marker, "blocked"), []byte("injected activation failure"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e == nil {
		t.Fatal("activation failure accepted")
	}
	passive := *target
	passive.Config = target.Config
	passive.Config.WorkspaceID = w.Config.WorkspaceID
	passive.Config.CredentialNamespaceID = target.CredentialNamespace()
	store, e := catalog.OpenWorkspace(ctx, &passive, nil, true)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.PutNamedSetting(ctx, contracts.ID(), "passive", 0, json.RawMessage(`{}`)); e == nil {
		t.Fatal("failed activation became writable after operational lease release")
	}
	if _, e = store.BeginTransfer(ctx, contracts.ID(), time.Second); e == nil {
		t.Fatal("ordinary transfer bypassed pending activation")
	}
	wrong := passive
	wrong.Config = passive.Config
	wrong.Config.Profiles.Storage.ID = contracts.ID()
	m, e := Inspect(ctx, dir)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.BeginPortableActivation(ctx, m.CatalogDigest, &wrong); e == nil {
		t.Fatal("different destination profile reconciled pending activation")
	}
	relocatedRoot := passive
	relocatedRoot.Control = t.TempDir()
	if _, e = store.BeginPortableActivation(ctx, m.CatalogDigest, &relocatedRoot); e == nil {
		t.Fatal("different physical local destination reconciled pending activation")
	}
	store.Close()
	if e = os.Remove(filepath.Join(marker, "blocked")); e != nil {
		t.Fatal(e)
	}
	if e = os.Remove(marker); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(marker, original, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
		t.Fatal("activation recovery", e)
	}
	store, e = catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if _, e = store.PutNamedSetting(ctx, contracts.ID(), "active", 0, json.RawMessage(`{}`)); e != nil {
		t.Fatal("completed activation remained fenced", e)
	}
	if _, e = store.Export(ctx); e != nil {
		t.Fatal("activation proof invalid", e)
	}
}

func TestRestoreEmptyTargetWithSameWorkspaceIdentity(t *testing.T) {
	ctx := context.Background()
	w, db, _ := fixture(t)
	dir := filepath.Join(t.TempDir(), "bundle")
	if _, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir}); e != nil {
		t.Fatal(e)
	}
	target, e := workspace.Init(t.TempDir(), "Same identity")
	if e != nil {
		t.Fatal(e)
	}
	target.Config.CredentialNamespaceID = target.CredentialNamespace()
	target.Config.WorkspaceID = w.Config.WorkspaceID
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
		t.Fatal(e)
	}
	store, e := catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	if _, e = store.PutNamedSetting(ctx, contracts.ID(), "same-identity-active", 0, json.RawMessage(`{}`)); e != nil {
		t.Fatal(e)
	}
}

func TestRestoredAuthorityCanBeBackedUpAndRestoredAgain(t *testing.T) {
	ctx := context.Background()
	w, db, _ := fixture(t)
	first := filepath.Join(t.TempDir(), "first")
	if _, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: first}); e != nil {
		t.Fatal(e)
	}
	target, e := workspace.Init(t.TempDir(), "First target")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = RestoreWorkspace(ctx, target, nil, first); e != nil {
		t.Fatal(e)
	}
	store, e := catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	second := filepath.Join(t.TempDir(), "second")
	if _, e = Create(ctx, target, store, nil, CreateOptions{ID: contracts.ID(), Directory: second}); e != nil {
		t.Fatal("second backup", e)
	}
	final, e := workspace.Init(t.TempDir(), "Second target")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = RestoreWorkspace(ctx, final, nil, second); e != nil {
		t.Fatal("second restore", e)
	}
	finalStore, e := catalog.OpenWorkspace(ctx, final, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer finalStore.Close()
	if _, e = finalStore.PutNamedSetting(ctx, contracts.ID(), "active-again", 0, json.RawMessage(`{}`)); e != nil {
		t.Fatal(e)
	}
	if _, e = finalStore.Export(ctx); e != nil {
		t.Fatal(e)
	}
}

func TestInterruptedCapturePinsRecoverByIDAndRetry(t *testing.T) {
	ctx := context.Background()
	w, db, s := fixture(t)
	p := publish(t, s, "source-recoverable", "canonical-audio")
	owner := p
	owner.State = "retiring"
	if e := s.Store.DeleteUnreferenced(ctx, owner); e != nil {
		t.Fatal(e)
	}
	id := contracts.ID()
	dir := filepath.Join(t.TempDir(), "retry")
	if _, e := Create(ctx, w, db, nil, CreateOptions{ID: id, Directory: dir}); e == nil {
		t.Fatal("missing source accepted")
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("failed capture advertised")
	}
	retained, e := db.Publication(ctx, p.ID)
	if e != nil || !retained.References[id] {
		t.Fatal("lost interrupted lifetime pin", e)
	}
	source, e := os.CreateTemp(t.TempDir(), "restored-source")
	if e != nil {
		t.Fatal(e)
	}
	source.Write([]byte("source-recoverable"))
	source.Seek(0, 0)
	republish := p
	republish.State = "pending"
	if _, e = s.Store.PublishImmutable(ctx, republish, source, func(v catalog.Publication) (catalog.Publication, error) { return v, nil }); e != nil {
		t.Fatal(e)
	}
	source.Close()
	if _, e = Create(ctx, w, db, nil, CreateOptions{ID: id, Directory: dir}); e != nil {
		t.Fatal("retry after source recovery", e)
	}
	retained, e = db.Publication(ctx, p.ID)
	if e != nil || retained.References[id] {
		t.Fatal("completed recovery pin leaked", e)
	}
	other := contracts.ID()
	snapshot, e := db.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	publications, e := catalog.BackupPublications(snapshot)
	if e != nil {
		t.Fatal(e)
	}
	if e = db.PinBackup(ctx, other, publications); e != nil {
		t.Fatal(e)
	}
	if e = ReleaseID(ctx, db, other); e != nil {
		t.Fatal("release incomplete ID", e)
	}
	if e = ReleaseID(ctx, db, contracts.ID()); e == nil {
		t.Fatal("unproven backup ID released")
	}
}

func TestRestoreDropsAllSourceBackupPinsAndPreservesManualReferences(t *testing.T) {
	ctx := context.Background()
	w, db, s := fixture(t)
	p := publish(t, s, "explicit-retention", "canonical-audio")
	manual := contracts.ID()
	if e := db.ArtifactReference(ctx, p.ID, manual, false); e != nil {
		t.Fatal(e)
	}
	firstDir := filepath.Join(t.TempDir(), "first")
	first, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: firstDir, Mode: "reference-only"})
	if e != nil {
		t.Fatal(e)
	}
	secondDir := filepath.Join(t.TempDir(), "second")
	second, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: secondDir})
	if e != nil {
		t.Fatal(e)
	}
	target := destination(t)
	if _, e = RestoreWorkspace(ctx, target, nil, secondDir); e != nil {
		t.Fatal(e)
	}
	restored, e := catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	got, e := restored.Publication(ctx, p.ID)
	if e != nil || got.References[first.ID] || got.References[second.ID] || !got.References[manual] {
		t.Fatal("backup and manual retention conflated", e, got.References)
	}
}

func TestBundleOmitsUnacceptedLocalInputsWithoutMutatingSource(t *testing.T) {
	ctx := context.Background()
	w, db, _ := fixture(t)
	input := `A:\private pending input\source.wav`
	payload, _ := json.Marshal(map[string]any{"items": []any{map[string]any{"source": input}}})
	work, e := db.EnqueueWork(ctx, contracts.ID(), "media.import", payload)
	if e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(t.TempDir(), "portable")
	m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(filepath.Join(dir, "catalog.json"))
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(raw), "private pending input") {
		t.Fatal("pending original path copied")
	}
	if m.SourceCatalogRevision >= m.Revision || m.SourceCatalogDigest == m.CatalogDigest {
		t.Fatal("source capture not bound to transformed authority")
	}
	original, e := db.Work(ctx, work.ID)
	if e != nil || string(original.Payload) != string(work.Payload) || original.State != "pending" {
		t.Fatal("source mutated", e)
	}
	if _, e = RestoreWorkspace(ctx, destination(t), nil, dir); e != nil {
		t.Fatal("sanitized authority invalid", e)
	}
}
func publish(t *testing.T, s *artifact.Service, data, kind string) catalog.Publication {
	t.Helper()
	path := filepath.Join(t.TempDir(), "original ü.file")
	if e := os.WriteFile(path, []byte(data), 0600); e != nil {
		t.Fatal(e)
	}
	p, e := s.Publish(context.Background(), contracts.ID(), path, kind)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func destination(t *testing.T) *workspace.Workspace {
	t.Helper()
	w, e := workspace.Init(t.TempDir(), "Recovered")
	if e != nil {
		t.Fatal(e)
	}
	return w
}

func TestCompleteBackupRestoresWithoutSourceAndResetsGraph(t *testing.T) {
	ctx := context.Background()
	w, db, s := fixture(t)
	p := publish(t, s, "canonical-track-data", "canonical-audio")
	model := publish(t, s, "durable-model-bytes", "model")
	scratch := publish(t, s, "disposable-input", "inference-preparation")
	if _, e := db.PutNamedSetting(ctx, contracts.ID(), "precision", 0, json.RawMessage(`{"unix_ns":1781029324123456789,"state":"unknown"}`)); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(t.TempDir(), "complete")
	m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
	if e != nil {
		t.Fatal("create", e)
	}
	if len(m.Artifacts) != 2 {
		t.Fatal("wrong current manifest", len(m.Artifacts))
	}
	if _, e = Inspect(ctx, dir); e != nil {
		t.Fatal(e)
	}
	if _, e = Create(ctx, w, db, nil, CreateOptions{ID: m.ID, Directory: dir}); e != nil {
		t.Fatal("idempotent", e)
	}
	if _, e = Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir}); e == nil {
		t.Fatal("overwrote bundle")
	}
	// Source artifacts become unavailable independently of the retained bundle.
	p.State = "retiring"
	model.State = "retiring"
	if e = s.Store.DeleteUnreferenced(ctx, p); e != nil {
		t.Fatal(e)
	}
	if e = s.Store.DeleteUnreferenced(ctx, model); e != nil {
		t.Fatal(e)
	}
	target := destination(t)
	oldID := target.Config.WorkspaceID
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
		t.Fatal("restore", e)
	}
	if target.Config.WorkspaceID == oldID || target.Config.WorkspaceID != w.Config.WorkspaceID {
		t.Fatal("identity activation")
	}
	restored, e := catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	svc, e := artifact.NewService(ctx, target, restored, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer svc.Close()
	for _, want := range []catalog.Publication{p, model} {
		if e = svc.Verify(ctx, want.ID); e != nil {
			t.Fatal("restored bytes", e)
		}
		got, e := restored.Publication(ctx, want.ID)
		if e != nil || got.ProfileID != target.Config.Profiles.Storage.ID || got.Digest != want.Digest || got.References[m.ID] {
			t.Fatal("relocation identity or source-only pin", e)
		}
	}
	missing, e := restored.Publication(ctx, scratch.ID)
	if e != nil || missing.State != "missing" {
		t.Fatal("transient bytes restored", e)
	}
	setting, _, e := restored.NamedSetting(ctx, "precision")
	if e != nil || string(setting) != `{"state":"unknown","unix_ns":1781029324123456789}` {
		t.Fatal("integer changed", string(setting), e)
	}
	status, e := restored.GraphStatus(ctx)
	if e != nil || status["checkpoint"] != 0 || status["pending_events"] == 0 {
		t.Fatal("graph authority not reset", e, status)
	}
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
		t.Fatal("activation retry", e)
	}
}

func TestEmptyBackup(t *testing.T) {
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "empty")
	if e != nil {
		t.Fatal(e)
	}
	s, e := catalog.OpenWorkspace(ctx, w, nil, true)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	snap, e := s.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = catalog.ValidateSnapshot(ctx, snap); e != nil {
		t.Fatal("initial snapshot", e)
	}
	lease, e := s.BeginTransfer(ctx, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	snap, e = s.Export(catalog.WithTransfer(ctx, lease))
	if e != nil {
		t.Fatal(e)
	}
	if e = catalog.ValidateSnapshot(ctx, snap); e != nil {
		t.Fatal("leased snapshot", e)
	}
	s.EndTransfer(ctx, lease)
	if _, e = Create(ctx, w, s, nil, CreateOptions{ID: contracts.ID(), Directory: filepath.Join(t.TempDir(), "empty")}); e != nil {
		t.Fatal("create", e)
	}
}

func TestBackupCorruptionLeavesTargetInactive(t *testing.T) {
	ctx := context.Background()
	w, db, s := fixture(t)
	publish(t, s, "immutable", "canonical-audio")
	dir := filepath.Join(t.TempDir(), "bundle")
	m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
	if e != nil {
		t.Fatal(e)
	}
	target := destination(t)
	original := target.Config.WorkspaceID
	if e = os.WriteFile(filepath.Join(dir, m.Artifacts[0].File), []byte("corrupted"), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e == nil {
		t.Fatal("corrupt bytes accepted")
	}
	reopened, e := workspace.Open(target.Root)
	if e != nil || reopened.Config.WorkspaceID != original {
		t.Fatal("target authority changed", e)
	}
	if _, e = Inspect(ctx, dir); e == nil {
		t.Fatal("corrupt verify passed")
	}
}

func TestReferenceOnlyPinsReleaseAndDependencyLoss(t *testing.T) {
	ctx := context.Background()
	w, db, s := fixture(t)
	p := publish(t, s, "pinned", "canonical-audio")
	dir := filepath.Join(t.TempDir(), "reference")
	m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir, Mode: "reference-only"})
	if e != nil {
		t.Fatal(e)
	}
	if m.Dependency == "none" || m.Artifacts[0].File != "" {
		t.Fatal("source dependency hidden")
	}
	if _, e = db.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Minute); e == nil {
		t.Fatal("backup dependency retired")
	}
	target := destination(t)
	if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
		t.Fatal("reference recovery", e)
	}
	if e = Release(ctx, db, dir); e != nil {
		t.Fatal(e)
	}
	if e = Release(ctx, db, dir); e != nil {
		t.Fatal("release replay", e)
	}
	if _, e = db.ClaimRetirement(ctx, p.ID, contracts.ID(), 0, time.Minute); e != nil {
		t.Fatal("released dependency retained", e)
	}
	p.State = "retiring"
	if e = s.Store.DeleteUnreferenced(ctx, p); e != nil {
		t.Fatal(e)
	}
	if _, e = Verify(ctx, dir, nil); e == nil {
		t.Fatal("lost source dependency unnoticed")
	}
}

func TestManifestTraversalAndFalseCatalogProofRejected(t *testing.T) {
	for _, attack := range []string{"path", "catalog"} {
		t.Run(attack, func(t *testing.T) {
			ctx := context.Background()
			w, db, s := fixture(t)
			publish(t, s, "bytes", "canonical-audio")
			dir := filepath.Join(t.TempDir(), "bundle")
			m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
			if e != nil {
				t.Fatal(e)
			}
			if attack == "path" {
				m.Artifacts[0].File = "../outside"
			} else {
				m.CatalogDigest = "forged"
			}
			m.Digest, _ = manifestDigest(m)
			raw, _ := json.Marshal(m)
			if e = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0600); e != nil {
				t.Fatal(e)
			}
			if _, e = Inspect(ctx, dir); e == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
}

func TestManifestPublicationTamperingRejectedBeforeTargetWrites(t *testing.T) {
	for _, attack := range []string{"retention-reference", "kind", "owner", "journal-proof", "events"} {
		t.Run(attack, func(t *testing.T) {
			ctx := context.Background()
			w, db, service := fixture(t)
			publish(t, service, "authoritative publication bytes", "canonical-audio")
			dir := filepath.Join(t.TempDir(), "bundle")
			m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
			if e != nil {
				t.Fatal(e)
			}
			p := &m.Artifacts[0].Publication
			switch attack {
			case "retention-reference":
				p.References[contracts.ID()] = true
			case "kind":
				p.Kind = "model"
			case "owner":
				p.Owner = contracts.ID()
			case "journal-proof":
				p.JournalReceiptID = contracts.ID()
			case "events":
				p.Events = append(p.Events, "forged-history")
			}
			m.Digest, e = manifestDigest(m)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := json.Marshal(m)
			if e = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0600); e != nil {
				t.Fatal(e)
			}
			if _, e = Inspect(ctx, dir); e == nil {
				t.Fatal("recomputed manifest accepted altered publication")
			}
			target := destination(t)
			marker := filepath.Join(target.Root, ".insonic", "workspace.json")
			before, e := os.ReadFile(marker)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = RestoreWorkspace(ctx, target, nil, dir); e == nil {
				t.Fatal("altered authority restored")
			}
			after, e := os.ReadFile(marker)
			if e != nil || string(before) != string(after) {
				t.Fatal("target configuration changed", e)
			}
			if _, e = os.Stat(filepath.Join(target.Control, "catalog.sqlite")); !os.IsNotExist(e) {
				t.Fatal("target catalog opened before publication proof validation")
			}
			if _, e = os.Stat(filepath.Join(target.Control, "backup-restore-progress")); !os.IsNotExist(e) {
				t.Fatal("target progress created before publication proof validation")
			}
		})
	}
}

func TestSelfContainedOriginRootsRestoreAcrossHostPathSyntax(t *testing.T) {
	for _, origin := range []string{`C:\origin\artifacts`, `D:/origin/artifacts`, `\\source-server\audio-share\artifacts`, "/origin/artifacts"} {
		t.Run(origin, func(t *testing.T) {
			ctx := context.Background()
			w, db, service := fixture(t)
			p := publish(t, service, "portable cross-host bytes", "canonical-audio")
			dir := filepath.Join(t.TempDir(), "bundle")
			m, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir})
			if e != nil {
				t.Fatal(e)
			}
			// The captured profile uses an originating-relative root. Its resolved
			// locator in the manifest uses the originating host's path syntax.
			m.SourceProfiles[0].Configuration["root"] = origin
			m.Digest, e = manifestDigest(m)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := json.Marshal(m)
			if e = os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0600); e != nil {
				t.Fatal(e)
			}
			if _, e = Inspect(ctx, dir); e != nil {
				t.Fatal("origin syntax rejected", e)
			}
			target := destination(t)
			if _, e = RestoreWorkspace(ctx, target, nil, dir); e != nil {
				t.Fatal("cross-host restore", e)
			}
			store, e := catalog.OpenWorkspace(ctx, target, nil, false)
			if e != nil {
				t.Fatal(e)
			}
			defer store.Close()
			artifacts, e := artifact.NewService(ctx, target, store, nil, contracts.ID())
			if e != nil {
				t.Fatal(e)
			}
			defer artifacts.Close()
			if e = artifacts.Verify(ctx, p.ID); e != nil {
				t.Fatal("restored cross-host bytes", e)
			}
		})
	}
}

func TestOriginRootClassificationDoesNotRebaseDependencies(t *testing.T) {
	for _, invalid := range []string{"artifacts", `C:artifacts`, `\artifacts`, `\\server`, `\\server\`, `\\..\share`, "", "C:\\bad\x00root"} {
		if absoluteOriginPath(invalid) {
			t.Fatal("relative or invalid origin root accepted", invalid)
		}
	}
	for _, origin := range []string{`C:\origin\artifacts`, `\\source-server\audio-share\artifacts`, "/origin/artifacts"} {
		if !absoluteOriginPath(origin) {
			t.Fatal("absolute origin rejected", origin)
		}
		if filepath.IsAbs(origin) {
			continue
		}
		p := workspace.Profile{Adapter: "filesystem", Configuration: map[string]any{"root": origin}}
		if store, e := openStorage(context.Background(), p, t.TempDir(), nil); e == nil {
			store.Close()
			t.Fatal("foreign dependency rebased onto current host", origin)
		}
	}
}

func TestCancelledCreateAndNonemptyRestore(t *testing.T) {
	ctx := context.Background()
	w, db, s := fixture(t)
	publish(t, s, "bytes", "canonical-audio")
	dir := filepath.Join(t.TempDir(), "bundle")
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, e := Create(cancelled, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir}); e == nil {
		t.Fatal("cancel accepted")
	}
	if _, e := os.Stat(dir); !os.IsNotExist(e) {
		t.Fatal("incomplete bundle advertised")
	}
	if _, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: dir}); e != nil {
		t.Fatal(e)
	}
	target, targetDB, targetService := fixture(t)
	publish(t, targetService, "target-owned", "canonical-audio")
	before, _ := targetDB.Revision(ctx)
	original := target.Config.WorkspaceID
	if _, e := RestoreWorkspace(ctx, target, nil, dir); e == nil {
		t.Fatal("occupied target restored")
	}
	after, _ := targetDB.Revision(ctx)
	if before != after || target.Config.WorkspaceID != original {
		t.Fatal("occupied target altered")
	}
}
