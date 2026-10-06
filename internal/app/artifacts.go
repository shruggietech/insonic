// SPDX-License-Identifier: Apache-2.0
package app

import (
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

func (a *App) artifactService() (*artifact.Service, error) {
	a.artifactMu.Lock()
	defer a.artifactMu.Unlock()
	if a.Artifacts == nil {
		service, e := artifact.NewService(a.ctx, a.Workspace, a.Catalog, a.secrets, a.Session)
		if e != nil {
			return nil, e
		}
		a.Artifacts = service
	}
	return a.Artifacts, nil
}
func (a *App) artifactDispatch(req contracts.Request) (result any, err error) {
	defer func() {
		if publication, ok := result.(catalog.Publication); ok {
			result = artifact.ReceiptOf(publication)
		}
	}()
	if req.Operation == "artifacts.show" {
		return a.Catalog.Publication(a.ctx, req.PublicationID)
	}
	if req.Operation == "artifacts.retain" || req.Operation == "artifacts.release-reference" {
		e := a.Catalog.ArtifactReference(a.ctx, req.PublicationID, req.ReferenceID, req.Operation == "artifacts.release-reference")
		return map[string]bool{"accepted": e == nil}, e
	}
	s, e := a.artifactService()
	if e != nil {
		return nil, e
	}
	switch req.Operation {
	case "artifacts.publish":
		if req.PublicationID != "" || req.LeaseID != "" || req.ReferenceID != "" {
			return nil, contracts.Fail("invalid_request")
		}
		return s.Publish(a.ctx, req.RequestID, req.SourcePath, req.ArtifactKind)
	case "artifacts.show":
		return a.Catalog.Publication(a.ctx, req.PublicationID)
	case "artifacts.verify":
		e = s.Verify(a.ctx, req.PublicationID)
		return map[string]bool{"verified": e == nil}, e
	case "artifacts.materialize":
		if req.MaxBytes > 0 {
			return s.MaterializeBound(a.ctx, req.PublicationID, req.MaxBytes)
		}
		return s.Materialize(a.ctx, req.PublicationID)
	case "artifacts.cache-prune":
		return s.Prune(a.ctx, req.PublicationID)
	case "artifacts.reconcile":
		return s.Reconcile(a.ctx, req.PublicationID)
	case "artifacts.abort":
		return s.Abort(a.ctx, req.PublicationID)
	case "artifacts.retire":
		return s.Retire(a.ctx, req.PublicationID)
	case "artifacts.lease-renew":
		return s.Renew(a.ctx, req.PublicationID, req.LeaseID)
	case "artifacts.lease-release":
		e = s.Release(a.ctx, req.PublicationID, req.LeaseID)
		return map[string]bool{"released": e == nil}, e
	case "artifacts.retain", "artifacts.release-reference":
		e = a.Catalog.ArtifactReference(a.ctx, req.PublicationID, req.ReferenceID, req.Operation == "artifacts.release-reference")
		return map[string]bool{"accepted": e == nil}, e
	}
	return nil, contracts.Fail("invalid_request")
}
func (a *App) storageDiagnostics() (contracts.Capability, any) {
	cap := contracts.Capability{AdapterID: a.Workspace.Config.Profiles.Storage.Adapter, ContractVersion: contracts.Version, State: "unavailable", Operations: []string{}}
	s, e := a.artifactService()
	if e != nil {
		return cap, map[string]string{"state": "unavailable"}
	}
	cap.State = "configured"
	if cap.AdapterID == "filesystem" {
		cap.State = "available"
	}
	cap.Operations = []string{"publish", "stat", "range", "verify", "materialize", "reconcile", "retire"}
	return cap, map[string]any{"capabilities": s.Store.Capabilities(), "remote_reachability": "not-probed"}
}
func artifactRequestValid(req contracts.Request) bool {
	isArtifact := strings.HasPrefix(req.Operation, "artifacts.")
	if !isArtifact {
		return req.PublicationID == "" && req.SourcePath == "" && req.ArtifactKind == "" && req.LeaseID == "" && req.ReferenceID == "" && req.MaxBytes == 0
	}
	if req.Operation == "artifacts.publish" {
		return req.SourcePath != "" && req.ArtifactKind != "" && req.PublicationID == "" && req.LeaseID == "" && req.ReferenceID == "" && req.MaxBytes == 0
	}
	if !contracts.ValidID(req.PublicationID) || req.SourcePath != "" || req.ArtifactKind != "" || req.MaxBytes < 0 || (req.Operation != "artifacts.materialize" && req.MaxBytes != 0) {
		return false
	}
	lease := req.Operation == "artifacts.lease-renew" || req.Operation == "artifacts.lease-release"
	ref := req.Operation == "artifacts.retain" || req.Operation == "artifacts.release-reference"
	return (!lease && req.LeaseID == "" || lease && contracts.ValidID(req.LeaseID)) && (!ref && req.ReferenceID == "" || ref && contracts.ValidID(req.ReferenceID))
}
