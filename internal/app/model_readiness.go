// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
)

// Readiness is a response view of durable facts and this inspection. It never
// promotes a registration to availability or persists a verification claim.
func (a *App) modelReadiness(id string) (map[string]any, error) {
	install, err := a.Catalog.BaseModel(a.ctx, id)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(install)
	view := map[string]any{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&view) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	view["model_id"] = install.ID
	view["manifest_digest"] = install.Digest
	view["registry_state"] = install.State
	view["readiness"] = "not_acquired"
	view["verification"] = "not_checked"
	if install.State == "available" {
		view["readiness"] = "available"
	}
	var manifest models.Manifest
	if models.DecodeManifest(install.Manifest, &manifest) != nil {
		view["readiness"], view["verification"] = "damaged", "failed"
		return view, contracts.Condition("conflict", "model_damaged")
	}
	acquisition := models.AcquisitionID(a.Workspace.Config.WorkspaceID, manifest)
	if work, err := a.Catalog.Work(a.ctx, acquisition); err == nil {
		view["acquisition_work_id"] = work.ID
		view["acquisition_state"] = work.State
		if install.State != "available" && (work.State == "pending" || work.State == "running") {
			view["readiness"] = "acquiring"
		}
	}
	return view, nil
}

func (a *App) inspectModel(req contracts.Request) (any, error) {
	view, err := a.modelReadiness(req.ItemID)
	if err != nil {
		return view, err
	}
	service, err := a.modelService()
	if err != nil {
		return view, err
	}
	if req.Operation == "models.verify" {
		err = service.Verify(a.ctx, req.ItemID)
		view["verified"] = err == nil
	} else {
		files, e := service.Materialize(a.ctx, req.ItemID)
		err = e
		if err == nil {
			install, e := a.Catalog.BaseModel(a.ctx, req.ItemID)
			var manifest models.Manifest
			if e != nil || models.DecodeManifest(install.Manifest, &manifest) != nil || len(files) != len(manifest.Files) {
				// Release every acquired lease if the named response cannot be built.
				for _, file := range files {
					service.Artifacts.Release(a.ctx, file.PublicationID, file.Lease.ID)
				}
				return view, contracts.Condition("conflict", "model_damaged")
			}
			named := []any{}
			for i, file := range files {
				declaration := manifest.Files[i]
				named = append(named, map[string]any{"model_id": req.ItemID, "manifest_digest": manifest.Digest(), "role": declaration.Role, "size": declaration.Size, "sha256": declaration.SHA256, "publication_id": file.PublicationID, "path": file.Path, "lease_id": file.Lease.ID, "lease": file.Lease})
			}
			view["files"] = named
			view["materialized"] = true
		}
	}
	if err == nil {
		view["verification"] = "verified"
		view["readiness"] = "verified"
	} else {
		view["verification"] = "failed"
		if typed, ok := err.(*contracts.Error); ok {
			switch typed.Reason {
			case "model_damaged":
				view["readiness"] = "damaged"
			case "model_storage_unavailable":
				view["readiness"] = "unavailable"
			}
		}
	}
	return view, err
}
