// SPDX-License-Identifier: Apache-2.0
package backendsetup

import (
	"errors"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
	"strings"
)

type FieldError struct {
	Field      string `json:"field"`
	Correction string `json:"correction"`
}
type PlanError struct{ Fields []FieldError }

func (e *PlanError) Error() string { return "invalid backend plan" }
func (e *PlanError) Unwrap() error { return contracts.Condition("invalid_request", "backend_plan") }
func schemaPlanError(err error) error {
	out := &PlanError{Fields: []FieldError{}}
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return contracts.Condition("invalid_request", "backend_plan")
	}
	seen := map[string]bool{}
	var visit func(*jsonschema.ValidationError)
	visit = func(v *jsonschema.ValidationError) {
		if len(out.Fields) >= 20 {
			return
		}
		if len(v.Causes) > 0 {
			for _, cause := range v.Causes {
				visit(cause)
			}
			return
		}
		parts := []string{}
		for _, part := range v.InstanceLocation {
			if !strings.Contains("|schema_version|kind|workspace_id|display_name|control_directory|credential_namespace_id|profiles|storage|catalog|graph|profile_id|profile_revision|adapter_id|contract_version|configuration|expected_backend_version|root|path|endpoint|bucket|prefix|region|addressing_style|authentication|credential_id|host|port|database|schema|tls_mode|ca_file|preferred_dialect|", "|"+part+"|") {
				break
			}
			parts = append(parts, part)
		}
		field := strings.Join(parts, ".")
		if field == "" {
			field = "plan"
		}
		if !seen[field] {
			seen[field] = true
			out.Fields = append(out.Fields, FieldError{field, "Provide the required fields and types for this selected backend; values were not echoed."})
		}
	}
	visit(validation)
	return out
}
func Failure(err error) map[string]any {
	var p *PlanError
	if errors.As(err, &p) {
		return map[string]any{"field_errors": p.Fields, "valid": false}
	}
	return map[string]any{"valid": false}
}
