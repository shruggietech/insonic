// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeParameterMembersAreUnique(t *testing.T) {
	dir := t.TempDir()
	query := filepath.Join(dir, "q.cypher")
	params := filepath.Join(dir, "p.json")
	os.WriteFile(query, []byte("RETURN $value AS exact"), 0600)
	for _, raw := range []string{
		`{"value":{"type":"string","value":"first"},"value":{"type":"string","value":"second"}}`,
		`{"value":{"type":"string","type":"integer","value":1}}`,
		`{"value":{"type":"string","value":"first","value":"second"}}`,
		`{"value":{"type":"object","value":{"nested":1,"nested":2}}}`,
	} {
		os.WriteFile(params, []byte(raw), 0600)
		if _, _, _, e := parseExploreAlias([]string{"query", "native", "--dialect", "ladybug-cypher", "--file", query, "--parameters", params}); e == nil {
			t.Fatal("accepted duplicate", raw)
		}
	}
	os.WriteFile(params, []byte(`{"value":{"type":"integer","value":9223372036854775807}}`), 0600)
	_, _, raw, e := parseExploreAlias([]string{"query", "native", "--dialect", "ladybug-cypher", "--file", query, "--parameters", params})
	if e != nil || !strings.Contains(string(raw), "9223372036854775807") {
		t.Fatal(string(raw), e)
	}
}
func TestSearchConsumesCursor(t *testing.T) {
	_, _, raw, e := parseExploreAlias([]string{"search", "phrase", "--after", "opaque", "--limit", "25"})
	var q contracts.QueryInput
	json.Unmarshal(raw, &q)
	if e != nil || q.Definition.Pagination == nil || q.Definition.Pagination.Cursor != "opaque" {
		t.Fatal(string(raw), e)
	}
}
func TestHumanQueryContinuationPrecedesDetails(t *testing.T) {
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = invocation{Path: []string{"search"}, Args: []string{"search", "phrase"}}
	items := []any{}
	for i := 0; i < 100; i++ {
		items = append(items, map[string]any{"id": contracts.ID(), "text": strings.Repeat("long detail ", 40)})
	}
	out, e := renderHuman(contracts.Response{Result: map[string]any{"items": items, "total": 607, "omitted": 507, "next_cursor": "exact-cursor"}})
	if e != nil || !strings.Contains(out, "exact-cursor") || !strings.Contains(out, "--after") || strings.Index(out, "exact-cursor") > strings.Index(out, "Item 1") {
		t.Fatal(out, e)
	}
}
