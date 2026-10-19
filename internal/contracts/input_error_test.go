// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestJSONFailuresHaveGeneratedCausesWithoutPrivateInput(t *testing.T) {
	for _, tc := range []struct{ raw, reason string }{
		{"", "json_empty"},
		{"{", "json_syntax"},
		{`{"private-secret-key":"private-secret-value","private-secret-key":false}`, "json_duplicate_key"},
		{`{} {"private-secret-key":"private-secret-value"}`, "json_extra_document"},
		{string([]byte{'"', 0xff, '"'}), "json_encoding"},
	} {
		t.Run(tc.reason, func(t *testing.T) {
			err := ValidateJSON([]byte(tc.raw))
			var typed *Error
			if !errors.As(err, &typed) || typed.Reason != tc.reason {
				t.Fatalf("got %v; want generated %s", err, tc.reason)
			}
			raw, _ := json.Marshal(SafeError(err))
			if strings.Contains(string(raw), "private-secret") {
				t.Fatal("private JSON text escaped into the generated error")
			}
		})
	}
}

func TestValidationLocationsDoNotRepeatDynamicKeys(t *testing.T) {
	for _, key := range []string{"private-secret-member", "0000012345", "12345"} {
		err := InputFailure("input_type", InputViolation{Pointer: "/data/parameters/" + key + "/value", Expected: "string"})
		raw, _ := json.Marshal(SafeError(err))
		if strings.Contains(string(raw), key) || !strings.Contains(string(raw), "/data/parameters/*/value") {
			t.Fatal(string(raw))
		}
	}
	err := &Error{Code: "invalid_request", Reason: "input_enum", Message: "private-secret-value", Validation: &InputViolation{Pointer: "/private-secret-member", Expected: "private-secret-value", Constraint: "private-secret-value", Offset: -1}}
	raw, _ := json.Marshal(SafeError(err))
	if strings.Contains(string(raw), "private-secret") {
		t.Fatal(string(raw))
	}
}

func TestJSONSyntaxPositionUsesTheDecoderFailureByte(t *testing.T) {
	for _, raw := range []string{`{"a":truX}`, `{"a": "bad\xescape"}`, `{} truX`} {
		var out any
		decoder := json.NewDecoder(strings.NewReader(raw))
		var syntax *json.SyntaxError
		err := decoder.Decode(&out)
		if err == nil {
			err = decoder.Decode(&out)
		}
		if !errors.As(err, &syntax) {
			t.Fatalf("fixture has no syntax position: %v", err)
		}
		var got *Error
		if !errors.As(ValidateJSON([]byte(raw)), &got) || got.Validation == nil || got.Validation.Offset != syntax.Offset {
			t.Fatalf("syntax position for %q: %+v; want %d", raw, got, syntax.Offset)
		}
	}
}

func TestInputBoundsPreserveSignedClockLimits(t *testing.T) {
	for _, bound := range []int64{-9223372036854775807, -1, 0, 9223372036854775807} {
		err := SafeError(InputFailure("input_range", InputViolation{Pointer: "/data/unix_ns", Constraint: "minimum", Bound: bound}))
		if err.Validation == nil || err.Validation.Constraint != "minimum" || err.Validation.Bound != bound {
			t.Fatalf("lost exact signed bound: %+v", err)
		}
	}
}
