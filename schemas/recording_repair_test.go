// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRecordingRepairRequestContracts(t *testing.T) {
	base := map[string]any{"schema_version": "1.0.3", "kind": "runtime-request", "workspace_id": "10000000-0000-4000-8000-000000000001", "request_id": "10000000-0000-4000-8000-000000000002", "item_id": "10000000-0000-4000-8000-000000000003", "operation": "recordings.repair", "data": map[string]any{"expected_revision": 1, "source_revision": 1, "document_digest": strings.Repeat("a", 64), "automatic": true, "processing": map[string]any{"pipeline_id": "10000000-0000-4000-8000-000000000004"}}}
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
		valid  bool
	}{
		{"automatic", func(map[string]any) {}, true},
		{"explicit", func(d map[string]any) {
			delete(d, "automatic")
			d["ranges"] = []any{map[string]any{"stream_index": 0, "start_sample": 0, "end_sample": 16000}}
		}, true},
		{"no-selection", func(d map[string]any) { delete(d, "automatic") }, false},
		{"ambiguous-selection", func(d map[string]any) { d["ranges"] = []any{map[string]any{"stream_index": 0, "end_sample": 16000}} }, false},
		{"unfenced", func(d map[string]any) { delete(d, "document_digest") }, false},
		{"unbounded-attempts", func(d map[string]any) { d["policy"] = map[string]any{"max_attempts": 4} }, false},
		{"partial-comparison", func(d map[string]any) { d["coverage"] = map[string]any{"after": 1} }, false},
		{"first-comparison", func(d map[string]any) { d["coverage"] = map[string]any{"after": 0} }, true},
		{"reuse-text", func(d map[string]any) { d["processing"].(map[string]any)["transcription"] = "reuse" }, false},
		{"unknown-policy", func(d map[string]any) { d["policy"] = map[string]any{"unlimited": true} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, _ := json.Marshal(base)
			var value map[string]any
			json.Unmarshal(encoded, &value)
			tc.change(value["data"].(map[string]any))
			encoded, _ = json.Marshal(value)
			if err := ValidateRequest(encoded); (err == nil) != tc.valid {
				t.Fatalf("valid=%v: %v", tc.valid, err)
			}
		})
	}
	base["operation"] = "recordings.coverage"
	base["data"] = map[string]any{"after": 1, "limit": 100}
	encoded, _ := json.Marshal(base)
	if err := ValidateRequest(encoded); err != nil {
		t.Fatal("read-only pagination rejected", err)
	}
}

func TestSavedCoverageCannotSkipRepairDiagnostics(t *testing.T) {
	raw, err := registry.ReadFile("v1.0.3/pipeline-config.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Examples []map[string]any `json:"examples"`
	}
	if json.Unmarshal(raw, &schema) != nil || len(schema.Examples) == 0 {
		t.Fatal("missing example")
	}
	value := schema.Examples[0]
	configuration := value["configuration"].(map[string]any)
	for _, offset := range []int{0, 1} {
		configuration["coverage"] = map[string]any{"after": offset}
		raw, _ = json.Marshal(configuration)
		if err := ValidatePipelineConfiguration(raw); (err == nil) != (offset == 0) {
			t.Fatalf("offset %d: %v", offset, err)
		}
	}
}
