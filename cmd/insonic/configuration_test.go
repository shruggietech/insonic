// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigurationCLIReadAndMutationShapes(t *testing.T) {
	id := contracts.ID()
	for _, family := range []string{"pipelines", "speakers", "terms"} {
		for _, args := range [][]string{{family, "list"}, {family, "show", id}} {
			op, item, raw, err := parseConfiguration(args)
			if err != nil || op != family+"."+args[1] || len(raw) != 0 || (args[1] == "show" && item != id) {
				t.Fatalf("%v: %s %s %s %v", args, op, item, raw, err)
			}
		}
		for _, args := range [][]string{{family, "set", id}, {family, "show"}, {family, "list", id}, {family, "unknown", id}} {
			if _, _, _, err := parseConfiguration(args); err == nil {
				t.Fatalf("invalid arguments admitted: %v", args)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "input.json")
	write := func(raw string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"max_hint_bytes":200,"speaker_ids":[]}`)
	op, item, raw, err := parseConfiguration([]string{"terms", "compile", "--input", path})
	if err != nil || op != "terms.compile" || item != "" || !json.Valid(raw) {
		t.Fatal(op, item, string(raw), err)
	}
	if _, _, _, err := parseConfiguration([]string{"terms", "compile", id, "--input", path}); err == nil {
		t.Fatal("compile accepted item identity")
	}
	write(`{"max_hint_bytes":200,"max_hint_bytes":500}`)
	if _, _, _, err := parseConfiguration([]string{"terms", "compile", "--input", path}); err == nil {
		t.Fatal("duplicate input keys admitted")
	}
	write(`{"expected_revision":0,"speaker":{"id":"` + id + `","name":"Example Person"},"aliases":[]}`)
	if _, _, _, err := parseConfiguration([]string{"speakers", "set", id, "--input", path}); err != nil {
		t.Fatal(err)
	}
	write(`{"expected_revision":0,"speaker":{"id":"` + id + `","name":"Example Person"},"aliases":[],"unexpected":true}`)
	if _, _, _, err := parseConfiguration([]string{"speakers", "set", id, "--input", path}); err == nil {
		t.Fatal("unknown mutation data admitted")
	}
}
