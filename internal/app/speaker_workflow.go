// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/pipeline"
	sd "github.com/shruggietech/insonic/internal/speakerdiscovery"
	"github.com/shruggietech/insonic/internal/voicemodels"
)

type SpeakerWorkflowInput struct {
	Configuration    *pipeline.SpeakerWorkflow          `json:"configuration,omitempty"`
	Training         *voicemodels.TrainingConfiguration `json:"training,omitempty"`
	PipelineID       string                             `json:"pipeline_id,omitempty"`
	PipelineRevision int64                              `json:"pipeline_revision,omitempty"`
}
type speakerWorkflowPayload struct {
	RecordingID       string                   `json:"recording_id"`
	RequestDigest     string                   `json:"request_digest"`
	Configuration     pipeline.SpeakerWorkflow `json:"configuration"`
	Discovery         discoveryPayload         `json:"discovery"`
	ModelDependencies []string                 `json:"model_dependencies,omitempty"`
	Training          voicemodels.TrainPayload `json:"training"`
	ProfileRevisions  map[string]int64         `json:"profile_revisions"`
	PipelineID        string                   `json:"pipeline_id,omitempty"`
	PipelineRevision  int64                    `json:"pipeline_revision,omitempty"`
}
type speakerWorkflowResult struct {
	Stage             string                      `json:"stage"`
	Awaiting          string                      `json:"awaiting_work_id,omitempty"`
	DiscoveryID       string                      `json:"discovery_work_id"`
	DiscoveryRevision int64                       `json:"discovery_revision"`
	Application       *catalog.SpeakerApplication `json:"application,omitempty"`
	Index             int                         `json:"speaker_index"`
	Enrollments       []workflowEnrollment        `json:"enrollments"`
}
type workflowEnrollment struct {
	SpeakerID string `json:"speaker_id"`
	DatasetID string `json:"dataset_id"`
	WorkID    string `json:"work_id,omitempty"`
	VersionID string `json:"version_id,omitempty"`
	State     string `json:"state"`
}

var errSpeakerWorkflowYield = errors.New("speaker workflow yielded")

func workflowID(parent, stage, person string) string {
	return models.StableID("speaker-workflow:" + parent + ":" + stage + ":" + person)
}
func (a *App) requestSpeakerWorkflow(ctx context.Context, req contracts.Request) (catalog.Work, error) {
	var in SpeakerWorkflowInput
	var empty catalog.Work
	if !contracts.ValidID(req.RequestID) || !contracts.ValidID(req.ItemID) || strictPayload(req.Data, &in) != nil {
		return empty, contracts.Fail("invalid_request")
	}
	digest := sd.Digest([]any{req.ItemID, in})
	if old, e := a.Catalog.Work(ctx, req.RequestID); e == nil {
		var p speakerWorkflowPayload
		if old.Kind != "speakers.run" || strictPayload(old.Payload, &p) != nil || p.RequestDigest != digest {
			return empty, contracts.Fail("conflict")
		}
		return old, nil
	} else if contracts.SafeError(e).Code != "not_found" {
		return empty, e
	}
	if in.PipelineID != "" {
		if !contracts.ValidID(in.PipelineID) || in.PipelineRevision < 1 || in.Configuration != nil || in.Training != nil {
			return empty, contracts.Fail("invalid_request")
		}
		saved, e := a.Catalog.Pipeline(ctx, in.PipelineID)
		if e != nil {
			return empty, e
		}
		if saved.Revision != in.PipelineRevision {
			return empty, contracts.Fail("conflict")
		}
		bound, e := a.bindPortableConfiguration(saved.Configuration)
		if e != nil {
			return empty, e
		}
		var d pipeline.Definition
		if strictPayload(bound, &d) != nil || pipeline.Validate(saved.Preset, d) != nil || d.SpeakerWorkflow == nil {
			return empty, contracts.Fail("invalid_request")
		}
		in.Configuration = d.SpeakerWorkflow
		if in.Configuration.Enroll && len(d.SpeakerTraining) > 0 {
			in.Training = &voicemodels.TrainingConfiguration{}
			if strictPayload(d.SpeakerTraining, in.Training) != nil {
				return empty, contracts.Fail("invalid_request")
			}
		}
	} else if in.PipelineRevision != 0 {
		return empty, contracts.Fail("invalid_request")
	}
	if in.Configuration == nil {
		return empty, contracts.Fail("invalid_request")
	}
	c, e := pipeline.NormalizeSpeakerWorkflow(*in.Configuration)
	if e != nil {
		return empty, e
	}
	if c.Enroll != (in.Training != nil) {
		return empty, contracts.Fail("invalid_request")
	}
	f, e := a.Catalog.FreezeSpeakerDiscovery(ctx, req.ItemID, c.Limits.MaxCues, "")
	if e != nil {
		return empty, e
	}
	p := speakerWorkflowPayload{RecordingID: req.ItemID, RequestDigest: digest, Configuration: c, ProfileRevisions: map[string]int64{}, PipelineID: in.PipelineID, PipelineRevision: in.PipelineRevision}
	p.Discovery = discoveryPayload{RecordingID: req.ItemID, SourceEpoch: f.SourceEpoch, MaxCues: c.Limits.MaxCues, ExpectedRevision: f.ExpectedRevision, RequestDigest: digest, Configuration: c.Provider, Limits: c.Limits, Policy: c.Policy}
	if c.Enroll {
		raw, _ := json.Marshal(in.Training)
		if voicemodels.ValidateTrainingConfiguration(raw) != nil {
			return empty, contracts.Fail("invalid_request")
		}
		tr := in.Training
		id, hash, selections, _, err := a.freezeSpeakerBase(ctx, tr.BaseModelID, "speaker-model-training")
		if err != nil {
			return empty, err
		}
		tools, err := a.freezeSpeakerProcessingTools()
		if err != nil {
			return empty, err
		}
		deps, err := a.queueModelSelections(ctx, selections)
		if err != nil {
			return empty, err
		}
		options := voicemodels.TrainOptions{Name: "Contextual speaker profile", Adapter: tr.Adapter, Kind: tr.Kind, BaseModelID: id, BaseDigest: hash, Parameters: tr.Parameters, Device: tr.Device, PipelineID: in.PipelineID, PipelineRevision: in.PipelineRevision}
		if options.Adapter.ID == "pyannote-profile" && options.Device == "" {
			options.Device = "auto"
		}
		p.ModelDependencies = deps
		p.Training = voicemodels.TrainPayload{Options: options, ProcessingTools: tools, ModelSelections: selections, ModelDependencies: deps}
		if c.SelectProfile {
			for _, id := range f.Context.Identities {
				profile, err := a.Catalog.SpeakerProfile(ctx, id.ID)
				if err != nil {
					return empty, err
				}
				p.ProfileRevisions[id.ID] = profile.Revision
			}
		}
	}
	raw, _ := json.Marshal(p)
	w, e := a.Catalog.EnqueueWork(ctx, req.RequestID, "speakers.run", raw)
	if e == nil {
		a.recoverWork()
	}
	return w, e
}
func (a *App) speakerWorkflowReady(item catalog.Work) bool {
	if item.Kind != "speakers.run" {
		return true
	}
	var r speakerWorkflowResult
	if json.Unmarshal(item.Result, &r) != nil || r.Awaiting == "" {
		return true
	}
	child, e := a.Catalog.Work(a.ctx, r.Awaiting)
	return e != nil || child.State == "succeeded" || child.State == "failed" || child.State == "cancelled"
}
func (a *App) processSpeakerWorkflow(ctx context.Context, claim catalog.Work) (any, error) {
	var p speakerWorkflowPayload
	if strictPayload(claim.Payload, &p) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	r := speakerWorkflowResult{Stage: "discovery", DiscoveryID: workflowID(claim.ID, "discovery", ""), Enrollments: []workflowEnrollment{}}
	current, e := a.Catalog.Work(ctx, claim.ID)
	if e != nil {
		return r, e
	}
	if string(current.Result) != "null" {
		if json.Unmarshal(current.Result, &r) != nil {
			return r, contracts.Fail("invalid_request")
		}
	}
	checkpoint := func(stage string) error {
		r.Stage = stage
		r.Awaiting = ""
		raw, _ := json.Marshal(r)
		_, e := a.Catalog.CheckpointWork(ctx, claim, stage, "running", raw, leaseTTL)
		return e
	}
	wait := func(id string) (any, error) {
		r.Awaiting = id
		raw, _ := json.Marshal(r)
		if _, e := a.Catalog.YieldSpeakerWorkflow(ctx, claim, "waiting-"+r.Stage, raw); e != nil {
			return r, e
		}
		return r, errSpeakerWorkflowYield
	}
	terminal := func(w catalog.Work) error {
		if w.State == "failed" || w.State == "cancelled" {
			return &contracts.Error{Code: "operation_failed", Message: "A speaker workflow stage failed or was cancelled. Inspect the child work, correct its prerequisites, then retry the workflow."}
		}
		return nil
	}
	if r.Stage == "discovery" {
		raw, _ := json.Marshal(p.Discovery)
		child, e := a.Catalog.EnqueueChildWork(ctx, claim, r.DiscoveryID, "speakers.discover", raw)
		if e != nil {
			return r, e
		}
		if e = terminal(child); e != nil {
			return r, e
		}
		if child.State != "succeeded" {
			return wait(child.ID)
		}
		d, e := a.Catalog.SpeakerDiscovery(ctx, p.RecordingID)
		if e != nil {
			return r, e
		}
		if d.WorkID != child.ID {
			return r, contracts.Fail("conflict")
		}
		r.DiscoveryRevision = d.Revision
		if !p.Configuration.Apply {
			r.Stage = "complete"
			r.Awaiting = ""
			return r, nil
		}
		if e = checkpoint("application"); e != nil {
			return r, e
		}
	}
	if r.Stage == "application" {
		applied, e := a.Catalog.ApplySpeakerDiscovery(ctx, workflowID(claim.ID, "apply", ""), p.RecordingID, r.DiscoveryRevision, &claim)
		if e != nil {
			return r, e
		}
		r.Application = &applied
		if !p.Configuration.Enroll {
			r.Stage = "complete"
			r.Awaiting = ""
			return r, nil
		}
		if e = checkpoint("enrollment"); e != nil {
			return r, e
		}
	}
	if r.Stage != "enrollment" || r.Application == nil {
		return r, contracts.Fail("invalid_request")
	}
	for r.Index < len(r.Application.SpeakerIDs) {
		person := r.Application.SpeakerIDs[r.Index]
		service, e := a.speakerModelService()
		if e != nil {
			return r, e
		}
		ds, e := service.CreateDataset(ctx, workflowID(claim.ID, "dataset", person), voicemodels.DatasetOptions{SpeakerID: person, Recipe: p.Configuration.Recipe}, claim)
		if e != nil {
			return r, e
		}
		if ds.Dataset.State != "current" {
			return r, contracts.Fail("conflict")
		}
		if len(ds.References) == 0 {
			r.Enrollments = append(r.Enrollments, workflowEnrollment{SpeakerID: person, DatasetID: ds.Dataset.ID, State: "no-eligible-evidence"})
			r.Index++
			if e = checkpoint("enrollment"); e != nil {
				return r, e
			}
			continue
		}
		payload := p.Training
		payload.Options.DatasetID = ds.Dataset.ID
		payload.RequestDigest = sd.Digest([]any{claim.ID, person, ds.Dataset.ID, p.Training})
		payload.Options, e = service.ValidateTrain(ctx, payload.Options)
		if e != nil {
			return r, e
		}
		raw, _ := json.Marshal(payload)
		child, e := a.Catalog.EnqueueChildWork(ctx, claim, workflowID(claim.ID, "enroll", person), "models.train", raw)
		if e != nil {
			return r, e
		}
		if e = terminal(child); e != nil {
			return r, e
		}
		if child.State != "succeeded" {
			return wait(child.ID)
		}
		var result struct {
			VersionID string `json:"version_id"`
		}
		if json.Unmarshal(child.Result, &result) != nil || !contracts.ValidID(result.VersionID) {
			return r, contracts.Fail("invalid_request")
		}
		if p.Configuration.SelectProfile {
			expected, ok := p.ProfileRevisions[person]
			if !ok {
				return r, contracts.Fail("conflict")
			}
			if _, e = a.Catalog.SetWorkflowSpeakerProfile(ctx, claim, workflowID(claim.ID, "profile", person), person, expected, result.VersionID); e != nil {
				return r, e
			}
		}
		r.Enrollments = append(r.Enrollments, workflowEnrollment{SpeakerID: person, DatasetID: ds.Dataset.ID, WorkID: child.ID, VersionID: result.VersionID, State: "enrolled"})
		r.Index++
		if e = checkpoint("enrollment"); e != nil {
			return r, e
		}
	}
	r.Stage = "complete"
	r.Awaiting = ""
	return r, nil
}
