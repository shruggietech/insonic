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
	t.Run("a claimed accelerator stays handled", func(t *testing.T) {
		var handled []uintptr
		args := &ICoreWebView2AcceleratorKeyPressedEventArgs{vtbl: &_ICoreWebView2AcceleratorKeyPressedEventArgsVtbl{
			GetKeyEventKind: NewComProc(func(this uintptr, output *COREWEBVIEW2_KEY_EVENT_KIND) uintptr {
				*output = COREWEBVIEW2_KEY_EVENT_KIND_KEY_DOWN
				return 0
			}),
			GetVirtualKey: NewComProc(func(this uintptr, output *uint32) uintptr { *output = 0x46; return 0 }),
			GetPhysicalKeyStatus: NewComProc(func(this uintptr, output *internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS) uintptr {
				*output = internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS{}
				return 0
			}),
			PutHandled: NewComProc(func(this, value uintptr) uintptr { handled = append(handled, value); return 0 }),
		}}
		view := &Chromium{AcceleratorKeyCallback: func(key uint) bool { return key == 0x46 }, globalErrorCallback: func(err error) { t.Error(err) }}
		view.AcceleratorKeyPressed(nil, args)
		if len(handled) != 1 || handled[0] != 1 {
			t.Fatalf("claimed native accelerator was reset: %v", handled)
		}
		runtime.KeepAlive(args)
	})
	t.Run("native system key keeps its owned UI context and copied flags", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		owner := windows.GetCurrentThreadId()
		handle, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
		if handle == 0 {
			t.Fatal("system-key message-only fixture unavailable")
		}
		defer user.NewProc("DestroyWindow").Call(handle)
		view := &Chromium{hwnd: handle, globalErrorCallback: func(err error) { t.Error(err) }}
		if err := view.attachWindowLifetime(handle); err != nil {
			t.Fatal(err)
		}
		ownedWindows.Lock()
		previous := ownedWindows.systemKeyHandler
		ownedWindows.Unlock()
		defer SetNativeSystemKeyHandler(previous)
		const flags = 0x20210001
		var dispositions []uintptr
		args := &ICoreWebView2AcceleratorKeyPressedEventArgs{vtbl: &_ICoreWebView2AcceleratorKeyPressedEventArgsVtbl{
			GetKeyEventKind: NewComProc(func(this uintptr, output *COREWEBVIEW2_KEY_EVENT_KIND) uintptr {
				*output = COREWEBVIEW2_KEY_EVENT_KIND_SYSTEM_KEY_DOWN
				return 0
			}),
			GetVirtualKey:     NewComProc(func(this uintptr, output *uint32) uintptr { *output = 0x46; return 0 }),
			GetKeyEventLParam: NewComProc(func(this uintptr, output *uint32) uintptr { *output = flags; return 0 }),
			GetPhysicalKeyStatus: NewComProc(func(this uintptr, output *internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS) uintptr {
				*output = internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS{IsMenuKeyDown: 1}
				return 0
			}),
			PutHandled: NewComProc(func(this, value uintptr) uintptr { dispositions = append(dispositions, value); return 0 }),
		}}
		nativeCalls, legacyCalls := 0, 0
		view.AcceleratorKeyCallback = func(uint) bool { legacyCalls++; return false }
		SetNativeSystemKeyHandler(func(hwnd uintptr, event NativeSystemKeyEvent) bool {
			nativeCalls++
			if hwnd != handle || windows.GetCurrentThreadId() != owner || event.VirtualKey != 0x46 || event.LParam != flags || !event.PhysicalKeyStatus.IsMenuKeyDown {
				t.Error("native system-key metadata or UI ownership changed", event)
			}
			return true
		})
		view.AcceleratorKeyPressed(nil, args)
		if nativeCalls != 1 || legacyCalls != 0 || len(dispositions) != 1 || dispositions[0] != 1 {
			t.Fatal("native claimed event was duplicated or reset", nativeCalls, legacyCalls, dispositions)
		}
		SetNativeSystemKeyHandler(func(uintptr, NativeSystemKeyEvent) bool { return false })
		dispositions = nil
		view.AcceleratorKeyPressed(nil, args)
		if legacyCalls != 1 || len(dispositions) != 1 || dispositions[0] != 0 {
			t.Fatal("unclaimed event lost its existing handler", legacyCalls, dispositions)
		}
		other := make(chan bool, 1)
		go func() { other <- view.nativeSystemKeyHandler() == nil }()
		if !<-other {
			t.Fatal("another thread acquired the native system-key handler")
		}
		view.hwnd = 0
		if view.nativeSystemKeyHandler() != nil {
			t.Fatal("unowned native window acquired the system-key handler")
		}
		view.hwnd = handle
		runtime.KeepAlive(args)
	})
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
