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
	"strings"
)

func parseRecording(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 3 || args[0] != "recordings" || !contracts.ValidID(args[2]) {
		return fail()
	}
	operation := "recordings." + args[1]
	if !strings.Contains("|show|document|mappings|process|assemble|map-speaker|export|resolve-segment|", "|"+args[1]+"|") {
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
func processingToolsCommand(w *workspace.Workspace, path string, machine bool) int {
	return configureToolsCommand(w, path, "processing_tools", machine)
}
