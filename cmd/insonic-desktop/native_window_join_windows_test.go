//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestNativeWindowOperationWaitsForItsOwnedUIDispatch(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	original, _ := windows.UTF16PtrFromString("queued operation pending")
	hwnd := exitWinCall("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(original)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("unshown message-only fixture unavailable")
	}
	defer exitWinCall("DestroyWindow", hwnd)
	var record nativeWindowMessage
	if unsafe.Sizeof(record) != 48 || unsafe.Offsetof(record.WParam) != 16 || unsafe.Offsetof(record.LParam) != 24 {
		t.Fatal("MSG ABI differs")
	}
	if err := awaitNativeWindowOperation(context.Background(), 0, func() {}); err == nil {
		t.Fatal("unowned UI completion accepted")
	}
	// No message pump runs here. Queuing cannot establish completion.
	blocked, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	failure := make(chan error, 1)
	go func() { failure <- awaitNativeWindowOperation(blocked, hwnd, func() {}) }()
	if err := <-failure; !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("unprocessed UI queue conferred success: %v", err)
	}
	title, _ := windows.UTF16PtrFromString("queued operation completed")
	var pin runtime.Pinner
	pin.Pin(title)
	defer pin.Unpin()
	ctx, stop := context.WithTimeout(context.Background(), 3*time.Second)
	defer stop()
	done := make(chan error, 1)
	uiThread := windows.GetCurrentThreadId()
	var operationThread uint32
	go func() {
		done <- awaitNativeWindowOperation(ctx, hwnd, func() {
			operationThread = windows.GetCurrentThreadId()
			exitWinCall("SetWindowTextW", hwnd, uintptr(unsafe.Pointer(title)))
		})
	}()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			if operationThread != uiThread {
				t.Fatal("native operation ran outside its owning UI thread")
			}
			var value [64]uint16
			exitWinCall("GetWindowTextW", hwnd, uintptr(unsafe.Pointer(&value[0])), uintptr(len(value)))
			if windows.UTF16ToString(value[:]) != "queued operation completed" {
				t.Fatal("completion preceded the actual queued native operation")
			}
			return
		default:
		}
		var message nativeWindowMessage
		if exitWinCall("PeekMessageW", uintptr(unsafe.Pointer(&message)), hwnd, 0, 0, 1) != 0 {
			exitWinCall("TranslateMessage", uintptr(unsafe.Pointer(&message)))
			exitWinCall("DispatchMessageW", uintptr(unsafe.Pointer(&message)))
		} else {
			time.Sleep(time.Millisecond)
		}
	}
	t.Fatal("owned UI join test exceeded its deadline")
}
