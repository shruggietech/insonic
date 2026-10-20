//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestQualificationProfileRequiresFreshSupervisorOwnedPath(t *testing.T) {
	for _, value := range []string{"", "relative", filepath.Join(t.TempDir(), "missing", "webview")} {
		t.Setenv("INSONIC_NATIVE_WEBVIEW_PROFILE", value)
		if _, err := qualificationWebviewProfile(); err == nil {
			t.Fatal("unowned or missing profile accepted")
		}
	}
	profile := filepath.Join(t.TempDir(), "webview")
	t.Setenv("INSONIC_NATIVE_WEBVIEW_PROFILE", profile)
	if got, err := qualificationWebviewProfile(); err != nil || got != profile {
		t.Fatal(got, err)
	}
	if err := os.Mkdir(profile, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := qualificationWebviewProfile(); err == nil {
		t.Fatal("reused profile accepted")
	}
}
