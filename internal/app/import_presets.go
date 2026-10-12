// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"strconv"
	"strings"
)

func (a *App) resolveImportPreset(selection string) (catalog.ImportPreset, error) {
	all, e := a.Catalog.ImportPresets(a.ctx)
	if e != nil {
		if contracts.SafeError(e).Code == "incompatible_version" {
			return catalog.ImportPreset{}, &contracts.Error{Code: "incompatible_version", Message: "The saved import preset format is incompatible. Open this catalog with a matching application version or restore a compatible catalog backup before importing."}
		}
		return catalog.ImportPreset{}, e
	}
	matches := []catalog.ImportPreset{}
	for _, p := range all {
		if p.State == "active" && (p.ID == selection || p.Name == selection) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 0 {
		return catalog.ImportPreset{}, &contracts.Error{Code: "not_found", Message: "The saved import preset is missing or deleted. Use import-presets list, then select its UUID or exact name; create a preset before using --preset."}
	}
	if len(matches) != 1 {
		return catalog.ImportPreset{}, &contracts.Error{Code: "conflict", Message: "The saved import preset selection is ambiguous. Use import-presets list to select an unambiguous UUID or exact name, or rename the colliding preset."}
	}
	return matches[0], nil
}
func (a *App) importPresetDispatch(req contracts.Request) (any, error) {
	switch req.Operation {
	case "import-presets.list":
		p, e := pageInput(req.Data)
		if e != nil {
			return nil, e
		}
		all, e := a.Catalog.ImportPresets(a.ctx)
		if e != nil {
			return nil, e
		}
		active := []catalog.ImportPreset{}
		for _, preset := range all {
			if preset.State == "active" {
				active = append(active, preset)
			}
		}
		return pageViews(active, p, func(p catalog.ImportPreset) string { return p.ID }, func(p catalog.ImportPreset) any { return p }), nil
	case "import-presets.show":
		var input struct {
			Selection string `json:"selection"`
		}
		if strictPayload(req.Data, &input) != nil || strings.TrimSpace(input.Selection) == "" {
			return nil, contracts.Fail("invalid_request")
		}
		return a.resolveImportPreset(input.Selection)
	default:
		var input struct {
			Expected int64                `json:"expected_revision"`
			Preset   catalog.ImportPreset `json:"preset"`
		}
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		action := strings.TrimPrefix(req.Operation, "import-presets.")
		if action == "delete" {
			input.Preset = catalog.ImportPreset{ID: req.ItemID}
		} else if action == "update" && req.ItemID != input.Preset.ID {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.MutateImportPreset(a.ctx, req.RequestID, action, input.Expected, input.Preset)
	}
}
func (a *App) freezeImportSelection(defaults, item, overrides library.Options, supplied bool) (library.ImportSelection, error) {
	explicit := library.MergeOptions(library.MergeOptions(defaults, item), overrides)
	selection := library.ImportSelection{}
	if explicit.Preset != "" {
		p, e := a.resolveImportPreset(explicit.Preset)
		if e != nil {
			return selection, e
		}
		if strictPayload(p.Options, &selection.Options) != nil {
			return selection, contracts.Fail("incompatible_version")
		}
		selection.PresetID = p.ID
		selection.PresetRevision = p.Revision
	}
	selection.Options = library.MergeOptions(library.MergeOptions(library.MergeOptions(selection.Options, defaults), item), overrides)
	if selection.Options.Destination == nil {
		p := a.Workspace.Config.Profiles.Storage
		selection.Options.Destination = &p
	}
	profile := selection.Options.Destination
	record, e := a.Catalog.Profile(a.ctx, profile.ID, profile.Revision)
	if e != nil && contracts.SafeError(e).Code == "not_found" {
		// Register a newly elected complete profile once; existing revisions stay read-only.
		selectedWorkspace := *a.Workspace
		selectedWorkspace.Config.Profiles.Storage = *profile
		if e = a.Catalog.RegisterWorkspace(a.ctx, &selectedWorkspace); e != nil {
			return selection, e
		}
		record, e = a.Catalog.Profile(a.ctx, profile.ID, profile.Revision)
	}
	if e != nil {
		return selection, e
	}
	if record.Role != "storage" {
		return selection, contracts.Fail("invalid_request")
	}
	// Explicit configuration must match the immutable admitted profile revision.
	registered, e := artifact.WorkspaceProfile(record)
	if e != nil {
		return selection, e
	}
	if documentHash(configurationBytes(registered)) != documentHash(configurationBytes(profile)) {
		return selection, contracts.Fail("conflict")
	}
	artifacts, e := a.artifactService()
	if e != nil {
		return selection, e
	}
	if e = artifacts.Destination(a.ctx, profile.ID, profile.Revision); e != nil {
		return selection, &contracts.Error{Code: contracts.SafeError(e).Code, Message: "The elected import destination is unavailable. Inspect its storage configuration and credential reference before retrying."}
	}
	if selection.Options.PipelineID != nil && *selection.Options.PipelineID != "" {
		plan, e := a.freezeImportProcessing(*selection.Options.PipelineID, supplied)
		if e != nil {
			return selection, &contracts.Error{Code: contracts.SafeError(e).Code, Message: "The elected import pipeline is not ready. Inspect the configured pipeline, processing tools and model selections before importing."}
		}
		plan.Destination = selection.Options.Destination
		selection.Processing = configurationBytes(plan)
		selection.ProcessingDigest = documentHash(selection.Processing)
	}
	return selection, nil
}
func (a *App) freezeImportProcessing(id string, supplied bool) (recordingPayload, error) {
	mode := "generate"
	if supplied {
		mode = "supplied"
	}
	options, election, e := a.electRecordingOptions(RecordingOptions{PipelineID: id, Transcription: mode, Diarization: "run"})
	if e != nil {
		return recordingPayload{}, e
	}
	if e = validateRecordingOptions(options); e != nil {
		return recordingPayload{}, e
	}
	tools, e := a.electedProcessingTools()
	if e != nil {
		return recordingPayload{}, e
	}
	options, election, selections, e := a.electProcessingModels(options, election)
	if e != nil {
		return recordingPayload{}, e
	}
	digests := map[string]string{}
	for _, selected := range selections {
		digests[selected.Target.ID] = selected.Digest
	}
	return recordingPayload{Options: options, Election: election, Tools: tools, ModelSelections: selections, ModelDigests: digests}, nil
}
func (a *App) resolveImportChoices(input library.ImportRequest) (library.ImportRequest, error) {
	input.Selections = map[int]string{}
	input.Elections = map[string]library.ImportSelection{}
	resolved := map[string]library.ImportSelection{}
	transcripts := map[string]bool{}
	for i := range input.Items {
		item := input.Items[i]
		hasText, known := transcripts[item.Record]
		if !known {
			hasText = a.importHasTranscript(item.Record)
			transcripts[item.Record] = hasText
		}
		supplied := item.Transcript != "" || item.Subtitle != "" || hasText
		requestKey := documentHash(configurationBytes(struct {
			Defaults, Item, Overrides library.Options
			Supplied                  bool
		}{input.Defaults, item.Options, input.Overrides, supplied}))
		selection, found := resolved[requestKey]
		if !found {
			var e error
			selection, e = a.freezeImportSelection(input.Defaults, item.Options, input.Overrides, supplied)
			if e != nil {
				return input, e
			}
			resolved[requestKey] = selection
		}
		key := documentHash(configurationBytes(selection))
		input.Selections[i] = key
		input.Elections[key] = selection
		input.Items[i].Options = selection.Options
	}
	input.Overrides = library.Options{}
	input.Defaults = library.Options{}
	prepared, e := library.PrepareImport(input)
	if e != nil {
		return prepared, e
	}
	elections := map[string]library.ImportSelection{}
	for i := range prepared.Items {
		key := prepared.Selections[i]
		selected := prepared.Elections[key]
		selected.Options = library.MergeOptions(prepared.Defaults, prepared.Items[i].Options)
		next := documentHash(configurationBytes(selected))
		prepared.Selections[i] = next
		elections[next] = selected
	}
	prepared.Elections = elections
	return prepared, nil
}
func (a *App) queueImportProcessing(ctx context.Context, parent catalog.Work, ordinal int, mediaID string, selection library.ImportSelection) ([]string, error) {
	if len(selection.Processing) == 0 {
		return []string{}, nil
	}
	id := library.DerivedID(parent.ID, "processing-"+strconv.Itoa(ordinal))
	digest := selection.ProcessingDigest
	if digest == "" {
		digest = documentHash(selection.Processing)
	}
	if existing, e := a.Catalog.Work(ctx, id); e == nil {
		var frozen recordingPayload
		if strictPayload(existing.Payload, &frozen) != nil || existing.Kind != "recordings.process" || frozen.MediaID != mediaID || frozen.RequestDigest != digest {
			return nil, contracts.Fail("conflict")
		}
		return []string{id}, nil
	} else if contracts.SafeError(e).Code != "not_found" {
		return nil, e
	}
	var plan recordingPayload
	if strictPayload(selection.Processing, &plan) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	entry, e := a.Catalog.Library(ctx, mediaID)
	if e != nil {
		return nil, e
	}
	plan.MediaID = mediaID
	plan.SourceRevision = entry.Revision
	plan.RequestDigest = digest
	current, e := a.Catalog.Recording(ctx, mediaID)
	if e == nil {
		plan.Expected = current.Revision
	} else if contracts.SafeError(e).Code != "not_found" {
		return nil, e
	}
	dependencies, e := a.queueModelSelections(ctx, plan.ModelSelections)
	if e != nil {
		return nil, e
	}
	plan.ModelDependencies = dependencies
	work, e := a.Catalog.EnqueueChildWork(ctx, parent, id, "recordings.process", configurationBytes(plan))
	if e != nil {
		return nil, e
	}
	a.recoverWork()
	return []string{work.ID}, nil
}

func (a *App) importHasTranscript(id string) bool {
	if id == "" {
		return false
	}
	entry, e := library.ResolveTarget(a.ctx, a.Catalog, id)
	if e != nil {
		return false
	}
	if entry.SubtitlePublicationID != nil {
		return true
	}
	record, e := a.Catalog.Recording(a.ctx, entry.ID)
	return e == nil && record.State == "ready"
}
