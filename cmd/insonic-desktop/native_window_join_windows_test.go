//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"runtime"
	"sync/atomic"
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

// The same queued operation must acknowledge actual UI-thread completion after
// a delay beyond the historical two-second allowance. No GUI is shown.
func TestNativeWindowOperationCalibratedAllowanceJoinsActualCompletion(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := windows.UTF16PtrFromString("STATIC")
	hwnd := exitWinCall("CreateWindowExW", 0, uintptr(unsafe.Pointer(class)), 0, 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("message-only fixture unavailable")
	}
	defer exitWinCall("DestroyWindow", hwnd)
	completed := make(chan error, 1)
	var calls atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		completed <- awaitNativeWindowOperation(nativeWindowJoinContext(ctx, 0), hwnd, func() { calls.Add(1) })
	}()
	time.Sleep(2200 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatal("queued work ran without the owning thread pump")
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-completed:
			if err != nil || calls.Load() != 1 {
				t.Fatalf("actual delayed completion was not joined: calls=%d err=%v", calls.Load(), err)
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
	t.Fatal("delayed owned completion unavailable")
}

func TestNativeDesktopObservationsPreserveUnknownAPIResults(t *testing.T) {
	unknown := observeNativeUserObject(0, false)
	if unknown.Name != "" || unknown.ReceivesInput != nil || unknown.Flags != nil || len(unknown.Errors) == 0 {
		t.Fatal("failed desktop queries invented observed values", unknown)
	}
	snapshot := nativeDesktopSnapshot(0, "unshown-test")
	if snapshot["owned"] != false || snapshot["errors"].(map[string]string)["ui_desktop"] == "" {
		t.Fatal("missing owned UI desktop was not explicit", snapshot)
	}
}
