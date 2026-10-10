// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shruggietech/insonic/internal/artifact"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/models"
)

func portableTrainingTarget(t *testing.T, source *Service) (*Service, *catalog.Store) {
	t.Helper()
	ctx := context.Background()
	snapshot, err := source.Catalog.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = catalog.PortableSnapshot(ctx, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	db, err := catalog.OpenSQLite(ctx, filepath.Join(t.TempDir(), "restored.sqlite"), snapshot.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Restore(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	// Immutable artifact relocation is covered by backup service fixtures. Keep
	// the byte store here to isolate successive native catalog/work relocation.
	assets, err := artifact.NewService(ctx, source.Artifacts.Workspace, db, nil, contracts.ID())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { assets.Close(); db.Close() })
	target := NewService(assets, db, nil)
	target.Prepare = source.Prepare
	target.LocalRunner = source.LocalRunner
	return target, db
}

func portableTrainingInvocation(t *testing.T, service *Service, db *catalog.Store, id string, options TrainOptions, targetPath string) catalog.Work {
	t.Helper()
	stored, err := db.Work(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	portable, digest, err := db.PortableWorkInput(context.Background(), stored)
	if err != nil || !portable || digest == "" {
		t.Fatal("native adapter identity unavailable", err)
	}
	service.PortableAdapterDigest = digest
	claim, err := db.ClaimWork(context.Background(), id, contracts.ID(), time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	options.Adapter.Executable.Path = targetPath
	claim.Payload, _ = json.Marshal(TrainPayload{Options: options})
	return claim
}

func TestPortableTrainingCheckpointSurvivesSuccessiveRestores(t *testing.T) {
	ctx := context.Background()
	source, originalDB := fixture(t)
	dataset, _ := datasetFixture(t, source, originalDB, 2)
	options := TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Portable checkpoint", Kind: "weights", Adapter: Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "fixture", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Executable: libraryPin(), SupportsResume: true}, Parameters: json.RawMessage(`{"epochs":2}`)}
	var err error
	options, err = source.ValidateTrain(ctx, options)
	if err != nil {
		t.Fatal(err)
	}
	producerID := contracts.ID()
	checkpointID := models.StableID("speaker-checkpoint:" + producerID + ":1:0")
	raw, _ := json.Marshal(TrainPayload{Options: options})
	if _, err = originalDB.EnqueueWork(ctx, producerID, "models.train", raw); err != nil {
		t.Fatal(err)
	}
	resumeOptions := options
	resumeOptions.CheckpointID = checkpointID
	resumeRaw, _ := json.Marshal(TrainPayload{Options: resumeOptions})
	resumeID := contracts.ID()
	if _, err = originalDB.EnqueueWork(ctx, resumeID, "models.train", resumeRaw); err != nil {
		t.Fatal(err)
	}
	checkpointBytes := []byte("portable optimizer state")
	source.LocalRunner = func(_ context.Context, _ Adapter, request AdapterRequest) (Output, error) {
		output := Output{ContractVersion: "1", Kind: "weights", Architecture: "fixture", Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}}
		if request.Operation == "train" {
			output.State = "checkpointed"
			output.Checkpoints = []Checkpoint{{Step: 1, Compatibility: "optimizer-v1", Artifact: Artifact{Role: "optimizer.bin", Format: "binary", Data: checkpointBytes, SHA256: hash(checkpointBytes), Size: int64(len(checkpointBytes))}}}
			return output, nil
		}
		if request.Operation != "resume" || request.Checkpoint == nil || request.Checkpoint.ID != checkpointID {
			t.Fatal("frozen checkpoint not resumed", request.Operation)
		}
		bytes, err := os.ReadFile(request.Checkpoint.Artifact.Path)
		if err != nil || string(bytes) != string(checkpointBytes) {
			t.Fatal("checkpoint bytes changed", err)
		}
		weights := []byte("finished portable weights")
		output.Artifacts = []Artifact{{Role: "weights.bin", Format: "binary", Data: weights, SHA256: hash(weights), Size: int64(len(weights))}}
		return output, nil
	}
	first, firstDB := portableTrainingTarget(t, source)
	firstPath := filepath.Join(t.TempDir(), "first-destination-adapter")
	claim := portableTrainingInvocation(t, first, firstDB, producerID, options, firstPath)
	if result, err := first.ExecuteTrain(ctx, claim); err == nil || result == nil {
		t.Fatal("checkpoint-only training did not checkpoint", result, err)
	}
	// A newly elected destination resume has no portable-origin proof. It must
	// still accept identical pinned bytes while rejecting changed settings.
	first.PortableAdapterDigest = ""
	localResume := resumeOptions
	localResume.Adapter.Executable.Path = firstPath
	if _, err = first.ValidateTrain(ctx, localResume); err != nil {
		t.Fatal("new destination election cannot resume portable checkpoint", err)
	}
	changed := localResume
	changed.Adapter.Arguments = []string{"--different-recipe"}
	if _, err = first.ValidateTrain(ctx, changed); err == nil {
		t.Fatal("changed adapter settings accepted checkpoint")
	}
	second, secondDB := portableTrainingTarget(t, first)
	secondPath := filepath.Join(t.TempDir(), "second-destination-adapter")
	claim = portableTrainingInvocation(t, second, secondDB, resumeID, resumeOptions, secondPath)
	result, err := second.ExecuteTrain(ctx, claim)
	if err != nil || result == nil {
		t.Fatal("second relocation cannot resume proven checkpoint", result, err)
	}
	checkpoint, err := secondDB.SpeakerCheckpoint(ctx, checkpointID)
	if err != nil || checkpoint.AdapterDigest != checkpointAdapterDigest(options.Adapter) {
		t.Fatal("checkpoint identity depends on destination path", checkpoint.AdapterDigest, err)
	}
	original, err := originalDB.Work(ctx, producerID)
	if err != nil || original.State != "pending" || string(original.Payload) != string(raw) {
		t.Fatal("source work changed", err)
	}
}
