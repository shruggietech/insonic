// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/schemas"
)

func portableInstant(t time.Time) catalog.Instant {
	t = t.UTC()
	return catalog.Instant{ISO: t.Format(time.RFC3339Nano), UnixNS: t.UnixNano()}
}

func portableDigest(v any) string {
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func portableArtifact(p catalog.Publication) map[string]any {
	return map[string]any{"artifact_id": p.ArtifactID, "sha256": p.Digest, "byte_length": p.Size}
}

// BuildDatasetManifest contains current evidence references, never assignments or text.
func BuildDatasetManifest(workspaceID, datasetID string, speaker catalog.Speaker, revision int64, recipe Recipe, refs []catalog.CurrentReference, summary Summary) (json.RawMessage, error) {
	members := []any{}
	sources := map[string]bool{}
	for i, ref := range refs {
		member := map[string]any{"ordinal": i, "segment_id": catalog.SpeakerDatasetSegmentID(datasetID, ref), "segment_revision": ref.RecordingRevision, "recording_id": ref.RecordingID, "document_digest": ref.DocumentDigest, "cue_id": ref.CueID, "local_speaker_id": ref.LocalSpeakerID}
		if ref.SourceDigest != "" {
			member["source_digest"] = ref.SourceDigest
		}
		if ref.SourceMapDigest != "" {
			member["source_map_digest"] = ref.SourceMapDigest
		}
		if ref.MappingRevision > 0 {
			member["mapping_revision"] = ref.MappingRevision
		}
		members = append(members, member)
		sources[ref.RecordingID] = true
	}
	languages := append([]string{}, recipe.Languages...)
	var maxDuration any
	if recipe.MaxDurationUS > 0 {
		maxDuration = recipe.MaxDurationUS
	}
	counts := map[string]int{}
	for _, diagnostic := range summary.Diagnostics {
		count := int(diagnostic.Count)
		if count == 0 {
			count = 1
		}
		counts[diagnostic.Code] += count
	}
	manifest := map[string]any{
		"schema_version": contracts.Version, "kind": "speaker-dataset", "workspace_id": workspaceID, "dataset_snapshot_id": datasetID,
		"speaker_id": speaker.ID, "speaker_identity_revision": speaker.Revision, "catalog_revision": revision, "created_at": portableInstant(time.Now()), "manifest_sha256": "", "state": "current",
		"selection":   map[string]any{"recipe_id": models.StableID("speaker-recipe:" + portableDigest(recipe)), "recipe_revision": 1, "preset": "current-independent", "preset_version": "1", "attribution_bases": []string{"manual"}, "source_asset_ids": []string{}, "languages": languages, "minimum_duration_us": recipe.MinDurationUS, "maximum_duration_us": maxDuration, "explicitly_excluded_segment_ids": []string{}, "options": recipe},
		"preparation": nil, "members": members, "exclusions": []any{},
		"summary": map[string]any{"segment_count": len(refs), "source_asset_count": len(sources), "original_duration_us": summary.DurationUS, "prepared_duration_us": nil, "excluded_segment_count": summary.Excluded, "diagnostic_counts": counts, "languages": languages},
	}
	manifest["manifest_sha256"] = portableDigest(manifest)
	raw, err := json.Marshal(manifest)
	if err != nil || schemas.ValidateSpeakerDocument(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return raw, nil
}

// BuildModelManifest describes immutable output origin and compatible consumers.
// Current dataset validity is rendered independently during inspection/retrieval.
func BuildModelManifest(workspaceID string, work catalog.Work, dataset catalog.SpeakerDataset, options TrainOptions, output Output, published []catalog.Publication) (json.RawMessage, error) {
	if len(published) < len(output.Artifacts) {
		return nil, contracts.Fail("invalid_request")
	}
	familyID := options.FamilyID
	if familyID == "" {
		familyID = models.StableID("speaker-family:" + work.ID)
	}
	versionID := models.StableID("speaker-version:" + work.ID)
	artifacts := []any{}
	formats := []string{}
	seenFormats := map[string]bool{}
	operations := append([]string{}, output.SupportedOperations...)
	if len(operations) == 0 {
		return nil, contracts.Fail("invalid_engine_output")
	}
	for i, a := range output.Artifacts {
		p := published[i]
		if p.State != "available" || p.Digest != a.SHA256 || p.Size != a.Size {
			return nil, contracts.Fail("invalid_request")
		}
		artifacts = append(artifacts, map[string]any{"role": a.Role, "format": a.Format, "artifact": portableArtifact(p), "required_for": operations, "dependencies": []any{}, "extensions": map[string]any{"insonic.publication": map[string]any{"publication_id": p.ID}}})
		if !seenFormats[a.Format] {
			formats = append(formats, a.Format)
			seenFormats[a.Format] = true
		}
	}
	if len(formats) == 0 {
		formats = []string{"provider-handle"}
	}
	consumers := []any{}
	for _, id := range output.Consumers {
		consumers = append(consumers, map[string]any{"adapter_id": id, "contract_version": "1", "implementation_version_range": "1", "accepted_formats": formats})
	}
	outputs := map[string]any{"artifacts": artifacts}
	if output.HostedHandle != "" {
		outputs["hosted_model"] = map[string]any{"provider_id": options.Adapter.ID, "provider_model_id": output.HostedHandle, "downloadable": false, "supported_operations": operations, "retrieval_adapter_id": nil}
	}
	execution := options.Adapter.Mode
	provider := map[string]any{"execution": execution, "provider_id": options.Adapter.ID, "adapter_id": options.Adapter.ID, "contract_version": options.Adapter.ContractVersion, "adapter_version": options.Adapter.ContractVersion}
	if execution == "hosted" {
		provider["endpoint"] = options.Adapter.Endpoint
		if options.Adapter.CredentialID != "" {
			provider["credential_id"] = options.Adapter.CredentialID
		}
	}
	if options.Adapter.Executable.SHA256 != "" {
		provider["implementation_sha256"] = options.Adapter.Executable.SHA256
	}
	var base any
	if options.BaseModelID != "" {
		var revision any
		if options.Adapter.UpstreamRevision != "" {
			revision = options.Adapter.UpstreamRevision
		}
		base = map[string]any{"model_id": options.BaseModelID, "revision": revision, "sha256": options.BaseDigest, "format": options.Adapter.Architecture}
	}
	pipelineID, pipelineRevision := options.PipelineID, options.PipelineRevision
	if pipelineID == "" {
		pipelineID = models.StableID("speaker-adapter-config:" + portableDigest(options.Adapter))
		pipelineRevision = 1
	}
	parameters := options.Parameters
	if len(parameters) == 0 {
		parameters = json.RawMessage(`{}`)
	}
	attemptID := models.StableID("speaker-attempt:" + work.ID + ":" + strconv.FormatInt(work.Generation, 10))
	checkpoints := []any{}
	for i, checkpoint := range output.Checkpoints {
		index := len(output.Artifacts) + i
		if index >= len(published) {
			return nil, contracts.Fail("invalid_request")
		}
		p := published[index]
		if p.Digest != checkpoint.Artifact.SHA256 || p.Size != checkpoint.Artifact.Size || p.State != "available" {
			return nil, contracts.Fail("invalid_request")
		}
		checkpoints = append(checkpoints, map[string]any{"checkpoint_id": checkpoint.ID, "producing_attempt_id": attemptID, "training_step": checkpoint.Step, "format": checkpoint.Artifact.Format, "artifact": portableArtifact(p), "resume_supported": options.Adapter.SupportsResume, "compatible_adapter_id": options.Adapter.ID, "compatible_adapter_version": options.Adapter.ContractVersion, "base_model": base})
	}
	now := portableInstant(time.Now())
	var started, completed any
	if output.StartedAt != nil {
		started = output.StartedAt
	}
	if output.CompletedAt != nil {
		completed = output.CompletedAt
	}
	diagnostics := []any{}
	for _, diagnostic := range output.Diagnostics {
		details := map[string]any{"count": diagnostic.Count}
		if diagnostic.Value != nil {
			details["value"] = *diagnostic.Value
		}
		diagnostics = append(diagnostics, map[string]any{"code": diagnostic.Code, "severity": "info", "message": strings.ReplaceAll(diagnostic.Code, "_", " "), "method": options.Adapter.ID, "method_version": options.Adapter.ContractVersion, "details": details})
	}
	var sampleRate, channels any
	if options.Adapter.ID == "pyannote-profile" {
		sampleRate, channels = 16000, 1
	}
	var preparation any
	if p := output.PreparationPublication; p != nil {
		if p.State != "available" || p.Size < 1 {
			return nil, contracts.Fail("invalid_request")
		}
		preparation = map[string]any{"preparation_manifest_id": p.ID, "manifest_sha256": p.Digest, "artifact": portableArtifact(*p), "input_artifact_ids": []string{}, "adapter_id": "insonic-canonical-audio", "adapter_version": "1", "options": map[string]any{"recipe_sha256": portableDigest(json.RawMessage(dataset.Recipe)), "sample_rate_hz": 16000, "channel_count": 1, "normalize": false}}
	}
	manifest := map[string]any{
		"schema_version": contracts.Version, "kind": "speaker-model", "workspace_id": workspaceID, "model_family_id": familyID, "model_version_id": versionID, "originating_speaker_id": dataset.Dataset.SpeakerID, "originating_speaker_identity_revision": dataset.SpeakerRevision, "created_at": now, "manifest_sha256": "", "display_name": options.Name, "state": "current",
		"training": map[string]any{"training_run_id": catalog.SpeakerTrainingRunID(work.ID), "job_id": work.ID, "producing_attempt_id": attemptID, "pipeline_id": pipelineID, "pipeline_revision": pipelineRevision, "dataset": map[string]any{"dataset_snapshot_id": dataset.Dataset.ID, "manifest_sha256": dataset.ManifestDigest, "originating_speaker_id": dataset.Dataset.SpeakerID, "speaker_identity_revision": dataset.SpeakerRevision, "state": "current"}, "preparation": preparation, "provider": provider, "base_model": base, "parameters": parameters, "started_at": started, "completed_at": completed},
		"outputs":  outputs, "compatibility": map[string]any{"model_kind": output.Kind, "architecture": output.Architecture, "supported_operations": operations, "consumers": consumers, "runtime_dependencies": []any{}, "sample_rate_hz": sampleRate, "channel_count": channels, "languages": []string{}},
		"checkpoints": checkpoints, "license_declarations": []any{map[string]any{"scope": "output", "declared_license": nil, "attribution": nil, "source_basis": "configured-adapter"}}, "evaluations": []any{}, "diagnostics": diagnostics,
	}
	if options.Adapter.License != "" {
		manifest["license_declarations"].([]any)[0].(map[string]any)["declared_license"] = options.Adapter.License
	}
	manifest["manifest_sha256"] = portableDigest(manifest)
	raw, err := json.Marshal(manifest)
	if err != nil || schemas.ValidateSpeakerDocument(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return raw, nil
}

type ManifestArtifactDescriptor struct {
	Role          string
	Format        string
	ArtifactID    string
	PublicationID string
	Digest        string
	Size          int64
}

func ManifestArtifactDescriptors(raw json.RawMessage) ([]ManifestArtifactDescriptor, error) {
	if catalog.ValidateJSON(raw) != nil || schemas.ValidateSpeakerDocument(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	var m struct {
		Outputs struct {
			Artifacts []struct {
				Role     string `json:"role"`
				Format   string `json:"format"`
				Artifact struct {
					ID     string `json:"artifact_id"`
					Digest string `json:"sha256"`
					Size   int64  `json:"byte_length"`
				} `json:"artifact"`
				Extensions map[string]struct {
					PublicationID string `json:"publication_id"`
				} `json:"extensions"`
			} `json:"artifacts"`
		} `json:"outputs"`
	}
	if json.Unmarshal(raw, &m) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	out := []ManifestArtifactDescriptor{}
	for _, a := range m.Outputs.Artifacts {
		out = append(out, ManifestArtifactDescriptor{a.Role, a.Format, a.Artifact.ID, a.Extensions["insonic.publication"].PublicationID, a.Artifact.Digest, a.Artifact.Size})
	}
	return out, nil
}
