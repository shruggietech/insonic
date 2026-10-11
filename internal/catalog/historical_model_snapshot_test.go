// SPDX-License-Identifier: Apache-2.0
package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestHistoricalBaseModelManifestSnapshotRestore(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	rawSchema, err := os.ReadFile("../../schemas/v1.0.0/base-model-manifest.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Examples []map[string]any `json:"examples"`
	}
	if err = json.Unmarshal(rawSchema, &schema); err != nil || len(schema.Examples) == 0 {
		t.Fatal("historical fixture", err)
	}
	raw, err := json.Marshal(schema.Examples[0])
	if err != nil {
		t.Fatal(err)
	}
	m := BaseModelInstall{ID: contracts.ID(), Name: schema.Examples[0]["name"].(string), Version: schema.Examples[0]["model_version"].(string), Digest: hash(raw), Manifest: raw, PublicationIDs: json.RawMessage(`[]`), State: "registered"}
	claim := speakerWork(t, s, "models.register")
	accepted, err := s.CommitBaseModel(ctx, claim, m)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	validateSnapshotSchema(t, snap)
	dest := localStore(t, s.workspace)
	if err = dest.Restore(ctx, snap); err != nil {
		t.Fatal("historical model restore", err)
	}
	restored, err := dest.BaseModel(ctx, accepted.ID)
	if err != nil || restored.Digest != accepted.Digest || restored.Revision != accepted.Revision || !bytes.Equal(restored.Manifest, accepted.Manifest) {
		t.Fatal("historical manifest or proof changed", err)
	}
}

func TestHistoricalSpeakerManifestSnapshotRestore(t *testing.T) {
	ctx := context.Background()
	s := localStore(t, contracts.ID())
	dataset, _, _, _ := speakerDatasetFixture(t, s)
	manifest, weights, preparation := availableArtifact(t, s), availableArtifact(t, s), availableArtifact(t, s)
	publications, err := json.Marshal([]string{manifest.ID, weights.ID, preparation.ID})
	if err != nil {
		t.Fatal(err)
	}
	claim := speakerWork(t, s, "speaker.train")
	value := SpeakerOutput{ID: contracts.ID(), ModelID: contracts.ID(), SpeakerID: dataset.Dataset.SpeakerID, DatasetID: dataset.Dataset.ID, WorkID: claim.ID, Kind: "voice-embedding", Name: "Historical", PublicationIDs: publications}
	var metadata map[string]json.RawMessage
	if err = json.Unmarshal(speakerMetadataFixture(t, value), &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["schema_version"] = json.RawMessage(`"1.0.0"`)
	rawSchema, err := os.ReadFile("../../schemas/v1.0.0/speaker-model.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var historical struct {
		Examples []map[string]json.RawMessage `json:"examples"`
	}
	if err = json.Unmarshal(rawSchema, &historical); err != nil || len(historical.Examples) == 0 {
		t.Fatal("historical speaker fixture", err)
	}
	var training, originalTraining map[string]json.RawMessage
	if err = json.Unmarshal(metadata["training"], &training); err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(historical.Examples[0]["training"], &originalTraining); err != nil {
		t.Fatal(err)
	}
	var prepared map[string]json.RawMessage
	if err = json.Unmarshal(originalTraining["preparation"], &prepared); err != nil {
		t.Fatal(err)
	}
	prepared["artifact"], err = json.Marshal(map[string]any{"artifact_id": preparation.ArtifactID, "sha256": preparation.Digest, "byte_length": preparation.Size})
	if err != nil {
		t.Fatal(err)
	}
	prepared["input_artifact_ids"] = json.RawMessage(`[]`)
	training["preparation"], err = json.Marshal(prepared)
	if err != nil {
		t.Fatal(err)
	}
	metadata["training"], err = json.Marshal(training)
	if err != nil {
		t.Fatal(err)
	}
	value.Metadata, err = json.Marshal(metadata)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := s.CommitSpeakerOutput(ctx, claim, value)
	if err != nil {
		t.Fatal("historical speaker output admission", err)
	}
	snap, err := s.Export(ctx)
	if err != nil {
		t.Fatal(err)
	}
	validateSnapshotSchema(t, snap)
	dest := localStore(t, s.workspace)
	if err = dest.Restore(ctx, snap); err != nil {
		t.Fatal("historical speaker output restore", err)
	}
	restored, err := dest.Export(ctx)
	if err != nil || len(restored.Records.SpeakerOutputs) != 1 {
		t.Fatal("missing historical output", err)
	}
	out := restored.Records.SpeakerOutputs[0]
	if out.ID != accepted.ID || !bytes.Equal(out.Metadata, accepted.Metadata) || !bytes.Equal(out.PublicationIDs, accepted.PublicationIDs) {
		t.Fatal("historical speaker manifest or proof changed")
	}
}
