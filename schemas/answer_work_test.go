// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestAnswerWorkflowRequestSchema(t *testing.T) {
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "answers.request", Data: json.RawMessage(`{"request":{"version":"1.3.0","question":"What was agreed?"}}`)}
	raw, _ := json.Marshal(request)
	if err := ValidateRequest(raw); err != nil {
		t.Fatal(err)
	}
	request.Data = json.RawMessage(`{"request":{"version":"1.3.0","question":"q"},"configuration":{"provider":{"enabled":true,"route":"local","adapter":"insonic-http","endpoint":"http://127.0.0.1:1","model":"fixture"},"max_repairs":2}}`)
	raw, _ = json.Marshal(request)
	if ValidateRequest(raw) == nil {
		t.Fatal("unbounded repairs")
	}
	request.Operation = "answers.retry"
	request.Data = nil
	request.ItemID = contracts.ID()
	raw, _ = json.Marshal(request)
	if err := ValidateRequest(raw); err != nil {
		t.Fatal(err)
	}
	request.ItemID = ""
	raw, _ = json.Marshal(request)
	if ValidateRequest(raw) == nil {
		t.Fatal("retry without identity")
	}
	pipeline := []byte(`{"answer_synthesis":{"provider":{"enabled":true,"route":"local","adapter":"insonic-http","endpoint":"http://127.0.0.1:1","model":"fixture"}}}`)
	if err := ValidatePipelineConfiguration(pipeline); err != nil {
		t.Fatal(err)
	}
}
