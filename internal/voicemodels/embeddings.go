// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
)

func (s *Service) embedInputs(ctx context.Context, inputs []PreparedInput, modelID, digest string) ([][]float64, error) {
	if len(inputs) > 64 {
		return nil, contracts.Fail("input_limit")
	}
	if s.EmbedBatch != nil {
		out, err := s.EmbedBatch(ctx, inputs, modelID, digest)
		if err != nil {
			return nil, err
		}
		if len(out) != len(inputs) {
			return nil, contracts.Fail("invalid_engine_output")
		}
		return out, nil
	}
	if s.Embed == nil {
		return nil, contracts.Fail("unavailable")
	}
	out := make([][]float64, 0, len(inputs))
	for _, input := range inputs {
		vector, err := s.Embed(ctx, input.Path, input.SourceMap, modelID, digest)
		if err != nil {
			return nil, err
		}
		out = append(out, vector)
	}
	return out, nil
}

func (s *Service) embedInputsWithPolicy(ctx context.Context, inputs []PreparedInput, modelID, digest, device string) (processing.EmbeddingResult, error) {
	if !processing.ValidDevice(device) || len(inputs) < 1 || len(inputs) > 64 {
		return processing.EmbeddingResult{}, contracts.Fail("invalid_request")
	}
	if device == "" {
		device = "cpu"
	}
	if s.EmbedBatchWithPolicy == nil {
		vectors, err := s.embedInputs(ctx, inputs, modelID, digest)
		return processing.EmbeddingResult{Vectors: vectors}, err
	}
	out, err := s.EmbedBatchWithPolicy(ctx, inputs, modelID, digest, device)
	if err != nil {
		return out, err
	}
	if len(out.Vectors) != len(inputs) || len(out.Executions) != 1 || !out.Executions[0].Valid() || out.Executions[0].Count != 1 || out.Executions[0].RequestedDevice != device {
		return processing.EmbeddingResult{}, contracts.Fail("invalid_engine_output")
	}
	return out, nil
}
