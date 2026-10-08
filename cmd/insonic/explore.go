// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"os"
	"strconv"
)

func parseExplore(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	op := args[0] + "." + args[1]
	if !contracts.ExploreOperation(op) {
		return fail()
	}
	index := 2
	item := ""
	switch op {
	case "evidence.extract", "evidence.show", "timeline.recording", "query.show", "query.save", "views.show", "views.save":
		if len(args) < 3 || !contracts.ValidID(args[2]) {
			return fail()
		}
		item = args[2]
		index = 3
	}
	var raw json.RawMessage
	if (op == "query.run" || op == "query.explain") && len(args) > index && args[index] == "--saved" {
		if (len(args) != index+2 && len(args) != index+4) || !contracts.ValidID(args[index+1]) {
			return fail()
		}
		input := map[string]any{"saved_id": args[index+1]}
		if len(args) == index+4 {
			if args[index+2] != "--revision" {
				return fail()
			}
			n, e := strconv.ParseInt(args[index+3], 10, 64)
			if e != nil || n < 1 {
				return fail()
			}
			input["revision"] = n
		}
		raw, _ = json.Marshal(input)
	} else if len(args) > index {
		if len(args) != index+2 || args[index] != "--input" {
			return fail()
		}
		f, e := os.Open(args[index+1])
		if e != nil {
			return "", "", nil, contracts.Fail("unavailable")
		}
		defer f.Close()
		raw, e = io.ReadAll(io.LimitReader(f, (1<<20)+1))
		if e != nil || len(raw) > 1<<20 || catalog.ValidateJSON(raw) != nil {
			return fail()
		}
	}
	if op == "timeline.calendar" && len(raw) == 0 {
		raw = json.RawMessage(`{"filter":{"include_undated":true}}`)
	}
	return op, item, raw, nil
}
