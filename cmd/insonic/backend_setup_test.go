// SPDX-License-Identifier: Apache-2.0
package main

import (
	"strings"
	"testing"
)

func TestBootstrapAndServiceCommandInventory(t *testing.T) {
	for _, args := range [][]string{{"workspace", "plan", "--input", "plan.json"}, {"workspace", "preflight", "C:/new", "--input", "plan.json", "--secret-input"}, {"workspace", "provision", "C:/new", "--input", "plan.json", "--credential-backend", "session", "--secret-input"}, {"query", "services"}, {"query", "assistance-test", "--input", "test.json"}} {
		if _, e := parseInvocation(args); e != nil {
			t.Fatal(args, e)
		}
	}
	for _, action := range []string{"plan", "preflight", "provision"} {
		help := commandHelp([]string{"workspace", action})
		if !strings.Contains(help, "postgresql") || !strings.Contains(help, "arcadedb") {
			t.Fatal("incomplete bootstrap help", help)
		}
	}
}
