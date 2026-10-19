//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MSG from winuser.h on Windows amd64. Native callbacks receive the pointer
// directly, avoiding an integer-to-Go-pointer conversion.
type nativeMenuMessage struct {
	Window         uintptr
	Message        uint32
	_              uint32
	WParam, LParam uintptr
	Time           uint32
	X, Y           int32
	Private        uint32
}

func nativeFileMnemonic(message uint32, key, flags uintptr) bool {
	return message == 0x104 && key == 0x46 && flags&(1<<29) != 0 && flags&(1<<30) == 0
}

// Wails forwards WebView accelerator keys as WM_KEYDOWN, dropping their Alt
// system-key context. Route the actual Alt+F on this UI thread before that
// conversion. No desktop-wide hotkey or other-process hook is installed.
func installNativeMenuRouting(diagnostic bool) (func() error, error) {
	runtime.LockOSThread()
	thread := windows.GetCurrentThreadId()
	if thread == 0 {
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("native menu UI thread unavailable")
	}
	api := windows.NewLazySystemDLL("user32.dll")
	next := api.NewProc("CallNextHookEx")
	ancestor := api.NewProc("GetAncestor")
	owner := api.NewProc("GetWindowThreadProcessId")
	foreground := api.NewProc("GetForegroundWindow")
	getMenu := api.NewProc("GetMenu")
	keyState := api.NewProc("GetKeyState")
	if diagnostic {
		fmt.Fprintf(os.Stderr, "native-menu-thread=%d\n", thread)
	}
	debugCount := 0
	callbackFunction := func(code int32, remove uintptr, msg *nativeMenuMessage) uintptr {
		if diagnostic && code >= 0 && remove&1 != 0 && msg != nil && debugCount < 32 && (msg.Message == 0x100 || msg.Message == 0x101 || msg.Message == 0x104 || msg.Message == 0x105) && (msg.WParam == 0x12 || msg.WParam == 0x46 || msg.WParam == 0x58 || msg.WParam == 0x73) {
			debugCount++
			fmt.Fprintf(os.Stderr, "native-menu-input message=%x key=%x flags=%x window=%x\n", msg.Message, msg.WParam, msg.LParam, msg.Window)
		}
		if code >= 0 && remove&1 != 0 && msg != nil && nativeFileMnemonic(msg.Message, msg.WParam, msg.LParam) {
			root, _, _ := ancestor.Call(msg.Window, 2)
			var pid uint32
			owner.Call(root, uintptr(unsafe.Pointer(&pid)))
			active, _, _ := foreground.Call()
			menu, _, _ := getMenu.Call(root)
			control, _, _ := keyState.Call(0x11)
			if root != 0 && pid == uint32(os.Getpid()) && active == root && menu != 0 && control&0x8000 == 0 {
				msg.Window = root
				msg.Message = 0x112 // WM_SYSCOMMAND
				msg.WParam = 0xf100 // SC_KEYMENU
				msg.LParam = 'f'
				if diagnostic {
					fmt.Fprintln(os.Stderr, "native-exit-stage=menu-mnemonic")
				}
			}
		}
		value, _, _ := next.Call(0, uintptr(code), remove, uintptr(unsafe.Pointer(msg)))
		return value
	}
	callback := syscall.NewCallback(callbackFunction)
	hook, _, nativeError := api.NewProc("SetWindowsHookExW").Call(3, callback, 0, uintptr(thread))
	if hook == 0 {
		runtime.UnlockOSThread()
		return nil, fmt.Errorf("native menu thread routing unavailable: %v", nativeError)
	}
	return func() error {
		removed, _, err := api.NewProc("UnhookWindowsHookEx").Call(hook)
		runtime.KeepAlive(callbackFunction)
		runtime.UnlockOSThread()
		if removed == 0 {
			return fmt.Errorf("native menu thread routing cleanup failed: %v", err)
		}
		return nil
	}, nil
}
