// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/processing"
)

func TestSpeakerEnrollmentAndMatchingRetainDeviceExecution(t *testing.T) {
	for _, policy := range []string{"", "cpu", "auto", "cuda"} {
		t.Run(policy, func(t *testing.T) {
			s, db := fixture(t)
			ctx := context.Background()
			dataset, speaker := datasetFixture(t, s, db, 65)
			base, digest := contracts.ID(), strings.Repeat("b", 64)
			want := policy
			if want == "" {
				want = "cpu"
			}
			calls := 0
			s.EmbedBatchWithPolicy = func(_ context.Context, inputs []PreparedInput, id, hash, device string) (processing.EmbeddingResult, error) {
				calls++
				if id != base || hash != digest || device != want || len(inputs) < 1 || len(inputs) > 64 {
					t.Fatalf("election changed: %s %s %s %d", id, hash, device, len(inputs))
				}
				observed := processing.DeviceExecution{RequestedDevice: device, AttemptedDevices: []string{"cpu"}, Device: "cpu", Count: 1}
				if device != "cpu" {
					observed.AttemptedDevices = []string{"cuda", "cpu"}
					observed.FallbackReason = "cuda_resource_exhausted"
				}
				out := processing.EmbeddingResult{Vectors: make([][]float64, len(inputs)), Executions: []processing.DeviceExecution{observed}}
				for i := range inputs {
					out.Vectors[i] = []float64{1, 0}
				}
				return out, nil
			}
			options := TrainOptions{Device: policy, DatasetID: dataset.Dataset.ID, Name: "Observed profile", Kind: "voice-embedding", Adapter: builtin(), BaseModelID: base, BaseDigest: digest}
			claim := work(t, db, "models.train", TrainPayload{Options: options})
			result, err := s.ExecuteTrain(ctx, claim)
			if err != nil {
				t.Fatal(err)
			}
			observed := result.(map[string]any)["device_executions"].([]processing.DeviceExecution)
			if calls != 2 || len(observed) != 1 || observed[0].Count != 2 || observed[0].RequestedDevice != want {
				t.Fatalf("observations=%+v calls=%d", observed, calls)
			}
			replayed, err := s.ExecuteTrain(ctx, claim)
			if err != nil {
				t.Fatal(err)
			}
			replayObservations, ok := replayed.(map[string]any)["device_executions"].([]processing.DeviceExecution)
			if !ok || len(replayObservations) != 1 || replayObservations[0].Count != 2 || calls != 2 {
				t.Fatalf("replay lost durable observations: %#v", replayed)
			}
			version := result.(map[string]any)["version_id"].(string)
			stored, err := s.Show(ctx, version)
			if err != nil {
				t.Fatal(err)
			}
			var manifest struct {
				Training struct {
					Executions []processing.DeviceExecution `json:"device_executions"`
				} `json:"training"`
			}
			if json.Unmarshal(stored.Metadata, &manifest) != nil || len(manifest.Training.Executions) != 1 || manifest.Training.Executions[0].Count != 2 {
				t.Fatalf("durable provenance missing: %s", stored.Metadata)
			}
			if _, err = s.Fetch(ctx, version, filepath.Join(t.TempDir(), "fetched")); err != nil {
				t.Fatal(err)
			}
			if _, err = db.SetSpeakerProfile(ctx, contracts.ID(), speaker.ID, 0, version); err != nil {
				t.Fatal(err)
			}
			target, _ := recording(t, db, 1)
			if _, err = db.MutateRoster(ctx, contracts.ID(), target.ID, 0, "replace", []string{speaker.ID}); err != nil {
				t.Fatal(err)
			}
			payload, err := s.ValidateMatch(ctx, MatchOptions{Device: policy, RecordingID: target.ID, ModelID: base, ModelDigest: digest, Adapter: builtin(), Threshold: 0.8, Margin: 0.05, MinEvidenceUS: 1000})
			if err != nil {
				t.Fatal(err)
			}
			matched, err := s.ExecuteMatch(ctx, work(t, db, "recordings.match", payload))
			if err != nil {
				t.Fatal(err)
			}
			matchObservations := matched.(map[string]any)["device_executions"].([]processing.DeviceExecution)
			if len(matchObservations) != 1 || matchObservations[0].Count != 1 || matchObservations[0].RequestedDevice != want {
				t.Fatal(matchObservations)
			}
		})
	}
}

func TestEmbeddingRejectsContradictoryPolicyReceipt(t *testing.T) {
	s := &Service{EmbedBatchWithPolicy: func(context.Context, []PreparedInput, string, string, string) (processing.EmbeddingResult, error) {
		return processing.EmbeddingResult{Vectors: [][]float64{{1, 0}}, Executions: []processing.DeviceExecution{{RequestedDevice: "cpu", AttemptedDevices: []string{"cpu"}, Device: "cpu", Count: 1}}}, nil
	}}
	if _, err := s.embedInputsWithPolicy(context.Background(), []PreparedInput{{}}, "model", "digest", "auto"); err == nil {
		t.Fatal("accepted mismatched requested policy")
	}
}
