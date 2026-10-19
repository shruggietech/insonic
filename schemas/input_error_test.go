// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestRequestValidationReportsSafeFieldConstraints(t *testing.T) {
	base := func() map[string]any {
		return map[string]any{"kind": "runtime-request", "schema_version": contracts.Version, "workspace_id": contracts.ID(), "request_id": contracts.ID(), "operation": "jobs.start", "duration_ms": 20}
	}
	for _, tc := range []struct {
		name, reason, pointer string
		change                func(map[string]any)
	}{
		{"range", "input_range", "/duration_ms", func(m map[string]any) { m["duration_ms"] = 600001 }},
		{"type", "input_type", "/duration_ms", func(m map[string]any) { m["duration_ms"] = "private-secret-value" }},
		{"version", "input_version", "/schema_version", func(m map[string]any) { m["schema_version"] = "private-secret-version" }},
		{"unknown", "input_unknown_field", "", func(m map[string]any) { m["private-secret-member"] = "private-secret-value" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := base()
			tc.change(m)
			raw, _ := json.Marshal(m)
			err := ValidateRequest(raw)
			var e *contracts.Error
			if !errors.As(err, &e) || e.Reason != tc.reason {
				t.Fatalf("%v", err)
			}
			out, _ := json.Marshal(contracts.SafeError(err))
			if strings.Contains(string(out), "private-secret") {
				t.Fatal(string(out))
			}
			if tc.pointer != "" && (e.Validation == nil || e.Validation.Pointer != tc.pointer) {
				t.Fatalf("location: %+v", e)
			}
		})
	}
}

func TestValidationChoosesTheSelectedUnionMode(t *testing.T) {
	for _, tc := range []struct {
		mode, field string
		definition  map[string]any
	}{
		{"native", "text", map[string]any{"mode": "native", "dialect": "ladybug-cypher"}},
		{"normalized", "operation", map[string]any{"mode": "normalized"}},
	} {
		raw, _ := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "query.run", Data: mustRaw(t, map[string]any{"definition": tc.definition})})
		err := ValidateRequest(raw)
		var e *contracts.Error
		if !errors.As(err, &e) || e.Reason != "input_required" || e.Validation == nil || e.Validation.Pointer != "/data/definition/"+tc.field {
			t.Fatalf("%s selected unrelated union correction: %v", tc.mode, err)
		}
	}
}

func mustRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestNativeUnionCorrectionPreservesActualTypeFailure(t *testing.T) {
	raw, _ := json.Marshal(contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "query.run", Data: mustRaw(t, map[string]any{"definition": map[string]any{"mode": "native", "dialect": "ladybug-cypher", "text": 9}})})
	var e *contracts.Error
	if !errors.As(ValidateRequest(raw), &e) || e.Reason != "input_type" || e.Validation == nil || e.Validation.Pointer != "/data/definition/text" {
		t.Fatalf("selected unrelated union correction: %v", e)
	}
}
