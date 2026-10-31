// SPDX-License-Identifier: Apache-2.0
package app

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/pipeline"
	"github.com/shruggietech/insonic/internal/processing"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"github.com/shruggietech/insonic/internal/workers"
)

type ReadinessInput struct {
	PipelineID        string `json:"pipeline_id,omitempty"`
	PipelineRevision  int64  `json:"pipeline_revision,omitempty"`
	RecognitionModel  string `json:"recognition_model_id,omitempty"`
	DiarizationModel  string `json:"diarization_model_id,omitempty"`
	RecognitionDevice string `json:"recognition_device,omitempty"`
	DiarizationDevice string `json:"diarization_device,omitempty"`
	DatasetID         string `json:"dataset_id,omitempty"`
}
type Prerequisite struct {
	Name       string `json:"name"`
	State      string `json:"state"`
	Available  bool   `json:"available"`
	Correction string `json:"correction,omitempty"`
	Identity   any    `json:"identity,omitempty"`
}
type TaskReadiness struct {
	State         string         `json:"state"`
	Usable        bool           `json:"usable"`
	Prerequisites []Prerequisite `json:"prerequisites"`
	Actions       []string       `json:"actions"`
}
type Readiness struct {
	Tasks            map[string]TaskReadiness      `json:"tasks"`
	Environments     map[string]workers.Inspection `json:"environments"`
	Verification     string                        `json:"verification"`
	PipelineID       string                        `json:"pipeline_id,omitempty"`
	PipelineRevision int64                         `json:"pipeline_revision,omitempty"`
	PipelineUsable   bool                          `json:"pipeline_usable"`
}

func composeTask(p []Prerequisite) TaskReadiness {
	r := TaskReadiness{State: "prerequisites_available", Usable: true, Prerequisites: p, Actions: []string{}}
	for _, v := range p {
		if !v.Available {
			r.Usable = false
			if v.State == "configured_not_probed" || v.State == "unknown" {
				if r.State != "needs_attention" {
					r.State = "unknown"
				}
			} else {
				r.State = "needs_attention"
			}
			if v.Correction != "" {
				r.Actions = append(r.Actions, v.Correction)
			}
		}
	}
	return r
}
func checkedPin(ctx context.Context, name string, p library.PinnedFile) Prerequisite {
	r := Prerequisite{Name: name, State: "not_configured", Correction: "Configure verified installed " + name + " tools."}
	if p.Path == "" {
		return r
	}
	r.State = "configured"
	r.Identity = p
	if ctx.Err() != nil {
		r.State = "unknown"
		r.Correction = "Verification exhausted its enclosing budget or was cancelled; inspect readiness again."
		return r
	}
	actual, e := workers.Hash(ctx, p.Path)
	if e == nil && actual.SHA256 == p.SHA256 {
		r.State = "file_verified"
		r.Available = true
		r.Correction = ""
	} else if ctx.Err() != nil {
		r.State = "unknown"
		r.Correction = "Verification exhausted its enclosing budget or was cancelled; inspect readiness again."
	} else {
		r.State = "file_unavailable_or_changed"
	}
	return r
}
func (a *App) processingReadiness(ctx context.Context, input ReadinessInput) (Readiness, error) {
	var d *pipeline.Definition
	if input.PipelineRevision < 0 || input.PipelineRevision != 0 && input.PipelineID == "" || input.PipelineID != "" && !contracts.ValidID(input.PipelineID) || input.DatasetID != "" && !contracts.ValidID(input.DatasetID) {
		return Readiness{}, contracts.Fail("invalid_request")
	}
	if input.PipelineID != "" {
		p, e := a.Catalog.Pipeline(ctx, input.PipelineID)
		if e != nil {
			return Readiness{}, e
		}
		if input.PipelineRevision != 0 && p.Revision != input.PipelineRevision {
			return Readiness{}, contracts.Fail("conflict")
		}
		value, e := a.configuredPipelineDefinition(p)
		if e != nil {
			return Readiness{}, e
		}
		d = &value
		input.PipelineRevision = p.Revision
	}
	return a.readinessDefinition(ctx, input, d)
}
func (a *App) readinessDefinition(ctx context.Context, input ReadinessInput, d *pipeline.Definition) (Readiness, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	r := Readiness{Tasks: map[string]TaskReadiness{}, Environments: map[string]workers.Inspection{}, Verification: "prerequisite_bytes_and_library_probe; model_inference_not_run", PipelineID: input.PipelineID, PipelineRevision: input.PipelineRevision}
	for _, device := range []string{input.RecognitionDevice, input.DiarizationDevice} {
		if !processing.ValidDevice(device) {
			return r, contracts.Fail("invalid_request")
		}
	}
	for _, ref := range []string{input.RecognitionModel, input.DiarizationModel} {
		if ref != "" && !models.ValidateReference(ref) {
			return r, contracts.Fail("invalid_request")
		}
	}
	c, _ := ReadProcessingTools(a.Workspace)
	media, _ := ReadMediaTools(a.Workspace)
	imports := []Prerequisite{checkedPin(ctx, "ffprobe", library.PinnedFile{Path: media.FFprobe.Path, SHA256: media.FFprobe.SHA256}), checkedPin(ctx, "exiftool", library.PinnedFile{Path: media.ExifTool.Path, SHA256: media.ExifTool.SHA256}), checkedPin(ctx, "ffmpeg", library.PinnedFile{Path: media.FFmpeg.Path, SHA256: media.FFmpeg.SHA256})}
	storage, _ := a.storageDiagnostics()
	storageNeed := Prerequisite{Name: "storage", State: storage.State, Available: storage.State == "available"}
	if !storageNeed.Available {
		storageNeed.Correction = "Inspect doctor storage diagnostics and correct the elected storage backend."
	}
	imports = append(imports, storageNeed)
	r.Tasks["import"] = composeTask(imports)
	cueson := checkedPin(ctx, "cueson", library.PinnedFile{Path: c.Cueson.Executable, SHA256: c.Cueson.ExecutableSHA256})
	r.Tasks["existing_subtitles"] = composeTask([]Prerequisite{cueson, storageNeed})
	ffmpeg := checkedPin(ctx, "ffmpeg", library.PinnedFile{Path: c.Processing.FFmpeg.Path, SHA256: c.Processing.FFmpeg.SHA256})
	if c.Processing.FFmpeg.Path == "" {
		ffmpeg = checkedPin(ctx, "ffmpeg", library.PinnedFile{Path: media.FFmpeg.Path, SHA256: media.FFmpeg.SHA256})
	}
	worker := checkedPin(ctx, "processing worker", c.Processing.Worker)
	defaults := models.Defaults()
	if input.RecognitionModel == "" {
		input.RecognitionModel = models.InstallationID(defaults[0].Manifest)
	}
	if input.DiarizationModel == "" {
		input.DiarizationModel = models.InstallationID(defaults[1].Manifest)
	}
	if d != nil && pipeline.HasProcessing(*d) {
		input.RecognitionModel = d.Recognition.ModelID
		input.DiarizationModel = d.Diarization.ModelID
		input.RecognitionDevice = d.Recognition.Recognition.Device
		input.DiarizationDevice = d.Diarization.Diarization.Device
	}
	for _, role := range []string{"recognition", "diarization"} {
		python, receipt, ref, device, task := c.Processing.RecognitionPython, c.Processing.RecognitionEnvironment, input.RecognitionModel, input.RecognitionDevice, "transcription"
		var stage *pipeline.Stage
		if role == "diarization" {
			python, receipt, ref, device, task = c.Processing.DiarizationPython, c.Processing.DiarizationEnvironment, input.DiarizationModel, input.DiarizationDevice, "diarization"
		}
		if d != nil && pipeline.HasProcessing(*d) {
			stage = &d.Recognition
			if role == "diarization" {
				stage = &d.Diarization
			}
		}
		if stage != nil && stage.Mode == "hosted" {
			r.Tasks[task] = composeTask([]Prerequisite{{Name: "hosted worker", State: "configured_not_probed", Correction: "Test the elected hosted worker's audio protocol and credentials; configuration alone is not availability."}})
			continue
		}
		if device == "" {
			device = "auto"
		}
		var inspection workers.Inspection
		if ctx.Err() != nil {
			inspection = workers.UnattemptedInspection(ctx, role)
		} else {
			var e error
			inspection, e = a.workerService().Inspect(ctx, role, python.Path, false, receipt)
			if e != nil {
				return r, e
			}
		}
		r.Environments[role] = inspection
		engine := Prerequisite{Name: role + " engine", State: "engine_unavailable", Available: inspection.Compatible && inspection.EngineAvailable, Correction: "Use workers plan/setup to prepare or reuse a compatible " + role + " environment."}
		if inspection.Correction != "" {
			engine.Correction = inspection.Correction
		}
		if inspection.Probe != nil && (inspection.Probe.Reason == "deadline" || inspection.Probe.Reason == "cancelled") {
			engine.State = "unknown"
		}
		if engine.Available {
			engine.State = "engine_available"
			engine.Correction = ""
		}
		identity := checkedPin(ctx, role+" interpreter", python)
		if receipt.Path != "" && inspection.Probe != nil && inspection.Probe.Stage == "receipt_verification" && inspection.Error != "" {
			identity.Available = false
			identity.State = "environment_unavailable"
			identity.Correction = inspection.Correction
			if inspection.Probe.Reason == "deadline" || inspection.Probe.Reason == "cancelled" {
				identity.State = "unknown"
			} else if inspection.Error == "checksum_mismatch" {
				identity.State = "environment_changed"
			}
		}
		deviceNeed := devicePrerequisite(device, inspection.Devices)
		if !deviceNeed.Available && deviceNeed.Correction == "" {
			deviceNeed.Correction = "Prepare compatible CPU dependencies, or explicitly select a supported device."
		}
		model := Prerequisite{Name: "model", State: "unknown", Correction: "The enclosing readiness budget expired before model verification; retry readiness."}
		if ctx.Err() == nil {
			model = a.modelPrerequisite(ctx, ref, task)
		}
		if engine.State == "unknown" {
			deviceNeed.State = "unknown"
			deviceNeed.Correction = engine.Correction
		}
		r.Tasks[task] = composeTask([]Prerequisite{ffmpeg, cueson, worker, identity, engine, model, deviceNeed, storageNeed})
	}
	if ctx.Err() != nil {
		r.Tasks["training"] = composeTask([]Prerequisite{{Name: "training verification", State: "unknown", Correction: "Readiness exhausted its enclosing budget or was cancelled; retry readiness."}})
		return r, nil
	}
	training := r.Tasks["diarization"]
	training.Prerequisites = append([]Prerequisite{}, training.Prerequisites...)
	adapter := Prerequisite{Name: "training adapter", State: "not_selected", Correction: "Select a saved pipeline with an elected training adapter."}
	if d != nil && len(d.SpeakerTraining) > 0 {
		var selected voicemodels.TrainingConfiguration
		if json.Unmarshal(d.SpeakerTraining, &selected) != nil {
			return r, contracts.Fail("invalid_request")
		}
		if selected.Adapter.ID != "pyannote-profile" {
			need := Prerequisite{Name: "elected training adapter", State: "configured_not_probed", Correction: "Verify the elected hosted adapter protocol and credentials; local inspection does not contact it."}
			if selected.Adapter.Mode == "local" {
				need.State = "file_verified"
				need.Available = true
				need.Correction = ""
				if err := voicemodels.VerifyLocalAdapter(ctx, selected.Adapter); err != nil {
					if ctx.Err() != nil {
						return r, contracts.Fail("cancelled")
					}
					need.State = "file_unavailable_or_changed"
					need.Available = false
					need.Correction = contracts.SafeError(err).Message
				}
			}
			training = composeTask([]Prerequisite{ffmpeg, storageNeed, need})
			if selected.BaseModelID != "" {
				training.Prerequisites = append(training.Prerequisites, a.modelPrerequisite(ctx, selected.BaseModelID, "speaker-model-training"))
			}
		} else {
			for i, p := range training.Prerequisites {
				if p.Name == "model" && selected.BaseModelID != "" {
					training.Prerequisites[i] = a.modelPrerequisite(ctx, selected.BaseModelID, "speaker-model-training")
				}
				if strings.HasSuffix(p.Name, " device policy") {
					training.Prerequisites[i] = devicePrerequisite(selected.Device, r.Environments["diarization"].Devices)
					if p.State == "unknown" {
						training.Prerequisites[i].State = "unknown"
						training.Prerequisites[i].Available = false
						training.Prerequisites[i].Correction = p.Correction
					}
				}
			}
		}
		adapter.State, adapter.Available, adapter.Correction = "elected", true, ""
	}
	training.Prerequisites = append(training.Prerequisites, adapter)
	dataset := Prerequisite{Name: "training dataset", State: "not_selected", Correction: "Select a current nonempty speaker dataset and an elected training adapter."}
	if input.DatasetID != "" {
		value, e := a.Catalog.SpeakerDataset(ctx, input.DatasetID)
		if e == nil && value.Dataset.State == "current" && len(value.References) > 0 {
			dataset.State = "current_nonempty"
			dataset.Available = true
			dataset.Correction = ""
		} else {
			dataset.State = "unavailable_or_empty"
			dataset.Correction = "Rebuild or select a current nonempty speaker dataset."
		}
	}
	training.Prerequisites = append(training.Prerequisites, dataset)
	r.Tasks["training"] = composeTask(training.Prerequisites)
	r.PipelineUsable = electedPipelineUsable(d, r.Tasks)
	return r, nil
}

func electedPipelineUsable(d *pipeline.Definition, tasks map[string]TaskReadiness) bool {
	if d == nil {
		return false
	}
	elected, usable := false, true
	if pipeline.HasProcessing(*d) {
		elected = true
		usable = tasks["transcription"].Usable && tasks["diarization"].Usable
	}
	if len(d.SpeakerTraining) > 0 {
		elected = true
		usable = usable && tasks["training"].Usable
	}
	return elected && usable
}
func (a *App) modelPrerequisite(ctx context.Context, ref, operation string) Prerequisite {
	r := Prerequisite{Name: "model", State: "not_selected", Correction: "Select/register a compatible exact model, then acquire/verify it through model jobs."}
	interrupted := func() Prerequisite {
		r.State, r.Available = "unknown", false
		r.Correction = "Model verification exhausted its enclosing budget or was cancelled; retry readiness."
		return r
	}
	if ctx.Err() != nil {
		return interrupted()
	}
	if ref == "" {
		return r
	}
	selection, e := models.NewResolver(a.Catalog, a.secrets).Resolve(ctx, ref, operation)
	if ctx.Err() != nil {
		return interrupted()
	}
	if e != nil || selection.Target.Kind != "base" || !selection.Compatible {
		r.State = "missing_or_incompatible"
		return r
	}
	r.Identity = map[string]any{"model_id": selection.Target.ID, "manifest_digest": selection.Digest}
	r.State = "registered_not_acquired"
	service, e := a.modelService()
	if e != nil {
		r.State = "storage_unavailable"
		return r
	}
	e = service.Verify(ctx, selection.Target.ID)
	if ctx.Err() != nil {
		return interrupted()
	}
	if e == nil {
		r.Available = true
		r.State = "model_verified"
		r.Correction = ""
	} else if typed, ok := e.(*contracts.Error); ok && typed.Reason != "" {
		r.State = typed.Reason
	}
	return r
}

func devicePrerequisite(policy string, devices map[string]workers.Device) Prerequisite {
	if policy == "" {
		policy = "auto"
	}
	selected := policy
	if selected == "auto" {
		selected = "cuda"
	}
	dev := devices[selected]
	out := Prerequisite{Name: policy + " device policy", State: dev.DependencyState, Available: dev.Available, Correction: dev.Correction}
	if !dev.Available && selected == "cuda" && devices["cpu"].Available {
		out.State = "cpu_fallback_available"
		out.Available = true
		out.Correction = "CUDA is unavailable (" + dev.DependencyState + "); execution may use CPU fallback. Inspect workers for compatible NVIDIA prerequisites."
	}
	if !out.Available && out.Correction == "" {
		out.Correction = "Prepare compatible CPU dependencies, or explicitly select a supported device."
	}
	return out
}
