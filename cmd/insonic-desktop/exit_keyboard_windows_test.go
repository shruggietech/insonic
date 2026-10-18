//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"testing"
	"unsafe"
)

func TestNativeInputABIAndInvalidOwnershipGuard(t *testing.T) {
	var input exitKeyboardInput
	if unsafe.Sizeof(input) != 40 || unsafe.Offsetof(input.Key) != 8 || unsafe.Offsetof(input.Flags) != 12 || unsafe.Offsetof(input.Extra) != 24 {
		t.Fatal("native INPUT/KEYBDINPUT layout differs from Windows amd64 ABI")
	}
	if err := requireNativeInputTarget(0); err == nil {
		t.Fatal("invalid HWND accepted for global keyboard input")
	}
	foreign := exitWinCall("GetForegroundWindow")
	if foreign != 0 {
		if err := requireNativeInputTarget(foreign); err == nil {
			t.Fatal("test process must not own the operator's foreground window")
		}
	}
}
