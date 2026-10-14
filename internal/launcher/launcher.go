// SPDX-License-Identifier: Apache-2.0
// Package launcher contains installation-only selected-client launch contracts.
package launcher

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type Plan struct {
	Mode        string            `json:"mode"`
	Executable  string            `json:"executable"`
	CommandLine string            `json:"command_line,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

func Build(directory, mode, systemDirectory string) (Plan, error) {
	p := Plan{Mode: mode}
	if !filepath.IsAbs(directory) || !filepath.IsAbs(systemDirectory) {
		return p, errors.New("absolute program and system directories are required")
	}
	directory = filepath.Clean(directory)
	target := filepath.Join(directory, "insonic-desktop.exe")
	if mode == "cli" {
		target = filepath.Join(directory, "insonic.exe")
	} else if mode != "desktop" {
		return p, errors.New("choose desktop or cli")
	}
	if f, e := os.Stat(target); e != nil || !f.Mode().IsRegular() {
		return p, errors.New("the selected client is not installed; use installer Change or Repair")
	}
	p.Executable = target
	if mode == "cli" {
		script := filepath.Join(directory, "insonic-session.cmd")
		if f, e := os.Stat(script); e != nil || !f.Mode().IsRegular() {
			return p, errors.New("the CLI session helper is missing; use installer Repair")
		}
		p.Executable = filepath.Join(systemDirectory, "cmd.exe")
		p.CommandLine = "\"" + p.Executable + "\" /D /Q /V:OFF /K \"\"%INSONIC_LAUNCH_SCRIPT%\"\""
		p.Environment = map[string]string{"INSONIC_LAUNCH_SCRIPT": script, "INSONIC_LAUNCH_CLI": target, "INSONIC_PROGRAM_DIRECTORY": directory}
	}
	return p, nil
}
func ValidateSelection(selection string) error {
	switch strings.ToLower(selection) {
	case "cli", "desktop", "both", "desktop,cli", "cli,desktop":
		return nil
	}
	return errors.New("select Desktop, CLI or both")
}
func ValidateDestination(directory string) error {
	if !filepath.IsAbs(directory) {
		return errors.New("choose an absolute program directory")
	}
	directory = filepath.Clean(directory)
	if directory == filepath.VolumeName(directory)+string(os.PathSeparator) {
		return errors.New("a volume root is not a program directory")
	}
	for p := directory; ; p = filepath.Dir(p) {
		if _, e := os.Stat(filepath.Join(p, ".insonic", "workspace.json")); e == nil {
			return errors.New("program files must be outside a library")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	if f, e := os.Lstat(directory); e == nil {
		if !f.IsDir() || f.Mode()&os.ModeSymlink != 0 {
			return errors.New("program destination must be a real directory")
		}
		entries, e := os.ReadDir(directory)
		if e != nil {
			return e
		}
		if len(entries) > 0 {
			b, e := os.ReadFile(filepath.Join(directory, "insonic-msi-ownership.json"))
			var marker struct {
				Kind   string `json:"kind"`
				Family string `json:"family"`
			}
			if (e != nil || json.Unmarshal(b, &marker) != nil || marker.Kind != "windows-msi-installation" || marker.Family != "insonic") && !registeredInstallation(directory) {
				return errors.New("choose an empty directory; an unrelated installation or user files already occupy this location")
			}
		}
	} else if !os.IsNotExist(e) {
		return e
	}
	return nil
}
func Preflight(directory, selection string) error {
	if e := ValidateSelection(selection); e != nil {
		return e
	}
	if e := ValidateDestination(directory); e != nil {
		return e
	}
	return NoActiveClients(directory)
}
