// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/answers"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestAnswerRequestAndEnvelopeSchemas(t *testing.T) {
	for _, scenario := range []struct {
		raw   string
		valid bool
	}{
		{`{"version":"1.3.1","question":"What changed?"}`, true},
		{`{"version":"1.3.1","question":"What changed?","filters":{"source_interval":{"start_us":"9007199254740993","end_us":"9007199254741993"}}}`, true},
		{`{"version":"1.3.1","question":"q","provider":{"credential":"private"}}`, false},
		{`{"version":"1.3.1","question":"q","budget":{"rows":100001}}`, false},
		{`{"version":"1.3.1","question":"q","filters":{"source_interval":{"start_us":9007199254740993,"end_us":9007199254741993}}}`, false},
	} {
		if (ValidateAnswerRequest([]byte(scenario.raw)) == nil) != scenario.valid {
			t.Fatal(scenario)
		}
	}
	e := answers.Envelope{Version: contracts.Version, PackageID: contracts.ID(), State: "insufficient", Claims: []answers.Claim{{ID: "c1", Text: "No current evidence", Support: "unsupported", Uncertainty: "No matching cues"}}}
	raw, _ := json.Marshal(e)
	if err := ValidateAnswerEnvelope(raw); err != nil {
		t.Fatal(err)
	}
	base := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: "answers.prepare", Data: json.RawMessage(`{"version":"1.3.1","question":"q"}`)}
	raw, _ = json.Marshal(base)
	if err := ValidateRequest(raw); err != nil {
		t.Fatal(err)
	}
	base.Operation = "answers.show"
	base.ItemID = contracts.ID()
	base.Data = nil
	raw, _ = json.Marshal(base)
	if err := ValidateRequest(raw); err != nil {
		t.Fatal(err)
	}
	base.Data = json.RawMessage(`{}`)
	raw, _ = json.Marshal(base)
	if ValidateRequest(raw) == nil {
		t.Fatal("show admitted unexpected input")
	}
	for _, op := range []string{"answers.prepare", "answers.validate"} {
		example, schema := InputHelp(op)
		if len(example) == 0 || len(schema) == 0 {
			t.Fatal("missing help", op)
		}
	}
}
