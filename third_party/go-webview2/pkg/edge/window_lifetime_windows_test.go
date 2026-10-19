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
	var releases []string
	release := func(name string) ComProc {
		return NewComProc(func(uintptr) uintptr {
			if windows.GetCurrentThreadId() != thread {
				t.Error("COM Release ran outside its UI thread")
			}
			releases = append(releases, name)
			return 0
		})
	}
	vtbl := &_ICoreWebView2ControllerVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("controller")}, Close: NewComProc(func(this uintptr) uintptr {
		calls++
		if windows.GetCurrentThreadId() != thread {
			t.Error("controller Close ran outside its UI thread")
		}
		return 0
	})}
	controller := &ICoreWebView2Controller{vtbl: vtbl}
	chromium := &Chromium{controller: controller, webview: &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("webview")}}}, environment: &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("environment")}}}, globalErrorCallback: func(err error) { t.Error(err) }}
	if err := chromium.attachWindowLifetime(hwnd); err != nil {
		t.Fatal(err)
	}
	if accepted, _, _ := user.NewProc("DestroyWindow").Call(hwnd); accepted == 0 {
		t.Fatal("normal native destruction failed")
	}
	if len(releases) != 3 || releases[0] != "webview" || releases[1] != "controller" || releases[2] != "environment" || chromium.webview != nil || chromium.controller != nil || chromium.environment != nil {
		t.Fatalf("owned COM references survived close: %v", releases)
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
		failedReleases := 0
		failedTable := &_ICoreWebView2ControllerVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { failedReleases++; return 0 })}, Close: NewComProc(func(uintptr) uintptr { attempts++; return 0x80004005 })}
		failedView := &Chromium{controller: &ICoreWebView2Controller{vtbl: failedTable}, globalErrorCallback: func(error) { reported = true }}
		if err := failedView.attachWindowLifetime(failed); err != nil {
			t.Fatal(err)
		}
		if accepted, _, _ := user.NewProc("DestroyWindow").Call(failed); accepted == 0 {
			t.Fatal("failed Close suppressed destruction")
		}
		if failedReleases != 1 {
			t.Fatal("failed Close retained its owned interface")
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
	t.Run("owned COM apartment is balanced on its original thread", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		query := windows.NewLazySystemDLL("ole32.dll").NewProc("CoGetApartmentType")
		apartment := func() uintptr {
			var kind, qualifier uint32
			hr, _, _ := query.Call(uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&qualifier)))
			return uintptr(uint32(hr))
		}
		if apartment() != 0x800401f0 {
			t.Fatal("fixture needs a fresh uninitialized worker apartment")
		}
		ownedWindows.Lock()
		priorThread, priorOwned, priorFinalized := ownedWindows.apartmentThread, ownedWindows.apartmentOwned, ownedWindows.apartmentFinalized
		ownedWindows.Unlock()
		if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if apartment() == 0 {
				windows.CoUninitialize()
			}
			ownedWindows.Lock()
			ownedWindows.apartmentThread, ownedWindows.apartmentOwned, ownedWindows.apartmentFinalized = priorThread, priorOwned, priorFinalized
			ownedWindows.Unlock()
		}()
		recordOwnedCOMApartment()
		other := make(chan error, 1)
		go func() { other <- FinalizeCOMApartment() }()
		if err := <-other; err == nil {
			t.Fatal("another thread finalized the COM apartment")
		}
		if err := FinalizeCOMApartment(); err != nil {
			t.Fatal(err)
		}
		if apartment() != 0x800401f0 {
			t.Fatal("the binding's COM initialization survived finalization")
		}
		if err := FinalizeCOMApartment(); err != nil {
			t.Fatal("idempotent finalization failed", err)
		}
	})

}
