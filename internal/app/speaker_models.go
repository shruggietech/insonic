// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/voicemodels"
)

func speakerWorkOperation(op string) bool {
	switch op {
	case "models.dataset.create", "models.dataset.list", "models.dataset.show", "models.train", "models.speaker.list", "models.speaker.show", "models.speaker.fetch", "models.profile.set", "recordings.match":
		return true
	}
	return false
}
func speakerWorkRequestValid(req contracts.Request) bool {
	if !speakerWorkOperation(req.Operation) || req.JobID != "" || req.DurationMS != 0 || req.AfterGeneration != 0 || req.PublicationID != "" || req.SourcePath != "" || req.ArtifactKind != "" || req.LeaseID != "" || req.ReferenceID != "" || req.MaxBytes != 0 || len(req.Data) > contracts.MaxWorkPayload {
		return false
	}
	switch req.Operation {
	case "models.dataset.list", "models.speaker.list":
		return req.ItemID == ""
	case "models.dataset.create", "models.train":
		return req.ItemID == "" && len(req.Data) > 0
	case "models.dataset.show", "models.speaker.show":
		return contracts.ValidID(req.ItemID) && len(req.Data) == 0
	default:
		return contracts.ValidID(req.ItemID) && len(req.Data) > 0
	}
}

func (a *App) speakerModelService() (*voicemodels.Service, error) {
	return a.speakerModelServiceWithTools(nil)
}
func (a *App) speakerModelServiceWithTools(tools *ProcessingTools) (*voicemodels.Service, error) {
	artifacts, err := a.artifactService()
	if err != nil {
		return nil, err
	}
	service := voicemodels.NewService(artifacts, a.Catalog, a.secrets)
	service.BindPortableConfiguration = a.bindPortableConfiguration
	service.Client = a.hostedClient
	if a.speakerFactory != nil {
		return a.speakerFactory(service), nil
	}
	service.Prepare = func(ctx context.Context, entry catalog.LibraryEntry, options processing.AudioOptions) (*voicemodels.Audio, error) {
		executor, err := a.recordingExecutorWithTools(tools)
		if err != nil {
			return nil, err
		}
		audio, err := executor.Prepare(ctx, entry, options)
		if err != nil {
			return nil, err
		}
		raw, _ := json.Marshal(audio.SourceMap)
		var source processing.SourceMap
		if strictPayload(raw, &source) != nil {
			audio.Close()
			return nil, contracts.Fail("invalid_request")
		}
		return &voicemodels.Audio{Path: audio.Path, SourceMap: source, Close: audio.Close}, nil
	}
	service.Embed = func(ctx context.Context, path string, source processing.SourceMap, id, digest string) ([]float64, error) {
		config := tools
		if config == nil {
			selected, err := ReadProcessingTools(a.Workspace)
			if err != nil {
				return nil, err
			}
			config = &selected
		}
		engine := &processing.Service{Artifacts: artifacts, Catalog: a.Catalog, Config: config.Processing}
		return engine.Embed(ctx, path, source, id, digest)
	}
	service.EmbedBatch = func(ctx context.Context, inputs []voicemodels.PreparedInput, id, digest string) ([][]float64, error) {
		config := tools
		if config == nil {
			selected, err := ReadProcessingTools(a.Workspace)
			if err != nil {
				return nil, err
			}
			config = &selected
		}
		engine := &processing.Service{Artifacts: artifacts, Catalog: a.Catalog, Config: config.Processing}
		selected := make([]processing.EmbeddingInput, len(inputs))
		for index, input := range inputs {
			selected[index] = processing.EmbeddingInput{Path: input.Path, SourceMap: input.SourceMap}
		}
		return engine.EmbedBatch(ctx, selected, id, digest)
	}
	return service, nil
}

func (a *App) freezeSpeakerBase(ctx context.Context, reference, operation string) (string, string, []models.Resolution, []string, error) {
	if reference == "" {
		return "", "", nil, nil, nil
	}
	selected, err := models.NewResolver(a.Catalog, a.secrets).Resolve(ctx, reference, operation)
	if err != nil {
		return "", "", nil, nil, err
	}
	if selected.Target.Kind != "base" || !selected.Compatible {
		return "", "", nil, nil, contracts.Fail("unsupported_capability")
	}
	return selected.Target.ID, selected.Digest, []models.Resolution{selected}, nil, nil
}

func (a *App) speakerWorkDispatch(req contracts.Request) (any, error) {
	switch req.Operation {
	case "models.dataset.show":
		dataset, err := a.Catalog.SpeakerDataset(a.ctx, req.ItemID)
		return speakerDatasetView(dataset), err
	case "models.profile.set":
		var input struct {
			Expected int64  `json:"expected_revision"`
			Version  string `json:"version_id"`
		}
		if !speakerInputFields(req.Data, "expected_revision", "version_id") || strictPayload(req.Data, &input) != nil || input.Expected < 0 || input.Version != "" && !contracts.ValidID(input.Version) {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.SetSpeakerProfile(a.ctx, req.RequestID, req.ItemID, input.Expected, input.Version)
	case "models.dataset.list", "models.speaker.list":
		var input struct {
			Speaker string `json:"speaker_id,omitempty"`
			After   string `json:"after_id,omitempty"`
			Limit   int    `json:"limit,omitempty"`
		}
		if len(req.Data) > 0 && strictPayload(req.Data, &input) != nil || input.Speaker != "" && !contracts.ValidID(input.Speaker) {
			return nil, contracts.Fail("invalid_request")
		}
		raw, _ := json.Marshal(listPage{After: input.After, Limit: input.Limit})
		page, err := pageInput(raw)
		if err != nil {
			return nil, err
		}
		if req.Operation == "models.dataset.list" {
			items, err := a.Catalog.SpeakerDatasets(a.ctx, input.Speaker)
			return pageViews(items, page, func(d catalog.SpeakerDataset) string { return d.Dataset.ID }, speakerDatasetSummary), err
		}
		items, err := a.Catalog.SpeakerOutputs(a.ctx, input.Speaker)
		if err != nil {
			return nil, err
		}
		view := pageViews(items, page, func(o catalog.SpeakerOutput) string { return o.ID }, speakerOutputSummary).(map[string]any)
		if input.Speaker != "" {
			profile, err := a.Catalog.SpeakerProfile(a.ctx, input.Speaker)
			if err != nil {
				return nil, err
			}
			view["profile"] = profile
		}
		return view, nil
	case "models.dataset.create":
		var input voicemodels.DatasetOptions
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		service, err := a.speakerModelService()
		if err != nil {
			return nil, err
		}
		dataset, err := service.CreateDataset(a.ctx, req.RequestID, input)
		return speakerDatasetView(dataset), err
	case "models.speaker.show":
		output, err := a.Catalog.SpeakerOutput(a.ctx, req.ItemID)
		if err != nil {
			return nil, err
		}
		profile, err := a.Catalog.SpeakerProfile(a.ctx, output.SpeakerID)
		if err != nil {
			return nil, err
		}
		if len(output.Metadata) > 512<<10 {
			return map[string]any{"output": speakerOutputSummary(output), "profile": profile, "metadata_inline": false, "metadata_bytes": len(output.Metadata)}, nil
		}
		return map[string]any{"output": output, "profile": profile, "metadata_inline": true}, nil
	case "models.speaker.fetch":
		var input struct {
			Destination string `json:"destination"`
		}
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		service, err := a.speakerModelService()
		if err != nil {
			return nil, err
		}
		return service.Fetch(a.ctx, req.ItemID, input.Destination)
	case "models.train", "recordings.match":
		return a.enqueueSpeakerWork(req)
	}
	return nil, contracts.Fail("invalid_request")
}

func speakerDatasetView(dataset catalog.SpeakerDataset) any {
	view := speakerDatasetSummary(dataset).(map[string]any)
	view["references_inline"] = len(dataset.References) <= 100
	view["recipe_inline"] = len(dataset.Recipe) <= 16<<10
	if len(dataset.Recipe) <= 16<<10 {
		view["recipe"] = dataset.Recipe
	}
	if len(dataset.References) <= 100 {
		view["references"] = dataset.References
	}
	return view
}
func speakerDatasetSummary(dataset catalog.SpeakerDataset) any {
	d := dataset.Dataset
	return map[string]any{"dataset": map[string]any{"id": d.ID, "speaker_id": d.SpeakerID, "state": d.State, "manifest_artifact_id": d.ManifestArtifactID, "invalidated_revision": d.InvalidatedRevision, "invalidated_by": d.InvalidatedBy}, "speaker_revision": dataset.SpeakerRevision, "epoch": dataset.Epoch, "summary": dataset.Summary, "reference_count": len(dataset.References)}
}
func speakerOutputSummary(output catalog.SpeakerOutput) any {
	return map[string]any{"id": output.ID, "model_id": output.ModelID, "speaker_id": output.SpeakerID, "dataset_id": output.DatasetID, "work_id": output.WorkID, "kind": output.Kind, "name": output.Name, "publication_ids": output.PublicationIDs}
}

func (a *App) enqueueSpeakerWork(req contracts.Request) (any, error) {
	// Replay reconciles the original intent before selecting newer corpus/profile
	// state or resolving an alias. The request UUID never means "latest again".
	requestDigest := documentHash(configurationBytes(struct {
		Item  string          `json:"item_id"`
		Input json.RawMessage `json:"input"`
	}{req.ItemID, req.Data}))
	if previous, err := a.Catalog.Work(a.ctx, req.RequestID); err == nil {
		var frozen struct {
			RequestDigest string `json:"request_digest"`
		}
		if json.Unmarshal(previous.Payload, &frozen) != nil || previous.Kind != req.Operation || frozen.RequestDigest != requestDigest {
			return nil, contracts.Fail("conflict")
		}
		return a.modelWorkView(previous), nil
	} else if typed, ok := err.(*contracts.Error); !ok || typed.Code != "not_found" {
		return nil, err
	}
	service, err := a.speakerModelService()
	if err != nil {
		return nil, err
	}
	var raw []byte
	if req.Operation == "models.train" {
		var options voicemodels.TrainOptions
		if !speakerInputFieldsAbsent(req.Data, "base_digest") || strictPayload(req.Data, &options) != nil || options.BaseDigest != "" {
			return nil, contracts.Fail("invalid_request")
		}
		options, err = service.ResolveTrainingPipeline(a.ctx, options)
		if err != nil {
			return nil, err
		}
		id, digest, selections, _, err := a.freezeSpeakerBase(a.ctx, options.BaseModelID, "speaker-model-training")
		if err != nil {
			return nil, err
		}
		options.BaseModelID, options.BaseDigest = id, digest
		options, err = service.ValidateTrain(a.ctx, options)
		if err != nil {
			return nil, err
		}
		tools, err := a.freezeSpeakerProcessingTools()
		if err != nil {
			return nil, err
		}
		dependencies, err := a.queueModelSelections(a.ctx, selections)
		if err != nil {
			return nil, err
		}
		raw, err = json.Marshal(voicemodels.TrainPayload{Options: options, ProcessingTools: tools, RequestDigest: requestDigest, ModelSelections: selections, ModelDependencies: dependencies})
	} else {
		var options voicemodels.MatchOptions
		if !speakerInputFieldsAbsent(req.Data, "recording_id", "model_digest") || !speakerInputFields(req.Data, "model_id", "adapter", "threshold", "ambiguity_margin", "min_evidence_us") || strictPayload(req.Data, &options) != nil || options.RecordingID != "" || options.ModelDigest != "" {
			return nil, contracts.Fail("invalid_request")
		}
		options.RecordingID = req.ItemID
		id, digest, selections, _, e := a.freezeSpeakerBase(a.ctx, options.ModelID, "voice-matching")
		if e != nil {
			return nil, e
		}
		options.ModelID, options.ModelDigest = id, digest
		frozen, e := service.ValidateMatch(a.ctx, options)
		if e != nil {
			return nil, e
		}
		tools, e := a.freezeSpeakerProcessingTools()
		if e != nil {
			return nil, e
		}
		frozen.ProcessingTools = tools
		dependencies, e := a.queueModelSelections(a.ctx, selections)
		if e != nil {
			return nil, e
		}
		frozen.RequestDigest = requestDigest
		frozen.ModelSelections = selections
		frozen.ModelDependencies = dependencies
		raw, err = json.Marshal(frozen)
	}
	if err != nil {
		return nil, contracts.Fail("invalid_request")
	}
	work, err := a.Catalog.EnqueueWork(a.ctx, req.RequestID, req.Operation, raw)
	if err == nil {
		a.recoverWork()
	}
	return a.modelWorkView(work), err
}

func (a *App) freezeSpeakerProcessingTools() (json.RawMessage, error) {
	if a.speakerFactory != nil {
		return nil, nil
	}
	tools, err := a.electedProcessingTools()
	if err != nil {
		return nil, err
	}
	if tools == nil {
		return nil, nil
	}
	raw, err := json.Marshal(tools)
	return raw, err
}
func speakerPayloadTools(claim catalog.Work) (*ProcessingTools, error) {
	var envelope struct {
		Tools json.RawMessage `json:"processing_tools"`
	}
	if json.Unmarshal(claim.Payload, &envelope) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if len(envelope.Tools) == 0 {
		return nil, nil
	}
	var tools ProcessingTools
	if strictPayload(envelope.Tools, &tools) != nil || ValidateProcessingTools(tools) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return &tools, nil
}

func speakerInputFields(raw []byte, fields ...string) bool {
	var input map[string]json.RawMessage
	if catalog.ValidateJSON(raw) != nil || json.Unmarshal(raw, &input) != nil || input == nil {
		return false
	}
	for _, field := range fields {
		value, ok := input[field]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return false
		}
	}
	return true
}
func speakerInputFieldsAbsent(raw []byte, fields ...string) bool {
	var input map[string]json.RawMessage
	if catalog.ValidateJSON(raw) != nil || json.Unmarshal(raw, &input) != nil || input == nil {
		return false
	}
	for _, field := range fields {
		if _, exists := input[field]; exists {
			return false
		}
	}
	return true
}
