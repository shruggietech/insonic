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

func TestSpeakerDiscoveryCommands(t *testing.T) {
	id := contracts.ID()
	input := filepath.Join(t.TempDir(), "discovery.json")
	if e := os.WriteFile(input, []byte("{}"), 0600); e != nil {
		t.Fatal(e)
	}
	for _, action := range []string{"discover", "hypotheses", "resolve"} {
		args := []string{"speakers", action, id}
		if action != "hypotheses" {
			args = append(args, "--input", input)
		}
		op, item, _, e := parseExplore(args)
		if e != nil || op != "speakers."+action || item != id {
			t.Fatal(action, op, item, e)
		}
		help := commandHelp([]string{"speakers", action})
		if !strings.Contains(help, "RECORDING_UUID") {
			t.Fatal(help)
		}
	}
}
func TestSpeakerDiscoveryHumanSummary(t *testing.T) {
	prior := operatorContext
	defer func() { operatorContext = prior }()
	operatorContext.Path = []string{"speakers", "hypotheses"}
	var result any
	json.Unmarshal([]byte(`{"id":"recording","revision":1,"aggregate":{"coverage":{"total_cues":3,"selected_cues":1,"omitted_cues":2},"hypotheses":[{}],"decisions":[{"voice_id":"local","outcome":"supported-contextual","identity_status":"unregistered","name":"Alex","reasons":["evidence:self-identification"]}]}}`), &result)
	text, e := renderHuman(contracts.Response{Result: result})
	if e != nil {
		t.Fatal(e)
	}
	for _, part := range []string{"1 of 3", "2 omitted", "supported-contextual", "Alex", "unchanged"} {
		if !strings.Contains(text, part) {
			t.Fatalf("missing %q in %s", part, text)
		}
	}
}
