//go:build windows

// SPDX-License-Identifier: Apache-2.0
package launcher

import "testing"

func TestDocumentationShellDispatchPolicy(t *testing.T) {
	// SHELLEXECUTEINFO requires NO_CONSOLE to inherit the parent console
	// instead of creating a new one for console association handlers.
	if documentationShellMask&0x00008000 == 0 {
		t.Fatal("documentation dispatch can create a console window")
	}
	if documentationShellMask&0x00800000 != 0 {
		t.Fatal("documentation dispatch bypasses zone checks")
	}
	if documentationShellMask&0x00000100 == 0 || documentationShellMask&0x00000400 == 0 {
		t.Fatal("documentation dispatch must finish before COM cleanup and report errors through its caller")
	}
}
