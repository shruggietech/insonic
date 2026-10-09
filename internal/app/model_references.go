// SPDX-License-Identifier: Apache-2.0
package app

import (
	"strings"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
)

func modelReferenceOperation(op string) bool {
	return op == "models.resolve" || op == "models.discover" || strings.HasPrefix(op, "models.alias.") || strings.HasPrefix(op, "models.source.")
}

func modelReferenceRequestValid(req contracts.Request) bool {
	if req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 {
		return false
	}
	switch req.Operation {
	case "models.resolve", "models.discover":
		return req.ItemID == "" && len(req.Data) > 0
	case "models.alias.list", "models.source.list":
		return req.ItemID == ""
	case "models.alias.show", "models.source.show":
		return contracts.ValidID(req.ItemID) && len(req.Data) == 0
	case "models.alias.set", "models.alias.remove", "models.source.set", "models.source.remove":
		return contracts.ValidID(req.ItemID) && len(req.Data) > 0
	default:
		return false
	}
}

func (a *App) modelReferenceDispatch(req contracts.Request) (any, error) {
	resolver := models.NewResolver(a.Catalog, a.secrets)
	switch req.Operation {
	case "models.resolve":
		var input struct {
			Reference string `json:"reference"`
			Operation string `json:"operation"`
		}
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return resolver.Resolve(a.ctx, input.Reference, input.Operation)
	case "models.discover":
		var input struct {
			SourceID string `json:"source_id"`
		}
		if strictPayload(req.Data, &input) != nil || !contracts.ValidID(input.SourceID) {
			return nil, contracts.Fail("invalid_request")
		}
		items, err := resolver.Discover(a.ctx, input.SourceID)
		return map[string]any{"items": items}, err
	case "models.alias.list", "models.source.list":
		page, err := pageInput(req.Data)
		if err != nil {
			return nil, err
		}
		if req.Operation == "models.alias.list" {
			all, err := a.Catalog.ModelAliases(a.ctx)
			return pageViews(all, page, func(v catalog.ModelAlias) string { return v.ID }, func(v catalog.ModelAlias) any { return v }), err
		}
		all, err := a.Catalog.ModelSources(a.ctx)
		return pageViews(all, page, func(v catalog.ModelSource) string { return v.ID }, func(v catalog.ModelSource) any { return v }), err
	case "models.alias.show":
		return a.Catalog.ModelAlias(a.ctx, req.ItemID)
	case "models.source.show":
		return a.Catalog.ModelSource(a.ctx, req.ItemID)
	case "models.alias.set":
		var input struct {
			Expected int64              `json:"expected_revision"`
			Alias    catalog.ModelAlias `json:"alias"`
		}
		if strictPayload(req.Data, &input) != nil || input.Alias.ID != req.ItemID || input.Alias.Revision != 0 {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.PutModelAlias(a.ctx, req.RequestID, input.Expected, input.Alias)
	case "models.source.set":
		var input struct {
			Expected int64               `json:"expected_revision"`
			Source   catalog.ModelSource `json:"source"`
		}
		if strictPayload(req.Data, &input) != nil || input.Source.ID != req.ItemID || input.Source.Revision != 0 {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.PutModelSource(a.ctx, req.RequestID, input.Expected, input.Source)
	case "models.alias.remove", "models.source.remove":
		var input struct {
			Expected int64 `json:"expected_revision"`
		}
		if strictPayload(req.Data, &input) != nil || input.Expected < 1 {
			return nil, contracts.Fail("invalid_request")
		}
		if req.Operation == "models.alias.remove" {
			return a.Catalog.RemoveModelAlias(a.ctx, req.RequestID, input.Expected, req.ItemID)
		}
		return a.Catalog.RemoveModelSource(a.ctx, req.RequestID, input.Expected, req.ItemID)
	}
	return nil, contracts.Fail("invalid_request")
}
