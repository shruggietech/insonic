//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"golang.org/x/sys/windows"
	"runtime"
	"sync"
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

func TestNativeMenuInputCannotTargetAnUnownedWindow(t *testing.T) {
	if err := sendNativeKeyboard(0, "Alt+F", []exitKeyboardInput{{Kind: 1, Key: 0x12}, {Kind: 1, Key: 0x46}}); err == nil {
		t.Fatal("menu opening input accepted without the owned foreground target")
	}
	if err := sendNativeKeyboard(0, "File/Exit X", []exitKeyboardInput{{Kind: 1, Key: 0x58}}); err == nil {
		t.Fatal("menu selection input accepted without the owned foreground target")
	}
}

//go:noinline
func menuLabelOnNewStack(depth int, menu uintptr) string {
	var frame [127]byte
	frame[depth%len(frame)] = byte(depth)
	var value string
	if depth == 0 {
		value = exitWinLabel(menu, 0)
	} else {
		value = menuLabelOnNewStack(depth-1, menu)
	}
	runtime.KeepAlive(frame)
	return value
}

func TestNativeMenuLabelsSurviveGoStackGrowth(t *testing.T) {
	menu := exitWinCall("CreateMenu")
	if menu == 0 {
		t.Fatal("fixture menu creation failed")
	}
	defer exitWinCall("DestroyMenu", menu)
	label, _ := windows.UTF16PtrFromString("&File")
	if exitWinCall("AppendMenuW", menu, 0, 1, uintptr(unsafe.Pointer(label))) == 0 {
		t.Fatal("fixture menu append failed")
	}
	var workers sync.WaitGroup
	failures := make(chan string, 128)
	for depth := 0; depth < 128; depth++ {
		workers.Add(1)
		go func(depth int) {
			defer workers.Done()
			if value := menuLabelOnNewStack(depth, menu); value != "&File" {
				failures <- value
			}
		}(depth)
	}
	workers.Wait()
	close(failures)
	for value := range failures {
		t.Errorf("native menu label corrupted across stack growth: %q", value)
	}
}
