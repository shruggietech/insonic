//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var lifetimeComctl = windows.NewLazySystemDLL("comctl32.dll")
var lifetimeSetSubclass = lifetimeComctl.NewProc("SetWindowSubclass")
var lifetimeRemoveSubclass = lifetimeComctl.NewProc("RemoveWindowSubclass")
var lifetimeDefSubclass = lifetimeComctl.NewProc("DefSubclassProc")
var lifetimeGetOwner = windows.NewLazySystemDLL("user32.dll").NewProc("GetWindowThreadProcessId")

// NativeSystemKeyEvent copies accelerator metadata; no borrowed COM pointer escapes.
type NativeSystemKeyEvent struct {
	VirtualKey        uint
	LParam            uint32
	PhysicalKeyStatus COREWEBVIEW2_PHYSICAL_KEY_STATUS
}

// SetNativeSystemKeyHandler configures the single host application's native
// system-key route. Call before running the UI and clear after its run returns.
// Handlers must only post native work, never enter modal UI inside a COM callback.
func SetNativeSystemKeyHandler(handler func(uintptr, NativeSystemKeyEvent) bool) {
	ownedWindows.Lock()
	ownedWindows.systemKeyHandler = handler
	ownedWindows.Unlock()
}

func (e *Chromium) nativeSystemKeyHandler() func(uintptr, NativeSystemKeyEvent) bool {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[e.hwnd]
	if entry == nil || entry.chromium != e || entry.closed || entry.thread != windows.GetCurrentThreadId() {
		return nil
	}
	return ownedWindows.systemKeyHandler
}

type windowLifetime struct {
	chromium *Chromium
	thread   uint32
	closed   bool
}

// Native code retains only HWNDs and a static callback, never a Go object pointer.
var ownedWindows = struct {
	sync.Mutex
	values                             map[uintptr]*windowLifetime
	failure                            error
	closedControllers                  uint32
	apartmentThread                    uint32
	apartmentOwned, apartmentFinalized bool
	systemKeyHandler                   func(uintptr, NativeSystemKeyEvent) bool
}{values: make(map[uintptr]*windowLifetime)}

var lifetimeSubclass uintptr

func init() { lifetimeSubclass = syscall.NewCallback(lifetimeWindowProc) }

func lifetimeWindowProc(hwnd, message, wparam, lparam, id, data uintptr) uintptr {
	if message == 0x0002 || message == 0x0082 { // WM_DESTROY / WM_NCDESTROY
		ownedWindows.Lock()
		entry := ownedWindows.values[hwnd]
		closeController := entry != nil && !entry.closed
		if closeController {
			entry.closed = true
		}
		if message == 0x0082 {
			delete(ownedWindows.values, hwnd)
		}
		ownedWindows.Unlock()
		if closeController {
			entry.chromium.ShuttingDown()
			if controller := entry.chromium.controller; controller != nil {
				if err := controller.Close(); err != nil {
					failure := fmt.Errorf("close WebView controller: %w", err)
					ownedWindows.Lock()
					if ownedWindows.failure == nil {
						ownedWindows.failure = failure
					}
					ownedWindows.Unlock()
					entry.chromium.globalErrorCallback(failure)
				} else {
					ownedWindows.Lock()
					ownedWindows.closedControllers++
					ownedWindows.Unlock()
				}
			}
			entry.chromium.releaseOwnedInterfaces()
		}
		if message == 0x0082 {
			lifetimeRemoveSubclass.Call(hwnd, lifetimeSubclass, id)
		}
	}
	result, _, _ := lifetimeDefSubclass.Call(hwnd, message, wparam, lparam)
	return result
}

// WindowCloseError preserves a failed Close without interrupting ordinary
// native destruction or the application's remaining shutdown callbacks.
func WindowCloseError() error {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	return ownedWindows.failure
}

// ClosedControllerCount reports completed Close operations, not merely requests.
func ClosedControllerCount() uint32 {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	return ownedWindows.closedControllers
}

func (e *Chromium) attachWindowLifetime(hwnd uintptr) error {
	for _, procedure := range []*windows.LazyProc{lifetimeSetSubclass, lifetimeRemoveSubclass, lifetimeDefSubclass, lifetimeGetOwner} {
		if err := procedure.Find(); err != nil {
			return err
		}
	}
	var pid uint32
	thread, _, _ := lifetimeGetOwner.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || pid != uint32(os.Getpid()) || uint32(thread) != windows.GetCurrentThreadId() {
		return fmt.Errorf("WebView lifetime requires its owned UI-thread window")
	}
	ownedWindows.Lock()
	if ownedWindows.values[hwnd] != nil {
		ownedWindows.Unlock()
		return fmt.Errorf("WebView window lifetime already registered")
	}
	ownedWindows.values[hwnd] = &windowLifetime{chromium: e, thread: uint32(thread)}
	ownedWindows.Unlock()
	if accepted, _, _ := lifetimeSetSubclass.Call(hwnd, lifetimeSubclass, lifetimeSubclass, 0); accepted == 0 {
		ownedWindows.Lock()
		delete(ownedWindows.values, hwnd)
		ownedWindows.Unlock()
		return fmt.Errorf("WebView window lifetime registration failed")
	}
	return nil
}

// Close synchronously releases the WebView's browser resources and event cycles.
// It must run on the owning UI thread before the parent window is destroyed.
func (i *ICoreWebView2Controller) Close() error {
	hr, _, _ := i.vtbl.Close.Call(uintptr(unsafe.Pointer(i)))
	if windows.Handle(hr) != windows.S_OK {
		return windows.Errno(hr)
	}
	return nil
}

// Close stops the WebView; Release balances the interfaces retained by Embed.
func (e *Chromium) releaseOwnedInterfaces() {
	if value := e.webview; value != nil {
		e.webview = nil
		value.vtbl.Release.Call(uintptr(unsafe.Pointer(value)))
	}
	if value := e.controller; value != nil {
		e.controller = nil
		value.Release()
	}
	if value := e.environment; value != nil {
		e.environment = nil
		value.vtbl.Release.Call(uintptr(unsafe.Pointer(value)))
	}
}

func recordOwnedCOMApartment() {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	ownedWindows.apartmentThread = windows.GetCurrentThreadId()
	ownedWindows.apartmentOwned = true
	ownedWindows.apartmentFinalized = false
}

// FinalizeCOMApartment balances the binding's existing successful CoInitializeEx
// after all owned windows are destroyed. It must run on that original UI thread.
// Close errors remain separate; failure cleanup must still release its apartment.
func FinalizeCOMApartment() error {
	ownedWindows.Lock()
	if len(ownedWindows.values) != 0 {
		ownedWindows.Unlock()
		return fmt.Errorf("WebView windows remain before COM apartment cleanup")
	}
	if ownedWindows.apartmentFinalized {
		ownedWindows.Unlock()
		return nil
	}
	if !ownedWindows.apartmentOwned || ownedWindows.apartmentThread != windows.GetCurrentThreadId() {
		ownedWindows.Unlock()
		return fmt.Errorf("WebView COM cleanup requires its initialized owning UI thread")
	}
	ownedWindows.apartmentOwned = false
	ownedWindows.apartmentFinalized = true
	ownedWindows.Unlock()
	windows.CoUninitialize()
	return nil
}
