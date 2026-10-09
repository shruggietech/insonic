// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestSavedTrainingElectionRequiresExplicitPositiveRevisionBeforeCatalogRead(t *testing.T) {
	service := &Service{}
	for _, revision := range []int64{0, -1} {
		_, err := service.ResolveTrainingPipeline(context.Background(), TrainOptions{PipelineID: contracts.ID(), PipelineRevision: revision})
		if failure, ok := err.(*contracts.Error); !ok || failure.Code != "invalid_request" {
			t.Fatalf("mutable training pipeline accepted revision %d: %v", revision, err)
		}
	}
}
