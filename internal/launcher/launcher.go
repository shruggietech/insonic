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
	if mode == "documentation" {
		if _, err := DocumentationEntry(directory); err != nil {
			return p, err
		}
		p.Executable = filepath.Join(directory, "insonic-launcher.exe")
		return p, nil
	}
	target := filepath.Join(directory, "insonic-desktop.exe")
	if mode == "cli" {
		target = filepath.Join(directory, "insonic.exe")
	} else if mode != "desktop" {
		return p, errors.New("choose desktop, cli or documentation")
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

// canonicalPath resolves the nearest existing ancestor and then appends only
// the missing suffix. A fresh destination beneath an alias therefore retains
// its physical ancestry without requiring creation of any directory.
func canonicalPath(name string) (string, error) {
	if !filepath.IsAbs(name) {
		return "", errors.New("an absolute path is required")
	}
	current := filepath.Clean(name)
	var missing []string
	for {
		_, e := os.Lstat(current)
		if e == nil {
			resolved, e := filepath.EvalSymlinks(current)
			if e != nil {
				return "", e
			}
			if len(missing) > 0 {
				f, e := os.Stat(resolved)
				if e != nil {
					return "", e
				}
				if !f.IsDir() {
					return "", errors.New("the existing parent is not a directory")
				}
			}
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(e) {
			return "", e
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", e
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func containsPath(directory, name string) bool {
	relative, e := filepath.Rel(directory, name)
	return e == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(os.PathSeparator)) && !filepath.IsAbs(relative)
}

func canonicalProcessImage(image string) (string, error) {
	if resolved, e := canonicalPath(image); e == nil {
		return resolved, nil
	}
	// Protected system images can deny file metadata even when their process
	// reports its name. Resolve the directory without demanding file access.
	directory, e := canonicalPath(filepath.Dir(image))
	if e != nil {
		return "", e
	}
	return filepath.Join(directory, filepath.Base(image)), nil
}

func outsideLibraries(directory string) error {
	for p := directory; ; p = filepath.Dir(p) {
		if _, e := os.Stat(filepath.Join(p, ".insonic", "workspace.json")); e == nil {
			return errors.New("program files must be outside a library")
		} else if !os.IsNotExist(e) {
			return errors.New("cannot inspect library ancestry; choose an accessible program directory")
		}
		if filepath.Dir(p) == p {
			break
		}
	}
	return nil
}

func ValidateDestination(directory string) error {
	if !filepath.IsAbs(directory) {
		return errors.New("choose an absolute program directory")
	}
	directory = filepath.Clean(directory)
	if directory == filepath.VolumeName(directory)+string(os.PathSeparator) {
		return errors.New("a volume root is not a program directory")
	}
	physical, e := canonicalPath(directory)
	if e != nil {
		return errors.New("cannot resolve the program directory; choose an accessible directory with valid aliases")
	}
	if physical == filepath.VolumeName(physical)+string(os.PathSeparator) {
		return errors.New("a volume root is not a program directory")
	}
	if e := outsideLibraries(directory); e != nil {
		return e
	}
	if physical != directory {
		if e := outsideLibraries(physical); e != nil {
			return e
		}
	}
	if f, e := os.Stat(directory); e == nil {
		if !f.IsDir() {
			return errors.New("program destination must resolve to a directory")
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

func maintenanceRoots(directory, registered string) ([]string, error) {
	physical, e := canonicalPath(directory)
	if e != nil {
		return nil, errors.New("cannot resolve the program directory; correct its aliases before maintenance")
	}
	roots := []string{physical}
	if registered == "" {
		return roots, nil
	}
	if !filepath.IsAbs(registered) || filepath.Dir(filepath.Clean(registered)) == filepath.Clean(registered) {
		return nil, errors.New("registered installation directory is invalid; correct it before maintenance")
	}
	previous, e := canonicalPath(registered)
	if e != nil {
		return nil, errors.New("cannot resolve the registered installation directory before maintenance")
	}
	if !strings.EqualFold(physical, previous) {
		roots = append(roots, previous)
	}
	return roots, nil
}

func shortcutCollision(path string, owned bool) error {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".lnk") {
		return errors.New("installer shortcut path is invalid")
	}
	info, e := os.Lstat(path)
	if os.IsNotExist(e) {
		return nil
	}
	if e != nil {
		return errors.New("cannot inspect the selected shortcut before installation")
	}
	if !info.Mode().IsRegular() || !owned {
		return errors.New("the selected shortcut already exists outside MSI ownership; deselect that integration or move the existing shortcut before retrying")
	}
	return nil
}
