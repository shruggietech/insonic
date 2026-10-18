//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var exitUser32 = windows.NewLazySystemDLL("user32.dll")

func exitWinCall(name string, args ...uintptr) uintptr {
	value, _, _ := exitUser32.NewProc(name).Call(args...)
	return value
}
func exitWinLabel(handle uintptr, index uintptr) string {
	var label [64]uint16
	length := exitWinCall("GetMenuStringW", handle, index, uintptr(unsafe.Pointer(&label[0])), uintptr(len(label)), 0x400)
	return windows.UTF16ToString(label[:min(int(length), len(label))])
}
func exitOwnedWindow() (uintptr, error) {
	deadline := time.Now().Add(5 * time.Second)
	var handles []uintptr
	var owned, visible, active int
	callback := syscall.NewCallback(func(hwnd, unused uintptr) uintptr {
		var pid uint32
		exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != uint32(os.Getpid()) || exitWinCall("GetMenu", hwnd) == 0 {
			return 1
		}
		var title [64]uint16
		exitWinCall("GetWindowTextW", hwnd, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
		if windows.UTF16ToString(title[:]) == "insonic" {
			owned++
			shown := exitWinCall("IsWindowVisible", hwnd) != 0
			foreground := exitWinCall("GetForegroundWindow") == hwnd
			if shown {
				visible++
			}
			if foreground {
				active++
			}
			if shown && exitWinCall("IsWindowEnabled", hwnd) != 0 {
				handles = append(handles, hwnd)
			}
		}
		return 1
	})
	for time.Now().Before(deadline) {
		handles = nil
		owned, visible, active = 0, 0, 0
		exitWinCall("EnumWindows", callback, 0)
		if len(handles) > 1 {
			return 0, fmt.Errorf("ambiguous owned native window")
		}
		if len(handles) == 1 {
			return handles[0], nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 0, fmt.Errorf("owned native window unavailable (owned=%d, visible=%d, active=%d)", owned, visible, active)
}
func activateNativeExit(action string) error {
	hwnd, err := exitOwnedWindow()
	if err != nil {
		return err
	}
	root := exitWinCall("GetMenu", hwnd)
	file := exitWinCall("GetSubMenu", root, 0)
	if exitWinLabel(root, 0) != "&File" || file == 0 || exitWinLabel(file, 0) != "E&xit" || exitWinCall("GetMenuState", file, 0, 0x400)&3 != 0 {
		return fmt.Errorf("native File/Exit menu unavailable")
	}
	post := func(message, wparam, lparam uintptr) error {
		if exitWinCall("PostMessageW", hwnd, message, wparam, lparam) == 0 {
			return fmt.Errorf("native keyboard message rejected")
		}
		return nil
	}
	switch action {
	case "close":
		return post(0x10, 0, 0)
	case "alt-f4":
		return sendNativeAltF4(hwnd)
	case "menu":
		// The normal system keyboard-menu route opens File with Alt+F. No direct
		// command ID or callback fallback may satisfy this qualification.
		if err = post(0x112, 0xf100, 'f'); err != nil {
			return err
		}
		type guiThreadInfo struct {
			Size, Flags                                        uint32
			Active, Focus, Capture, MenuOwner, MoveSize, Caret uintptr
			CaretRect                                          struct{ Left, Top, Right, Bottom int32 }
		}
		thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
		deadline := time.Now().Add(2 * time.Second)
		var queried, active, owner bool
		var flags uint32
		for time.Now().Before(deadline) {
			info := guiThreadInfo{}
			info.Size = uint32(unsafe.Sizeof(info))
			queried = exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) != 0
			active, owner, flags = info.Active == hwnd, info.MenuOwner == hwnd, info.Flags
			if queried && info.Flags&4 != 0 && owner {
				return post(0x102, 'x', 1)
			}
			time.Sleep(20 * time.Millisecond)
		}
		return fmt.Errorf("Alt+F did not enter the owned native menu (query=%t, flags=%d, active=%t, owner=%t)", queried, flags, active, owner)
	}
	return fmt.Errorf("unknown native Exit action")
}

// INPUT/KEYBDINPUT use the Windows amd64 ABI (40-byte INPUT, union at byte8).
type exitKeyboardInput struct {
	Kind        uint32
	_           uint32
	Key, Scan   uint16
	Flags, Time uint32
	_           uint32
	Extra       uintptr
	_           [8]byte
}

func requireNativeInputTarget(hwnd uintptr) error {
	var pid uint32
	exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || pid != uint32(os.Getpid()) || exitWinCall("GetForegroundWindow") != hwnd || exitWinCall("IsWindowVisible", hwnd) == 0 || exitWinCall("IsWindowEnabled", hwnd) == 0 {
		return fmt.Errorf("native input requires this process's visible foreground fixture window")
	}
	return nil
}

// This explicit disposable-UI probe uses the real input system. Posted messages
// alone cannot establish the system keyboard processing used by Alt+F4.
func sendNativeAltF4(hwnd uintptr) error {
	if err := requireNativeInputTarget(hwnd); err != nil {
		return err
	}
	keys := []exitKeyboardInput{{Kind: 1, Key: 0x12}, {Kind: 1, Key: 0x73}, {Kind: 1, Key: 0x73, Flags: 2}, {Kind: 1, Key: 0x12, Flags: 2}}
	sent, _, nativeError := exitUser32.NewProc("SendInput").Call(uintptr(len(keys)), uintptr(unsafe.Pointer(&keys[0])), unsafe.Sizeof(keys[0]))
	runtime.KeepAlive(keys)
	if sent != uintptr(len(keys)) {
		// Release modifiers even if the system accepted only part of the chord.
		releases := []exitKeyboardInput{{Kind: 1, Key: 0x73, Flags: 2}, {Kind: 1, Key: 0x12, Flags: 2}}
		exitWinCall("SendInput", uintptr(len(releases)), uintptr(unsafe.Pointer(&releases[0])), unsafe.Sizeof(releases[0]))
		runtime.KeepAlive(releases)
		var code uintptr
		if typed, ok := nativeError.(syscall.Errno); ok {
			code = uintptr(typed)
		}
		return fmt.Errorf("native Alt+F4 input rejected (sent=%d, error=%d)", sent, code)
	}
	return nil
}
