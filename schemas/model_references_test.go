// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"testing"
)

func TestModelReferencesCompileOfflineAndShareElectionSyntax(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	for _, reference := range []string{"speech", "speech-v2", "base:" + id, "speaker:" + id, "source:archive/release/1"} {
		data := map[string]any{"kind": "runtime-request", "schema_version": "0.0.0", "workspace_id": id, "request_id": id, "operation": "models.resolve", "data": map[string]any{"reference": reference, "operation": "transcription"}}
		raw, _ := json.Marshal(data)
		if err := ValidateRequest(raw); err != nil {
			t.Fatalf("%s: %v", reference, err)
		}
		d := localPipelineConfiguration()
		d["recognition"].(map[string]any)["model_id"] = reference
		raw, _ = json.Marshal(d)
		if err := ValidatePipelineConfiguration(raw); err != nil {
			t.Fatalf("pipeline %s: %v", reference, err)
		}
	}
	for _, reference := range []string{"Speech", " speech", "base:invalid", "source:archive/", "source:archive/\ninvalid", "https://example.org/model"} {
		data := map[string]any{"kind": "runtime-request", "schema_version": "0.0.0", "workspace_id": id, "request_id": id, "operation": "models.resolve", "data": map[string]any{"reference": reference, "operation": "transcription"}}
		raw, _ := json.Marshal(data)
		if ValidateRequest(raw) == nil {
			t.Fatal("invalid reference accepted", reference)
		}
	}
}

func TestModelReferenceMutationsFenceRevisionsAndRejectEnvelopeDrift(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	base := map[string]any{"kind": "runtime-request", "schema_version": "0.0.0", "workspace_id": id, "request_id": id, "operation": "models.alias.set", "item_id": id, "data": map[string]any{"expected_revision": 0, "alias": map[string]any{"id": id, "name": "speech", "state": "active", "target": map[string]any{"kind": "base", "id": id, "operation": "transcription"}}}}
	raw, _ := json.Marshal(base)
	if e := ValidateRequest(raw); e != nil {
		t.Fatal(e)
	}
	base["publication_id"] = id
	raw, _ = json.Marshal(base)
	if ValidateRequest(raw) == nil {
		t.Fatal("mutation accepted unrelated publication authority")
	}
	delete(base, "publication_id")
	base["operation"] = "models.alias.remove"
	base["data"] = map[string]any{"expected_revision": 0}
	raw, _ = json.Marshal(base)
	if ValidateRequest(raw) == nil {
		t.Fatal("remove accepted zero expected revision")
	}
	base["data"] = map[string]any{"expected_revision": 1}
	raw, _ = json.Marshal(base)
	if e := ValidateRequest(raw); e != nil {
		t.Fatal(e)
	}
}
