// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"math"
	"path/filepath"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/firstsetup"
	"github.com/shruggietech/insonic/internal/workers"
	"github.com/shruggietech/insonic/internal/workspace"
)

const setupSetting = "first-run-setup"

type setupDocument struct {
	Kind        string           `json:"kind"`
	Version     string           `json:"schema_version"`
	WorkspaceID string           `json:"workspace_id"`
	State       firstsetup.State `json:"state"`
}

// setupState is metadata only. Configuration and work keep their existing
// authorities, revisions and request identities throughout the journey.
func (a *App) setupState(ctx context.Context) (firstsetup.State, int64, error) {
	raw, revision, err := a.Catalog.NamedSetting(ctx, setupSetting)
	if err != nil {
		return firstsetup.State{}, revision, err
	}
	if len(raw) == 0 {
		return firstsetup.DefaultState(), revision, nil
	}
	var saved setupDocument
	if strictPayload(raw, &saved) != nil || saved.Kind != setupSetting || saved.WorkspaceID != a.Workspace.Config.WorkspaceID {
		return firstsetup.State{}, revision, contracts.Fail("invalid_request")
	}
	if !contracts.SupportedDocumentVersion(saved.Version) {
		return firstsetup.State{}, revision, contracts.Fail("incompatible_version")
	}
	if err := firstsetup.ValidateState(saved.State); err != nil {
		return firstsetup.State{}, revision, err
	}
	return saved.State, revision, nil
}

// setupObservations are digests of current declarations, not successful engine
// or authentication claims. Expensive task probes remain setup.readiness.
func (a *App) setupObservations(ctx context.Context) (map[string]string, error) {
	settings, err := a.settingsShow()
	if err != nil {
		return nil, err
	}
	sections := settings.(map[string]any)["sections"].(map[string]any)
	service, serviceRevision, err := a.Catalog.NamedSetting(ctx, assistanceSetting)
	if err != nil {
		return nil, err
	}
	test, testRevision, err := a.Catalog.NamedSetting(ctx, "query-assistance-test")
	if err != nil {
		return nil, err
	}
	processing, processingErr := ReadProcessingTools(a.Workspace)
	media, mediaErr := ReadMediaTools(a.Workspace)
	profiles := a.Workspace.Config.Profiles
	return map[string]string{
		"workspace":  digestJSON(map[string]any{"workspace_id": a.Workspace.Config.WorkspaceID, "display_name": a.Workspace.Config.DisplayName, "schema_version": a.Workspace.Config.Version}),
		"backends":   digestJSON(profiles),
		"processing": digestJSON(map[string]any{"section": sections["processing_tools"], "tools": processing, "error": setupObservationError(processingErr)}),
		"services":   digestJSON(map[string]any{"configuration": service, "revision": serviceRevision, "last_test": test, "test_revision": testRevision}),
		"import":     digestJSON(map[string]any{"section": sections["media_tools"], "tools": media, "error": setupObservationError(mediaErr), "storage": profiles.Storage}),
	}, nil
}

func (a *App) setupPipelineCorrection(ctx context.Context, choice firstsetup.Choices) string {
	if choice.PipelineID == "" {
		return ""
	}
	pipeline, err := a.Catalog.Pipeline(ctx, choice.PipelineID)
	if err != nil {
		return "Inspect Pipelines and explicitly select an available saved pipeline. The previous election remains intact."
	}
	if pipeline.Revision != choice.PipelineRevision {
		return "The selected pipeline changed. Inspect its current revision and explicitly save the intended election."
	}
	if _, err := a.configuredPipelineDefinition(pipeline); err != nil {
		return "Correct the selected pipeline configuration or its destination bindings in Settings and Pipelines."
	}
	return ""
}

func setupObservationError(err error) any {
	if err == nil {
		return nil
	}
	return contracts.SafeError(err)
}

func (a *App) setupPaths() (map[string]any, error) {
	platform, err := workspace.PlatformPaths()
	if err != nil {
		return nil, err
	}
	paths := map[string]any{"workspace": a.Workspace.Root, "control": a.Workspace.Control, "cache": platform.Cache, "runtime": platform.Runtime, "capacity": "unknown"}
	if a.Workspace.Config.Profiles.Storage.Adapter == "filesystem" {
		if root, ok := a.Workspace.Config.Profiles.Storage.Configuration["root"].(string); ok {
			if !filepath.IsAbs(root) {
				root = filepath.Join(a.Workspace.Control, root)
			}
			paths["artifacts"] = root
		}
	} else {
		paths["artifact_storage"] = a.Workspace.Config.Profiles.Storage.Adapter
	}
	if available, err := workers.FreeSpace(a.Workspace.Root); err == nil {
		paths["available_space_bytes"] = available
		paths["capacity"] = "observed_workspace_volume; remote_capacity_unknown"
	}
	return paths, nil
}

func (a *App) setupShow(ctx context.Context) (map[string]any, error) {
	state, revision, err := a.setupState(ctx)
	if err != nil {
		return nil, err
	}
	observations, err := a.setupObservations(ctx)
	if err != nil {
		return nil, err
	}
	stages := make([]map[string]any, 0, len(state.Stages))
	pipelineCorrection := a.setupPipelineCorrection(ctx, state.Choices)
	for i := range state.Stages {
		stage := &state.Stages[i]
		// The offer is not a saved assertion. Stamp its initial declarations so
		// accepting a fresh default does not manufacture stale observations.
		if revision == 0 {
			stage.Proof = observations[stage.ID]
		}
		view := map[string]any{"id": stage.ID, "state": stage.State}
		if stage.State != "skipped" && stage.Proof != observations[stage.ID] {
			view["state"] = "needs_attention"
			view["correction"] = "Refresh this stage, inspect its current configuration and save its current observation. Ordinary operations remain available."
		}
		if stage.ID == "processing" && stage.State != "skipped" && pipelineCorrection != "" {
			view["state"] = "needs_attention"
			view["correction"] = pipelineCorrection
		}
		stages = append(stages, view)
	}
	paths, err := a.setupPaths()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"kind": setupSetting, "schema_version": contracts.Version, "workspace_id": a.Workspace.Config.WorkspaceID,
		"revision": revision, "state": state, "observations": observations, "stages": stages, "paths": paths,
		"actions": []string{"Open Library to import media", "Open Settings to configure optional capabilities or reopen setup", "Open Jobs to inspect, retry or cancel elected work", "Use setup readiness to inspect current task prerequisites"},
	}, nil
}

func (a *App) setupSave(ctx context.Context, req contracts.Request) (map[string]any, error) {
	var envelope struct {
		ExpectedRevision *int64           `json:"expected_revision"`
		State            firstsetup.State `json:"state"`
	}
	if strictPayload(req.Data, &envelope) != nil || envelope.ExpectedRevision == nil || *envelope.ExpectedRevision < 0 || *envelope.ExpectedRevision == math.MaxInt64 {
		return nil, contracts.Fail("invalid_request")
	}
	input := firstsetup.SaveInput{ExpectedRevision: *envelope.ExpectedRevision, State: envelope.State}
	if err := firstsetup.ValidateState(input.State); err != nil {
		return nil, err
	}
	for _, stage := range input.State.Stages {
		if stage.State != "skipped" && !hashString.MatchString(stage.Proof) {
			return nil, contracts.Fail("invalid_request")
		}
	}
	document := setupDocument{Kind: setupSetting, Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, State: input.State}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, contracts.Fail("invalid_request")
	}
	// The catalog checks stable request identity before expected_revision.
	// A replay therefore succeeds even after another client saves newer state.
	receipt, err := a.Catalog.PutNamedSetting(ctx, req.RequestID, setupSetting, input.ExpectedRevision, raw)
	if err != nil {
		return nil, err
	}
	current, err := a.setupShow(ctx)
	if err != nil {
		return nil, err
	}
	current["accepted_revision"] = receipt.Revision
	current["accepted_receipt"] = receipt
	return current, nil
}

func (a *App) setupDispatch(ctx context.Context, req contracts.Request) (any, error) {
	switch req.Operation {
	case "setup.show":
		if len(req.Data) != 0 {
			return nil, contracts.Fail("invalid_request")
		}
		return a.setupShow(ctx)
	case "setup.save":
		return a.setupSave(ctx, req)
	case "setup.readiness":
		if len(req.Data) != 0 {
			return nil, contracts.Fail("invalid_request")
		}
		state, _, err := a.setupState(ctx)
		if err != nil {
			return nil, err
		}
		return a.processingReadiness(ctx, ReadinessInput{PipelineID: state.Choices.PipelineID, PipelineRevision: state.Choices.PipelineRevision})
	default:
		return nil, contracts.Fail("invalid_request")
	}
}
