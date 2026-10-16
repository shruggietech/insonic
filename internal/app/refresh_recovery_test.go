// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/workspace"
)

func TestNativeRefreshDeferredCleanupRecoversOnRuntimeRestart(t *testing.T) {
	selected := os.Getenv("INSONIC_LIBRARY_TOOLS_FILE")
	if selected == "" {
		t.Skip("pinned native tools not selected")
	}
	tools, err := os.ReadFile(selected)
	if err != nil {
		t.Fatal(err)
	}
	w, err := workspace.Init(t.TempDir(), "refresh recovery")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(w.Control, "media-tools.json"), tools, 0600); err != nil {
		t.Fatal(err)
	}
	a, err := New(w)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if a != nil {
			a.Close()
		}
	}()
	ctx := context.Background()
	source, err := filepath.Abs("../../tests/fixtures/media/speech.flac")
	if err != nil {
		t.Fatal(err)
	}
	queued := realRequest(a, "media.import", "", library.ImportRequest{Items: []library.Item{{Source: source}}})
	if queued.Error != nil {
		t.Fatal(queued.Error)
	}
	work := awaitWork(t, a, queued.Result.(map[string]any)["work_id"].(string))
	var imported library.ImportResult
	if work.State != "succeeded" || json.Unmarshal(work.Result, &imported) != nil {
		t.Fatal(work.State, string(work.Result))
	}
	entry, err := a.Catalog.Library(ctx, imported.Items[0].MediaID)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := catalog.PublicationIDs(entry.ReportPublicationIDs)
	if err != nil || len(ids) == 0 {
		t.Fatal("missing reports", err)
	}
	artifacts, err := a.artifactService()
	if err != nil {
		t.Fatal(err)
	}
	lease, err := artifacts.Materialize(ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if artifacts != nil {
			artifacts.Release(ctx, ids[0], lease.Lease.ID)
		}
	}()
	queued = realRequest(a, "media.refresh", entry.ID, library.Options{})
	if queued.Error != nil {
		t.Fatal(queued.Error)
	}
	work = awaitWork(t, a, queued.Result.(map[string]any)["work_id"].(string))
	if work.State != "succeeded" {
		t.Fatal("leased refresh failed", work.Error, string(work.Result))
	}
	accepted, err := a.Catalog.Library(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := a.Catalog.Publication(ctx, ids[0])
	if err != nil || pub.State != "available" {
		t.Fatal("leased report retired", err)
	}
	if err = artifacts.Release(ctx, ids[0], lease.Lease.ID); err != nil {
		t.Fatal(err)
	}
	artifacts = nil
	a.Close()
	a, err = New(w)
	if err != nil {
		t.Fatal(err)
	}
	// Startup schedules recovery independently of application readiness. Observe
	// its result without calling the sweeper or relying on a fixed delay.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		pub, err = a.Catalog.Publication(ctx, ids[0])
		if err != nil || pub.State == "retired" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil || pub.State != "retired" {
		t.Fatal("startup did not recover pending cleanup", err, pub.State)
	}
	current, err := a.Catalog.Library(ctx, entry.ID)
	if err != nil || current.Revision != accepted.Revision || current.Digest != accepted.Digest {
		t.Fatal("restart changed accepted authority", err)
	}
	stored, err := a.Catalog.Work(ctx, work.ID)
	if err != nil || stored.State != "succeeded" {
		t.Fatal("restart lost accepted work outcome", err)
	}
}
