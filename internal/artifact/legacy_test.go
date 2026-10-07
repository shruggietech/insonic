// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type secondLegacyReadFailure struct {
	Store
	reads int
}

func (s *secondLegacyReadFailure) OpenRange(ctx context.Context, p catalog.Publication, offset, length int64) (io.ReadCloser, error) {
	s.reads++
	if s.reads == 2 {
		return nil, contracts.Fail("unavailable")
	}
	return s.Store.OpenRange(ctx, p, offset, length)
}

func TestLegacyFailedSecondReadbackRetainsBytesAndRetries(t *testing.T) {
	s := testService(t)
	p := legacyDerivedFixture(t, s)
	ctx := context.Background()
	original := s.Store
	s.Store = &secondLegacyReadFailure{Store: original}
	if e := s.RecoverLegacy(ctx); e == nil {
		t.Fatal("failed second readback admitted")
	}
	current, e := s.Catalog.Publication(ctx, p.ID)
	if e != nil || current.State != "pending" || current.AdmissionID != "" {
		t.Fatal("failed readback invented admission", current, e)
	}
	if _, e = original.Stat(ctx, p); e != nil {
		t.Fatal("unverified legacy bytes deleted", e)
	}
	if e = s.Catalog.FinishCleanup(ctx, p.ID); e == nil {
		t.Fatal("unverified legacy cleanup completed")
	}
	if e = s.RecoverLegacy(ctx); e != nil {
		t.Fatal("readback retry failed", e)
	}
	current, e = s.Catalog.Publication(ctx, p.ID)
	if e != nil || current.State != "available" {
		t.Fatal("successful recovery did not admit", current, e)
	}
}

func restoreLegacyService(t *testing.T, s *Service, snap catalog.Snapshot) {
	t.Helper()
	db, e := catalog.OpenSQLite(context.Background(), filepath.Join(t.TempDir(), "restored.sqlite"), s.Workspace.Config.WorkspaceID)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if e = db.Restore(context.Background(), snap); e != nil {
		t.Fatal("legacy snapshot restore", e)
	}
	s.Catalog = db
}

func legacyDerivedFixture(t *testing.T, s *Service) catalog.Publication {
	t.Helper()
	ctx := context.Background()
	data := []byte("old derived assignments to retire")
	sum := sha256.Sum256(data)
	profile := s.Workspace.Config.Profiles.Storage
	p := catalog.Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: "", ProfileID: profile.ID, ProfileRevision: profile.Revision, Digest: hex.EncodeToString(sum[:]), Size: int64(len(data)), Kind: "derived-manifest", Key: "legacy/" + contracts.ID(), Owner: s.Owner, State: "pending"}
	p.LocationID = p.ID
	source := bytesFile(t, data)
	if _, e := s.Store.PublishImmutable(ctx, p, source, func(next catalog.Publication) (catalog.Publication, error) { return next, nil }); e != nil {
		t.Fatal(e)
	}
	rev, e := s.Catalog.Revision(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Catalog.Commit(ctx, catalog.Mutation{OperationID: contracts.ID(), Expected: rev, Records: catalog.Records{Artifacts: []catalog.Artifact{{ID: p.ArtifactID, Digest: p.Digest, Size: p.Size, Kind: p.Kind}}, Locations: []catalog.ArtifactLocation{{ID: p.ID, ArtifactID: p.ArtifactID, ProfileID: p.ProfileID, ProfileRevision: p.ProfileRevision, Key: p.Key, State: "available"}}, Speakers: []catalog.Speaker{{ID: contracts.ID(), Name: "Legacy identity"}}}}); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Catalog.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	snap.CatalogSchema = 3
	snap.Records.Datasets = []catalog.Dataset{{ID: contracts.ID(), SpeakerID: snap.Records.Speakers[0].ID, ManifestArtifactID: &p.ArtifactID, Options: json.RawMessage(`{"attribution":"legacy payload"}`)}}
	snap.Digest = ""
	raw, e := json.Marshal(snap)
	if e != nil {
		t.Fatal(e)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var canonical map[string]any
	if e = dec.Decode(&canonical); e != nil {
		t.Fatal(e)
	}
	// Construct the actual historical entity shape, without schema5 identity proofs.
	records := canonical["records"].(map[string]any)
	for _, value := range records["speakers"].([]any) {
		speaker := value.(map[string]any)
		delete(speaker, "revision")
		delete(speaker, "state")
	}
	for _, value := range canonical["state"].([]any) {
		table := value.(map[string]any)
		if table["name"] != "operation_receipt" {
			continue
		}
		for _, entry := range table["rows"].([]any) {
			row := entry.([]any)
			var result map[string]any
			if json.Unmarshal([]byte(row[3].(string)), &result) != nil {
				t.Fatal("invalid fixture receipt")
			}
			delete(result, "speaker_proofs")
			text, _ := json.Marshal(result)
			row[3] = string(text)
		}
	}
	normalized, _ := json.Marshal(canonical)
	sum = sha256.Sum256(normalized)
	canonical["digest"] = hex.EncodeToString(sum[:])
	raw, _ = json.Marshal(canonical)
	migrated, e := catalog.ReadSnapshotReader(bytes.NewReader(raw))
	if e != nil {
		t.Fatal("legacy snapshot conversion", e)
	}
	restoreLegacyService(t, s, migrated)
	cleanups, e := s.Catalog.Cleanups(ctx)
	if e != nil || len(cleanups) != 1 || cleanups[0].LegacyLocationID == nil {
		t.Fatal("legacy location obligation missing", cleanups, e)
	}
	return p
}

func legacyPhysicalRetirementSuite(t *testing.T, s *Service) {
	t.Helper()
	ctx := context.Background()
	p := legacyDerivedFixture(t, s)
	if _, e := s.Catalog.Publication(ctx, p.ID); e == nil {
		t.Fatal("location invented admission")
	}
	if e := s.RecoverLegacy(ctx); e != nil {
		t.Fatal(e)
	}
	admitted, e := s.Catalog.Publication(ctx, p.ID)
	if e != nil || admitted.Verification != "sha256-readback" || admitted.State != "available" {
		t.Fatal("legacy bytes not independently admitted", admitted, e)
	}
	if _, e = s.RetireCurrent(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.Catalog.FinishCleanup(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Store.Stat(ctx, p); e == nil {
		t.Fatal("derived bytes remain after completion")
	}
	cleanups, _ := s.Catalog.Cleanups(ctx)
	if cleanups[0].State != "done" {
		t.Fatal("physical cleanup not recorded")
	}
	snap, e := s.Catalog.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restoreLegacyService(t, s, snap)
}
func TestLegacyDerivedPhysicalRetirement(t *testing.T) {
	legacyPhysicalRetirementSuite(t, testService(t))
}
func TestLegacyDerivedUnavailableKeepsObligation(t *testing.T) {
	s := testService(t)
	p := legacyDerivedFixture(t, s)
	ctx := context.Background()
	selected := s.Workspace.Config.Profiles.Storage
	s.Workspace.Config.Profiles.Storage.ID = contracts.ID()
	if e := s.RecoverLegacy(ctx); e != nil {
		t.Fatal(e)
	}
	if _, e := s.Catalog.Publication(ctx, p.ID); e == nil {
		t.Fatal("unelected profile adopted")
	}
	s.Workspace.Config.Profiles.Storage = selected
	p.State = "retiring" // Simulate selected storage loss without completing catalog retirement.
	if e := s.Store.DeleteUnreferenced(ctx, p); e != nil {
		t.Fatal(e)
	}
	if e := s.RecoverLegacy(ctx); e == nil {
		t.Fatal("absent bytes admitted")
	}
	c, _ := s.Catalog.Cleanups(ctx)
	if c[0].State != "pending" {
		t.Fatal("unavailable obligation completed")
	}
	if e := s.Catalog.FinishCleanup(ctx, p.ID); e == nil {
		t.Fatal("physical deletion invented admission/completion")
	}
}
func TestAbortedMappedCandidatePhysicalCleanup(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	profile := s.Workspace.Config.Profiles.Storage
	p := catalog.Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), LocationID: contracts.ID(), ProfileID: profile.ID, ProfileRevision: profile.Revision, Digest: hex.EncodeToString(make([]byte, 32)), Size: 10, Kind: "mapped-audio", Key: "objects/" + contracts.ID(), Owner: s.Owner}
	var e error
	p, e = s.Catalog.BeginPublication(ctx, p, s.TTL)
	if e != nil {
		t.Fatal(e)
	}
	entry, e := s.Catalog.UpdateLibrary(ctx, contracts.ID(), 0, catalog.LibraryEntry{ID: contracts.ID(), AssetID: contracts.ID(), Digest: hex.EncodeToString(make([]byte, 32)), Size: 0, Mode: "reference", Title: "Source", Class: "audio", Facts: json.RawMessage(`{}`), Metadata: json.RawMessage(`{}`), Dates: json.RawMessage(`{}`), ReportPublicationIDs: json.RawMessage(`[]`)})
	if e != nil {
		t.Fatal(e)
	}
	durable, e := s.scratch.OpenFile(p.ID+".source", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	durable.Write([]byte("private source"))
	durable.Close()
	if e = s.Catalog.QueueDerivedCleanup(ctx, contracts.ID(), entry.ID, p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Abort(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = s.scratch.Stat(p.ID + ".source"); !os.IsNotExist(e) {
		t.Fatal("private source remained", e)
	}
	if e = s.Catalog.FinishCleanup(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	snap, e := s.Catalog.Export(ctx)
	if e != nil {
		t.Fatal(e)
	}
	restoreLegacyService(t, s, snap)
}
