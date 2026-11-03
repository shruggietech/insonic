// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnswerCLIInputsAndHumanLimitations(t *testing.T) {
	file := filepath.Join(t.TempDir(), "question.json")
	if err := os.WriteFile(file, []byte(`{"version":"1.3.0","question":"What changed?"}`), 0600); err != nil {
		t.Fatal(err)
	}
	inv, err := parseInvocation([]string{"answers", "prepare", "--input", file})
	if err != nil {
		t.Fatal(err)
	}
	op, id, raw, err := parseExplore(inv.Args)
	if err != nil || op != "answers.prepare" || id != "" || len(raw) == 0 {
		t.Fatal(op, id, err)
	}
	if _, err = parseInvocation([]string{"answers", "prepare"}); err == nil {
		t.Fatal("missing request accepted")
	}
	for _, action := range []string{"show", "export"} {
		id := contracts.ID()
		inv, err := parseInvocation([]string{"answers", action, id})
		if err != nil {
			t.Fatal(err)
		}
		_, got, _, err := parseExplore(inv.Args)
		if err != nil || got != id {
			t.Fatal(got, err)
		}
	}
	previous := operatorContext
	defer func() { operatorContext = previous }()
	operatorContext = inv
	out, err := renderHuman(contracts.Response{Result: map[string]any{"freshness": "stale", "cache_hit": false, "package": map[string]any{"kind": "answer-evidence-context", "id": contracts.ID(), "state": "partial", "items": []any{}, "next": "exact-continuation", "independent_families": 0, "diagnostics": []any{"graph_lag"}, "observation": map[string]any{"graph_state": "lagging"}, "context": map[string]any{"omitted": 12, "bytes": 256, "tokens": 256}}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"stale", "partial", "lagging", "12", "exact-continuation"} {
		if !strings.Contains(out, text) {
			t.Fatal("hidden limitation", text, out)
		}
	}
	for _, command := range []string{"answers prepare", "answers validate"} {
		help := commandHelp(strings.Fields(command))
		if !strings.Contains(help, "question") && !strings.Contains(help, "claims") {
			t.Fatal("missing shared input help", help)
		}
	}
}

func TestGroundedAnswerHumanSummary(t *testing.T) {
	old := operatorContext
	defer func() { operatorContext = old }()
	operatorContext.Path = []string{"answers", "show"}
	id := contracts.ID()
	out, err := renderHuman(contracts.Response{Result: map[string]any{"freshness": "stale", "accepted": true, "answer": map[string]any{"kind": "grounded-answer", "id": id, "work_id": contracts.ID(), "package_id": contracts.ID(), "route": "local", "provider": map[string]any{"model": "fixture"}, "evidence_count": 2, "independent_families": 1, "answer": map[string]any{"state": "partial", "claims": []any{map[string]any{"id": "c1", "kind": "paraphrase", "support": "uncertain", "text": "Sources disagree.", "uncertainty": "Conflicting evidence"}}}, "diagnostics": []any{"graph_lag"}}}})
	if err != nil || !strings.Contains(out, "current source freshness: stale") || !strings.Contains(out, "Sources disagree.") || !strings.Contains(out, "semantic support") {
		t.Fatal(out, err)
	}
}
