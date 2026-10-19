//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"golang.org/x/sys/windows"
	"testing"
	"unsafe"
)

func TestNativeFileMnemonicRequiresActualAltSystemKey(t *testing.T) {
	for _, tc := range []struct {
		message    uint32
		key, flags uintptr
		want       bool
	}{
		{0x104, 0x46, 1 << 29, true},
		{0x100, 0x46, 1 << 29, false},
		{0x104, 0x46, 0, false},
		{0x104, 0x58, 1 << 29, false},
		{0x104, 0x46, (1 << 29) | (1 << 30), false},
		{0x105, 0x46, 1 << 29, false},
	} {
		if got := nativeFileMnemonic(tc.message, tc.key, tc.flags); got != tc.want {
			t.Fatalf("message=%x key=%x flags=%x: %t", tc.message, tc.key, tc.flags, got)
		}
	}
	var msg nativeMenuMessage
	if unsafe.Sizeof(msg) != 48 || unsafe.Offsetof(msg.WParam) != 16 || unsafe.Offsetof(msg.LParam) != 24 {
		t.Fatal("MSG differs from Windows amd64 ABI")
	}
}

func TestNativeMenuThreadRoutingInstallsAndCleansUpWithoutGlobalInput(t *testing.T) {
	stop, err := installNativeMenuRouting(false)
	if err != nil {
		t.Fatal(err)
	}
	// Exercise the actual callback ABI on this thread's private message queue.
	var msg nativeMenuMessage
	exitWinCall("PeekMessageW", uintptr(unsafe.Pointer(&msg)), 0, 0, 0, 0)
	for _, kind := range []uintptr{0x8001, 0x104} {
		if exitWinCall("PostThreadMessageW", uintptr(windows.GetCurrentThreadId()), kind, 0x46, 1<<29) == 0 {
			t.Fatal("owned thread message rejected")
		}
		if exitWinCall("PeekMessageW", uintptr(unsafe.Pointer(&msg)), 0, kind, kind, 1) == 0 || uintptr(msg.Message) != kind || msg.Window != 0 {
			t.Fatal("thread-only message was changed or lost", msg)
		}
	}
	if err = stop(); err != nil {
		t.Fatal(err)
	}
}
