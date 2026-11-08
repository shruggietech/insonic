// SPDX-License-Identifier: Apache-2.0
package schemas

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/shruggietech/insonic/internal/contracts"
)

// AnswerSynthesisSchemaBytes specializes the packaged answer contract for one
// generation. It contains no remote or dangling references. Public historical
// answer envelopes retain their separate compatibility rules.
func AnswerSynthesisSchemaBytes(packageID string) ([]byte, error) {
	raw, err := registry.ReadFile("v" + contracts.Version + "/answer-evidence.schema.json")
	if err != nil {
		return nil, err
	}
	var doc map[string]any
	if err = json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	defs := doc["$defs"].(map[string]any)
	envelope := defs["envelope"].(map[string]any)
	properties := envelope["properties"].(map[string]any)
	if packageID != "" {
		properties["package_id"].(map[string]any)["const"] = packageID
	}
	properties["state"].(map[string]any)["enum"] = []string{"complete", "partial", "insufficient"}
	properties["claims"].(map[string]any)["maxItems"] = 128
	claim := defs["claim"].(map[string]any)
	claim["required"] = append(claim["required"].([]any), "kind")
	var expand func(any, int) (any, error)
	expand = func(value any, depth int) (any, error) {
		if depth > 32 {
			return nil, errors.New("answer schema reference depth exceeded")
		}
		switch n := value.(type) {
		case map[string]any:
			out := map[string]any{}
			if ref, ok := n["$ref"].(string); ok {
				if !strings.HasPrefix(ref, "#/$defs/") {
					return nil, errors.New("answer schema reference is not local")
				}
				target, ok := defs[strings.TrimPrefix(ref, "#/$defs/")]
				if !ok {
					return nil, errors.New("answer schema reference is missing")
				}
				expanded, err := expand(target, depth+1)
				if err != nil {
					return nil, err
				}
				out = expanded.(map[string]any)
			}
			for key, child := range n {
				if key == "$ref" || key == "examples" {
					continue
				}
				expanded, err := expand(child, depth+1)
				if err != nil {
					return nil, err
				}
				out[key] = expanded
			}
			return out, nil
		case []any:
			out := make([]any, len(n))
			for i, child := range n {
				expanded, err := expand(child, depth+1)
				if err != nil {
					return nil, err
				}
				out[i] = expanded
			}
			return out, nil
		default:
			return value, nil
		}
	}
	expanded, err := expand(envelope, 0)
	if err != nil {
		return nil, err
	}
	return json.Marshal(expanded)
}

var synthesisOnce sync.Once
var synthesisSchema *jsonschema.Schema
var synthesisErr error

// ValidateAnswerSynthesis checks raw types before Go decoding can erase nulls.
// Package identity and citation semantics are checked against current authority
// by the answer validator, independently of this generation shape.
func ValidateAnswerSynthesis(data []byte) error {
	synthesisOnce.Do(func() {
		raw, err := AnswerSynthesisSchemaBytes("")
		if err != nil {
			synthesisErr = err
			return
		}
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
		if err != nil {
			synthesisErr = err
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.AssertFormat()
		compiler.UseLoader(offlineLoader{})
		if err = compiler.AddResource("urn:insonic:answer-synthesis", doc); err != nil {
			synthesisErr = err
			return
		}
		synthesisSchema, synthesisErr = compiler.Compile("urn:insonic:answer-synthesis")
	})
	if synthesisErr != nil {
		return contracts.Fail("operation_failed")
	}
	if err := contracts.ValidateJSON(data); err != nil {
		return err
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return inputError(err)
	}
	return inputError(synthesisSchema.Validate(doc))
}
