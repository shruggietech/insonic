// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"strings"
	"testing"
	"unicode/utf16"
)

func TestLimitedAccountRejectsElevationAndAccountSubstitution(t *testing.T) {
	if e := limitedAccount("S-1-5-21-fixture", "S-1-5-21-fixture", false); e != nil {
		t.Fatal(e)
	}
	for _, test := range []struct {
		current, limited string
		elevated         bool
	}{{"S-1-5-21-fixture", "S-1-5-21-other", false}, {"S-1-5-21-fixture", "S-1-5-21-fixture", true}, {"", "S-1-5-21-fixture", false}} {
		if e := limitedAccount(test.current, test.limited, test.elevated); e == nil || !strings.Contains(e.Error(), "open insonic normally") {
			t.Fatal("accepted elevated/different account or missing correction", e)
		}
	}
}

func TestLimitedEnvironmentKeepsUnicodePathsAndReplacesPrivateVariables(t *testing.T) {
	base := []string{"SystemRoot=C:\\Windows", "insonic_launch_cli=old", "Path=limited-user-path", "=C:=C:\\Users\\Fixture"}
	block, e := unicodeEnvironment(base, map[string]string{"INSONIC_LAUNCH_CLI": "C:\\insonic & %TEMP% é\\insonic.exe", "INSONIC_LAUNCH_SCRIPT": "C:\\insonic & %TEMP% é\\insonic-session.cmd"})
	if e != nil {
		t.Fatal(e)
	}
	text := string(utf16.Decode(block))
	if !strings.HasSuffix(text, "\x00\x00") || strings.Count(strings.ToUpper(text), "INSONIC_LAUNCH_CLI=") != 1 || strings.Contains(text, "old") || !strings.Contains(text, "%TEMP% é") || !strings.Contains(text, "=C:=C:\\Users\\Fixture") {
		t.Fatalf("incorrect Unicode environment %q", text)
	}
	if !strings.Contains(text, "Path=limited-user-path") {
		t.Fatal("lost limited user's path")
	}
}

func TestLimitedEnvironmentRejectsNulAndUnapprovedOverrides(t *testing.T) {
	for _, test := range []struct {
		base    []string
		private map[string]string
	}{{[]string{"PATH=one\x00INJECTED=two"}, nil}, {[]string{"not-an-environment-entry"}, nil}, {nil, map[string]string{"PATH": "arbitrary"}}, {nil, map[string]string{"INSONIC_LAUNCH_CLI": "one\x00two"}}} {
		if _, e := unicodeEnvironment(test.base, test.private); e == nil {
			t.Fatal("accepted malformed environment")
		}
	}
	block, e := unicodeEnvironment(nil, nil)
	if e != nil || len(block) != 2 || block[0] != 0 || block[1] != 0 {
		t.Fatal("empty environment is not double terminated")
	}
}
