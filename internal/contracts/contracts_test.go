package contracts

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestErrorBoundaryAndIdentities(t *testing.T) {
	for _, code := range []string{"invalid_request", "incompatible_version", "workspace_mismatch", "conflict", "unavailable", "secret-value"} {
		if strings.Contains(Fail(code).Error(), "secret-value") {
			t.Fatal("untrusted error text returned")
		}
	}
	seen := map[string]bool{}
	for range 1000 {
		id := ID()
		if !ValidID(id) || seen[id] {
			t.Fatal("bad identity")
		}
		seen[id] = true
	}
}

func TestStructuredErrorDiagnosticsRedactAndCount(t *testing.T) {
	original := Fail("operation_failed")
	input := []Diagnostic{{Code: "consumer_speaker_attribution_omitted", Pointer: "/cues/0/speaker_attributions/0", Message: "private-speaker-secret"}, {Code: "consumer_speaker_attribution_omitted", Pointer: "/cues/1/speaker_attributions/0", Message: "private-speaker-secret"}, {Code: "private-secret-marker", Pointer: "/private/filesystem/path", Message: "private-speaker-secret"}}
	err := WithDiagnostics(original, input)
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if strings.Contains(string(data), "private") || len(err.Diagnostics) != 2 || err.Diagnostics[0].Count != 2 || err.Diagnostics[0].Pointer != "/cues/0/speaker_attributions/0" {
		t.Fatalf("bad redacted diagnostics %s", data)
	}
	if len(original.Diagnostics) != 0 {
		t.Fatal("mutated original error")
	}
	many := make([]Diagnostic, 8192)
	for i := range many {
		many[i] = input[0]
	}
	bounded := WithDiagnostics(original, many)
	data, _ = json.Marshal(bounded)
	if len(data) > 65536 || bounded.Diagnostics[0].Count != 8192 {
		t.Fatal("unbounded or incomplete aggregate")
	}
	if WithDiagnostics(nil, input) != nil {
		t.Fatal("fabricated error")
	}
}

func TestProcessingFailureCodesRemainActionable(t *testing.T) {
	if failure := Fail("model_acquisition_failed"); failure.Code != "model_acquisition_failed" || failure.Message == "The operation failed." {
		t.Fatal("acquisition failure lost its actionable outcome")
	}
	for _, code := range []string{"timing_unavailable", "invalid_audio", "audio_limit", "unsupported_audio", "model_unavailable", "unsupported_capability", "input_limit", "invalid_engine_output", "invalid_timing", "engine_ci_forbidden", "engine_unavailable", "unsupported_device", "invalid_embedding"} {
		if Fail(code).Code != code {
			t.Fatalf("collapsed %s", code)
		}
	}
}

func TestSpeakerWorkFailuresRetainDistinctRecoveryOutcomes(t *testing.T) {
	for _, code := range []string{"unsupported_retrieval", "empty_dataset", "unsupported_resume", "incompatible_checkpoint", "training_incomplete", "text_unavailable"} {
		if failure := Fail(code); failure.Code != code || failure.Message == "The operation failed." {
			t.Fatal("speaker recovery diagnostic collapsed", code, failure)
		}
	}
}
