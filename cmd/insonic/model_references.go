// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"io"
	"os"
	"strconv"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
)

func modelInput(path string) (json.RawMessage, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, contracts.Fail("unavailable")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, contracts.MaxWorkPayload+1))
	if err != nil || len(raw) > contracts.MaxWorkPayload || catalog.ValidateJSON(raw) != nil {
		return nil, contracts.Fail("invalid_request")
	}
	return raw, nil
}

func parseModelReferences(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 || args[0] != "models" {
		return fail()
	}
	if args[1] == "resolve" {
		if len(args) != 5 || args[2] == "" || args[3] != "--operation" || !catalog.ModelOperation(args[4]) {
			return fail()
		}
		raw, err := json.Marshal(map[string]string{"reference": args[2], "operation": args[4]})
		return "models.resolve", "", raw, err
	}
	if args[1] == "discover" {
		if len(args) != 3 || !contracts.ValidID(args[2]) {
			return fail()
		}
		raw, err := json.Marshal(map[string]string{"source_id": args[2]})
		return "models.discover", "", raw, err
	}
	if (args[1] != "alias" && args[1] != "source") || len(args) < 3 {
		return fail()
	}
	op := "models." + args[1] + "." + args[2]
	if args[2] == "list" {
		if len(args) == 3 {
			return op, "", nil, nil
		}
		if len(args) == 5 && args[3] == "--input" {
			raw, err := modelInput(args[4])
			return op, "", raw, err
		}
		return fail()
	}
	if len(args) < 4 || !contracts.ValidID(args[3]) {
		return fail()
	}
	switch args[2] {
	case "show":
		if len(args) == 4 {
			return op, args[3], nil, nil
		}
	case "set":
		if len(args) == 6 && args[4] == "--input" {
			raw, err := modelInput(args[5])
			return op, args[3], raw, err
		}
	case "remove":
		if len(args) == 6 && args[4] == "--expected-revision" {
			rev, err := strconv.ParseInt(args[5], 10, 64)
			if err != nil || rev < 1 {
				return fail()
			}
			raw, err := json.Marshal(map[string]int64{"expected_revision": rev})
			return op, args[3], raw, err
		}
	}
	return fail()
}
