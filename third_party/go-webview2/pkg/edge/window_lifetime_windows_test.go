//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"runtime"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestOwnedWindowDestructionClosesControllerOnceOnItsUIThread(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread := windows.GetCurrentThreadId()
	before := ClosedControllerCount()
	defer func() { ownedWindows.Lock(); ownedWindows.closedControllers = before; ownedWindows.Unlock() }()
	user := windows.NewLazySystemDLL("user32.dll")
	class, _ := windows.UTF16PtrFromString("STATIC")
	title, _ := windows.UTF16PtrFromString("owned WebView lifetime fixture")
	// A message-only window is never shown or activated.
	hwnd, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("message-only fixture window unavailable")
	}
	defer user.NewProc("DestroyWindow").Call(hwnd)
	if err := (&Chromium{}).attachWindowLifetime(0); err == nil {
		t.Fatal("invalid native window acquired a lifetime owner")
	}
	foreignThread := make(chan error, 1)
	go func() { foreignThread <- (&Chromium{}).attachWindowLifetime(hwnd) }()
	if err := <-foreignThread; err == nil {
		t.Fatal("another thread acquired the native window lifetime")
	}
	calls := 0
	vtbl := &_ICoreWebView2ControllerVtbl{Close: NewComProc(func(this uintptr) uintptr {
		calls++
		if windows.GetCurrentThreadId() != thread {
			t.Error("controller Close ran outside its UI thread")
		}
		return 0
	})}
	controller := &ICoreWebView2Controller{vtbl: vtbl}
	chromium := &Chromium{controller: controller, globalErrorCallback: func(err error) { t.Error(err) }}
	if err := chromium.attachWindowLifetime(hwnd); err != nil {
		t.Fatal(err)
	}
	if accepted, _, _ := user.NewProc("DestroyWindow").Call(hwnd); accepted == 0 {
		t.Fatal("normal native destruction failed")
	}
	if calls != 1 || !chromium.shuttingDown || ClosedControllerCount() != before+1 {
		t.Fatalf("controller Close calls=%d, shuttingDown=%t", calls, chromium.shuttingDown)
	}
	ownedWindows.Lock()
	retained := ownedWindows.values[hwnd] != nil
	ownedWindows.Unlock()
	if retained {
		t.Fatal("destroyed window retained its Go lifetime owner")
	}
	if alive, _, _ := user.NewProc("IsWindow").Call(hwnd); alive != 0 {
		t.Fatal("subclass suppressed the normal native window procedure")
	}
	if WindowCloseError() != nil {
		t.Fatal("successful Close retained an error")
	}
	t.Run("failed Close preserves destruction and error", func(t *testing.T) {
		ownedWindows.Lock()
		original := ownedWindows.failure
		ownedWindows.Unlock()
		defer func() { ownedWindows.Lock(); ownedWindows.failure = original; ownedWindows.Unlock() }()
		failed, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
		if failed == 0 {
			t.Fatal("message-only failure fixture unavailable")
		}
		defer user.NewProc("DestroyWindow").Call(failed)
		attempts := 0
		reported := false
		failedTable := &_ICoreWebView2ControllerVtbl{Close: NewComProc(func(uintptr) uintptr { attempts++; return 0x80004005 })}
		failedView := &Chromium{controller: &ICoreWebView2Controller{vtbl: failedTable}, globalErrorCallback: func(error) { reported = true }}
		if err := failedView.attachWindowLifetime(failed); err != nil {
			t.Fatal(err)
		}
		if accepted, _, _ := user.NewProc("DestroyWindow").Call(failed); accepted == 0 {
			t.Fatal("failed Close suppressed destruction")
		}
		if attempts != 1 || !reported || WindowCloseError() == nil {
			t.Fatal("failed Close lost its result", attempts, reported, WindowCloseError())
		}
		ownedWindows.Lock()
		retained := ownedWindows.values[failed] != nil
		ownedWindows.Unlock()
		if retained {
			t.Fatal("failed Close stranded the window owner")
		}
	})
}
