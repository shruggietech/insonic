// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
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
