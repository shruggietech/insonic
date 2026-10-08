// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDesktopCLISharedRoutes(t *testing.T) {
	id := contracts.ID()
	for _, args := range [][]string{{"settings", "show"}, {"recordings", "cues", id}} {
		operation, item, _, e := parseDesktopOperation(args)
		if e != nil || operation != args[0]+"."+args[1] || args[0] == "recordings" && item != id {
			t.Fatal("shared desktop CLI route", args, e)
		}
	}
	path := filepath.Join(t.TempDir(), "input.json")
	if os.WriteFile(path, []byte(`{"revision":1}`), 0600) != nil {
		t.Fatal("fixture")
	}
	if operation, item, _, e := parseDesktopOperation([]string{"media", "playback", id, "--input", path}); e != nil || operation != "media.playback" || item != id {
		t.Fatal("playback CLI", e)
	}
	if os.WriteFile(path, []byte(`{"revision":1,"path":"arbitrary"}`), 0600) != nil {
		t.Fatal("fixture")
	}
	if _, _, _, e := parseDesktopOperation([]string{"media", "playback", id, "--input", path}); e == nil {
		t.Fatal("playback path accepted")
	}
	if _, _, _, e := parseDesktopOperation([]string{"settings", "set"}); e == nil {
		t.Fatal("mutation without typed input")
	}
	for _, section := range []string{"media_tools", "processing_tools", "appearance"} {
		raw := `{"section":"` + section + `","revision":"` + strings.Repeat("a", 64) + `","value":null}`
		if e := os.WriteFile(path, []byte(raw), 0600); e != nil {
			t.Fatal(e)
		}
		if operation, _, _, e := parseDesktopOperation([]string{"settings", "set", "--input", path}); e != nil || operation != "settings.set" {
			t.Fatal("settings reset CLI parity", section, e)
		}
	}
}
