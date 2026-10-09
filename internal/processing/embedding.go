// SPDX-License-Identifier: Apache-2.0
package processing

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"math"
	"os"
)

type EmbeddingInput struct {
	Path      string
	SourceMap SourceMap
}

// EmbedBatch loads one elected bundle for a bounded group of exact clips.
func (s *Service) EmbedBatch(ctx context.Context, inputs []EmbeddingInput, modelID, digest string) ([][]float64, error) {
	if inEngineCI() {
		return nil, contracts.Fail("engine_ci_forbidden")
	}
	if len(inputs) < 1 || len(inputs) > 64 {
		return nil, contracts.Fail("input_limit")
	}
	paths := make([]string, len(inputs))
	for i, input := range inputs {
		if err := ValidateMappedAudio(input.Path, input.SourceMap); err != nil {
			return nil, err
		}
		paths[i] = input.Path
	}
	directory, err := os.MkdirTemp(s.Artifacts.Workspace.Control, "speaker-embedding-")
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	if err = workspace.SecureDirectory(directory, true); err != nil {
		return nil, err
	}
	session := &Session{MappedPath: inputs[0].Path, SourceMap: inputs[0].SourceMap, service: s, directory: directory}
	raw, _, err := session.workerSelected(ctx, "embed-batch", modelID, map[string]any{"device": "cpu", "audio_paths": paths}, nil, digest)
	if err != nil {
		return nil, err
	}
	var out struct {
		Vectors     [][]float64    `json:"vectors"`
		Provenance  map[string]any `json:"provenance"`
		Diagnostics []Diagnostic   `json:"diagnostics"`
	}
	if strict(raw, &out) != nil || len(out.Vectors) != len(inputs) || !validProvenance(out.Provenance) || !validDiagnostics(out.Diagnostics) {
		return nil, contracts.Fail("invalid_engine_output")
	}
	skipped := int64(0)
	for _, vector := range out.Vectors {
		// A null batch member is an explicitly diagnosed short source interval.
		if vector == nil {
			skipped++
			continue
		}
		if len(vector) < 1 || len(vector) > 4096 {
			return nil, contracts.Fail("invalid_embedding")
		}
		norm := 0.0
		for _, v := range vector {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, contracts.Fail("invalid_embedding")
			}
			norm += v * v
		}
		if math.IsInf(norm, 0) || norm < 1e-24 {
			return nil, contracts.Fail("invalid_embedding")
		}
	}
	var diagnosed int64
	for _, diagnostic := range out.Diagnostics {
		if diagnostic.Code == "embedding_short_clip_excluded" {
			diagnosed += diagnostic.Count
		}
	}
	if diagnosed != skipped {
		return nil, contracts.Fail("invalid_engine_output")
	}
	return out.Vectors, nil
}

// Embed computes one acoustic embedding from an exact prepared audio clip.
// It loads the same elected offline pyannote bundle as diarization, but does
// not infer activity or change its caller's accepted intervals.
func (s *Service) Embed(ctx context.Context, path string, mapping SourceMap, modelID, digest string) ([]float64, error) {
	if inEngineCI() {
		return nil, contracts.Fail("engine_ci_forbidden")
	}
	if err := ValidateMappedAudio(path, mapping); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp(s.Artifacts.Workspace.Control, "speaker-embedding-")
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer os.RemoveAll(directory)
	if err = workspace.SecureDirectory(directory, true); err != nil {
		return nil, err
	}
	session := &Session{MappedPath: path, SourceMap: mapping, service: s, directory: directory}
	raw, _, err := session.workerSelected(ctx, "embed", modelID, map[string]any{"device": "cpu"}, nil, digest)
	if err != nil {
		return nil, err
	}
	var output struct {
		Vector      []float64      `json:"vector"`
		Provenance  map[string]any `json:"provenance"`
		Diagnostics []Diagnostic   `json:"diagnostics"`
	}
	if strict(raw, &output) != nil || len(output.Vector) < 1 || len(output.Vector) > 4096 || !validProvenance(output.Provenance) || !validDiagnostics(output.Diagnostics) {
		return nil, contracts.Fail("invalid_engine_output")
	}
	norm := 0.0
	for _, v := range output.Vector {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, contracts.Fail("invalid_embedding")
		}
		norm += v * v
	}
	if math.IsInf(norm, 0) || norm < 1e-24 {
		return nil, contracts.Fail("invalid_embedding")
	}
	return output.Vector, nil
}
func inEngineCI() bool {
	for _, name := range []string{"CI", "GITHUB_ACTIONS", "TF_BUILD", "BUILD_BUILDID", "JENKINS_URL"} {
		if os.Getenv(name) != "" {
			return true
		}
	}
	return false
}
