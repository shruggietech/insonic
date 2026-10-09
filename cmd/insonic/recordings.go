// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"os"
	"strconv"
	"strings"
)

func parseRecording(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) >= 2 && args[0] == "recordings" && args[1] == "roster" {
		return parseRoster(args)
	}
	if len(args) < 3 || args[0] != "recordings" || !contracts.ValidID(args[2]) {
		return fail()
	}
	operation := "recordings." + args[1]
	if !strings.Contains("|show|document|mappings|process|assemble|map-speaker|match|export|resolve-segment|", "|"+args[1]+"|") {
		return fail()
	}
	var raw json.RawMessage
	if len(args) == 3 && (args[1] == "show" || args[1] == "document" || args[1] == "mappings") {
		return operation, args[2], nil, nil
	}
	if len(args) != 5 || args[3] != "--input" || args[1] == "show" {
		return fail()
	}
	file, e := os.Open(args[4])
	if e != nil {
		return "", "", nil, contracts.Fail("unavailable")
	}
	defer file.Close()
	raw, e = io.ReadAll(io.LimitReader(file, contracts.MaxWorkPayload+1))
	if e != nil || len(raw) > contracts.MaxWorkPayload || catalog.ValidateJSON(raw) != nil {
		return fail()
	}
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: operation, ItemID: args[2], Data: raw}
	encoded, _ := json.Marshal(request)
	if schemas.ValidateRequest(encoded) != nil {
		return fail()
	}
	return operation, args[2], raw, nil
}
func parseRoster(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 4 || !contracts.ValidID(args[3]) {
		return fail()
	}
	mode := args[2]
	op := "recordings.roster." + mode
	if mode == "show" {
		if len(args) != 4 {
			return fail()
		}
		return op, args[3], nil, nil
	}
	if mode != "add" && mode != "remove" && mode != "replace" && mode != "clear" {
		return fail()
	}
	refs := []string{}
	expected := int64(0)
	seen := false
	for i := 4; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return fail()
		}
		switch args[i] {
		case "--speaker":
			if mode == "clear" || len(refs) >= 1000 {
				return fail()
			}
			refs = append(refs, args[i+1])
		case "--expected-revision":
			if seen {
				return fail()
			}
			n, e := strconv.ParseInt(args[i+1], 10, 64)
			if e != nil || n < 0 {
				return fail()
			}
			expected = n
			seen = true
		default:
			return fail()
		}
	}
	if !seen || (mode == "add" || mode == "remove") && len(refs) == 0 {
		return fail()
	}
	raw, e := json.Marshal(map[string]any{"expected_revision": expected, "speakers": refs})
	return op, args[3], raw, e
}
func processingToolsCommand(w *workspace.Workspace, path string, machine bool) int {
	return configureToolsCommand(w, path, "processing_tools", machine)
}
