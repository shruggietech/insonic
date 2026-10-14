// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestInputTemplatesMatchPackagedOperations(t *testing.T) {
	raw, _ := registry.ReadFile("v" + contracts.Version + "/runtime-request.schema.json")
	var request map[string]any
	_ = json.Unmarshal(raw, &request)
	for _, value := range request["properties"].(map[string]any)["operation"].(map[string]any)["enum"].([]any) {
		op := value.(string)
		template, _ := InputHelp(op)
		if template == nil {
			continue
		}
		t.Run(op, func(t *testing.T) {
			req := map[string]any{"kind": "runtime-request", "schema_version": contracts.Version, "workspace_id": contracts.ID(), "request_id": contracts.ID(), "operation": op, "data": json.RawMessage(template)}
			for _, value := range request["allOf"].([]any) {
				branch := value.(map[string]any)
				condition, _ := branch["if"].(map[string]any)
				props, _ := condition["properties"].(map[string]any)
				rule, _ := props["operation"].(map[string]any)
				matches := rule["const"] == op
				if choices, ok := rule["enum"].([]any); ok {
					for _, c := range choices {
						if c == op {
							matches = true
						}
					}
				}
				if !matches {
					continue
				}
				then, _ := branch["then"].(map[string]any)
				if required, ok := then["required"].([]any); ok {
					for _, field := range required {
						if field == "item_id" {
							req["item_id"] = contracts.ID()
						}
					}
				}
			}
			encoded, _ := json.Marshal(req)
			if err := ValidateRequest(encoded); err != nil {
				t.Fatalf("template %s: %v", template, err)
			}
		})
	}
}

func TestInputHelpHonorsMinimumArrayCardinality(t *testing.T) {
	value := helpExample(map[string]any{"type": "array", "minItems": json.Number("3"), "items": map[string]any{"type": "integer"}}).([]any)
	if len(value) != 3 {
		t.Fatalf("minimum items: %d", len(value))
	}
}
