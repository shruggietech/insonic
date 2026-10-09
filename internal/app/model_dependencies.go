// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/voicemodels"
)

// A shared acquisition is independent durable work. Waiting consumers never
// hold scheduler slots and cancelling one consumer never cancels another's bytes.
func (a *App) enqueueModelAcquisition(ctx context.Context, selected models.Resolution, repair bool) (catalog.Work, error) {
	if selected.Target.Kind != "base" || selected.Manifest == nil || selected.Manifest.Digest() != selected.Digest || models.InstallationID(*selected.Manifest) != selected.Target.ID {
		return catalog.Work{}, contracts.Fail("invalid_request")
	}
	id := models.AcquisitionID(a.Workspace.Config.WorkspaceID, *selected.Manifest)
	raw, _ := json.Marshal(models.Request{Manifest: *selected.Manifest})
	work, err := a.Catalog.EnqueueWork(ctx, id, "models.acquire", raw)
	if err != nil {
		return work, err
	}
	if repair && (work.State == "failed" || work.State == "cancelled" || work.State == "interrupted" || work.State == "succeeded") {
		work, err = a.Catalog.RetryWork(ctx, contracts.ID(), id)
		if typed, ok := err.(*contracts.Error); ok && typed.Code == "conflict" {
			// A concurrent consumer may already have restarted the same frozen bundle.
			current, e := a.Catalog.Work(ctx, id)
			if e == nil && (current.State == "pending" || current.State == "running") {
				return current, nil
			}
		}
	}
	return work, err
}

func (a *App) queueModelSelections(ctx context.Context, selections []models.Resolution) ([]string, error) {
	dependencies := []string{}
	for _, selected := range selections {
		if selected.Target.Kind != "base" {
			continue
		}
		if selected.Manifest != nil && slices.Contains(dependencies, models.AcquisitionID(a.Workspace.Config.WorkspaceID, *selected.Manifest)) {
			continue
		}
		s, err := a.modelService()
		if err != nil {
			return nil, err
		}
		verified := s.Verify(ctx, selected.Target.ID) == nil
		work, err := a.enqueueModelAcquisition(ctx, selected, false)
		if err == nil && (work.State == "failed" || work.State == "cancelled" || work.State == "interrupted" || work.State == "succeeded" && !verified) {
			work, err = a.enqueueModelAcquisition(ctx, selected, true)
		}
		if err != nil {
			return nil, err
		}
		if !slices.Contains(dependencies, work.ID) {
			dependencies = append(dependencies, work.ID)
		}
	}
	return dependencies, nil
}

func frozenModelDependencies(work catalog.Work) ([]models.Resolution, []string, error) {
	switch work.Kind {
	case "models.train":
		var p voicemodels.TrainPayload
		if strictPayload(work.Payload, &p) != nil {
			return nil, nil, contracts.Fail("invalid_request")
		}
		return p.ModelSelections, p.ModelDependencies, nil
	case "recordings.match":
		var p voicemodels.MatchPayload
		if strictPayload(work.Payload, &p) != nil {
			return nil, nil, contracts.Fail("invalid_request")
		}
		return p.ModelSelections, p.ModelDependencies, nil
	case "recordings.process", "models.ensure":
		var p recordingPayload
		if strictPayload(work.Payload, &p) != nil {
			return nil, nil, contracts.Fail("invalid_request")
		}
		return p.ModelSelections, p.ModelDependencies, nil
	case "media.import", "library.import":
		var p library.ImportRequest
		if strictPayload(work.Payload, &p) != nil {
			return nil, nil, contracts.Fail("invalid_request")
		}
		selected := []models.Resolution{}
		for _, raw := range p.ModelSelections {
			var model models.Resolution
			if strictPayload(raw, &model) != nil {
				return nil, nil, contracts.Fail("invalid_request")
			}
			selected = append(selected, model)
		}
		return selected, p.ModelDependencies, nil
	default:
		return nil, nil, nil
	}
}

func (a *App) dependenciesReady(work catalog.Work) (bool, error) {
	selections, ids, err := frozenModelDependencies(work)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		found := false
		for _, selection := range selections {
			if selection.Manifest != nil && selection.Target.Kind == "base" && models.AcquisitionID(a.Workspace.Config.WorkspaceID, *selection.Manifest) == id {
				found = true
				break
			}
		}
		if !found {
			return false, contracts.Fail("invalid_request")
		}
		dependency, e := a.Catalog.Work(a.ctx, id)
		if e != nil {
			return false, e
		}
		if dependency.Kind != "models.acquire" {
			return false, contracts.Fail("invalid_request")
		}
		if dependency.State == "failed" || dependency.State == "cancelled" {
			return false, contracts.Fail("model_acquisition_failed")
		}
		if dependency.State != "succeeded" {
			return false, nil
		}
	}
	return true, nil
}

func (a *App) verifyElectedModels(ctx context.Context, selections []models.Resolution) error {
	verified := map[string]bool{}
	for _, selected := range selections {
		if selected.Target.Kind != "base" {
			continue
		}
		install, err := a.Catalog.BaseModel(ctx, selected.Target.ID)
		if err != nil {
			return err
		}
		if selected.Manifest == nil || selected.Manifest.Digest() != selected.Digest || install.Digest != selected.Digest {
			return contracts.Fail("conflict")
		}
		if selected.Target.Operation != "" {
			if err = models.CheckCompatibility(*selected.Manifest, selected.Target.Operation, selected.Target.Adapter, selected.Target.ContractVersion); err != nil {
				return err
			}
		}
		key := selected.Target.ID + ":" + selected.Digest
		if verified[key] {
			continue
		}
		service, err := a.modelService()
		if err != nil {
			return err
		}
		if err = service.Verify(ctx, selected.Target.ID); err != nil {
			return err
		}
		verified[key] = true
	}
	return nil
}

func validateModelElection(p recordingPayload) error {
	for _, selection := range p.ModelSelections {
		if selection.Target.Kind != "base" || selection.Manifest == nil || models.InstallationID(*selection.Manifest) != selection.Target.ID || selection.Manifest.Digest() != selection.Digest || p.ModelDigests[selection.Target.ID] != selection.Digest {
			return contracts.Fail("invalid_request")
		}
		switch selection.Target.Operation {
		case "transcription":
			if p.Options.Transcription != "generate" || p.Options.RecognitionModelID != selection.Target.ID {
				return contracts.Fail("invalid_request")
			}
		case "diarization":
			if p.Options.Diarization == "reuse" || p.Options.DiarizationModelID != selection.Target.ID {
				return contracts.Fail("invalid_request")
			}
		default:
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

func modelSelectionSummaries(selections []models.Resolution) []models.Resolution {
	summaries := []models.Resolution{}
	for _, selection := range selections[:min(len(selections), 100)] {
		selection.Manifest = nil
		selection.Lineage = nil
		summaries = append(summaries, selection)
	}
	return summaries
}

func (a *App) electProcessingModels(o RecordingOptions, election *recordingElection) (RecordingOptions, *recordingElection, []models.Resolution, error) {
	// Historical deterministic fixtures deliberately replace model management.
	// Model orchestration fixtures explicitly retain real election and artifacts.
	if a.recordingFactory != nil && !a.enforceModelElection {
		return o, election, nil, nil
	}
	resolver := models.NewResolver(a.Catalog, a.secrets)
	selections := []models.Resolution{}
	for _, stage := range []struct {
		operation  string
		selected   bool
		id         *string
		configured *pipeline.Stage
	}{
		{"transcription", o.Transcription == "generate", &o.RecognitionModelID, nil},
		{"diarization", o.Diarization != "reuse", &o.DiarizationModelID, nil},
	} {
		if !stage.selected {
			continue
		}
		var configured *pipeline.Stage
		if election != nil {
			if stage.operation == "transcription" {
				configured = &election.Definition.Recognition
			} else {
				configured = &election.Definition.Diarization
			}
			if configured.Mode == "hosted" {
				continue
			}
		}
		selected, err := resolver.Resolve(a.ctx, *stage.id, stage.operation)
		if err != nil {
			return o, election, nil, err
		}
		if !selected.Compatible {
			return o, election, nil, &contracts.Error{Code: "unsupported_capability", Message: "Selected model is incompatible with the elected operation; inspect models resolve for diagnostics."}
		}
		if selected.Target.Kind != "base" {
			return o, election, nil, localModelSelectionError(selected)
		}
		adapter, version := selected.Target.Adapter, selected.Target.ContractVersion
		if configured != nil {
			adapter, version = configured.Adapter, configured.Version
		}
		if err = models.CheckCompatibility(*selected.Manifest, stage.operation, adapter, version); err != nil {
			return o, election, nil, err
		}
		*stage.id = selected.Target.ID
		if configured != nil {
			configured.ModelID = selected.Target.ID
		}
		selections = append(selections, selected)
	}
	return o, election, selections, nil
}

func (a *App) freezeImportModels(p library.ImportRequest) (library.ImportRequest, error) {
	resolver := models.NewResolver(a.Catalog, a.secrets)
	selected := []models.Resolution{}
	resolved := map[string]models.Resolution{}
	for i := range p.Items {
		item := &p.Items[i]
		attribution := item.Attribution
		if attribution == "" {
			attribution = p.Defaults.Attribution
		}
		if attribution != "diarize" {
			continue
		}
		ref := item.DiarizationModelID
		if ref == "" {
			ref = p.Defaults.DiarizationModelID
		}
		model, found := resolved[ref]
		if !found {
			var err error
			model, err = resolver.Resolve(a.ctx, ref, "diarization")
			if err != nil {
				return p, err
			}
			resolved[ref] = model
		}
		if !model.Compatible {
			return p, contracts.Fail("unsupported_capability")
		}
		if model.Target.Kind != "base" {
			return p, localModelSelectionError(model)
		}
		item.DiarizationModelID = model.Target.ID
		item.DiarizationModelDigest = model.Digest
		found = false
		for _, old := range selected {
			if old.Reference == model.Reference && old.Target.ID == model.Target.ID && old.Digest == model.Digest {
				found = true
			}
		}
		if !found {
			selected = append(selected, model)
		}
	}
	p.ModelSelections = nil
	for _, model := range selected {
		raw, _ := json.Marshal(model)
		p.ModelSelections = append(p.ModelSelections, raw)
	}
	var err error
	p.ModelDependencies, err = a.queueModelSelections(a.ctx, selected)
	return p, err
}

func localModelSelectionError(selection models.Resolution) error {
	if selection.Target.Kind == "hosted" {
		return &contracts.Error{Code: "unsupported_capability", Message: "This reference identifies a hosted provider handle, not a downloadable local model. Select an explicit configured hosted pipeline to use that provider."}
	}
	return &contracts.Error{Code: "unsupported_capability", Message: "This reference identifies a trained speaker version. This local transcription or diarization operation requires a compatible base-model bundle."}
}

func (a *App) modelWorkView(work catalog.Work) any {
	view := workView(work).(map[string]any)
	selections, dependencies, err := frozenModelDependencies(work)
	if err != nil {
		return view
	}
	if len(selections) > 0 {
		view["model_selections"] = modelSelectionSummaries(selections)
		view["model_selection_count"] = len(selections)
	}
	if len(dependencies) > 0 {
		view["acquisition_ids"] = dependencies[:min(len(dependencies), 100)]
		view["acquisition_count"] = len(dependencies)
		if work.State == "pending" || work.State == "interrupted" {
			ready, e := a.dependenciesReady(work)
			if !ready && e == nil {
				view["phase"] = "acquiring-models"
			}
		}
	}
	return view
}
