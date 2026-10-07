// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"testing"
)

func localPipelineConfiguration() map[string]any {
	return map[string]any{"recognition": map[string]any{"adapter": "faster-whisper", "contract_version": "1", "mode": "local", "model_id": "11111111-1111-4111-8111-111111111111"}, "diarization": map[string]any{"adapter": "pyannote", "contract_version": "1", "mode": "local", "model_id": "22222222-2222-4222-8222-222222222222"}}
}
func TestPipelineConfigurationPackagedDefaultsAndBounds(t *testing.T) {
	d := localPipelineConfiguration()
	raw, _ := json.Marshal(d)
	if err := ValidatePipelineConfiguration(raw); err != nil {
		t.Fatal("omitted limits and quality defaults rejected", err)
	}
	d["quality"] = map[string]any{"enabled": false, "min_speech_coverage": 0.5}
	raw, _ = json.Marshal(d)
	if err := ValidatePipelineConfiguration(raw); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{`{"recognition":{},"diarization":{}}`, `{"recognition":{"adapter":"faster-whisper","contract_version":"1","mode":"local","model_id":"11111111-1111-4111-8111-111111111111","limits":{"timeout_ms":0}},"diarization":{"adapter":"pyannote","contract_version":"1","mode":"local","model_id":"22222222-2222-4222-8222-222222222222"}}`} {
		if err := ValidatePipelineConfiguration([]byte(invalid)); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	d = localPipelineConfiguration()
	d["recognition"].(map[string]any)["credential_id"] = "33333333-3333-4333-8333-333333333333"
	raw, _ = json.Marshal(d)
	if err := ValidatePipelineConfiguration(raw); err == nil {
		t.Fatal("local stage accepted hosted credentials")
	}
}
func TestConfiguredRequestsShareOfflineSchema(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	envelope := func(op string, item bool, data any) []byte {
		d := map[string]any{"kind": "runtime-request", "schema_version": "0.0.0", "workspace_id": id, "request_id": id, "operation": op}
		if item {
			d["item_id"] = id
		}
		if data != nil {
			d["data"] = data
		}
		raw, _ := json.Marshal(d)
		return raw
	}
	for _, op := range []string{"pipelines.list", "speakers.list", "terms.list", "pipelines.show", "speakers.show", "terms.show", "pipelines.inspect", "speakers.aliases", "speakers.select", "speakers.diagnostics"} {
		item := op != "pipelines.list" && op != "speakers.list" && op != "terms.list"
		if err := ValidateRequest(envelope(op, item, nil)); err != nil {
			t.Fatal(op, err)
		}
	}
	if err := ValidateRequest(envelope("terms.compile", false, map[string]any{"max_hint_bytes": 200})); err != nil {
		t.Fatal(err)
	}
	for _, raw := range [][]byte{envelope("terms.compile", true, map[string]any{}), envelope("speakers.set", true, map[string]any{"speaker": map[string]any{"id": id, "name": "Person"}, "aliases": []any{}}), envelope("pipelines.list", true, nil), envelope("terms.compile", false, map[string]any{"max_hint_bytes": 8193}), envelope("speakers.select", true, map[string]any{"limit": 101})} {
		if err := ValidateRequest(raw); err == nil {
			t.Fatal("invalid configured request accepted", string(raw))
		}
	}
}

// Empty value-struct overrides are emitted by Go's JSON omitempty behavior.
// They must preserve direct managed-model client compatibility.
func TestDirectRecordingRequestAllowsEmptyEncodedOverrides(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	base := map[string]any{"kind": "runtime-request", "schema_version": "0.0.0", "workspace_id": id, "request_id": id, "operation": "recordings.process", "item_id": id}
	options := map[string]any{"transcription": "reuse", "diarization": "run", "diarization_model_id": id, "overrides": map[string]any{}}
	base["data"] = options
	raw, _ := json.Marshal(base)
	if err := ValidateRequest(raw); err != nil {
		t.Fatal("empty override narrowed direct request", err)
	}
	options["overrides"] = map[string]any{"quality": map[string]any{"enabled": false}}
	raw, _ = json.Marshal(base)
	if err := ValidateRequest(raw); err == nil {
		t.Fatal("nonempty override lost saved pipeline requirement")
	}
}
