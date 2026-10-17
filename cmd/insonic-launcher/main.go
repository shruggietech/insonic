// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/internal/launcher"
	"os"
	"path/filepath"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	mode := ""
	interactive := false
	fail := func(e error) int {
		if interactive && mode == "documentation" {
			launcher.ReportDocumentationError()
		}
		json.NewEncoder(os.Stderr).Encode(map[string]any{"status": "failed", "message": e.Error()})
		return 1
	}
	if len(args) == 3 && args[0] == "--preflight" {
		if e := launcher.Preflight(args[1], args[2]); e != nil {
			return fail(e)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "passed", "active_clients": "absent", "scope": "per-user"})
		return 0
	}
	if len(args) == 3 && args[0] == "--check-shortcut" {
		if e := launcher.CheckShortcut(args[1], args[2]); e != nil {
			return fail(e)
		}
		json.NewEncoder(os.Stdout).Encode(map[string]any{"status": "passed", "shortcut_ownership": "verified", "scope": "per-user"})
		return 0
	}
	describe := false
	for _, a := range args {
		switch a {
		case "--documentation":
			if mode != "" {
				return fail(fmt.Errorf("choose one action"))
			}
			mode = "documentation"
		case "--interactive":
			interactive = true
		case "--describe":
			describe = true
		case "--desktop":
			if mode != "" {
				return fail(fmt.Errorf("choose one client"))
			}
			mode = "desktop"
		case "--cli":
			if mode != "" {
				return fail(fmt.Errorf("choose one client"))
			}
			mode = "cli"
		default:
			return fail(fmt.Errorf("use --desktop, --cli, --documentation, --describe, --preflight DIRECTORY SELECTION or --check-shortcut FILE COMPONENT"))
		}
	}
	if interactive && mode != "documentation" {
		return fail(fmt.Errorf("interactive warnings require documentation mode"))
	}
	exe, e := os.Executable()
	if e != nil {
		return fail(e)
	}
	directory := filepath.Dir(exe)
	if mode == "" && describe {
		if _, e := os.Stat(filepath.Join(directory, "insonic-desktop.exe")); e == nil {
			mode = "desktop"
		} else {
			mode = "cli"
		}
	}
	system, e := launcher.SystemDirectory()
	if e != nil {
		return fail(e)
	}
	p, e := launcher.Build(directory, mode, system)
	if e != nil {
		return fail(e)
	}
	if describe {
		json.NewEncoder(os.Stdout).Encode(p)
		return 0
	}
	if e := launcher.Launch(p); e != nil {
		return fail(e)
	}
	return 0
}
