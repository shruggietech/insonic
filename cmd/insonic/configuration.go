// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"io"
	"os"
)

// parseConfiguration accepts explicit JSON files for bounded structured inputs.
// The same assembled envelope is validated by the runtime and desktop bridge.
func parseConfiguration(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	operation := args[0] + "." + args[1]
	if !contracts.ConfigurationOperation(operation) {
		return fail()
	}
	index := 2
	item := ""
	if args[1] != "list" && operation != "terms.compile" {
		if len(args) < 3 || !contracts.ValidID(args[2]) {
			return fail()
		}
		item = args[2]
		index = 3
	}
	var raw json.RawMessage
	if len(args) > index {
		if len(args) != index+2 || args[index] != "--input" {
			return fail()
		}
		file, err := os.Open(args[index+1])
		if err != nil {
			return "", "", nil, contracts.Fail("unavailable")
		}
		defer file.Close()
		raw, err = io.ReadAll(io.LimitReader(file, contracts.MaxWorkPayload+1))
		if err != nil || len(raw) > contracts.MaxWorkPayload || catalog.ValidateJSON(raw) != nil {
			return fail()
		}
	}
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: operation, ItemID: item, Data: raw}
	if !contracts.ConfigurationRequestValid(request) {
		return fail()
	}
	encoded, err := json.Marshal(request)
	if err != nil || schemas.ValidateRequest(encoded) != nil {
		return fail()
	}
	return operation, item, raw, nil
}
