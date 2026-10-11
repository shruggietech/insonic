// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestPopulatedHistoricalWorkspaceBackupKeepsSourceAuthority(t *testing.T) {
	ctx := context.Background()
	w, e := workspace.Init(t.TempDir(), "Historical source")
	if e != nil {
		t.Fatal(e)
	}
	w.Config.Version = "1.0.0"
	w.Config.Profiles.Storage.Version = "1.0.0"
	w.Config.Profiles.Catalog.Version = "1.0.0"
	w.Config.Profiles.Graph.Version = "1.0.0"
	raw, _ := json.Marshal(w.Config)
	config := filepath.Join(w.Control, "workspace.json")
	if e = os.WriteFile(config, raw, 0600); e != nil {
		t.Fatal(e)
	}
	w, e = workspace.Open(w.Root)
	if e != nil {
		t.Fatal(e)
	}
	db, e := catalog.OpenWorkspace(ctx, w, nil, true)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = db.RegisterWorkspace(ctx, w); e != nil {
		t.Fatal(e)
	}
	service, e := artifact.NewService(ctx, w, db, nil, contracts.ID())
	if e != nil {
		t.Fatal(e)
	}
	defer service.Close()
	source := filepath.Join(t.TempDir(), "original.bin")
	if e = os.WriteFile(source, []byte("immutable historical artifact"), 0600); e != nil {
		t.Fatal(e)
	}
	p, e := service.Publish(ctx, contracts.ID(), source, "other")
	if e != nil {
		t.Fatal(e)
	}
	work, e := db.EnqueueWork(ctx, contracts.ID(), "fixture.work", json.RawMessage("{}"))
	if e != nil {
		t.Fatal(e)
	}
	claim, e := db.ClaimWork(ctx, work.ID, contracts.ID(), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	work, e = db.CheckpointWork(ctx, claim, "historical-failure", "failed", json.RawMessage(`{"error":"conflict"}`), time.Minute)
	if e != nil {
		t.Fatal(e)
	}
	bundle := filepath.Join(t.TempDir(), "portable")
	manifest, e := Create(ctx, w, db, nil, CreateOptions{ID: contracts.ID(), Directory: bundle})
	if e != nil {
		t.Fatal(e)
	}
	if manifest.Config.Version != contracts.Version || manifest.SourceProfiles[0].Version != "1.0.0" {
		t.Fatal("portable/source versions", manifest)
	}
	after, _ := os.ReadFile(config)
	if !bytes.Equal(raw, after) {
		t.Fatal("historical source config rewritten")
	}
	// A historical backup envelope is admitted by its declared document version.
	manifest.Version = "1.0.0"
	manifest.Config.Version = "1.0.0"
	manifest.Digest, e = manifestDigest(manifest)
	if e != nil {
		t.Fatal(e)
	}
	manifestBytes, e := json.Marshal(manifest)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(bundle, "manifest.json"), manifestBytes, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = Inspect(ctx, bundle); e != nil {
		t.Fatal("historical backup envelope", e)
	}
	target, e := workspace.Init(t.TempDir(), "Restore")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = RestoreWorkspace(ctx, target, nil, bundle); e != nil {
		t.Fatal(e)
	}
	restored, e := catalog.OpenWorkspace(ctx, target, nil, false)
	if e != nil {
		t.Fatal(e)
	}
	defer restored.Close()
	got, e := restored.Work(ctx, work.ID)
	if e != nil || got.Error != "operation_failed" || !bytes.Equal(got.Result, work.Result) {
		t.Fatal("historical failure reinterpreted", got, e)
	}
	original, e := restored.Publication(ctx, p.ID)
	if e != nil || original.Digest != p.Digest {
		t.Fatal("artifact identity", original, e)
	}
	if _, e = restored.Export(ctx); e != nil {
		t.Fatal("restored proof", e)
	}
}
