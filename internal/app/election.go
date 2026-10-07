// SPDX-License-Identifier: Apache-2.0
package app

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/speakers"
	"slices"
)

func validateElection(p recordingPayload) error {
	o, e := p.Options, p.Election
	if o.PipelineID != "" {
		if e == nil || e.PipelineID != o.PipelineID || e.Revision != o.PipelineRevision || e.Revision < 1 || len(e.Digest) != 64 || pipeline.Validate(e.Preset, e.Definition) != nil {
			return contracts.Fail("invalid_request")
		}
		if o.RecognitionModelID != e.Definition.Recognition.ModelID || o.DiarizationModelID != e.Definition.Diarization.ModelID || !bytes.Equal(configurationBytes(o.Recognition), configurationBytes(e.Definition.Recognition.Recognition)) || o.Quality == nil || !bytes.Equal(configurationBytes(o.Quality), configurationBytes(e.Definition.Quality)) {
			return contracts.Fail("invalid_request")
		}
		attribution := e.Definition.Diarization.Diarization
		attribution.Quality = &e.Definition.Quality
		if !bytes.Equal(configurationBytes(o.Attribution), configurationBytes(attribution)) {
			return contracts.Fail("invalid_request")
		}
	}
	if e != nil && o.Transcription == "generate" {
		if o.Recognition.ContextDigest != e.Context.Digest || o.Recognition.ContextDigest != processing.HintsDigest(o.Recognition.Hints) || !slices.Equal(o.Recognition.Hints, e.Context.Hints) {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

type recordingElection struct {
	PipelineID string                   `json:"pipeline_id"`
	Revision   int64                    `json:"pipeline_revision"`
	Digest     string                   `json:"pipeline_digest"`
	Preset     string                   `json:"preset"`
	Definition pipeline.Definition      `json:"configuration"`
	Context    speakers.CompiledContext `json:"context"`
}

func (a *App) electRecordingOptions(o RecordingOptions) (RecordingOptions, *recordingElection, error) {
	if o.Quality != nil && o.Attribution.Quality != nil {
		return o, nil, contracts.Fail("invalid_request")
	}
	var election *recordingElection
	budget, supported := 200, true
	if o.PipelineID != "" {
		if !contracts.ValidID(o.PipelineID) || o.PipelineRevision < 0 {
			return o, nil, contracts.Fail("invalid_request")
		}
		p, e := a.Catalog.Pipeline(a.ctx, o.PipelineID)
		if e != nil {
			return o, nil, e
		}
		if o.PipelineRevision != 0 && o.PipelineRevision != p.Revision {
			return o, nil, contracts.Fail("conflict")
		}
		d, e := pipelineDefinition(p)
		if e != nil {
			return o, nil, e
		}
		d, e = pipeline.Elect(p.Preset, d, o.Overrides)
		if e != nil {
			return o, nil, e
		}
		// Full-stage overrides are the single route/model/options override contract.
		if o.RecognitionModelID != "" || o.DiarizationModelID != "" || o.Recognition.Device != "" || o.Recognition.Language != "" || len(o.Recognition.Hints) > 0 || o.Recognition.ContextDigest != "" || o.Attribution != (processing.DiarizationOptions{}) || o.Quality != nil {
			return o, nil, contracts.Fail("invalid_request")
		}
		capability, e := pipeline.Capabilities(d.Recognition)
		if e != nil {
			return o, nil, e
		}
		budget, supported = capability.MaxHintBytes, capability.SupportsHints
		o.PipelineRevision = p.Revision
		o.RecognitionModelID = d.Recognition.ModelID
		o.DiarizationModelID = d.Diarization.ModelID
		o.Recognition = d.Recognition.Recognition
		o.Attribution = d.Diarization.Diarization
		o.Quality = &d.Quality
		election = &recordingElection{PipelineID: p.ID, Revision: p.Revision, Digest: documentHash(configurationBytes(p)), Preset: p.Preset, Definition: d}
	} else if o.PipelineRevision != 0 || o.Overrides.Recognition != nil || o.Overrides.Diarization != nil || o.Overrides.Quality != nil {
		return o, nil, contracts.Fail("invalid_request")
	}
	if o.Transcription == "generate" {
		if o.Context.Language == "" {
			o.Context.Language = o.Recognition.Language
		}
		compiled, e := a.recognitionContext(o.Context, pipeline.Stage{Recognition: o.Recognition}, budget, supported)
		if e != nil {
			return o, nil, e
		}
		o.Recognition.Hints = compiled.Hints
		o.Recognition.ContextDigest = processing.HintsDigest(compiled.Hints)
		if _, e = processing.ValidateRecognitionOptions(o.Recognition, budget); e != nil {
			return o, nil, e
		}
		if election == nil {
			election = &recordingElection{Preset: "local", Definition: pipeline.Definition{Recognition: pipeline.Stage{Adapter: "faster-whisper", Version: "1", Mode: "local", ModelID: o.RecognitionModelID}, Diarization: pipeline.Stage{Adapter: "pyannote", Version: "1", Mode: "local", ModelID: o.DiarizationModelID}}}
		}
		election.Context = compiled
		election.Definition.Recognition.Recognition = o.Recognition
	}
	if o.Quality != nil {
		o.Attribution.Quality = o.Quality
	}
	return o, election, nil
}
func audioSourceMap(s *audioExecution) (processing.SourceMap, error) {
	if mapping, ok := s.SourceMap.(processing.SourceMap); ok {
		return mapping, nil
	}
	var mapping processing.SourceMap
	if json.Unmarshal(configurationBytes(s.SourceMap), &mapping) != nil {
		return mapping, contracts.Fail("invalid_timing")
	}
	return mapping, nil
}
func (a *App) electedRecognize(ctx context.Context, s *audioExecution, p recordingPayload) (processing.RecognitionResult, error) {
	options := p.Options.Recognition
	options.ExpectedModelDigest = p.ModelDigests[p.Options.RecognitionModelID]
	if p.Election != nil && p.Election.PipelineID != "" {
		l := p.Election.Definition.Recognition.Limits
		options.ExecutionLimits = &processing.ExecutionLimits{MaxAudioBytes: l.MaxAudioBytes, MaxResponseBytes: l.MaxResponseBytes, TimeoutMS: l.TimeoutMS}
	}
	if p.Election == nil || p.Election.Definition.Recognition.Mode == "local" {
		return s.Recognize(ctx, p.Options.RecognitionModelID, options)
	}
	mapping, e := audioSourceMap(s)
	if e != nil {
		return processing.RecognitionResult{}, e
	}
	return (&pipeline.Executor{Secrets: a.secrets, Client: a.hostedClient}).Recognize(ctx, s.Path, mapping, p.Election.Definition.Recognition, options)
}
func (a *App) electedDiarize(ctx context.Context, s *audioExecution, p recordingPayload) (processing.DiarizationResult, error) {
	options := p.Options.Attribution
	options.ExpectedModelDigest = p.ModelDigests[p.Options.DiarizationModelID]
	if p.Election != nil && p.Election.PipelineID != "" {
		l := p.Election.Definition.Diarization.Limits
		options.ExecutionLimits = &processing.ExecutionLimits{MaxAudioBytes: l.MaxAudioBytes, MaxResponseBytes: l.MaxResponseBytes, TimeoutMS: l.TimeoutMS}
	}
	if p.Election == nil || p.Election.Definition.Diarization.Mode == "local" {
		return s.Diarize(ctx, p.Options.DiarizationModelID, options)
	}
	mapping, e := audioSourceMap(s)
	if e != nil {
		return processing.DiarizationResult{}, e
	}
	return (&pipeline.Executor{Secrets: a.secrets, Client: a.hostedClient}).Diarize(ctx, s.Path, mapping, p.Election.Definition.Diarization, options)
}

func (a *App) electModelDigests(o RecordingOptions) (map[string]string, error) {
	// Deterministic recording fixtures replace the managed-model execution seam.
	if a.recordingFactory != nil {
		return nil, nil
	}
	out := map[string]string{}
	for _, stage := range []struct {
		id, capability string
		selected       bool
	}{
		{o.RecognitionModelID, "transcription", o.Transcription == "generate"},
		{o.DiarizationModelID, "diarization", o.Diarization != "reuse"},
	} {
		if !stage.selected || stage.id == "" {
			continue
		}
		install, e := a.Catalog.BaseModel(a.ctx, stage.id)
		if e != nil {
			return nil, e
		}
		var manifest models.Manifest
		if install.State != "available" || models.DecodeManifest(install.Manifest, &manifest) != nil || manifest.Digest() != install.Digest {
			return nil, contracts.Fail("model_unavailable")
		}
		capable := false
		for _, c := range manifest.Capabilities {
			if c == stage.capability {
				capable = true
			}
		}
		if !capable {
			return nil, contracts.Fail("unsupported_capability")
		}
		out[stage.id] = install.Digest
	}
	return out, nil
}
