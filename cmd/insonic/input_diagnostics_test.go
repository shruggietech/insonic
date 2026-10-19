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
	"github.com/shruggietech/insonic/internal/workspace"
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

func TestWorkspaceOpenFailuresRetainTheirActualCause(t *testing.T) {
	for _, mode := range []string{"absent-root", "absent-config", "malformed", "unsupported-version", "invalid-config"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			wantCode, wantReason := "invalid_request", ""
			switch mode {
			case "absent-root":
				root = filepath.Join(root, "absent")
				wantCode, wantReason = "unavailable", "workspace_missing"
			case "absent-config":
				wantCode, wantReason = "unavailable", "workspace_missing"
			default:
				w, err := workspace.Init(root, "Fixture")
				if err != nil {
					t.Fatal(err)
				}
				file := filepath.Join(w.Control, "workspace.json")
				raw := []byte("{broken PRIVATE_WORKSPACE_VALUE")
				if mode != "malformed" {
					raw, err = os.ReadFile(file)
					if err != nil {
						t.Fatal(err)
					}
					var config map[string]any
					if err = json.Unmarshal(raw, &config); err != nil {
						t.Fatal(err)
					}
					if mode == "unsupported-version" {
						config["schema_version"] = "999.0.0"
						wantCode = "incompatible_version"
					} else {
						config["display_name"] = nil
					}
					raw, err = json.Marshal(config)
					if err != nil {
						t.Fatal(err)
					}
				}
				if err = os.WriteFile(file, raw, 0600); err != nil {
					t.Fatal(err)
				}
			}
			stdout, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			defer stdout.Close()
			stderr, err := os.CreateTemp(t.TempDir(), "stderr")
			if err != nil {
				t.Fatal(err)
			}
			defer stderr.Close()
			previousOut, previousErr, previousContext := os.Stdout, os.Stderr, operatorContext
			defer func() { os.Stdout, os.Stderr, operatorContext = previousOut, previousErr, previousContext }()
			os.Stdout, os.Stderr = stdout, stderr
			code := execute([]string{"--workspace", root, "--json", "workspace", "show"})
			os.Stdout, os.Stderr = previousOut, previousErr
			if _, err = stderr.Seek(0, 0); err != nil {
				t.Fatal(err)
			}
			var response contracts.Response
			if err = json.NewDecoder(stderr).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.Error == nil || response.Error.Code != wantCode || response.Error.Reason != wantReason || response.Result != nil || code != exitCode(contracts.Fail(wantCode)) {
				t.Fatalf("%s lost its actual cause: code=%d response=%+v", mode, code, response)
			}
			if info, _ := stdout.Stat(); info.Size() != 0 {
				t.Fatal("failed workspace open wrote success output")
			}
			encoded, _ := json.Marshal(response)
			if strings.Contains(string(encoded), "PRIVATE_WORKSPACE_VALUE") {
				t.Fatal("workspace contents escaped into diagnostics")
			}
		})
	}
}
