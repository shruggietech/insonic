// SPDX-License-Identifier: Apache-2.0
// Package voicemodels executes explicitly elected speaker training and matching.
package voicemodels

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/processing"
)

type Limits struct {
	MaxInputBytes  int64 `json:"max_input_bytes,omitempty"`
	MaxOutputBytes int64 `json:"max_output_bytes,omitempty"`
	TimeoutMS      int64 `json:"timeout_ms,omitempty"`
}
type Adapter struct {
	ID                  string               `json:"id"`
	ContractVersion     string               `json:"contract_version"`
	Mode                string               `json:"mode"`
	Architecture        string               `json:"architecture"`
	OutputKinds         []string             `json:"output_kinds"`
	Consumers           []string             `json:"consumers"`
	SupportedOperations []string             `json:"supported_operations,omitempty"`
	NeedsText           bool                 `json:"needs_text,omitempty"`
	SupportsResume      bool                 `json:"supports_resume,omitempty"`
	SupportsCancel      bool                 `json:"supports_cancel,omitempty"`
	Executable          library.PinnedFile   `json:"executable,omitzero"`
	Arguments           []string             `json:"arguments,omitempty"`
	SupportFiles        []library.PinnedFile `json:"support_files,omitempty"`
	Endpoint            string               `json:"endpoint,omitempty"`
	RemoteModel         string               `json:"remote_model,omitempty"`
	UpstreamRevision    string               `json:"upstream_revision,omitempty"`
	CredentialID        string               `json:"credential_id,omitempty"`
	License             string               `json:"license,omitempty"`
	Limits              Limits               `json:"limits,omitzero"`
}
type Recipe struct {
	RecordingID         string   `json:"recording_id,omitempty"`
	RecordingIDs        []string `json:"recording_ids,omitempty"`
	Languages           []string `json:"languages,omitempty"`
	Channel             *int     `json:"channel,omitempty"`
	ExcludedReferences  []string `json:"excluded_references,omitempty"`
	MinRMS              *float64 `json:"min_rms,omitempty"`
	MaxClippingFraction *float64 `json:"max_clipping_fraction,omitempty"`
	MinDurationUS       int64    `json:"min_duration_us,omitempty"`
	MaxDurationUS       int64    `json:"max_duration_us,omitempty"`
	ExcludeOverlap      bool     `json:"exclude_overlap,omitempty"`
}
type DatasetOptions struct {
	SpeakerID string `json:"speaker_id"`
	Recipe    Recipe `json:"recipe,omitzero"`
}
type TrainOptions struct {
	DatasetID        string          `json:"dataset_id"`
	FamilyID         string          `json:"family_id,omitempty"`
	Name             string          `json:"name"`
	Kind             string          `json:"output_kind"`
	Adapter          Adapter         `json:"adapter"`
	BaseModelID      string          `json:"base_model_id,omitempty"`
	BaseDigest       string          `json:"base_digest,omitempty"`
	PipelineID       string          `json:"pipeline_id,omitempty"`
	PipelineRevision int64           `json:"pipeline_revision,omitempty"`
	Parameters       json.RawMessage `json:"parameters,omitempty"`
	CheckpointID     string          `json:"checkpoint_id,omitempty"`
}
type TrainingConfiguration struct {
	Adapter     Adapter         `json:"adapter"`
	BaseModelID string          `json:"base_model_id,omitempty"`
	Kind        string          `json:"output_kind"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}
type MatchOptions struct {
	RecordingID   string  `json:"recording_id"`
	ModelID       string  `json:"model_id"`
	ModelDigest   string  `json:"model_digest,omitempty"`
	Adapter       Adapter `json:"adapter"`
	Threshold     float64 `json:"threshold"`
	Margin        float64 `json:"ambiguity_margin"`
	MinEvidenceUS int64   `json:"min_evidence_us"`
}
type TrainPayload struct {
	ProcessingTools   json.RawMessage     `json:"processing_tools,omitempty"`
	RequestDigest     string              `json:"request_digest,omitempty"`
	Options           TrainOptions        `json:"options"`
	ModelSelections   []models.Resolution `json:"model_selections,omitempty"`
	ModelDependencies []string            `json:"model_dependencies,omitempty"`
}
type MatchPayload struct {
	ProcessingTools   json.RawMessage                 `json:"processing_tools,omitempty"`
	RequestDigest     string                          `json:"request_digest,omitempty"`
	Options           MatchOptions                    `json:"options"`
	Snapshot          catalog.SpeakerMatchingSnapshot `json:"snapshot"`
	ModelSelections   []models.Resolution             `json:"model_selections,omitempty"`
	ModelDependencies []string                        `json:"model_dependencies,omitempty"`
}
type Summary struct {
	References  int                     `json:"references"`
	Included    int                     `json:"included"`
	Excluded    int                     `json:"excluded"`
	DurationUS  int64                   `json:"duration_us"`
	Diagnostics []processing.Diagnostic `json:"diagnostics"`
}
type Audio struct {
	Path      string
	SourceMap processing.SourceMap
	Close     func() error
}
type Service struct {
	Catalog     catalog.Catalog
	Artifacts   *artifact.Service
	Secrets     contracts.SecretProvider
	Client      *http.Client
	Prepare     func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*Audio, error)
	Embed       func(context.Context, string, processing.SourceMap, string, string) ([]float64, error)
	EmbedBatch  func(context.Context, []PreparedInput, string, string) ([][]float64, error)
	LocalRunner func(context.Context, Adapter, AdapterRequest) (Output, error)
}

func NewService(artifacts *artifact.Service, db catalog.Catalog, secrets contracts.SecretProvider) *Service {
	return &Service{Catalog: db, Artifacts: artifacts, Secrets: secrets}
}

type Artifact struct {
	Role   string `json:"role"`
	Format string `json:"format"`
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Data   []byte `json:"data,omitempty"`
}
type Checkpoint struct {
	ID            string   `json:"id"`
	Step          int64    `json:"step"`
	BaseDigest    string   `json:"base_digest,omitempty"`
	Compatibility string   `json:"compatibility"`
	Artifact      Artifact `json:"artifact"`
}
type Output struct {
	PreparationPublication *catalog.Publication    `json:"-"`
	State                  string                  `json:"state,omitempty"`
	StartedAt              *catalog.Instant        `json:"-"`
	CompletedAt            *catalog.Instant        `json:"-"`
	ContractVersion        string                  `json:"contract_version"`
	Kind                   string                  `json:"kind"`
	Architecture           string                  `json:"architecture"`
	Consumers              []string                `json:"consumers"`
	SupportedOperations    []string                `json:"supported_operations"`
	Artifacts              []Artifact              `json:"artifacts"`
	HostedHandle           string                  `json:"hosted_handle,omitempty"`
	Checkpoints            []Checkpoint            `json:"checkpoints,omitempty"`
	Diagnostics            []processing.Diagnostic `json:"diagnostics,omitempty"`
}
type Profile struct {
	Kind        string    `json:"kind"`
	ModelID     string    `json:"model_id"`
	ModelDigest string    `json:"model_digest"`
	Vector      []float64 `json:"vector"`
	EvidenceUS  int64     `json:"evidence_us"`
}
type PreparedInput struct {
	ID                       string                    `json:"id"`
	Path                     string                    `json:"path,omitempty"`
	SHA256                   string                    `json:"sha256"`
	Size                     int64                     `json:"size"`
	DurationUS               int64                     `json:"duration_us"`
	SourceMap                processing.SourceMap      `json:"source_map"`
	SourceReference          *catalog.CurrentReference `json:"source_reference,omitempty"`
	OriginalStartNumerator   string                    `json:"original_start_numerator,omitempty"`
	OriginalStartDenominator string                    `json:"original_start_denominator,omitempty"`
	Text                     string                    `json:"text,omitempty"`
}
type AdapterRequest struct {
	ContractVersion    string          `json:"contract_version"`
	Operation          string          `json:"operation"`
	WorkID             string          `json:"work_id"`
	Attempt            int64           `json:"attempt"`
	BaseModelID        string          `json:"base_model_id,omitempty"`
	BaseDigest         string          `json:"base_digest,omitempty"`
	BaseModelDirectory string          `json:"base_model_directory,omitempty"`
	BaseFiles          []PreparedInput `json:"base_files,omitempty"`
	Parameters         json.RawMessage `json:"parameters"`
	Inputs             []PreparedInput `json:"inputs"`
	OutputDirectory    string          `json:"output_directory,omitempty"`
	Checkpoint         *Checkpoint     `json:"checkpoint,omitempty"`
	RemoteModel        string          `json:"remote_model,omitempty"`
	UpstreamRevision   string          `json:"upstream_revision,omitempty"`
}
