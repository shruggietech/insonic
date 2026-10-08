// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/speakers"
	"strings"
	"time"
)

func configuredOperation(op string) bool {
	return strings.HasPrefix(op, "pipelines.") || strings.HasPrefix(op, "speakers.") || strings.HasPrefix(op, "terms.")
}
func configuredRequestValid(req contracts.Request) bool {
	return contracts.ConfigurationRequestValid(req)
}
func pipelineDefinition(p catalog.Pipeline) (pipeline.Definition, error) {
	var d pipeline.Definition
	if strictPayload(p.Configuration, &d) != nil {
		return d, contracts.Fail("invalid_request")
	}
	return d, pipeline.Validate(p.Preset, d)
}

type compileInput struct {
	PipelineID       string   `json:"pipeline_id,omitempty"`
	PipelineRevision int64    `json:"pipeline_revision,omitempty"`
	SpeakerIDs       []string `json:"speaker_ids,omitempty"`
	Language         string   `json:"language,omitempty"`
	Context          string   `json:"context,omitempty"`
	MaxHintBytes     int      `json:"max_hint_bytes,omitempty"`
}

// recognitionContext previews the same selected hints and language filter used
// by generated recognition elections. It performs catalog reads only.
func effectiveRecognitionStage(stage pipeline.Stage) pipeline.Stage {
	if stage.Adapter == "faster-whisper" && stage.Mode == "local" && stage.Recognition.Language == "" {
		stage.Recognition.Language = "en"
	}
	return stage
}
func (a *App) recognitionContext(filter catalog.ContextFilter, stage pipeline.Stage, budget int, supported bool) (speakers.CompiledContext, error) {
	stage = effectiveRecognitionStage(stage)
	if filter.Language == "" {
		filter.Language = stage.Recognition.Language
	}
	snapshot, e := a.Catalog.ContextInputs(a.ctx, filter)
	if e != nil {
		return speakers.CompiledContext{}, e
	}
	snapshot.ExtraHints = stage.Recognition.Hints
	return speakers.CompileContext(snapshot, budget, supported)
}

func (a *App) compileRecognitionContext(input compileInput) (speakers.CompiledContext, error) {
	if input.MaxHintBytes < 0 || input.MaxHintBytes > 8192 || input.PipelineRevision < 0 || input.PipelineRevision != 0 && input.PipelineID == "" {
		return speakers.CompiledContext{}, contracts.Fail("invalid_request")
	}
	budget, supported := input.MaxHintBytes, true
	var stage pipeline.Stage
	if input.PipelineID != "" {
		p, e := a.Catalog.Pipeline(a.ctx, input.PipelineID)
		if e != nil {
			return speakers.CompiledContext{}, e
		}
		if input.PipelineRevision != 0 && p.Revision != input.PipelineRevision {
			return speakers.CompiledContext{}, contracts.Fail("conflict")
		}
		d, e := pipelineDefinition(p)
		if e != nil {
			return speakers.CompiledContext{}, e
		}
		stage = effectiveRecognitionStage(d.Recognition)
		capability, e := pipeline.Capabilities(stage)
		if e != nil {
			return speakers.CompiledContext{}, e
		}
		supported = capability.SupportsHints
		if budget == 0 || budget > capability.MaxHintBytes {
			budget = capability.MaxHintBytes
		}
	} else if budget == 0 {
		budget = 200
	}
	return a.recognitionContext(catalog.ContextFilter{Language: input.Language, Context: input.Context, SpeakerIDs: input.SpeakerIDs}, stage, budget, supported)
}
func (a *App) configuredDispatch(req contracts.Request) (any, error) {
	switch req.Operation {
	case "pipelines.list", "speakers.list", "terms.list":
		p, e := pageInput(req.Data)
		if e != nil {
			return nil, e
		}
		switch req.Operation {
		case "pipelines.list":
			items, next, e := a.Catalog.Pipelines(a.ctx, p.After, p.Limit)
			return map[string]any{"items": items, "next_id": next}, e
		case "speakers.list":
			items, next, e := a.Catalog.Speakers(a.ctx, p.After, p.Limit)
			return map[string]any{"items": items, "next_id": next}, e
		default:
			items, next, e := a.Catalog.Terms(a.ctx, p.After, p.Limit)
			return map[string]any{"items": items, "next_id": next}, e
		}
	case "pipelines.show":
		return a.Catalog.Pipeline(a.ctx, req.ItemID)
	case "pipelines.set":
		var input struct {
			Expected int64            `json:"expected_revision"`
			Pipeline catalog.Pipeline `json:"pipeline"`
		}
		if strictPayload(req.Data, &input) != nil || input.Pipeline.ID != req.ItemID {
			return nil, contracts.Fail("invalid_request")
		}
		if _, e := pipelineDefinition(input.Pipeline); e != nil {
			return nil, e
		}
		return a.Catalog.PutPipeline(a.ctx, req.RequestID, input.Expected, input.Pipeline)
	case "pipelines.inspect":
		var input struct {
			Revision  int64                 `json:"revision,omitempty"`
			Overrides pipeline.Override     `json:"overrides,omitempty"`
			Context   catalog.ContextFilter `json:"context,omitempty"`
		}
		if len(req.Data) > 0 && strictPayload(req.Data, &input) != nil || input.Revision < 0 {
			return nil, contracts.Fail("invalid_request")
		}
		p, e := a.Catalog.Pipeline(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		if input.Revision != 0 && input.Revision != p.Revision {
			return nil, contracts.Fail("conflict")
		}
		d, e := pipelineDefinition(p)
		if e != nil {
			return nil, e
		}
		d, e = pipeline.Elect(p.Preset, d, input.Overrides)
		if e != nil {
			return nil, e
		}
		d.Recognition = effectiveRecognitionStage(d.Recognition)
		recognition, e := pipeline.Capabilities(d.Recognition)
		if e != nil {
			return nil, e
		}
		diarization, e := pipeline.Capabilities(d.Diarization)
		if e != nil {
			return nil, e
		}
		context, e := a.recognitionContext(input.Context, d.Recognition, recognition.MaxHintBytes, recognition.SupportsHints)
		if e != nil {
			return nil, e
		}
		d.Recognition.Recognition.Hints = context.Hints
		d.Recognition.Recognition.ContextDigest = processing.HintsDigest(context.Hints)
		return map[string]any{"pipeline_id": p.ID, "revision": p.Revision, "preset": p.Preset, "configuration": d, "capabilities": []pipeline.Capability{recognition, diarization}, "context": context, "reachability": "not-probed"}, nil
	case "speakers.show":
		return a.Catalog.Speaker(a.ctx, req.ItemID)
	case "speakers.set":
		var input struct {
			Expected int64                  `json:"expected_revision"`
			Speaker  catalog.Speaker        `json:"speaker"`
			Aliases  []catalog.SpeakerAlias `json:"aliases"`
		}
		if strictPayload(req.Data, &input) != nil || input.Speaker.ID != req.ItemID {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.PutSpeaker(a.ctx, req.RequestID, input.Expected, catalog.SpeakerIdentity{Speaker: input.Speaker, Aliases: input.Aliases})
	case "speakers.aliases":
		current, e := a.Catalog.Speaker(a.ctx, req.ItemID)
		if e != nil {
			return nil, e
		}
		if len(req.Data) == 0 {
			return map[string]any{"speaker_id": req.ItemID, "revision": current.Speaker.Revision, "aliases": current.Aliases}, nil
		}
		var input struct {
			Expected int64                  `json:"expected_revision"`
			Aliases  []catalog.SpeakerAlias `json:"aliases"`
		}
		if strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		current.Aliases = input.Aliases
		return a.Catalog.PutSpeaker(a.ctx, req.RequestID, input.Expected, current)
	case "speakers.select", "speakers.diagnostics":
		var input struct {
			Cursor      string `json:"cursor,omitempty"`
			Limit       int    `json:"limit,omitempty"`
			RecordingID string `json:"recording_id,omitempty"`
			Quality     struct {
				Enabled *bool `json:"enabled,omitempty"`
			} `json:"quality,omitempty"`
		}
		if len(req.Data) > 0 && strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		ctx, stop := context.WithTimeout(a.ctx, 10*time.Second)
		defer stop()
		scope := catalog.SpeakerSelection{SpeakerID: req.ItemID, Cursor: input.Cursor, Limit: input.Limit, RecordingID: input.RecordingID}
		page, e := a.Catalog.CurrentSpeakerReferences(ctx, scope)
		if e != nil {
			return nil, e
		}
		resolved := make([]catalog.ResolvedEvidence, 0, len(page.References))
		for _, reference := range page.References {
			value, e := a.Catalog.ResolveEvidence(ctx, reference)
			if e != nil {
				return nil, e
			}
			resolved = append(resolved, value)
		}
		selected, e := speakers.SelectWithPrior(resolved, func(ref catalog.CurrentReference) ([]catalog.PriorEvidenceComparison, error) {
			return a.Catalog.ComparePriorSpeakerEvidence(ctx, scope, ref, page.Epoch)
		})
		if e != nil {
			return nil, e
		}
		selected.Partial = input.Cursor != "" || page.Next != ""
		enabled := input.Quality.Enabled == nil || *input.Quality.Enabled
		if !enabled {
			selected.Diagnostics = []speakers.Diagnostic{}
		}
		result := map[string]any{"speaker_id": req.ItemID, "revision": page.Revision, "next_cursor": page.Next, "selection": selected, "quality": map[string]bool{"enabled": enabled}}
		if len(configurationBytes(result)) > 512<<10 {
			return nil, contracts.Fail("output_limit")
		}
		return result, nil
	case "terms.show":
		return a.Catalog.Term(a.ctx, req.ItemID)
	case "terms.set":
		var input struct {
			Expected int64        `json:"expected_revision"`
			Term     catalog.Term `json:"term"`
		}
		if strictPayload(req.Data, &input) != nil || input.Term.ID != req.ItemID {
			return nil, contracts.Fail("invalid_request")
		}
		return a.Catalog.PutTerm(a.ctx, req.RequestID, input.Expected, input.Term)
	case "terms.compile":
		var input compileInput
		if len(req.Data) > 0 && strictPayload(req.Data, &input) != nil {
			return nil, contracts.Fail("invalid_request")
		}
		return a.compileRecognitionContext(input)
	}
	return nil, contracts.Fail("invalid_request")
}

// Keep effective definitions canonical for durable election comparison/digests.
func configurationBytes(value any) json.RawMessage { raw, _ := json.Marshal(value); return raw }
