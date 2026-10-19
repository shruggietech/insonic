// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestSettingsParserRetainsGeneratedInputCauses(t *testing.T) {
	valid := func(value string) string {
		return "{\"section\":\"appearance\",\"revision\":\"" + strings.Repeat("a", 64) + "\",\"value\":" + value + "}"
	}
	for _, tc := range []struct{ name, raw, reason, pointer string }{
		{"empty", "", "json_empty", ""},
		{"syntax", "{\"PRIVATE_INPUT_MEMBER\":", "json_syntax", ""},
		{"null", "null", "input_shape", "/data"},
		{"array", "[]", "input_shape", "/data"},
		{"extra-document", valid("null") + " {}", "json_extra_document", ""},
		{"duplicate", valid("{\"theme\":\"dark\",\"theme\":\"PRIVATE_INPUT_VALUE\",\"reduced_motion\":false}"), "json_duplicate_key", ""},
		{"unknown", valid("{\"theme\":\"dark\",\"reduced_motion\":false,\"PRIVATE_INPUT_MEMBER\":\"PRIVATE_INPUT_VALUE\"}"), "input_unknown_field", "/data/value"},
		{"type", valid("{\"theme\":9,\"reduced_motion\":false}"), "input_type", "/data/value/theme"},
		{"enum", valid("{\"theme\":\"PRIVATE_INPUT_VALUE\",\"reduced_motion\":false}"), "input_enum", "/data/value/theme"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "request.json")
			if err := os.WriteFile(file, []byte(tc.raw), 0600); err != nil {
				t.Fatal(err)
			}
			_, _, _, err := parseDesktopOperation([]string{"settings", "set", "--input", file})
			var typed *contracts.Error
			if !errors.As(err, &typed) || typed.Code != "invalid_request" || typed.Reason != tc.reason {
				t.Fatalf("got %v; want %s", err, tc.reason)
			}
			raw, _ := json.Marshal(typed)
			if strings.Contains(string(raw), "PRIVATE_INPUT") {
				t.Fatal("submitted input leaked in diagnostic")
			}
			if tc.pointer != "" && (typed.Validation == nil || typed.Validation.Pointer != tc.pointer) {
				t.Fatalf("wrong known field location: %+v", typed.Validation)
			}
			previous := operatorContext
			defer func() { operatorContext = previous }()
			for _, mode := range []string{"human", "quiet", "verbose", "debug", "json"} {
				operatorContext = invocation{Path: []string{"settings", "set"}, Args: []string{"settings", "set", "--input", file}, Quiet: mode == "quiet", Verbose: mode == "verbose", Debug: mode == "debug"}
				var out, errOut bytes.Buffer
				code := outputTo(nil, err, mode == "json", contracts.Response{}, &out, &errOut)
				if code != 2 || out.Len() != 0 || errOut.Len() == 0 || strings.Contains(errOut.String(), "PRIVATE_INPUT") {
					t.Fatalf("%s lost error stream/exit/privacy", mode)
				}
				if mode == "json" {
					var response contracts.Response
					if json.Unmarshal(errOut.Bytes(), &response) != nil || response.Error == nil || response.Error.Reason != tc.reason {
						t.Fatal("machine cause lost")
					}
				}
			}
		})
	}
}
