// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/explore"
	"github.com/shruggietech/insonic/internal/graph"
	"sort"
	"strings"
)

type queryRequest struct {
	contracts.QueryInput
	SavedID  string `json:"saved_id,omitempty"`
	Revision int64  `json:"revision,omitempty"`
}

func (a *App) queryInput(raw json.RawMessage) (contracts.QueryInput, error) {
	var req queryRequest
	if strictPayload(raw, &req) != nil {
		return contracts.QueryInput{}, contracts.Fail("invalid_request")
	}
	q := req.QueryInput
	if req.SavedID != "" {
		if q.Definition.Mode != "" || q.Title != "" || len(q.Parameters) > 0 {
			return q, contracts.Fail("invalid_request")
		}
		versions, e := a.Catalog.SavedQueries(a.ctx, req.SavedID)
		if e != nil {
			return q, e
		}
		found := false
		for _, v := range versions {
			if req.Revision == v.Revision || req.Revision == 0 {
				if strictPayload(v.Definition, &q) != nil {
					return q, contracts.Fail("operation_failed")
				}
				found = true
			}
		}
		if !found {
			return q, contracts.Fail("not_found")
		}
	} else if req.Revision != 0 {
		return q, contracts.Fail("invalid_request")
	}
	return q, q.Validate()
}
func queryDocument(w string, q catalog.SavedQuery) any {
	var input contracts.QueryInput
	_ = json.Unmarshal(q.Definition, &input)
	if input.Parameters == nil {
		input.Parameters = map[string]contracts.QueryParameter{}
	}
	return map[string]any{"kind": "graph-query", "schema_version": contracts.Version, "workspace_id": w, "query_id": q.ID, "revision": q.Revision, "title": input.Title, "access_mode": "read-only", "definition": input.Definition, "parameters": input.Parameters, "validations": q.Validations}
}
func (a *App) exploreDispatch(req contracts.Request) (any, error) {
	switch req.Operation {
	case "evidence.extract":
		return a.submitExtraction(req)
	case "evidence.show":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.Extraction(a.ctx, req.ItemID)
	case "query.list":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		all, e := a.Catalog.SavedQueries(a.ctx, "")
		if e != nil {
			return nil, e
		}
		latest := map[string]catalog.SavedQuery{}
		for _, v := range all {
			latest[v.ID] = v
		}
		items := []any{}
		ids := []string{}
		for id := range latest {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			items = append(items, queryDocument(req.WorkspaceID, latest[id]))
		}
		return map[string]any{"items": items}, nil
	case "query.show":
		var in struct {
			Revision int64 `json:"revision,omitempty"`
		}
		if len(req.Data) > 0 && strictPayload(req.Data, &in) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		versions, e := a.Catalog.SavedQueries(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		items := []any{}
		for _, v := range versions {
			if in.Revision == 0 || in.Revision == v.Revision {
				items = append(items, queryDocument(req.WorkspaceID, v))
			}
		}
		if len(items) == 0 {
			return nil, contracts.Fail("not_found")
		}
		return map[string]any{"items": items}, nil
	case "query.save":
		var in struct {
			ExpectedRevision int64                `json:"expected_revision"`
			Query            contracts.QueryInput `json:"query"`
		}
		if strictPayload(req.Data, &in) != nil || in.Query.Validate() != nil || strings.TrimSpace(in.Query.Title) == "" {
			return nil, contracts.Fail("invalid_request")
		}
		if in.Query.Definition.Mode == "native" {
			if e := graph.ValidateNative(in.Query.Definition.Dialect, in.Query.Definition.Text, in.Query.Parameters); e != nil {
				return nil, e
			}
		}
		raw, _ := json.Marshal(in.Query)
		q, e := a.Catalog.PutSavedQuery(a.ctx, req.RequestID, in.ExpectedRevision, catalog.SavedQuery{ID: req.ItemID, Definition: raw, Validations: a.queryCompatibility(in.Query)})
		if e != nil {
			return nil, e
		}
		return queryDocument(req.WorkspaceID, q), nil
	case "views.show":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.GraphLayout(a.ctx, req.ItemID)
	case "views.save":
		var in struct {
			ExpectedRevision int64           `json:"expected_revision"`
			QueryID          string          `json:"query_id"`
			Layout           json.RawMessage `json:"layout"`
		}
		if strictPayload(req.Data, &in) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.PutGraphLayout(a.ctx, req.RequestID, in.ExpectedRevision, catalog.GraphLayout{ID: req.ItemID, QueryID: in.QueryID, Layout: in.Layout})
	case "timeline.calendar":
		var in struct {
			Filter     contracts.CalendarFilter   `json:"filter"`
			Pagination *contracts.QueryPagination `json:"pagination,omitempty"`
		}
		if strictPayload(req.Data, &in) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		snap, e := a.Catalog.Export(a.ctx)
		if e != nil {
			return nil, e
		}
		c, e := explore.Build(snap)
		if e != nil {
			return nil, e
		}
		return explore.Calendar(c, in.Filter, in.Pagination)
	case "timeline.recording":
		var in struct {
			Pagination *contracts.QueryPagination `json:"pagination,omitempty"`
		}
		if len(req.Data) > 0 && strictPayload(req.Data, &in) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		snap, e := a.Catalog.Export(a.ctx)
		if e != nil {
			return nil, e
		}
		c, e := explore.Build(snap)
		if e != nil {
			return nil, e
		}
		return explore.Query(c, c.Refs, contracts.QueryInput{Definition: contracts.QueryDefinition{Mode: "normalized", Operation: "time-range", Filters: contracts.QueryFilter{MediaIDs: []string{req.ItemID}}, OrderBy: []contracts.QueryOrder{{Field: "source_start_us", Direction: "asc"}}, Pagination: in.Pagination}})
	case "query.validate":
		q, e := a.queryInput(req.Data)
		if e != nil {
			return nil, e
		}
		if q.Definition.Mode == "native" {
			if e = graph.ValidateNative(q.Definition.Dialect, q.Definition.Text, q.Parameters); e != nil {
				return nil, e
			}
		}
		return map[string]any{"definition_valid": true, "access_mode": "read-only", "schema_version": 1, "validations": a.queryCompatibility(q)}, nil
	}
	a.graphMu.Lock()
	defer a.graphMu.Unlock()
	switch req.Operation {
	case "graph.status":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		return a.graphState(a.ctx)
	case "graph.capabilities":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		g, e := a.openGraph(a.ctx)
		if e != nil {
			return map[string]any{"adapter_id": a.Workspace.Config.Profiles.Graph.Adapter, "state": "unavailable", "error": e, "operations": []string{}}, nil
		}
		return g.Capabilities(), nil
	case "graph.publish", "graph.rebuild":
		if len(req.Data) > 0 {
			return nil, contracts.Fail("invalid_request")
		}
		if req.Operation == "graph.rebuild" {
			if e := a.rebuildGraph(a.ctx, req.RequestID); e != nil {
				return nil, e
			}
			return a.graphState(a.ctx)
		}
		_, _, e := a.publishGraph(a.ctx)
		if e != nil {
			return nil, e
		}
		return a.graphState(a.ctx)
	case "query.run", "query.explain":
		q, e := a.queryInput(req.Data)
		if e != nil {
			return nil, e
		}
		if q.Definition.Mode == "native" {
			if e = graph.ValidateNative(q.Definition.Dialect, q.Definition.Text, q.Parameters); e != nil {
				return nil, e
			}
		}
		g, c, e := a.publishGraph(a.ctx)
		if e != nil {
			if q.Definition.Mode == "normalized" && req.Operation == "query.run" {
				return a.catalogQuery(q, e)
			}
			return nil, e
		}
		if q.Definition.Mode == "native" {
			var rows graph.Rows
			if req.Operation == "query.explain" {
				rows, e = g.Explain(a.ctx, q.Definition.Dialect, q.Definition.Text, q.Parameters)
			} else {
				rows, e = g.Query(a.ctx, q.Definition.Dialect, q.Definition.Text, q.Parameters)
			}
			if e != nil {
				return nil, e
			}
			status, e := a.Catalog.GraphStatus(a.ctx)
			if e != nil {
				return nil, e
			}
			if status["pending_events"] > 0 {
				return nil, contracts.Fail("conflict")
			}
			return map[string]any{"items": rows, "dialect": q.Definition.Dialect, "catalog_revision": c.Revision, "freshness": status}, nil
		}
		if req.Operation == "query.explain" {
			return map[string]any{"operation": q.Definition.Operation, "adapter": g.Capabilities(), "plan": "typed current-reference selection, catalog hydration, filtering, deterministic ordering and bounded pagination"}, nil
		}
		refs, e := g.ReadRefs(a.ctx, req.WorkspaceID)
		if e != nil {
			return nil, e
		}
		status, e := a.Catalog.GraphStatus(a.ctx)
		if e != nil {
			return nil, e
		}
		if status["pending_events"] > 0 {
			return nil, contracts.Fail("conflict")
		}
		return explore.Query(c, refs, q)
	}
	return nil, contracts.Fail("invalid_request")
}
