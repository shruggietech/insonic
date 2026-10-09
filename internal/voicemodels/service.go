// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
)

func (s *Service) ValidateTrain(ctx context.Context, o TrainOptions) (TrainOptions, error) {
	var normalizationError error
	o.Adapter.Limits, normalizationError = normalizeLimits(o.Adapter.Limits)
	if normalizationError != nil {
		return o, normalizationError
	}
	if !contracts.ValidID(o.DatasetID) || o.FamilyID != "" && !contracts.ValidID(o.FamilyID) || !text(o.Name, 512) || !text(o.Kind, 128) || ValidateAdapter(o.Adapter) != nil || o.BaseModelID != "" && !models.ValidateReference(o.BaseModelID) || o.BaseDigest != "" && !digestPattern.MatchString(o.BaseDigest) || o.PipelineID != "" && !contracts.ValidID(o.PipelineID) || o.PipelineRevision < 0 || (o.PipelineID == "") != (o.PipelineRevision == 0) || o.CheckpointID != "" && !contracts.ValidID(o.CheckpointID) {
		return o, contracts.Fail("invalid_request")
	}
	if !contains(o.Adapter.OutputKinds, o.Kind) || o.Adapter.ID == "pyannote-profile" && o.BaseModelID == "" {
		return o, contracts.Fail("unsupported_capability")
	}
	if len(o.Parameters) == 0 {
		o.Parameters = json.RawMessage(`{}`)
	}
	var parameters map[string]json.RawMessage
	if strict(o.Parameters, &parameters) != nil || len(o.Parameters) > 65536 {
		return o, contracts.Fail("invalid_request")
	}
	dataset, err := s.Catalog.SpeakerDataset(ctx, o.DatasetID)
	if err != nil {
		return o, err
	}
	if dataset.Dataset.State != "current" {
		return o, contracts.Fail("conflict")
	}
	if len(dataset.References) == 0 {
		return o, contracts.Fail("empty_dataset")
	}
	if o.CheckpointID != "" {
		if !o.Adapter.SupportsResume {
			return o, contracts.Fail("unsupported_resume")
		}
		checkpoint, err := s.Catalog.SpeakerCheckpoint(ctx, o.CheckpointID)
		if err != nil {
			return o, err
		}
		if checkpoint.AdapterDigest != adapterDigest(o.Adapter) || checkpoint.BaseDigest != o.BaseDigest {
			return o, contracts.Fail("incompatible_checkpoint")
		}
		producing, err := s.Catalog.Work(ctx, checkpoint.WorkID)
		if err != nil {
			return o, err
		}
		var original TrainPayload
		if strict(producing.Payload, &original) != nil || original.Options.DatasetID != o.DatasetID || hash(original.Options.Parameters) != hash(o.Parameters) {
			return o, contracts.Fail("incompatible_checkpoint")
		}
	}
	o.Adapter.Limits, err = normalizeLimits(o.Adapter.Limits)
	return o, err
}
func (s *Service) phase(ctx context.Context, claim catalog.Work, name string, result any) error {
	raw, _ := json.Marshal(result)
	_, err := s.Catalog.CheckpointWork(ctx, claim, name, "running", raw, 30*time.Second)
	return err
}
func (s *Service) publishBytes(ctx context.Context, id string, raw []byte, kind string) (catalog.Publication, error) {
	directory, err := s.scratch()
	if err != nil {
		return catalog.Publication{}, err
	}
	defer os.RemoveAll(directory)
	path := filepath.Join(directory, "content")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		return catalog.Publication{}, contracts.Fail("unavailable")
	}
	return s.Artifacts.Publish(ctx, id, path, kind)
}
func (s *Service) retire(ids []string, op string) {
	cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, id := range ids {
		_, _ = s.Artifacts.RetireCurrent(cleanup, id)
	}
}
func instant() *catalog.Instant {
	now := time.Now().UTC()
	return &catalog.Instant{ISO: now.Format(time.RFC3339Nano), UnixNS: now.UnixNano()}
}
func (s *Service) ExecuteTrain(ctx context.Context, claim catalog.Work) (any, error) {
	var payload TrainPayload
	if claim.Kind != "models.train" || strict(claim.Payload, &payload) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	if _, err := s.Catalog.RenewWork(ctx, claim, 30*time.Second); err != nil {
		return nil, err
	}
	versionID := models.StableID("speaker-version:" + claim.ID)
	if existing, err := s.Catalog.SpeakerOutput(ctx, versionID); err == nil {
		return map[string]any{"version_id": existing.ID, "model_id": existing.ModelID, "state": "available"}, nil
	}
	options, err := s.ValidateTrain(ctx, payload.Options)
	if err != nil {
		return nil, err
	}
	if options.BaseModelID != "" && (!contracts.ValidID(options.BaseModelID) || !digestPattern.MatchString(options.BaseDigest)) {
		return nil, contracts.Fail("conflict")
	}
	dataset, err := s.Catalog.SpeakerDataset(ctx, options.DatasetID)
	if err != nil {
		return nil, err
	}
	directory, err := s.scratch()
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	if err = s.phase(ctx, claim, "preparation", map[string]any{"dataset_id": options.DatasetID}); err != nil {
		return nil, err
	}
	inputs, summary, err := s.prepare(ctx, dataset, options, directory)
	if err != nil {
		return nil, err
	}
	preparation, err := s.publishPreparation(ctx, claim, dataset, inputs, summary)
	if err != nil {
		return nil, err
	}
	preparationAccepted := false
	defer func() {
		if !preparationAccepted {
			s.retire([]string{preparation.ID}, claim.ID)
		}
	}()
	if err = s.phase(ctx, claim, "training", map[string]any{"dataset_id": options.DatasetID, "prepared_inputs": len(inputs), "diagnostics": summary.Diagnostics}); err != nil {
		return nil, err
	}
	request := AdapterRequest{ContractVersion: "1", Operation: "train", WorkID: claim.ID, Attempt: claim.Generation, BaseModelID: options.BaseModelID, BaseDigest: options.BaseDigest, Parameters: options.Parameters, Inputs: inputs, OutputDirectory: directory}
	if options.Adapter.ID != "pyannote-profile" {
		request.BaseModelDirectory, request.BaseFiles, err = s.materializeBase(ctx, options, directory)
		if err != nil {
			return nil, err
		}
	}
	if options.CheckpointID != "" {
		checkpoint, err := s.Catalog.SpeakerCheckpoint(ctx, options.CheckpointID)
		if err != nil {
			return nil, err
		}
		materialized, err := s.Artifacts.MaterializeBound(ctx, checkpoint.PublicationID, options.Adapter.Limits.MaxInputBytes)
		if err != nil {
			return nil, err
		}
		defer s.Artifacts.Release(context.Background(), materialized.PublicationID, materialized.Lease.ID)
		publication, err := s.Catalog.Publication(ctx, checkpoint.PublicationID)
		if err != nil {
			return nil, err
		}
		var data []byte
		if options.Adapter.Mode == "hosted" {
			data, err = os.ReadFile(materialized.Path)
			if err != nil {
				return nil, contracts.Fail("unavailable")
			}
		}
		request.Checkpoint = &Checkpoint{ID: checkpoint.ID, Step: checkpoint.Step, BaseDigest: checkpoint.BaseDigest, Compatibility: checkpoint.Compatibility, Artifact: Artifact{Path: materialized.Path, SHA256: publication.Digest, Size: publication.Size, Data: data}}
		request.Operation = "resume"
	}
	inputBytes := int64(len(request.Parameters))
	for _, input := range request.Inputs {
		inputBytes += input.Size
	}
	for _, input := range request.BaseFiles {
		inputBytes += input.Size
	}
	if request.Checkpoint != nil {
		inputBytes += request.Checkpoint.Artifact.Size
	}
	if inputBytes > options.Adapter.Limits.MaxInputBytes {
		return nil, contracts.Fail("input_limit")
	}
	started := instant()
	var output Output
	if options.Adapter.ID == "pyannote-profile" {
		output, err = s.enroll(ctx, options, inputs, directory)
	} else {
		output, err = s.invoke(ctx, options.Adapter, request)
	}
	if err != nil {
		if (ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded)) && options.Adapter.Mode == "hosted" && options.Adapter.SupportsCancel {
			cancelCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
			request.Operation = "cancel"
			request.Inputs = nil
			request.BaseFiles = nil
			request.Checkpoint = nil
			controlAdapter := options.Adapter
			controlAdapter.Limits.TimeoutMS = 5000
			_, _ = s.invoke(cancelCtx, controlAdapter, request)
			stop()
		}
		return nil, err
	}
	output.StartedAt = started
	output.CompletedAt = instant()
	output.PreparationPublication = &preparation
	output.Diagnostics = append(output.Diagnostics, summary.Diagnostics...)
	if err = validateOutput(options.Adapter, output); err != nil {
		return nil, err
	}
	if output.Kind != options.Kind {
		return nil, contracts.Fail("invalid_engine_output")
	}
	var outputBytes int64
	for _, a := range output.Artifacts {
		outputBytes += a.Size
	}
	for _, c := range output.Checkpoints {
		outputBytes += c.Artifact.Size
	}
	if outputBytes > options.Adapter.Limits.MaxOutputBytes {
		return nil, contracts.Fail("output_limit")
	}
	if err = s.phase(ctx, claim, "validation", map[string]any{"artifacts": len(output.Artifacts), "checkpoints": len(output.Checkpoints)}); err != nil {
		return nil, err
	}
	published := []catalog.Publication{}
	ids := []string{}
	accepted := false
	defer func() {
		if !accepted {
			s.retire(ids, claim.ID)
		}
	}()
	for index, artifact := range output.Artifacts {
		publication, err := s.publishOutput(ctx, claim, artifact, directory, index, "speaker-model")
		if err != nil {
			return nil, err
		}
		published = append(published, publication)
		ids = append(ids, publication.ID)
	}
	for index, checkpoint := range output.Checkpoints {
		if checkpoint.Step < 0 || !text(checkpoint.Compatibility, 256) || checkpoint.BaseDigest != options.BaseDigest || !options.Adapter.SupportsResume {
			return nil, contracts.Fail("invalid_engine_output")
		}
		checkpoint.ID = models.StableID("speaker-checkpoint:" + claim.ID + ":" + strconv.FormatInt(claim.Generation, 10) + ":" + strconv.Itoa(index))
		output.Checkpoints[index].ID = checkpoint.ID
		publication, err := s.publishOutput(ctx, claim, checkpoint.Artifact, directory, len(output.Artifacts)+index, "speaker-checkpoint")
		if err != nil {
			return nil, err
		}
		published = append(published, publication)
		ids = append(ids, publication.ID)
		_, err = s.Catalog.CommitSpeakerCheckpoint(ctx, claim, catalog.SpeakerCheckpoint{ID: checkpoint.ID, WorkID: claim.ID, Attempt: claim.Generation, Step: checkpoint.Step, BaseDigest: checkpoint.BaseDigest, Compatibility: checkpoint.Compatibility, AdapterDigest: adapterDigest(options.Adapter), PublicationID: publication.ID})
		if err != nil {
			return nil, err
		}
	}
	if output.State == "checkpointed" {
		result := map[string]any{"state": "checkpointed", "checkpoint_ids": checkpointIDs(output.Checkpoints), "dataset_id": options.DatasetID, "diagnostics": output.Diagnostics}
		raw, _ := json.Marshal(result)
		if _, err = s.Catalog.CheckpointWork(ctx, claim, "checkpointed", "failed", raw, 30*time.Second); err != nil {
			return nil, err
		}
		return result, contracts.Fail("training_incomplete")
	}
	if err = s.phase(ctx, claim, "publication", map[string]any{"version_id": versionID}); err != nil {
		return nil, err
	}
	manifest, err := BuildModelManifest(s.Artifacts.Workspace.Config.WorkspaceID, claim, dataset, options, output, published)
	if err != nil {
		return nil, err
	}
	publication, err := s.publishBytes(ctx, models.StableID("speaker-output-manifest:"+claim.ID+":"+strconv.FormatInt(claim.Generation, 10)), manifest, "speaker-model")
	if err != nil {
		return nil, err
	}
	ids = append([]string{publication.ID, preparation.ID}, ids...)
	familyID := options.FamilyID
	if familyID == "" {
		familyID = models.StableID("speaker-family:" + claim.ID)
	}
	rawIDs, _ := json.Marshal(ids)
	got, err := s.Catalog.CommitSpeakerOutput(ctx, claim, catalog.SpeakerOutput{ID: versionID, ModelID: familyID, SpeakerID: dataset.Dataset.SpeakerID, DatasetID: dataset.Dataset.ID, WorkID: claim.ID, Kind: output.Kind, Name: options.Name, Metadata: manifest, PublicationIDs: rawIDs})
	if err != nil {
		return nil, err
	}
	accepted = true
	preparationAccepted = true
	return map[string]any{"version_id": got.ID, "model_id": got.ModelID, "state": "available", "checkpoint_ids": checkpointIDs(output.Checkpoints)}, nil
}
func checkpointIDs(v []Checkpoint) []string {
	out := []string{}
	for _, c := range v {
		out = append(out, c.ID)
	}
	return out
}
func (s *Service) enroll(ctx context.Context, options TrainOptions, inputs []PreparedInput, directory string) (Output, error) {
	if s.Embed == nil && s.EmbedBatch == nil {
		return Output{}, contracts.Fail("unavailable")
	}
	vector := []float64{}
	var duration int64
	var skipped int64
	for start := 0; start < len(inputs); start += 64 {
		end := start + 64
		if end > len(inputs) {
			end = len(inputs)
		}
		vectors, err := s.embedInputs(ctx, inputs[start:end], options.BaseModelID, options.BaseDigest)
		if err != nil {
			return Output{}, err
		}
		for index, input := range inputs[start:end] {
			if vectors[index] == nil {
				skipped++
				continue
			}
			values, err := normalized(vectors[index])
			if err != nil {
				return Output{}, err
			}
			if len(vector) == 0 {
				vector = make([]float64, len(values))
			}
			if len(vector) != len(values) {
				return Output{}, contracts.Fail("incompatible_version")
			}
			for i, v := range values {
				vector[i] += v * float64(input.DurationUS)
			}
			duration += input.DurationUS
		}
	}
	if duration == 0 {
		return Output{}, contracts.Fail("empty_dataset")
	}
	vector, err := normalized(vector)
	if err != nil {
		return Output{}, err
	}
	profile := Profile{Kind: "voice-embedding", ModelID: options.BaseModelID, ModelDigest: options.BaseDigest, Vector: vector, EvidenceUS: duration}
	raw, _ := json.Marshal(profile)
	path := filepath.Join(directory, "profile.json")
	if err = os.WriteFile(path, raw, 0600); err != nil {
		return Output{}, contracts.Fail("unavailable")
	}
	out := Output{ContractVersion: "1", Kind: "voice-embedding", Architecture: "pyannote", Consumers: []string{"voice-matching"}, SupportedOperations: []string{"voice-matching"}, Artifacts: []Artifact{{Role: "profile.json", Format: "application/json", Path: "profile.json", SHA256: hash(raw), Size: int64(len(raw))}}}
	if skipped > 0 {
		out.Diagnostics = []processing.Diagnostic{{Code: "embedding_short_clip_excluded", Count: skipped}}
	}
	return out, nil
}
func (s *Service) publishOutput(ctx context.Context, claim catalog.Work, a Artifact, directory string, index int, kind string) (catalog.Publication, error) {
	limits := int64(64 << 30)
	if a.Size < 1 || a.Size > limits || !digestPattern.MatchString(a.SHA256) || !models.ValidRole(a.Role) || !text(a.Format, 128) {
		return catalog.Publication{}, contracts.Fail("invalid_engine_output")
	}
	id := models.StableID("speaker-artifact:" + claim.ID + ":" + strconv.FormatInt(claim.Generation, 10) + ":" + strconv.Itoa(index) + ":" + a.SHA256)
	if len(a.Data) > 0 {
		if a.Path != "" || int64(len(a.Data)) != a.Size || hash(a.Data) != a.SHA256 {
			return catalog.Publication{}, contracts.Fail("invalid_engine_output")
		}
		return s.publishBytes(ctx, id, a.Data, kind)
	}
	if !models.ValidRole(a.Path) {
		return catalog.Publication{}, contracts.Fail("invalid_engine_output")
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return catalog.Publication{}, contracts.Fail("unavailable")
	}
	defer root.Close()
	f, err := root.Open(a.Path)
	if err != nil {
		return catalog.Publication{}, contracts.Fail("invalid_engine_output")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() != a.Size {
		return catalog.Publication{}, contracts.Fail("invalid_engine_output")
	}
	stage := filepath.Join(directory, "verified-"+strconv.Itoa(index))
	out, err := os.OpenFile(stage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return catalog.Publication{}, contracts.Fail("unavailable")
	}
	_, err = io.Copy(out, io.LimitReader(&contextReader{ctx: ctx, reader: f}, a.Size+1))
	syncErr := out.Sync()
	closeErr := out.Close()
	if err != nil || syncErr != nil || closeErr != nil {
		return catalog.Publication{}, contracts.Fail("unavailable")
	}
	p, err := s.Artifacts.Publish(ctx, id, stage, kind)
	if err == nil && (p.Digest != a.SHA256 || p.Size != a.Size) {
		s.retire([]string{p.ID}, claim.ID)
		return p, contracts.Fail("conflict")
	}
	return p, err
}
func (s *Service) List(ctx context.Context, speakerID string) ([]catalog.SpeakerOutput, error) {
	return s.Catalog.SpeakerOutputs(ctx, speakerID)
}
func (s *Service) Show(ctx context.Context, versionID string) (catalog.SpeakerOutput, error) {
	return s.Catalog.SpeakerOutput(ctx, versionID)
}
