// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestInvocationPreservesLiteralAndAtomicValues(t *testing.T) {
	cases := []struct {
		args    []string
		machine bool
		want    []string
	}{
		{[]string{"media", "import", "--title", "--json", "--", "--workspace"}, false, []string{"media", "import", "--title", "--json", "--", "--workspace"}},
		{[]string{"search", "--", "--json"}, false, []string{"search", "--json"}},
		{[]string{"catalog", "export", "--", "--help"}, false, []string{"catalog", "export", "--help"}},
		{[]string{"--json", "work", "wait", "00000000-0000-4000-8000-000000000001", "--timeout-ms=1"}, true, []string{"work", "wait", "00000000-0000-4000-8000-000000000001", "--timeout-ms", "1"}},
	}
	for _, c := range cases {
		got, e := parseInvocation(c.args)
		if e != nil || got.Machine != c.machine || !reflect.DeepEqual(got.Args, c.want) {
			t.Fatalf("%q: %#v %v", c.args, got, e)
		}
	}
}

func TestTranscriptInvocationRequiresTargetOnlyForSingleInput(t *testing.T) {
	valid := [][]string{
		{"transcript", "import", "--manifest", "captions.json"},
		{"transcript", "import", "caption.srt", "--record", "Recording title"},
		{"transcript", "import", "--record", "Recording title", "--legacy-sidecar"},
	}
	for _, args := range valid {
		if _, err := parseInvocation(args); err != nil {
			t.Fatalf("valid invocation %q: %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"transcript", "import", "caption.srt"},
		{"transcript", "import"},
		{"transcript", "import", "--manifest", "captions.json", "--record", "Recording title"},
		{"transcript", "import", "caption.srt", "--manifest", "captions.json"},
		{"transcript", "import", "--manifest", "captions.json", "--legacy-sidecar"},
	} {
		if _, err := parseInvocation(args); err == nil {
			t.Fatalf("invalid invocation accepted: %q", args)
		}
	}
	help := commandHelp([]string{"transcript", "import"})
	if !strings.Contains(help, "--manifest") || !strings.Contains(help, "per-item") {
		t.Fatal("manifest target workflow missing from help")
	}
}

func TestInvocationMachineErrorsIndependentOfOrder(t *testing.T) {
	for _, args := range [][]string{{"--request-id", "bad", "--json", "doctor"}, {"--json", "--request-id", "bad", "doctor"}, {"unknown", "--json"}, {"doctor", "--unknown", "--json"}} {
		got, e := parseInvocation(args)
		if e == nil || !got.Machine {
			t.Fatalf("%q: %#v %v", args, got, e)
		}
	}
}

func TestEveryCommandHasUsefulEarlyHelp(t *testing.T) {
	for _, command := range commandDefinitions {
		for _, form := range [][]string{append(append([]string{}, command.Path...), "--help"), append(append([]string{}, command.Path...), "-h"), append([]string{"help"}, command.Path...)} {
			got, e := parseInvocation(form)
			if e != nil || !got.Help {
				t.Fatalf("help %q: %v", form, e)
			}
			text := commandHelp(got.Path)
			if len(text) < 60 {
				t.Fatalf("empty help %q: %s", form, text)
			}
		}
	}
	for _, args := range [][]string{{"workspace", "init", "--help"}, {"catalog", "export", "--help"}, {"search", "--help"}, {"settings", "set", "--input", "missing.json", "--help"}} {
		got, e := parseInvocation(args)
		if e != nil || !got.Help {
			t.Fatalf("early help %q: %v", args, e)
		}
	}
}

func TestEverySourceOptionUsesTheSharedArityInventory(t *testing.T) {
	files, e := filepath.Glob("*.go")
	if e != nil {
		t.Fatal(e)
	}
	pattern := regexp.MustCompile(`"(--[a-z][a-z-]*)"`)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") || file == "commands.go" {
			continue
		}
		raw, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		for _, m := range pattern.FindAllStringSubmatch(string(raw), -1) {
			if _, ok := optionDefinitions[m[1]]; !ok {
				t.Fatalf("unregistered option %s in %s", m[1], file)
			}
		}
	}
	for key, option := range optionDefinitions {
		if key == "--help" || key == "-h" || key == "--version" {
			continue
		}
		var command []string
		global := containsOption([]string{"--json", "--workspace", "--request-id", "--verbose", "--debug", "--quiet", "--log-level", "--log-file"}, key)
		if global {
			command = []string{"doctor"}
		} else {
			for _, c := range commandDefinitions {
				if containsOption(c.Options, key) {
					command = c.Path
					break
				}
			}
		}
		if command == nil {
			t.Fatal("option has no command", key)
		}
		cases := [][]string{append(append([]string{}, command...), key+"=", key)}
		if option.Value {
			cases = append(cases, append(append([]string{}, command...), key), append(append([]string{}, command...), key, "", key, "value"))
		} else {
			cases = append(cases, append(append([]string{}, command...), key, key))
		}
		for _, args := range cases {
			_, e := parseInvocation(args)
			if e == nil || !strings.Contains(e.Error(), key) {
				t.Fatalf("option error %q: %v", args, e)
			}
		}
		if option.Value {
			got, _ := parseInvocation(append(append([]string{}, command...), key, "--help"))
			if got.Help {
				t.Fatal("option value became help", key)
			}
		}
	}
}
func TestLiteralHelpAndInvalidGlobalGroups(t *testing.T) {
	for _, args := range [][]string{{"--", "help"}, {"--unknown"}, {"models", "--unknown"}, {"doctor", "--help=true"}} {
		if _, e := parseInvocation(args); e == nil {
			t.Fatal("invalid input admitted", args)
		}
	}
}
