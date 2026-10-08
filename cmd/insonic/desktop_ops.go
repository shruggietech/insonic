// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"time"

	"github.com/shruggietech/insonic/internal/app"
	"github.com/shruggietech/insonic/internal/catalog"
	"github.com/shruggietech/insonic/internal/contracts"
	local "github.com/shruggietech/insonic/internal/runtime"
	"github.com/shruggietech/insonic/internal/workspace"
	"github.com/shruggietech/insonic/schemas"
)

func parseDesktopOperation(args []string) (string, string, json.RawMessage, error) {
	fail := func() (string, string, json.RawMessage, error) { return "", "", nil, contracts.Fail("invalid_request") }
	if len(args) < 2 {
		return fail()
	}
	op := args[0] + "." + args[1]
	if !contracts.DesktopOperation(op) {
		return fail()
	}
	index := 2
	item := ""
	if op != "settings.show" && op != "settings.set" {
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
		file, e := os.Open(args[index+1])
		if e != nil {
			return "", "", nil, contracts.Fail("unavailable")
		}
		defer file.Close()
		raw, e = io.ReadAll(io.LimitReader(file, (1<<20)+1))
		if e != nil || len(raw) > 1<<20 || catalog.ValidateJSON(raw) != nil {
			return fail()
		}
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, ItemID: item, Data: raw}
	bytes, _ := json.Marshal(req)
	if !contracts.DesktopRequestValid(req) || schemas.ValidateRequest(bytes) != nil {
		return fail()
	}
	return op, item, raw, nil
}

// Legacy convenience commands now use the same validated, revision-checked
// runtime mutation as desktop Settings instead of writing beside its owner.
func configureToolsCommand(w *workspace.Workspace, path, section string, machine bool) int {
	file, e := os.Open(path)
	if e != nil {
		return output(nil, contracts.Fail("unavailable"), machine)
	}
	raw, e := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	file.Close()
	if e != nil || len(raw) > 1<<20 {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	if e = app.ValidateSetting(section, raw); e != nil {
		return output(nil, e, machine)
	}
	ctx, stop := context.WithTimeout(context.Background(), 10*time.Minute)
	defer stop()
	if e = local.Ensure(ctx, w, ""); e != nil {
		return output(nil, e, machine)
	}
	req := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: w.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "settings.show"}
	shown, e := local.Call(ctx, w, req)
	if e != nil {
		return output(nil, e, machine)
	}
	if shown.Error != nil {
		return output(nil, shown.Error, machine)
	}
	bytes, _ := json.Marshal(shown.Result)
	var result struct {
		Sections map[string]struct {
			Revision string `json:"revision"`
		} `json:"sections"`
	}
	if json.Unmarshal(bytes, &result) != nil || result.Sections[section].Revision == "" {
		return output(nil, contracts.Fail("invalid_request"), machine)
	}
	req.Operation = "settings.set"
	req.RequestID = contracts.ID()
	req.Data, _ = json.Marshal(map[string]any{"section": section, "revision": result.Sections[section].Revision, "value": json.RawMessage(raw)})
	saved, e := local.Call(ctx, w, req)
	if e == nil && saved.Error != nil {
		e = saved.Error
	}
	return output(saved, e, machine)
}
