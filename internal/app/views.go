// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"sort"
)

type listPage struct {
	After string `json:"after_id,omitempty"`
	Limit int    `json:"limit,omitempty"`
}

func pageInput(raw []byte) (listPage, error) {
	p := listPage{Limit: 100}
	if len(raw) > 0 && strictPayload(raw, &p) != nil {
		return p, contracts.Fail("invalid_request")
	}
	if p.Limit == 0 {
		p.Limit = 100
	}
	if p.Limit < 1 || p.Limit > 100 || (p.After != "" && !contracts.ValidID(p.After)) {
		return p, contracts.Fail("invalid_request")
	}
	return p, nil
}
func pageViews[T any](all []T, p listPage, id func(T) string, view func(T) any) any {
	sort.Slice(all, func(i, j int) bool { return id(all[i]) < id(all[j]) })
	items := []any{}
	next := ""
	for _, item := range all {
		if id(item) <= p.After {
			continue
		}
		if len(items) == p.Limit {
			if len(items) > 0 {
				next = idFromLast(all, p, id)
			}
			break
		}
		items = append(items, view(item))
	}
	return map[string]any{"items": items, "next_id": next}
}
func idFromLast[T any](all []T, p listPage, id func(T) string) string {
	n := 0
	for _, item := range all {
		if id(item) <= p.After {
			continue
		}
		n++
		if n == p.Limit {
			return id(item)
		}
	}
	return ""
}
func workView(w catalog.Work) any {
	var result any
	if len(w.Result) <= 16<<10 {
		json.Unmarshal(w.Result, &result)
	} else {
		result = map[string]any{"state": "paged", "operation": "work.results"}
	}
	return map[string]any{"work_id": w.ID, "id": w.ID, "kind": w.Kind, "state": w.State, "generation": w.Generation, "phase": w.Phase, "result": result, "error": w.Error}
}
func mediaSummary(v library.EntryView) any {
	e := v.LibraryEntry
	return map[string]any{"media_id": e.ID, "id": e.ID, "title": e.Title, "class": e.Class, "mode": e.Mode, "digest": e.Digest, "size_bytes": e.Size, "duration_us": e.DurationUS, "availability": v.Availability, "revision": e.Revision}
}
func modelSummary(m catalog.BaseModelInstall) any {
	capabilities := []string{}
	compatibility := map[string]bool{"transcription": false, "diarization": false, "voice-matching": false, "speaker-model-training": false}
	var manifest models.Manifest
	if models.DecodeManifest(m.Manifest, &manifest) == nil && manifest.Digest() == m.Digest {
		capabilities = append(capabilities, manifest.Capabilities[:min(32, len(manifest.Capabilities))]...)
		for operation := range compatibility {
			adapter, version := models.DefaultAdapter(operation)
			compatibility[operation] = models.CheckCompatibility(manifest, operation, adapter, version) == nil
		}
	}
	return map[string]any{"model_id": m.ID, "id": m.ID, "name": m.Name, "model_version": m.Version, "manifest_digest": m.Digest, "state": m.State, "revision": m.Revision, "capabilities": capabilities, "operation_compatibility": compatibility}
}
func (a *App) workResults(req contracts.Request) (any, error) {
	var p struct {
		After int `json:"after_ordinal,omitempty"`
		Limit int `json:"limit,omitempty"`
	}
	p.Limit = 100
	if len(req.Data) > 0 && strictPayload(req.Data, &p) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if p.Limit == 0 {
		p.Limit = 100
	}
	if p.After < 0 || p.Limit < 1 || p.Limit > 100 {
		return nil, contracts.Fail("invalid_request")
	}
	w, e := a.Catalog.Work(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if w.Kind == "recordings.match" {
		var matching struct {
			Recording string            `json:"recording_id"`
			Decisions []json.RawMessage `json:"decisions"`
			Missing   []string          `json:"missing_profiles"`
			Mappings  int               `json:"mapping_count"`
			State     string            `json:"state"`
		}
		if len(w.Result) == 0 || json.Unmarshal(w.Result, &matching) != nil || matching.Decisions == nil {
			return workView(w), nil
		}
		if p.After > len(matching.Decisions) {
			return nil, contracts.Fail("invalid_request")
		}
		end := min(len(matching.Decisions), p.After+p.Limit)
		var next any
		if end < len(matching.Decisions) {
			next = end
		}
		return map[string]any{"work_id": w.ID, "state": w.State, "phase": w.Phase, "error": w.Error, "result": map[string]any{"recording_id": matching.Recording, "decisions": matching.Decisions[p.After:end], "missing_profiles": matching.Missing[:min(len(matching.Missing), 100)], "missing_profile_count": len(matching.Missing), "mapping_count": matching.Mappings, "state": matching.State}, "next_ordinal": next}, nil
	}
	var result library.ImportResult
	if w.Kind != "media.import" || json.Unmarshal(w.Result, &result) != nil {
		return workView(w), nil
	}
	if p.After > len(result.Items) {
		return nil, contracts.Fail("invalid_request")
	}
	end := min(len(result.Items), p.After+p.Limit)
	next := any(nil)
	if end < len(result.Items) {
		next = end
	}
	return map[string]any{"work_id": w.ID, "state": w.State, "items": result.Items[p.After:end], "next_ordinal": next, "partial": result.Partial, "captured_timezone": result.CapturedTimezone}, nil
}
