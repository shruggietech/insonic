//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"golang.org/x/sys/windows"
	"runtime"
	"testing"
	"unsafe"
)

func TestNativeInputObservationLeavesOwnedQueuedKeysUnchanged(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := startNativeExitInputTrace(0); err == nil {
		t.Fatal("unowned input observer accepted")
	}
	class, _ := windows.UTF16PtrFromString("STATIC")
	hwnd := exitWinCall("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("unshown owned fixture unavailable")
	}
	defer exitWinCall("DestroyWindow", hwnd)
	if err := startNativeExitInputTrace(hwnd); err != nil {
		t.Fatal(err)
	}
	defer stopNativeExitInputTrace()
	if exitWinCall("PostMessageW", hwnd, 0x100, 0x58, 0x12340001) == 0 {
		t.Fatal("owned key message rejected")
	}
	var message nativeWindowMessage
	if exitWinCall("PeekMessageW", uintptr(unsafe.Pointer(&message)), hwnd, 0x100, 0x100, 1) == 0 || message.Window != hwnd || message.Message != 0x100 || message.WParam != 0x58 || message.LParam != 0x12340001 {
		t.Fatal("input observation modified the original message")
	}
	trace := nativeExitInputTrace.Load()
	trace.mu.Lock()
	events := append([]nativeInputEvent(nil), trace.events...)
	trace.mu.Unlock()
	if len(events) != 1 || events[0].Hook != 3 || events[0].Window != hwnd || events[0].Key != 0x58 || events[0].Context != 0x12340001 {
		t.Fatal("owned key removal was not observed", events)
	}
	if err := stopNativeExitInputTrace(); err != nil {
		t.Fatal(err)
	}
	if exitWinCall("PostMessageW", hwnd, 0x100, 0x58, 0x12340002) == 0 {
		t.Fatal("post-removal key rejected")
	}
	exitWinCall("PeekMessageW", uintptr(unsafe.Pointer(&message)), hwnd, 0x100, 0x100, 1)
	trace.mu.Lock()
	defer trace.mu.Unlock()
	if len(trace.events) != 1 {
		t.Fatal("removed observer still collected messages")
	}
}

func TestNativeInputObservationBoundsAndFiltersItsQualificationKeys(t *testing.T) {
	trace := &nativeInputTrace{}
	trace.observe(3, &nativeWindowMessage{Message: 0x100, WParam: 'A'})
	trace.observe(3, &nativeWindowMessage{Message: 0x999, WParam: 'X'})
	if len(trace.events) != 0 {
		t.Fatal("recorded non-qualification input")
	}
	for i := 0; i < 100; i++ {
		trace.observe(3, &nativeWindowMessage{Message: 0x100, WParam: 'X'})
	}
	if len(trace.events) != 96 || trace.dropped != 4 {
		t.Fatal("input observation exceeded its bound")
	}
}
