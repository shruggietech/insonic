// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestRestoreProgressPreservesExactMultipartAndRejectsRebinding(t *testing.T) {
	w, e := workspace.Init(t.TempDir(), "Progress target")
	if e != nil {
		t.Fatal(e)
	}
	dir, e := restoreProgressDirectory(w, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	expected := catalog.Publication{ID: contracts.ID(), ArtifactID: contracts.ID(), ProfileID: contracts.ID(), ProfileRevision: 1, Key: "objects/immutable", Digest: "exact-hash", Size: 10 << 20, Kind: "canonical-audio", State: "pending"}
	progress := expected
	progress.UploadID = "known-upload"
	progress.PartSize = 8 << 20
	progress.Parts = []catalog.UploadPart{{Number: 1, ETag: "immutable-part-etag", Size: 8 << 20}}
	if _, e = saveRestoreProgress(dir, "bundle-proof", "destination-proof", progress); e != nil {
		t.Fatal(e)
	}
	got, e := loadRestoreProgress(dir, "bundle-proof", "destination-proof", expected)
	if e != nil || got.UploadID != progress.UploadID || got.PartSize != 8<<20 || len(got.Parts) != 1 || got.Parts[0].ETag != progress.Parts[0].ETag {
		t.Fatal("multipart recovery", got, e)
	}
	for _, mismatch := range []struct{ bundle, destination string }{{"another-bundle", "destination-proof"}, {"bundle-proof", "another-destination"}} {
		if _, e = loadRestoreProgress(dir, mismatch.bundle, mismatch.destination, expected); e == nil {
			t.Fatal("progress rebound")
		}
	}
	progress.PartSize = -1
	if _, e = saveRestoreProgress(dir, "bundle-proof", "destination-proof", progress); e != nil {
		t.Fatal(e)
	}
	if _, e = loadRestoreProgress(dir, "bundle-proof", "destination-proof", expected); e == nil {
		t.Fatal("invalid multipart geometry accepted")
	}
	progress.PartSize = 8 << 20
	progress.References = map[string]bool{contracts.ID(): true}
	if _, e = saveRestoreProgress(dir, "bundle-proof", "destination-proof", progress); e != nil {
		t.Fatal(e)
	}
	if _, e = loadRestoreProgress(dir, "bundle-proof", "destination-proof", expected); e == nil {
		t.Fatal("lifetime references changed by scratch journal")
	}
	expected.Key = "objects/different"
	if _, e = loadRestoreProgress(dir, "bundle-proof", "destination-proof", expected); e == nil {
		t.Fatal("immutable destination changed")
	}
	if e = os.WriteFile(filepath.Join(dir, expected.ID+".json"), []byte(`{"kind":"backup-restore-progress","kind":"changed"}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = loadRestoreProgress(dir, "bundle-proof", "destination-proof", expected); e == nil {
		t.Fatal("duplicate-key journal accepted")
	}
}
