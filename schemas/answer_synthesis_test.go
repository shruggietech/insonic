// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
)

func TestAnswerSynthesisSchemaIsCompleteAndStrict(t *testing.T) {
	id := contracts.ID()
	raw, err := AnswerSynthesisSchemaBytes(id)
	if err != nil || bytes.Contains(raw, []byte("\"$ref\"")) {
		t.Fatal("incomplete schema", err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat()
	compiler.UseLoader(offlineLoader{})
	if err = compiler.AddResource("urn:answer:test", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("urn:answer:test")
	if err != nil {
		t.Fatal(err)
	}
	valid := map[string]any{"version": contracts.Version, "package_id": id, "state": "insufficient", "claims": []any{map[string]any{"id": "one", "kind": "paraphrase", "text": "Unknown.", "support": "unsupported", "uncertainty": "No evidence."}}, "limitations": []any{"No evidence."}}
	if err = schema.Validate(valid); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"version", "package", "limitations", "kind", "uncertainty", "state", "reference", "excessive"} {
		t.Run(mode, func(t *testing.T) {
			clone, _ := json.Marshal(valid)
			var bad map[string]any
			json.Unmarshal(clone, &bad)
			claim := bad["claims"].([]any)[0].(map[string]any)
			switch mode {
			case "version":
				bad["version"] = "1"
			case "package":
				bad["package_id"] = contracts.ID()
			case "limitations":
				bad["limitations"] = []any{map[string]any{"text": "private-sentinel"}}
			case "kind":
				delete(claim, "kind")
			case "uncertainty":
				claim["uncertainty"] = nil
			case "state":
				bad["state"] = "cancelled"
			case "reference":
				claim["citations"] = []any{map[string]any{"evidence_id": "one", "reference": "source:private-sentinel", "quote": "Monday"}}
			case "excessive":
				claims := make([]any, 129)
				for i := range claims {
					claims[i] = claim
				}
				bad["claims"] = claims
			}
			if schema.Validate(bad) == nil {
				t.Fatal("invalid generation shape accepted")
			}
			if mode != "package" {
				body, _ := json.Marshal(bad)
				e := ValidateAnswerSynthesis(body)
				if e == nil {
					t.Fatal("runtime accepted invalid generation shape")
				}
				safe, _ := json.Marshal(contracts.SafeError(e))
				if strings.Contains(string(safe), "private-sentinel") {
					t.Fatal("provider value leaked")
				}
			}
		})
	}
}
