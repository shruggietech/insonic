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

type windowLifetime struct {
	chromium *Chromium
	closed   bool
}

// Native code retains only HWNDs and a static callback, never a Go object pointer.
var ownedWindows = struct {
	sync.Mutex
	values            map[uintptr]*windowLifetime
	failure           error
	closedControllers uint32
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
	ownedWindows.values[hwnd] = &windowLifetime{chromium: e}
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
