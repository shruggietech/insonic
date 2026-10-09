// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/subtitles"
)

type realWorker struct {
	claim  catalog.Work
	cancel context.CancelFunc
}

func strictPayload(raw []byte, out any) error {
	if catalog.ValidateJSON(raw) != nil {
		return contracts.Fail("invalid_request")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return contracts.Fail("invalid_request")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return contracts.Fail("invalid_request")
	}
	return nil
}

func (a *App) libraryService() (*library.Service, error) {
	artifacts, e := a.artifactService()
	if e != nil {
		return nil, e
	}
	tools, e := ReadMediaTools(a.Workspace)
	if e != nil {
		return nil, e
	}
	service := library.NewService(artifacts, a.Catalog, a.secrets, tools)
	service.NativeIngest = func(ctx context.Context, data []byte, format string) (json.RawMessage, error) {
		config, err := ReadProcessingTools(a.Workspace)
		if err != nil {
			return nil, err
		}
		driver, err := subtitles.New(config.Cueson)
		if err != nil {
			return nil, err
		}
		return driver.Ingest(ctx, data, format)
	}
	service.Diarize = func(ctx context.Context, entry catalog.LibraryEntry, document json.RawMessage, model string) (subtitles.Admission, json.RawMessage, error) {
		result := subtitles.Admission{Document: document, OriginalVersion: subtitles.SchemaVersion, AttributionBasis: []subtitles.Observation{}}
		if !contracts.ValidID(model) {
			return result, nil, contracts.Fail("invalid_request")
		}
		executor, err := a.recordingExecutor()
		if err != nil {
			return result, nil, err
		}
		session, err := executor.Prepare(ctx, entry, processing.AudioOptions{})
		if err != nil {
			return result, nil, err
		}
		defer session.Close()
		diarization, err := session.Diarize(ctx, model, processing.DiarizationOptions{})
		if err != nil {
			return result, nil, err
		}
		turns, err := localTurns(entry.ID, library.DerivedID(entry.ID, documentHash(document)), diarization.Turns)
		if err != nil {
			return result, nil, err
		}
		assembly, err := executor.Subtitles.Assemble(ctx, document, nil, turns, nil)
		if err != nil {
			return result, nil, err
		}
		var before, after map[string]json.RawMessage
		json.Unmarshal(document, &before)
		json.Unmarshal(assembly.Document, &after)
		if timing, ok := before["media_timing"]; ok {
			after["media_timing"] = timing
		} else {
			delete(after, "media_timing")
		}
		result.Document, _ = json.Marshal(after)
		if err = subtitles.ValidateDocument(result.Document); err != nil {
			return result, nil, err
		}
		provenance, _ := json.Marshal(map[string]any{"mode": "supplied-text;explicit-diarization;recognition-bypassed", "audio": session.Provenance, "source_map": session.SourceMap, "diarization": diarization.Provenance, "diagnostics": flattenDiagnostics([]any{session.Diagnostics, diarization.Diagnostics, assembly.Diagnostics})})
		return result, provenance, nil
	}
	return service, nil
}
func (a *App) modelService() (*models.Service, error) {
	artifacts, e := a.artifactService()
	if e != nil {
		return nil, e
	}
	return models.NewService(artifacts, a.Catalog, a.secrets), nil
}

// Real work has bounded parallelism and separate renewable durable authority.
// Recovery reconciles accepted bytes and checkpoints before executing effects.
func (a *App) recoverWork() {
	a.workMu.Lock()
	defer a.workMu.Unlock()
	if a.ctx.Err() != nil {
		return
	}
	for id, w := range a.workers {
		claim, e := a.Catalog.RenewWork(a.ctx, w.claim, leaseTTL)
		if e != nil {
			w.cancel()
			delete(a.workers, id)
		} else {
			w.claim = claim
		}
	}
	if len(a.workers) >= 4 {
		return
	}
	list, e := a.Catalog.Works(a.ctx)
	if e != nil {
		return
	}
	for _, item := range list {
		if item.Kind == "recordings.assemble" {
			continue
		}
		if len(a.workers) >= 4 {
			break
		}
		if a.workers[item.ID] != nil {
			continue
		}
		if item.State != "pending" && item.State != "interrupted" && item.State != "running" {
			continue
		}
		claim, e := a.Catalog.ClaimWork(a.ctx, item.ID, a.Session, leaseTTL)
		if e != nil {
			continue
		}
		ctx, cancel := context.WithCancel(a.ctx)
		worker := &realWorker{claim: claim, cancel: cancel}
		a.workers[item.ID] = worker
		a.wg.Add(1)
		go a.executeWork(ctx, worker, claim)
	}
}
func (a *App) executeWork(ctx context.Context, worker *realWorker, claim catalog.Work) {
	defer a.wg.Done()
	var result any
	var e error
	if claim.Kind == "evidence.extract" {
		result, e = a.processEvidence(ctx, claim)
	} else if claim.Kind == "recordings.process" {
		result, e = a.processRecording(ctx, claim)
	} else if strings.HasPrefix(claim.Kind, "models.") {
		var s *models.Service
		s, e = a.modelService()
		if e == nil {
			result, e = s.Execute(ctx, claim)
		}
	} else {
		var s *library.Service
		s, e = a.libraryService()
		if e == nil {
			result, e = s.Execute(ctx, claim)
		}
	}
	// Refresh exposes a full entry to direct callers, but journals hold references.
	if entry, ok := result.(catalog.LibraryEntry); ok {
		result = map[string]any{"media_id": entry.ID, "revision": entry.Revision, "state": "admitted"}
	}
	a.workMu.Lock()
	defer a.workMu.Unlock()
	if a.workers[claim.ID] != worker {
		return
	}
	defer delete(a.workers, claim.ID)
	defer worker.cancel()
	if ctx.Err() != nil {
		return
	}
	state := "succeeded"
	phase := "complete"
	if imported, ok := result.(library.ImportResult); ok && e == nil {
		for _, item := range imported.Items {
			if item.State == "failed" {
				state, phase = "failed", "completed-with-failures"
				break
			}
		}
	}
	if e != nil {
		state = "failed"
		phase = "operation-failed"
		code := "operation_failed"
		if typed, ok := e.(*contracts.Error); ok {
			code = typed.Code
		}
		result = map[string]any{"state": "failed", "error": code}
	}
	raw, _ := json.Marshal(result)
	_, _ = a.Catalog.CheckpointWork(ctx, worker.claim, phase, state, raw, leaseTTL)
}

func domainRequestValid(req contracts.Request) bool {
	if contracts.ExploreOperation(req.Operation) {
		return contracts.ExploreRequestValid(req)
	}
	if contracts.DesktopOperation(req.Operation) {
		return contracts.DesktopRequestValid(req)
	}
	if configuredOperation(req.Operation) {
		return configuredRequestValid(req)
	}
	if strings.HasPrefix(req.Operation, "recordings.") {
		return recordingRequestValid(req)
	}
	isDomain := strings.HasPrefix(req.Operation, "media.") || strings.HasPrefix(req.Operation, "models.") || strings.HasPrefix(req.Operation, "work.") || strings.HasPrefix(req.Operation, "credentials.")
	if !isDomain {
		return req.ItemID == "" && len(req.Data) == 0
	}
	if req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 {
		return false
	}
	list := req.Operation == "media.list" || req.Operation == "models.list" || req.Operation == "work.list"
	input := req.Operation == "media.import" || req.Operation == "models.register" || req.Operation == "models.acquire" || strings.HasPrefix(req.Operation, "credentials.")
	update := req.Operation == "media.refresh" || req.Operation == "media.set-origin" || req.Operation == "media.relocate"
	return (list && req.ItemID == "") || (req.Operation == "work.results" && contracts.ValidID(req.ItemID)) || (input && req.ItemID == "" && len(req.Data) > 0) || (update && contracts.ValidID(req.ItemID) && len(req.Data) > 0) || (!list && !input && !update && contracts.ValidID(req.ItemID) && len(req.Data) == 0)
}
func (a *App) domainDispatch(req contracts.Request) (result any, err error) {
	if configuredOperation(req.Operation) {
		return a.configuredDispatch(req)
	}
	defer func() {
		if w, ok := result.(catalog.Work); ok {
			result = workView(w)
		}
		if entry, ok := result.(catalog.LibraryEntry); ok {
			raw, _ := json.Marshal(entry)
			if len(raw) > 512<<10 {
				result = map[string]any{"media_id": entry.ID, "revision": entry.Revision, "metadata_inline": false, "report_publication_ids": entry.ReportPublicationIDs}
			}
		}
	}()
	if strings.HasPrefix(req.Operation, "recordings.") {
		if strings.HasPrefix(req.Operation, "recordings.roster.") {
			return a.rosterOperation(req)
		}
		return a.recordingDispatch(req)
	}
	switch req.Operation {
	case "work.list":
		p, e := pageInput(req.Data)
		if e != nil {
			return nil, e
		}
		all, e := a.Catalog.Works(a.ctx)
		if e != nil {
			return nil, e
		}
		return pageViews(all, p, func(w catalog.Work) string { return w.ID }, func(w catalog.Work) any { view := workView(w).(map[string]any); delete(view, "result"); return view }), nil
	case "work.results":
		return a.workResults(req)
	case "work.show":
		return a.Catalog.Work(a.ctx, req.ItemID)
	case "work.cancel":
		out, e := a.Catalog.CancelWork(a.ctx, req.RequestID, req.ItemID)
		a.workMu.Lock()
		if w := a.workers[req.ItemID]; w != nil && e == nil {
			w.cancel()
		}
		a.workMu.Unlock()
		return out, e
	case "work.retry":
		current, e := a.Catalog.Work(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		if current.Kind == "recordings.assemble" {
			return nil, &contracts.Error{Code: "invalid_request", Message: "Resubmit the assembly inputs through recordings.assemble to retry this job."}
		}
		out, e := a.Catalog.RetryWork(a.ctx, req.RequestID, req.ItemID)
		if e == nil {
			a.recoverWork()
		}
		return out, e
	case "models.register", "models.acquire":
		var input models.Request
		if strictPayload(req.Data, &input) != nil || input.Manifest.Validate() != nil {
			return nil, contracts.Fail("invalid_request")
		}
		out, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, req.Operation, req.Data)
		if e == nil {
			a.recoverWork()
		}
		return out, e
	case "models.list":
		p, e := pageInput(req.Data)
		if e != nil {
			return nil, e
		}
		all, e := a.Catalog.BaseModels(a.ctx)
		if e != nil {
			return nil, e
		}
		return pageViews(all, p, func(w catalog.BaseModelInstall) string { return w.ID }, modelSummary), nil
	case "models.show":
		return a.Catalog.BaseModel(a.ctx, req.ItemID)
	case "models.verify", "models.materialize":
		s, e := a.modelService()
		if e != nil {
			return nil, e
		}
		if req.Operation == "models.verify" {
			e = s.Verify(a.ctx, req.ItemID)
			return map[string]bool{"verified": e == nil}, e
		}
		return s.Materialize(a.ctx, req.ItemID)
	case "media.import":
		var input library.ImportRequest
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		prepared, e := library.PrepareImport(input)
		if e != nil {
			return nil, e
		}
		prepared.RequestDigest = ""
		for i := range prepared.Items {
			if prepared.Items[i].AcceptedReceipt != "" {
				return nil, contracts.Fail("invalid_request")
			}
			prepared.Items[i].TargetError = ""
			prepared.Items[i].ResolvedSpeakerIDs = nil
		}
		original, _ := json.Marshal(prepared)
		requestDigest := documentHash(original)
		if prior, err := a.Catalog.Work(a.ctx, req.RequestID); err == nil {
			var frozen library.ImportRequest
			if prior.Kind != req.Operation || json.Unmarshal(prior.Payload, &frozen) != nil || frozen.RequestDigest != requestDigest {
				return nil, contracts.Fail("conflict")
			}
			return workView(prior), nil
		} else {
			typed, ok := err.(*contracts.Error)
			if !ok || typed.Code != "not_found" {
				return nil, err
			}
		}
		prepared = library.FreezeTargets(a.ctx, a.Catalog, prepared)
		prepared.RequestDigest = requestDigest
		raw, _ := json.Marshal(prepared)
		out, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, req.Operation, raw)
		if e == nil {
			a.recoverWork()
		}
		return out, e
	case "media.refresh":
		var input library.Options
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		raw, _ := json.Marshal(library.RefreshRequest{MediaID: req.ItemID, Options: input})
		out, e := a.Catalog.EnqueueWork(a.ctx, req.RequestID, req.Operation, raw)
		if e == nil {
			a.recoverWork()
		}
		return out, e
	case "media.list", "media.show", "media.metadata", "media.raw", "media.set-origin", "media.relocate":
		s, e := a.libraryService()
		if e != nil {
			return nil, e
		}
		switch req.Operation {
		case "media.list":
			p, e := pageInput(req.Data)
			if e != nil {
				return nil, e
			}
			all, e := a.Catalog.Libraries(a.ctx)
			if e != nil {
				return nil, e
			}
			return pageViews(all, p, func(w catalog.LibraryEntry) string { return w.ID }, func(w catalog.LibraryEntry) any {
				v, e := s.Show(a.ctx, w.ID)
				if e != nil {
					return map[string]any{"media_id": w.ID, "availability": "unavailable"}
				}
				return mediaSummary(v)
			}), nil
		case "media.show":
			v, e := s.Show(a.ctx, req.ItemID)
			if e != nil {
				return nil, e
			}
			raw, _ := json.Marshal(v)
			if len(raw) > 512<<10 {
				view := mediaSummary(v).(map[string]any)
				view["metadata_inline"] = false
				view["report_publication_ids"] = v.ReportPublicationIDs
				return view, nil
			}
			return v, nil
		case "media.metadata":
			v, e := s.Metadata(a.ctx, req.ItemID)
			if e != nil {
				return nil, e
			}
			raw, _ := json.Marshal(v)
			if len(raw) > 512<<10 {
				req.Operation = "media.raw"
				reports, e := a.domainDispatch(req)
				return map[string]any{"encoding": "capture-bundle", "metadata_inline": false, "reports": reports}, e
			}
			return v, nil
		case "media.raw":
			entry, e := a.Catalog.Library(a.ctx, req.ItemID)
			if e != nil {
				return nil, e
			}
			ids, e := catalog.PublicationIDs(entry.ReportPublicationIDs)
			if e != nil {
				return nil, e
			}
			out := []any{}
			for _, id := range ids {
				m, e := s.Artifacts.MaterializeBound(a.ctx, id, 64<<20)
				if e != nil {
					return nil, e
				}
				out = append(out, m)
			}
			return out, nil
		case "media.set-origin":
			var input struct {
				Revision int64           `json:"revision"`
				Options  library.Options `json:"options"`
			}
			if strictPayload(req.Data, &input) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			return s.SetOrigin(a.ctx, req.RequestID, req.ItemID, input.Revision, input.Options)
		case "media.relocate":
			var input struct {
				Revision int64  `json:"revision"`
				Path     string `json:"path"`
			}
			if strictPayload(req.Data, &input) != nil {
				return nil, contracts.Fail("invalid_request")
			}
			return s.Relocate(a.ctx, req.RequestID, req.ItemID, input.Revision, input.Path)
		}
	}
	return nil, contracts.Fail("invalid_request")
}
