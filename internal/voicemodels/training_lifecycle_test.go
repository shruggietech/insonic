// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCheckpointOnlyTrainingResumesExactPreparedElection(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 2)
	checkpointBytes := []byte("optimizer state and architecture weights")
	adapter := Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "local", Architecture: "custom", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Executable: libraryPin(), SupportsResume: true}
	options := TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Actual generic protocol", Kind: "weights", Adapter: adapter, Parameters: json.RawMessage(`{"epochs":2}`)}
	calls := 0
	s.LocalRunner = func(ctx context.Context, a Adapter, request AdapterRequest) (Output, error) {
		calls++
		if len(request.Inputs) != 2 || request.Inputs[0].SourceReference == nil || request.Inputs[0].OriginalStartDenominator == "" || request.Inputs[0].Text != "" {
			t.Fatal("prepared election missing", request)
		}
		out := Output{ContractVersion: "1", Kind: "weights", Architecture: "custom", Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}}
		if request.Operation == "train" {
			out.State = "checkpointed"
			out.Checkpoints = []Checkpoint{{Step: 1, Compatibility: "optimizer-v1", Artifact: Artifact{Role: "optimizer.bin", Format: "binary", Data: checkpointBytes, SHA256: hash(checkpointBytes), Size: int64(len(checkpointBytes))}}}
			return out, nil
		}
		if request.Operation != "resume" || request.Checkpoint == nil {
			t.Fatal("checkpoint not resumed", request.Operation)
		}
		raw, err := os.ReadFile(request.Checkpoint.Artifact.Path)
		if err != nil || string(raw) != string(checkpointBytes) {
			t.Fatal("exact checkpoint unavailable", err)
		}
		weights := []byte("trained output after second epoch")
		out.Artifacts = []Artifact{{Role: "weights.bin", Format: "binary", Data: weights, SHA256: hash(weights), Size: int64(len(weights))}}
		return out, nil
	}
	claim := work(t, db, "models.train", TrainPayload{Options: options})
	result, err := s.ExecuteTrain(context.Background(), claim)
	if err == nil || result == nil {
		t.Fatal("checkpoint-only output reported completed", result, err)
	}
	stored, err := db.Work(context.Background(), claim.ID)
	if err != nil || stored.State != "failed" || stored.Phase != "checkpointed" {
		t.Fatal("checkpoint progress lost", stored, err)
	}
	ids := result.(map[string]any)["checkpoint_ids"].([]string)
	if len(ids) != 1 {
		t.Fatal(ids)
	}
	outputs, err := s.List(context.Background(), dataset.Dataset.SpeakerID)
	if err != nil || len(outputs) != 0 {
		t.Fatal("partial run published model", outputs, err)
	}
	options.CheckpointID = ids[0]
	wrong := options
	wrong.Parameters = json.RawMessage(`{"epochs":3}`)
	if _, err = s.ValidateTrain(context.Background(), wrong); err == nil {
		t.Fatal("resume changed training parameters")
	}
	result, err = s.ExecuteTrain(context.Background(), work(t, db, "models.train", TrainPayload{Options: options}))
	if err != nil || calls != 2 {
		t.Fatal("resume", result, err, calls)
	}
	output, err := s.Show(context.Background(), result.(map[string]any)["version_id"].(string))
	if err != nil || !strings.Contains(string(output.Metadata), "custom-inference") {
		t.Fatal("actual operation contract lost", err)
	}
}

func TestPreparationDiagnosesUnusableAudioWithoutPublishingInventedInputs(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 2)
	s.Prepare = func(context.Context, catalog.LibraryEntry, processing.AudioOptions) (*Audio, error) {
		return nil, contracts.Fail("invalid_audio")
	}
	directory, err := s.scratch()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(directory)
	inputs, summary, err := s.prepare(context.Background(), dataset, TrainOptions{Adapter: builtin()}, directory)
	if err == nil || len(inputs) != 0 || summary.Included != 0 || summary.Excluded != 2 || summary.DurationUS != 0 || len(summary.Diagnostics) != 1 || summary.Diagnostics[0].Code != "unusable_source_excluded" {
		t.Fatal(inputs, summary, err)
	}
}

func TestHostedCancellationUsesOnlySelectedEndpointAndWorkIdentity(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 1)
	started := make(chan struct{})
	cancelled := make(chan struct{}, 1)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]json.RawMessage
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("bad elected request")
			return
		}
		var operation string
		json.Unmarshal(request["operation"], &operation)
		if operation == "train" {
			calls.Add(1)
			close(started)
			<-r.Context().Done()
			return
		}
		if operation != "cancel" || request["work_id"] == nil || string(request["uploads"]) != "[]" {
			t.Error("cancel leaked prepared data or identity", operation, string(request["uploads"]))
		}
		cancelled <- struct{}{}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()
	s.Client = server.Client()
	options := TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Cancellable", Kind: "weights", Adapter: Adapter{ID: "insonic-speaker", ContractVersion: "1", Mode: "hosted", Architecture: "custom", OutputKinds: []string{"weights"}, Consumers: []string{"custom"}, SupportedOperations: []string{"custom-inference"}, Endpoint: server.URL, RemoteModel: "elected", UpstreamRevision: "immutable", SupportsCancel: true}}
	claim := work(t, db, "models.train", TrainPayload{Options: options})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := s.ExecuteTrain(ctx, claim); done <- err }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("training endpoint never started")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancel reported success")
		}
	case <-time.After(8 * time.Second):
		t.Fatal("cancel did not terminate work")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("declared provider cancel unsupported")
	}
	if calls.Load() != 1 {
		t.Fatal("unexpected route retry", calls.Load())
	}
}

func TestBatchEnrollmentLoadsCompatibleInputsInBoundedGroups(t *testing.T) {
	s, db := fixture(t)
	dataset, _ := datasetFixture(t, s, db, 105)
	groups := []int{}
	s.EmbedBatch = func(_ context.Context, inputs []PreparedInput, _, _ string) ([][]float64, error) {
		groups = append(groups, len(inputs))
		vectors := make([][]float64, len(inputs))
		for i := range inputs {
			vectors[i] = []float64{1, 0}
		}
		return vectors, nil
	}
	options := TrainOptions{DatasetID: dataset.Dataset.ID, Name: "Batched", Kind: "voice-embedding", Adapter: builtin(), BaseModelID: contracts.ID(), BaseDigest: strings.Repeat("b", 64)}
	if _, err := s.ExecuteTrain(context.Background(), work(t, db, "models.train", TrainPayload{Options: options})); err != nil {
		t.Fatal(err)
	}
	if len(groups) != 2 || groups[0] != 64 || groups[1] != 41 {
		t.Fatal("model initialized per clip", groups)
	}
}
