// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
)

func TestBridgePreservesGeneratedInputConstraintBeforeRuntimeStartup(t *testing.T) {
	b := Bridge{Workspace: &workspace.Workspace{Config: workspace.Config{WorkspaceID: contracts.ID()}}}
	request := contracts.Request{Operation: "jobs.start", DurationMS: 600001}
	result := b.operate(request)
	if result.Error == nil || result.Error.Code != "invalid_request" || result.Error.Reason != "input_range" || result.Error.Validation == nil || result.Error.Validation.Bound != 600000 {
		t.Fatalf("%+v", result)
	}
	raw, _ := json.Marshal(result)
	if err := schemas.ValidateResponse(raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "runtime_session_id") {
		t.Fatal("invalid input started a runtime")
	}
}
