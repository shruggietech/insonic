// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/schemas"
	"os"
	"path/filepath"
	"testing"
)

func TestExploreCLIContracts(t *testing.T) {
	p := filepath.Join(t.TempDir(), "query.json")
	if e := os.WriteFile(p, []byte(`{"definition":{"mode":"normalized","operation":"text-search","filters":{"text":"source"}}}`), 0600); e != nil {
		t.Fatal(e)
	}
	cases := [][]string{{"query", "run", "--input", p}, {"timeline", "calendar"}, {"graph", "status"}, {"query", "run", "--saved", contracts.ID()}, {"query", "show", contracts.ID()}}
	for _, args := range cases {
		op, id, data, e := parseExplore(args)
		if e != nil {
			t.Fatal(args, e)
		}
		r := contracts.Request{Kind: "runtime-request", Version: contracts.Version, WorkspaceID: contracts.ID(), RequestID: contracts.ID(), Operation: op, ItemID: id, Data: data}
		raw, _ := json.Marshal(r)
		if !contracts.ExploreRequestValid(r) || schemas.ValidateRequest(raw) != nil {
			t.Fatal(args, string(raw))
		}
	}
	_, _, _, e := parseExploreAlias([]string{"search", "phrase", "--speaker", contracts.ID()})
	if e != nil {
		t.Fatal(e)
	}
	for _, args := range [][]string{{"graph", "status", "--input"}, {"query", "run", "--saved", "bad"}, {"search", "phrase", "--unsupported", "yes"}} {
		_, _, _, e := parseExplore(args)
		if args[0] == "search" {
			_, _, _, e = parseExploreAlias(args)
		}
		if e == nil {
			t.Fatal("invalid CLI accepted", args)
		}
	}
}
