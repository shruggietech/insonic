// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/schemas"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestModelReadinessRolesLeasesAndVerificationFailureContext(t *testing.T) {
	a := configuredApp(t)
	m := dependencyManifest(t, "operator", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("synthetic model bytes")) }))
	registerDependencyManifest(t, a, m)
	id := models.InstallationID(m)
	view := configuredContractResult(t, realRequest(a, "models.show", id, nil))
	if view["readiness"] != "not_acquired" || view["verification"] != "not_checked" {
		t.Fatal(view)
	}
	rejected := realRequest(a, "models.verify", id, nil)
	raw, _ := json.Marshal(rejected)
	if rejected.Error == nil || rejected.Error.Reason != "model_not_acquired" || rejected.Result == nil || schemas.ValidateResponse(raw) != nil {
		t.Fatal(rejected)
	}
	selection, err := models.ResolveManifest(m, "transcription")
	if err != nil {
		t.Fatal(err)
	}
	work, err := a.enqueueModelAcquisition(a.ctx, selection, false)
	if err != nil {
		t.Fatal(err)
	}
	view = configuredContractResult(t, realRequest(a, "models.show", id, nil))
	if view["acquisition_work_id"] != work.ID || view["readiness"] != "acquiring" {
		t.Fatal(view)
	}
	claim, err := a.Catalog.ClaimWork(a.ctx, work.ID, a.Session, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	service, err := a.modelService()
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(a.ctx, claim)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(result)
	if _, err = a.Catalog.CheckpointWork(a.ctx, claim, "complete", "succeeded", raw, time.Minute); err != nil {
		t.Fatal(err)
	}
	view = configuredContractResult(t, realRequest(a, "models.materialize", id, nil))
	files := view["files"].([]any)
	if len(files) != len(m.Files) || view["verification"] != "verified" {
		t.Fatal(view)
	}
	for i, item := range files {
		file := item.(map[string]any)
		if file["role"] != m.Files[i].Role || file["sha256"] != m.Files[i].SHA256 || file["manifest_digest"] != m.Digest() || file["model_id"] != id || !contracts.ValidID(file["lease_id"].(string)) {
			t.Fatal(file)
		}
		data, err := os.ReadFile(file["path"].(string))
		if err != nil || !bytes.Equal(data, []byte("synthetic model bytes")) {
			t.Fatal(err)
		}
		if err = service.Artifacts.Release(a.ctx, file["publication_id"].(string), file["lease_id"].(string)); err != nil {
			t.Fatal(err)
		}
	}
	// Corrupt the authoritative artifact, not only its derived materialization.
	install, err := a.Catalog.BaseModel(a.ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	// A coherent publication with the wrong declared identity is proven damage,
	// while unavailable storage must remain a different condition.
	damagedService := models.NewService(service.Artifacts, operatorModelMetadataFixture{Catalog: a.Catalog}, service.Secrets)
	typed, ok := damagedService.Verify(a.ctx, id).(*contracts.Error)
	if !ok || typed.Reason != "model_damaged" {
		t.Fatal("metadata mismatch lost damage reason", typed)
	}
	malformed := &App{Catalog: operatorModelMetadataFixture{Catalog: a.Catalog, malformed: true}, Workspace: a.Workspace, ctx: a.ctx}
	malformedView, malformedErr := malformed.modelReadiness(id)
	if malformedErr == nil || malformedView["readiness"] != "damaged" || malformedView["verification"] != "failed" {
		t.Fatal(malformedView, malformedErr)
	}
	var ids []string
	json.Unmarshal(install.PublicationIDs, &ids)
	publication, err := a.Catalog.Publication(a.ctx, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(a.Workspace.Control, "artifacts", filepath.FromSlash(publication.Key))
	if _, err = os.Stat(artifactPath); err != nil {
		t.Fatal("fixture artifact location", err)
	}
	if err = os.WriteFile(artifactPath, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	rejected = realRequest(a, "models.verify", id, nil)
	raw, _ = json.Marshal(rejected)
	if rejected.Error == nil || rejected.Error.Reason != "model_damaged" || rejected.Result.(map[string]any)["readiness"] != "damaged" || schemas.ValidateResponse(raw) != nil {
		t.Fatal(rejected)
	}
}

type operatorModelMetadataFixture struct {
	catalog.Catalog
	malformed bool
}

func (f operatorModelMetadataFixture) BaseModel(ctx context.Context, id string) (catalog.BaseModelInstall, error) {
	model, err := f.Catalog.BaseModel(ctx, id)
	if err == nil && f.malformed {
		model.Manifest = json.RawMessage(`{}`)
	}
	return model, err
}
func (f operatorModelMetadataFixture) Publication(ctx context.Context, id string) (catalog.Publication, error) {
	publication, err := f.Catalog.Publication(ctx, id)
	if err == nil {
		publication.Digest = strings.Repeat("0", 64)
	}
	return publication, err
}
