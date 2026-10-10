// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"testing"
)

func TestSpeakerVersionRequestsRemainExactAndRejectEnvelopeDrift(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	for _, operation := range []string{"models.dataset.show", "models.speaker.show"} {
		request := map[string]any{"kind": "runtime-request", "schema_version": "1.0.0", "workspace_id": id, "request_id": id, "operation": operation, "item_id": id}
		check := func(valid bool) {
			t.Helper()
			raw, _ := json.Marshal(request)
			if err := ValidateRequest(raw); (err == nil) != valid {
				t.Fatalf("%s valid=%v: %v", operation, valid, err)
			}
		}
		check(true)
		request["data"] = map[string]any{"version_id": id}
		check(false)
		delete(request, "data")
		request["publication_id"] = id
		check(false)
		delete(request, "publication_id")
		request["item_id"] = "default-profile"
		check(false)
	}
}

func TestSpeakerFetchConfinesItsArgumentContract(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	request := map[string]any{"kind": "runtime-request", "schema_version": "1.0.0", "workspace_id": id, "request_id": id, "operation": "models.speaker.fetch", "item_id": id, "data": map[string]any{"destination": "/explicit/model-output"}}
	check := func(valid bool) {
		t.Helper()
		raw, _ := json.Marshal(request)
		if err := ValidateRequest(raw); (err == nil) != valid {
			t.Fatalf("valid=%v: %v", valid, err)
		}
	}
	check(true)
	request["data"].(map[string]any)["destination"] = ""
	check(false)
	request["data"] = map[string]any{"destination": "/explicit/model-output", "credential_value": "secret"}
	check(false)
	request["data"] = map[string]any{"destination": "/explicit/model-output"}
	delete(request, "item_id")
	check(false)
}

func TestSpeakerTrainingProfilesAndMatchingHaveStrictElections(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	adapter := map[string]any{"id": "pyannote-profile", "contract_version": "1", "mode": "local", "architecture": "pyannote", "output_kinds": []string{"voice-embedding"}, "consumers": []string{"voice-matching"}}
	profile := map[string]any{"speaker_training": map[string]any{"adapter": adapter, "output_kind": "voice-embedding", "base_model_id": id}}
	raw, _ := json.Marshal(profile)
	if err := ValidatePipelineConfiguration(raw); err != nil {
		t.Fatal(err)
	}
	profile["recognition"] = localPipelineConfiguration()["recognition"]
	raw, _ = json.Marshal(profile)
	if ValidatePipelineConfiguration(raw) == nil {
		t.Fatal("unpaired processing stage accepted")
	}
	request := map[string]any{"kind": "runtime-request", "schema_version": "1.0.0", "workspace_id": id, "request_id": id, "operation": "models.train", "data": map[string]any{"dataset_id": id, "name": "Training", "pipeline_id": id, "pipeline_revision": 1}}
	check := func(valid bool) {
		t.Helper()
		raw, _ := json.Marshal(request)
		if err := ValidateRequest(raw); (err == nil) != valid {
			t.Fatalf("valid=%v: %v", valid, err)
		}
	}
	check(true)
	delete(request["data"].(map[string]any), "pipeline_revision")
	check(false)
	request["operation"] = "recordings.match"
	request["item_id"] = id
	request["data"] = map[string]any{"model_id": id, "adapter": adapter, "threshold": 0, "ambiguity_margin": 0, "min_evidence_us": 1}
	check(true)
	request["data"].(map[string]any)["speaker_count"] = 1
	check(false)
	delete(request["data"].(map[string]any), "speaker_count")
	request["data"].(map[string]any)["model_digest"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	check(false)
	delete(request["data"].(map[string]any), "model_digest")
	delete(request["data"].(map[string]any), "threshold")
	check(false)
}
