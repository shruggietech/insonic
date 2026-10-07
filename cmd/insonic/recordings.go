// SPDX-License-Identifier: Apache-2.0
package main

import (
	"bytes"
	"encoding/json"
	"github.com/shruggietech/insonic/internal/app"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"os"
	"path/filepath"
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
	lock, e := w.Lock()
	if e != nil {
		return output(nil, e, machine)
	}
	defer lock.Unlock()
	file, e := os.Open(path)
	if e != nil {
		return output(nil, contracts.Fail("unavailable"), machine)
	}
	defer file.Close()
	raw, e := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 || catalog.ValidateJSON(raw) != nil {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if schemas.ValidateDocument(raw) != nil {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	var config app.ProcessingTools
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&config) != nil || decoder.Decode(new(any)) != io.EOF {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if e = app.ValidateProcessingTools(config); e != nil {
		return output(nil, e, machine)
	}
	temp, e := os.CreateTemp(w.Control, ".processing-tools-*")
	if e != nil {
		return output(nil, contracts.Fail("unavailable"), machine)
	}
	defer os.Remove(temp.Name())
	if _, e = temp.Write(raw); e == nil {
		e = temp.Sync()
	}
	closeErr := temp.Close()
	if e == nil {
		e = closeErr
	}
	if e == nil {
		e = os.Rename(temp.Name(), filepath.Join(w.Control, "processing-tools.json"))
	}
	if e != nil {
		return output(nil, contracts.Fail("unavailable"), machine)
	}
	return output(map[string]bool{"configured": true}, nil, machine)
}
