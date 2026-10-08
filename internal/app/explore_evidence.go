// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/evidence"
)

type extractionInput struct {
	ExpectedRevision int64           `json:"expected_revision"`
	DocumentDigest   string          `json:"document_digest"`
	Config           evidence.Config `json:"configuration"`
}
type extractionPayload struct {
	MediaID string          `json:"media_id"`
	Input   extractionInput `json:"input"`
}

func (a *App) submitExtraction(req contracts.Request) (any, error) {
	var in extractionInput
	if strictPayload(req.Data, &in) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	cfg, e := evidence.Normalize(in.Config)
	if e != nil {
		return nil, e
	}
	in.Config = cfg
	rec, e := a.Catalog.Recording(a.ctx, req.ItemID)
	if e != nil {
		return nil, e
	}
	if rec.Revision != in.ExpectedRevision || rec.DocumentDigest != in.DocumentDigest {
		return nil, contracts.Fail("conflict")
	}
	if rec.DocumentDigest == "" {
		return nil, contracts.Fail("invalid_request")
	}
	raw, _ := json.Marshal(extractionPayload{req.ItemID, in})
	w, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, "evidence.extract", raw)
	if e == nil {
		a.recoverWork()
	}
	return workView(w), e
}
func (a *App) processEvidence(ctx context.Context, claim catalog.Work) (any, error) {
	var payload extractionPayload
	if strictPayload(claim.Payload, &payload) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if _, accepted, e := a.Catalog.AcceptedExtractionWork(ctx, claim.ID, payload.MediaID); e != nil {
		return nil, e
	} else if accepted {
		return map[string]any{"media_id": payload.MediaID, "state": "accepted"}, nil
	}
	rec, e := a.Catalog.Recording(ctx, payload.MediaID)
	if e != nil {
		return nil, e
	}
	if rec.Revision != payload.Input.ExpectedRevision || rec.DocumentDigest != payload.Input.DocumentDigest {
		return nil, contracts.Fail("conflict")
	}
	cues, e := evidence.ParseCues(rec.Document)
	if e != nil {
		return nil, e
	}
	var adapter evidence.Adapter
	if payload.Input.Config.Adapter == "insonic-http" {
		adapter = &evidence.HTTPAdapter{Secrets: a.secrets}
	}
	extracted, e := evidence.Extract(ctx, cues, payload.Input.Config, adapter)
	if e != nil {
		return nil, e
	}
	assertions, _ := json.Marshal(extracted.Assertions)
	diagnostics, _ := json.Marshal(extracted.Diagnostics)
	// Elected route/model/version and context bounds are provenance; secret bytes
	// and complete extraction requests never enter the catalog or work journal.
	provenance, _ := json.Marshal(map[string]any{"configuration": payload.Input.Config, "chunks": extracted.Chunks, "work_id": claim.ID})
	v, e := a.Catalog.CommitExtraction(ctx, claim, catalog.Extraction{ID: rec.ID, RecordingRevision: rec.Revision, DocumentDigest: rec.DocumentDigest, Assertions: assertions, Provenance: provenance, Diagnostics: diagnostics})
	if e != nil {
		return nil, e
	}
	return map[string]any{"media_id": v.ID, "revision": v.Revision, "assertion_count": len(extracted.Assertions), "diagnostic_count": len(extracted.Diagnostics), "state": "accepted"}, nil
}
