// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestDatasetCreateReplayPreservesAcceptedNonemptyRecipeAfterInvalidation(t *testing.T) {
	service, db := fixture(t)
	initial, speaker := datasetFixture(t, service, db, 1)
	ctx := context.Background()
	operation := contracts.ID()
	options := DatasetOptions{SpeakerID: speaker.ID, Recipe: Recipe{MinDurationUS: 1, MaxDurationUS: 10000, ExcludeOverlap: true}}
	original, err := service.CreateDataset(ctx, operation, options)
	if err != nil {
		t.Fatal(err)
	}
	references := initial.References
	mappings, err := db.SpeakerMappings(ctx, references[0].RecordingID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.SetSpeakerMapping(ctx, contracts.ID(), references[0].RecordingRevision, mappings[0]); err != nil {
		t.Fatal(err)
	}
	invalidated, err := db.SpeakerDataset(ctx, original.Dataset.ID)
	if err != nil || invalidated.Dataset.State == "current" || len(invalidated.References) != 0 || string(invalidated.Recipe) != "{}" {
		t.Fatal("assignment-bearing recipe remained current", invalidated, err)
	}
	replay, err := service.CreateDataset(ctx, operation, options)
	if err != nil || replay.Dataset.ID != original.Dataset.ID || replay.Dataset.State != invalidated.Dataset.State || len(replay.References) != 0 || string(replay.Recipe) != "{}" {
		t.Fatal("accepted dataset replay changed or failed", replay, err)
	}
	changed := options
	changed.Recipe.MinDurationUS = 2
	if _, err = service.CreateDataset(ctx, operation, changed); errorCode(err) != "conflict" {
		t.Fatal("changed recipe accepted", err)
	}
	another, err := db.PutSpeaker(ctx, contracts.ID(), 0, catalog.SpeakerIdentity{Speaker: catalog.Speaker{ID: contracts.ID(), Name: "Another identity"}})
	if err != nil {
		t.Fatal(err)
	}
	changed = options
	changed.SpeakerID = another.Speaker.ID
	if _, err = service.CreateDataset(ctx, operation, changed); errorCode(err) != "conflict" {
		t.Fatal("changed speaker accepted", err)
	}
	if datasets, err := db.SpeakerDatasets(ctx, another.Speaker.ID); err != nil || len(datasets) != 0 {
		t.Fatal("changed intent created dataset", datasets, err)
	}
}

func errorCode(err error) string {
	if failure, ok := err.(*contracts.Error); ok {
		return failure.Code
	}
	return ""
}
