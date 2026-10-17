// SPDX-License-Identifier: Apache-2.0
package app

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"testing"
)

func TestSavedQueryHelpExampleExecutes(t *testing.T) {
	a := configuredApp(t)
	raw, _ := schemas.InputHelp("query.save")
	var input map[string]any
	if json.Unmarshal(raw, &input) != nil {
		t.Fatal(string(raw))
	}
	request := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: a.Workspace.Config.WorkspaceID, RequestID: contracts.ID(), Operation: "query.save", ItemID: contracts.ID(), Data: raw}
	encoded, _ := json.Marshal(request)
	if e := schemas.ValidateRequest(encoded); e != nil {
		t.Fatal(e)
	}
	if response := realRequest(a, "query.save", request.ItemID, input); response.Error != nil {
		t.Fatal("help example dispatch", response.Error, string(raw))
	}
	for _, title := range []any{nil, "", " \t\n", "\u0085\u00a0\u2003"} {
		query := input["query"].(map[string]any)
		if title == nil {
			delete(query, "title")
		} else {
			query["title"] = title
		}
		request.ItemID = contracts.ID()
		request.Data, _ = json.Marshal(input)
		encoded, _ = json.Marshal(request)
		if schemas.ValidateRequest(encoded) == nil {
			t.Fatalf("schema accepted blank title %q", title)
		}
		if realRequest(a, "query.save", request.ItemID, input).Error == nil {
			t.Fatalf("dispatcher accepted blank title %q", title)
		}
	}
}
