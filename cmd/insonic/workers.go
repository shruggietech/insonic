// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"io"
	"os"

	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
)

func parseWorkers(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	op := args[0] + "." + args[1]
	valid := map[string]bool{"workers.catalog": true, "workers.inspect": true, "workers.plan": true, "workers.setup": true, "processing.readiness": true, "models.defaults": true, "models.prepare": true}
	if !valid[op] {
		return fail()
	}
	var raw json.RawMessage
	if len(args) > 2 {
		if len(args) != 4 || args[2] != "--input" {
			return fail()
		}
		f, e := os.Open(args[3])
		if e != nil {
			return "", "", nil, contracts.Condition("unavailable", "input_missing")
		}
		defer f.Close()
		raw, e = io.ReadAll(io.LimitReader(f, (1<<20)+1))
		if e != nil || len(raw) > 1<<20 || catalog.ValidateJSON(raw) != nil {
			return fail()
		}
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, Data: raw}
	b, _ := json.Marshal(req)
	if schemas.ValidateRequest(b) != nil {
		return fail()
	}
	return op, "", raw, nil
}
