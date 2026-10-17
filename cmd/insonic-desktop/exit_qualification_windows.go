//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"fmt"
	"os"
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
	for time.Now().Before(deadline) {
		var handles []uintptr
		callback := syscall.NewCallback(func(hwnd, unused uintptr) uintptr {
			var pid uint32
			exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&pid)))
			if pid != uint32(os.Getpid()) || exitWinCall("GetMenu", hwnd) == 0 {
				return 1
			}
			var class [64]uint16
			exitWinCall("GetClassNameW", hwnd, uintptr(unsafe.Pointer(&class[0])), uintptr(len(class)))
			if windows.UTF16ToString(class[:]) == "winc_Form" {
				handles = append(handles, hwnd)
			}
			return 1
		})
		exitWinCall("EnumWindows", callback, 0)
		if len(handles) > 1 {
			return 0, fmt.Errorf("ambiguous owned native window")
		}
		if len(handles) == 1 {
			return handles[0], nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 0, fmt.Errorf("owned native window unavailable")
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
		if err = post(0x104, 0x73, 0x203e0001); err != nil {
			return err
		}
		return post(0x105, 0x73, 0xe03e0001)
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
		for time.Now().Before(deadline) {
			info := guiThreadInfo{}
			info.Size = uint32(unsafe.Sizeof(info))
			if exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) != 0 && info.Flags&4 != 0 && info.MenuOwner == hwnd {
				return post(0x102, 'x', 1)
			}
			time.Sleep(20 * time.Millisecond)
		}
		return fmt.Errorf("Alt+F did not enter the owned native menu")
	}
	return fmt.Errorf("unknown native Exit action")
}
