//go:build desktop

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"github.com/shruggietech/insonic/desktop"
	"path/filepath"
	"testing"
)

func TestNativeQualificationProfileDoesNotChangeProductionDefault(t *testing.T) {
	bridge := &desktop.Bridge{}
	production := applicationOptions(bridge, nil, nil, false, "")
	if production.Windows != nil && production.Windows.WebviewUserDataPath != "" {
		t.Fatal("production profile changed")
	}
	profile := filepath.Join(t.TempDir(), "webview")
	fixture := applicationOptions(bridge, nil, nil, false, profile)
	if fixture.Windows.WebviewUserDataPath != profile || fixture.StartHidden {
		t.Fatal("native fixture lost its explicit owned profile or visible native action")
	}
}
