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

func parseImportPresets(args []string) (string, string, json.RawMessage, error) {
	op, id, raw, e := parseImportPresetInput(args)
	if e != nil {
		return op, id, raw, e
	}
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, ItemID: id, Data: raw}
	encoded, e := json.Marshal(request)
	if e != nil || !contracts.ConfigurationRequestValid(request) || schemas.ValidateRequest(encoded) != nil {
		return "", "", nil, contracts.Fail("invalid_request")
	}
	return op, id, raw, nil
}
func parseImportPresetInput(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	op := "import-presets." + args[1]
	switch args[1] {
	case "list":
		if len(args) == 2 {
			return op, "", nil, nil
		}
		if len(args) != 4 || args[2] != "--input" {
			return fail()
		}
		raw, e := readStructuredInput(args[3])
		return op, "", raw, e
	case "show":
		if len(args) != 3 || args[2] == "" {
			return fail()
		}
		raw, _ := json.Marshal(map[string]string{"selection": args[2]})
		return op, "", raw, nil
	case "create", "update", "delete":
		index, id := 2, ""
		if args[1] != "create" {
			if len(args) < 3 || !contracts.ValidID(args[2]) {
				return fail()
			}
			id = args[2]
			index = 3
		}
		if len(args) != index+2 || args[index] != "--input" {
			return fail()
		}
		raw, e := readStructuredInput(args[index+1])
		return op, id, raw, e
	}
	return fail()
}

func readStructuredInput(path string) (json.RawMessage, error) {
	f, e := os.Open(path)
	if e != nil {
		return nil, contracts.Condition("unavailable", "input_missing")
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (1<<20)+1))
	if e != nil || len(raw) > 1<<20 {
		return nil, contracts.Fail("output_limit")
	}
	if catalog.ValidateJSON(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return raw, nil
}
