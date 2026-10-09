package schemas

import (
	"bytes"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"testing"
)

func TestRuntimeRequestArguments(t *testing.T) {
	base := map[string]any{"schema_version": "0.0.0", "kind": "runtime-request", "workspace_id": "10000000-0000-4000-8000-000000000001", "request_id": "10000000-0000-4000-8000-000000000002", "operation": "workspace.show"}
	for _, scenario := range []struct {
		name   string
		fields map[string]any
		valid  bool
	}{
		{"inspection", nil, true},
		{"unused-duration", map[string]any{"duration_ms": 0}, false},
		{"missing-duration", map[string]any{"operation": "jobs.start"}, false},
		{"start", map[string]any{"operation": "jobs.start", "duration_ms": 20}, true},
		{"unused-job", map[string]any{"job_id": "10000000-0000-4000-8000-000000000003"}, false},
		{"wrong-kind", map[string]any{"kind": "workspace-config"}, false},
		{"unknown-field", map[string]any{"credential": "fixture-secret"}, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			value := make(map[string]any)
			for key, field := range base {
				value[key] = field
			}
			for key, field := range scenario.fields {
				value[key] = field
			}
			data, _ := json.Marshal(value)
			if (ValidateRequest(data) == nil) != scenario.valid {
				t.Fatal("request contract outcome")
			}
		})
	}
}

func TestAssistantReceivesBareSelfContainedQuerySchema(t *testing.T) {
	raw := QuerySchemaBytes()
	if bytes.Contains(raw, []byte(`"$ref"`)) {
		t.Fatal("dangling provider reference")
	}
	doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	c := jsonschema.NewCompiler()
	if e = c.AddResource("urn:query", doc); e != nil {
		t.Fatal(e)
	}
	schema, e := c.Compile("urn:query")
	if e != nil {
		t.Fatal(e)
	}
	good, e := jsonschema.UnmarshalJSON(bytes.NewReader([]byte(`{"definition":{"mode":"normalized","operation":"media-list","pagination":{"limit":5}}}`)))
	if e != nil || schema.Validate(good) != nil {
		t.Fatal("bare query rejected", e)
	}
}
