// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
)

func ValidateTrainingConfiguration(raw json.RawMessage) error {
	var c TrainingConfiguration
	if strict(raw, &c) != nil || ValidateAdapter(c.Adapter) != nil || !contains(c.Adapter.OutputKinds, c.Kind) || c.BaseModelID != "" && !models.ValidateReference(c.BaseModelID) || c.Adapter.ID == "pyannote-profile" && c.BaseModelID == "" {
		return contracts.Fail("invalid_request")
	}
	if len(c.Parameters) > 0 {
		var params map[string]json.RawMessage
		if strict(c.Parameters, &params) != nil || len(c.Parameters) > 65536 {
			return contracts.Fail("invalid_request")
		}
	}
	return nil
}

// ResolveTrainingPipeline runs only at admission. Execution uses the elected
// adapter, parameters and exact base identity already frozen in its payload.
func (s *Service) ResolveTrainingPipeline(ctx context.Context, o TrainOptions) (TrainOptions, error) {
	if o.PipelineID == "" {
		if o.PipelineRevision != 0 {
			return o, contracts.Fail("invalid_request")
		}
		return o, nil
	}
	if !contracts.ValidID(o.PipelineID) || o.PipelineRevision < 1 {
		return o, contracts.Fail("invalid_request")
	}
	saved, err := s.Catalog.Pipeline(ctx, o.PipelineID)
	if err != nil {
		return o, err
	}
	if saved.Revision != o.PipelineRevision {
		return o, contracts.Fail("conflict")
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(saved.Configuration, &fields) != nil {
		return o, contracts.Fail("unsupported_capability")
	}
	if s.BindPortableConfiguration != nil {
		bound, e := s.BindPortableConfiguration(fields["speaker_training"])
		if e != nil {
			return o, e
		}
		fields["speaker_training"] = bound
	}
	if ValidateTrainingConfiguration(fields["speaker_training"]) != nil {
		return o, contracts.Fail("unsupported_capability")
	}
	var c TrainingConfiguration
	_ = json.Unmarshal(fields["speaker_training"], &c)
	if o.Adapter.ID != "" && adapterDigest(o.Adapter) != adapterDigest(c.Adapter) || o.BaseModelID != "" && o.BaseModelID != c.BaseModelID || o.Kind != "" && o.Kind != c.Kind || len(o.Parameters) > 0 && hash(o.Parameters) != hash(c.Parameters) {
		return o, contracts.Fail("conflict")
	}
	o.Adapter = c.Adapter
	o.BaseModelID = c.BaseModelID
	o.Kind = c.Kind
	o.Parameters = c.Parameters
	o.PipelineRevision = saved.Revision
	return o, nil
}
