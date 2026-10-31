// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// InputViolation contains generated contract metadata, never submitted values.
// Unknown instance keys are replaced with a wildcard before serialization.
type InputViolation struct {
	Pointer    string `json:"pointer,omitempty"`
	Offset     int64  `json:"offset,omitempty"`
	Expected   string `json:"expected,omitempty"`
	Constraint string `json:"constraint,omitempty"`
	Bound      int64  `json:"bound,omitempty"`
}

func inputReason(reason string) bool {
	switch reason {
	case "unsupported_language", "json_empty", "json_syntax", "json_encoding", "json_duplicate_key", "json_extra_document", "input_shape", "input_unknown_field", "input_required", "input_type", "input_enum", "input_range", "input_length", "input_version", "input_constraint", "input_bytes", "credential_value_limit", "credential_input_shape":
		return true
	}
	return false
}

// InputPointer exposes only property identifiers present in packaged contracts
// only. Numeric/dynamic instance tokens become wildcards because an error
// pointer alone cannot prove whether such a token is an array index or map key.
func InputPointer(parts []string) string {
	if len(parts) > 12 {
		parts = parts[:12]
	}
	var b strings.Builder
	for _, part := range parts {
		if !inputFields[part] {
			part = "*"
		}
		if b.Len()+len(part)+1 > 256 {
			break
		}
		b.WriteByte('/')
		b.WriteString(strings.ReplaceAll(strings.ReplaceAll(part, "~", "~0"), "/", "~1"))
	}
	return b.String()
}

func InputFailure(reason string, detail InputViolation) *Error {
	if !inputReason(reason) {
		return Condition("invalid_request", "input_invalid")
	}
	e := Condition("invalid_request", reason)
	if detail.Pointer != "" {
		detail.Pointer = InputPointer(strings.Split(strings.TrimPrefix(detail.Pointer, "/"), "/"))
	}
	if detail.Offset < 0 || detail.Offset > MaxWorkPayload+1 {
		detail.Offset = 0
	}
	switch detail.Expected {
	case "object", "array", "string", "number", "integer", "boolean", "null":
	default:
		detail.Expected = ""
	}
	switch detail.Constraint {
	case "minimum", "maximum", "minLength", "maxLength", "minItems", "maxItems", "minProperties", "maxProperties", "maxBytes", "maxDepth":
	default:
		detail.Constraint = ""
		detail.Bound = 0
	}
	if detail.Bound < 0 && detail.Constraint != "minimum" && detail.Constraint != "maximum" {
		detail.Constraint = ""
		detail.Bound = 0
	}
	if detail.Pointer != "" {
		e.Message += " Field: " + detail.Pointer + "."
	}
	if detail.Offset != 0 {
		e.Message += fmt.Sprintf(" Byte position: %d.", detail.Offset)
	}
	if detail.Expected != "" {
		e.Message += " Expected: " + detail.Expected + "."
	}
	if detail.Constraint != "" {
		e.Message += fmt.Sprintf(" Constraint: %s=%d.", detail.Constraint, detail.Bound)
	}
	e.Validation = &detail
	return e
}

// DecodeInput is strict and preserves generated JSON/type/member causes. It
// never returns encoding/json's messages, which can include private field text.
func DecodeInput(raw []byte, out any) error {
	if err := ValidateJSON(raw); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		var mismatch *json.UnmarshalTypeError
		if errors.As(err, &mismatch) {
			want := ""
			if mismatch.Type != nil {
				switch mismatch.Type.Kind().String() {
				case "struct", "map":
					want = "object"
				case "slice", "array":
					want = "array"
				case "string":
					want = "string"
				case "bool":
					want = "boolean"
				case "int", "int64", "uint64":
					want = "integer"
				case "float64":
					want = "number"
				}
			}
			return InputFailure("input_type", InputViolation{Pointer: InputPointer(strings.Split(mismatch.Field, ".")), Expected: want, Offset: mismatch.Offset})
		}
		if strings.HasPrefix(err.Error(), "json: unknown field ") {
			return InputFailure("input_unknown_field", InputViolation{})
		}
		return InputFailure("input_shape", InputViolation{})
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return InputFailure("json_extra_document", InputViolation{Offset: d.InputOffset()})
	}
	return nil
}
