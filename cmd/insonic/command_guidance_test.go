// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"github.com/shruggietech/insonic/internal/models"
	"github.com/shruggietech/insonic/internal/voicemodels"
	"github.com/shruggietech/insonic/schemas"
	"strings"
	"testing"
)

func TestEveryAcceptedCommandHasReviewedTaskGuidance(t *testing.T) {
	groups := map[string]bool{"": true}
	for _, c := range commandDefinitions {
		name := strings.Join(c.Path, " ")
		text := commandHelp(c.Path)
		for _, heading := range []string{"Purpose: ", "Workspace: ", "Paths: ", "Example: insonic ", "Placeholders: "} {
			if !strings.Contains(text, heading) {
				t.Errorf("%s lacks %s", name, heading)
			}
		}
		if c.Operands != "" && !strings.Contains(text, "Operands: ") {
			t.Errorf("%s lacks operand meaning", name)
		}
		for _, option := range c.Options {
			found := false
			for _, line := range strings.Split(text, "\n") {
				if strings.HasPrefix(line, "  "+option+" ") || strings.HasPrefix(line, "  "+option+";") {
					found = strings.Contains(line, "Meaning: ") && strings.Contains(line, "Default: ") && strings.Contains(line, "Status: ") && strings.Contains(line, "Repeat: ") && strings.Contains(line, "Conditions: ")
				}
			}
			if !found {
				t.Errorf("%s lacks complete guidance for %s", name, option)
			}
		}
		for n := 1; n < len(c.Path); n++ {
			groups[strings.Join(c.Path[:n], " ")] = true
		}
	}
	for group := range groups {
		if !strings.Contains(commandHelp(strings.Fields(group)), "Purpose: ") {
			t.Errorf("group %q lacks task guidance", group)
		}
	}
}

func TestActualEmittedExamplesPassParser(t *testing.T) {

	for _, c := range commandDefinitions {
		text := commandHelp(c.Path)
		found := false
		for _, line := range strings.Split(text, "\n") {
			if !strings.HasPrefix(line, "Example: insonic ") {
				continue
			}
			found = true
			args := strings.Fields(strings.TrimPrefix(line, "Example: insonic "))
			for i, v := range args {
				if strings.HasSuffix(v, "_UUID") || v == "UUID" {
					args[i] = "11111111-1111-4111-8111-111111111111"
				}
				switch v {
				case "DIRECTORY", "ABSOLUTE_DIRECTORY":
					args[i] = "/tmp/insonic-example"
				case "FILE", "INPUT_FILE", "MANIFEST_FILE":
					args[i] = "input.json"
				case "CURRENT_REVISION":
					args[i] = "1"
				case "REFERENCE":
					args[i] = "fixture"
				}
			}
			if _, err := parseInvocation(args); err != nil {
				t.Errorf("%s: %q: %v", strings.Join(c.Path, " "), args, err)
			}
		}
		if !found {
			t.Errorf("%s has no actual example", strings.Join(c.Path, " "))
		}
	}
}

func TestOperationSpecificGuidance(t *testing.T) {
	for _, action := range []string{"create", "show", "verify", "restore"} {
		text := commandHelp([]string{"backup", action})
		if strings.Contains(text, "DIRECTORY_OR_BACKUP_UUID") {
			t.Errorf("backup %s accepts only a directory", action)
		}
	}
	for _, path := range [][]string{{"query", "list"}, {"views", "show"}} {
		for _, c := range commandDefinitions {
			if strings.Join(c.Path, " ") == strings.Join(path, " ") && containsOption(c.Options, "--input") {
				t.Errorf("%v advertises unusable --input", path)
			}
		}
	}
	if text := commandHelp([]string{"recordings", "mappings"}); !strings.Contains(text, "mapping-row UUID") || strings.Contains(text, "Exclusive local speaker identity cursor") {
		t.Fatal("mapping help must describe the returned row cursor")
	}
	if strings.Contains(commandHelp([]string{"work", "results"}), "Omit --after ") {
		t.Fatal("work results advertises the wrong cursor")
	}
	for _, group := range []string{"alias", "source"} {
		_, e := parseInvocation([]string{"models", group, "remove", "11111111-1111-4111-8111-111111111111", "--expected-revision", "0"})
		if e == nil {
			t.Errorf("models %s remove parser accepts zero revision", group)
		}
	}
}

func TestReviewedHelpInventoryMatchesParser(t *testing.T) {
	if len(guidance.Commands) != len(commandDefinitions) || len(guidance.Options) != len(optionDefinitions) {
		t.Fatal("reviewed inventory and parser counts differ")
	}
	for _, c := range commandDefinitions {
		if strings.TrimSpace(guidance.Commands[strings.Join(c.Path, " ")]) == "" {
			t.Fatal("missing reviewed task", c.Path)
		}
	}
	for key := range optionDefinitions {
		m, ok := guidance.Options[key]
		if !ok || strings.TrimSpace(m.Meaning) == "" || strings.TrimSpace(m.Default) == "" || strings.TrimSpace(m.Conditions) == "" {
			t.Fatal("missing reviewed option contract", key)
		}
	}
}

func TestPositionalAndAdapterHelpTemplatesMeetSemanticContracts(t *testing.T) {
	for _, op := range []string{"media.tools", "processing.tools", "models.manifest"} {
		template, contract := schemas.InputHelp(op)
		if template == nil || contract == nil {
			t.Fatal("missing complete positional input", op)
		}
		switch op {
		case "media.tools":
			if err := schemas.ValidateMediaTools(template); err != nil {
				t.Fatal(err)
			}
		case "processing.tools":
			if err := schemas.ValidateDocument(template); err != nil {
				t.Fatal(err)
			}
		case "models.manifest":
			var manifest models.Manifest
			if err := models.DecodeManifest(template, &manifest); err != nil {
				t.Fatal(err)
			}
		}
	}
	raw, _ := schemas.InputHelp("recordings.match")
	var match struct{ Adapter voicemodels.Adapter }
	if err := json.Unmarshal(raw, &match); err != nil {
		t.Fatal(err)
	}
	if err := voicemodels.ValidateAdapter(match.Adapter); err != nil {
		t.Fatal("schema-valid help adapter fails runtime semantics", err)
	}
}

func TestGlobalHelpExposesValueArity(t *testing.T) {
	page := commandHelp([]string{"media", "list"})
	for _, flag := range []string{"--workspace", "--request-id", "--log-level", "--log-file"} {
		if !strings.Contains(page, "  "+flag+" VALUE; Meaning:") {
			t.Fatal("missing global value arity", flag)
		}
	}
	if !strings.Contains(page, "  --json; Meaning:") || strings.Contains(page, "  --json VALUE;") {
		t.Fatal("switch arity changed")
	}
}
