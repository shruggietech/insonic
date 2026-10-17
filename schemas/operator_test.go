// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestGeneratedOperatorReasonEnvelope(t *testing.T) {
	for _, reason := range contracts.OperatorReasons() {
		t.Run(reason, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]any{"kind": "runtime-response", "schema_version": contracts.Version, "error": contracts.Condition("operation_failed", reason)})
			if err := ValidateResponse(raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}
