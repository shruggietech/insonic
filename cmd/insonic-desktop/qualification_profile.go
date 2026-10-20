//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Only explicit native qualification modes consume the supervisor-owned path.
// Normal desktop launches retain the persistent product profile defaults.
func qualificationWebviewProfile() (string, error) {
	if runtime.GOOS != "windows" {
		return "", nil
	}
	profile := os.Getenv("INSONIC_NATIVE_WEBVIEW_PROFILE")
	if !filepath.IsAbs(profile) {
		return "", fmt.Errorf("owned native qualification profile is required")
	}
	parent, err := os.Stat(filepath.Dir(profile))
	if err != nil || !parent.IsDir() {
		return "", fmt.Errorf("owned native profile parent is unavailable")
	}
	if _, err = os.Lstat(profile); !os.IsNotExist(err) {
		return "", fmt.Errorf("native qualification requires a fresh owned profile")
	}
	return profile, nil
}
