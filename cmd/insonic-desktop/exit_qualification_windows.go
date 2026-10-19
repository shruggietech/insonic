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
)

func activateNativeExit(action string) error {
	hwnd, err := exitOwnedWindow()
	if err != nil {
		return err
	}
	threadID := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	fmt.Fprintf(os.Stderr, "native-input-target thread=%d window=%x foreground=%x\n", threadID, hwnd, exitWinCall("GetForegroundWindow"))
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
		// Real Alt+F enters the system menu input loop. A posted WM_CHAR can
		// reach the owner window without selecting its modal menu item.
		if err = sendNativeKeyboard(hwnd, "Alt+F", []exitKeyboardInput{{Kind: 1, Key: 0x12}, {Kind: 1, Key: 0x46}, {Kind: 1, Key: 0x46, Flags: 2}, {Kind: 1, Key: 0x12, Flags: 2}}); err != nil {
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
			if queried && active && owner && flags&4 != 0 {
				fmt.Fprintln(os.Stderr, "native-exit-stage=menu-open")
				if err = sendNativeKeyboard(hwnd, "File/Exit X", []exitKeyboardInput{{Kind: 1, Key: 0x58}, {Kind: 1, Key: 0x58, Flags: 2}}); err != nil {
					return err
				}
				fmt.Fprintln(os.Stderr, "native-exit-stage=menu-selected")
				return nil
			}
			time.Sleep(20 * time.Millisecond)
		}
		return fmt.Errorf("Alt+F did not enter the active owned native menu (query=%t, flags=%d, active=%t, owner=%t)", queried, flags, active, owner)
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
	return sendNativeKeyboard(hwnd, "Alt+F4", []exitKeyboardInput{{Kind: 1, Key: 0x12}, {Kind: 1, Key: 0x73}, {Kind: 1, Key: 0x73, Flags: 2}, {Kind: 1, Key: 0x12, Flags: 2}})
}

func sendNativeKeyboard(hwnd uintptr, label string, keys []exitKeyboardInput) error {
	if err := requireNativeInputTarget(hwnd); err != nil {
		return err
	}
	if len(keys) == 0 {
		return fmt.Errorf("native keyboard sequence is empty")
	}
	sent, _, nativeError := exitUser32.NewProc("SendInput").Call(uintptr(len(keys)), uintptr(unsafe.Pointer(&keys[0])), unsafe.Sizeof(keys[0]))
	runtime.KeepAlive(keys)
	if sent != uintptr(len(keys)) {
		// Release exactly the keys this sequence could have pressed, even when
		// the system accepted only part of the chord. No callback fallback.
		releases := []exitKeyboardInput{}
		for i := len(keys) - 1; i >= 0; i-- {
			if keys[i].Flags&2 == 0 {
				releases = append(releases, exitKeyboardInput{Kind: 1, Key: keys[i].Key, Flags: 2})
			}
		}
		if len(releases) > 0 {
			exitWinCall("SendInput", uintptr(len(releases)), uintptr(unsafe.Pointer(&releases[0])), unsafe.Sizeof(releases[0]))
			runtime.KeepAlive(releases)
		}
		var code uintptr
		if typed, ok := nativeError.(syscall.Errno); ok {
			code = uintptr(typed)
		}
		return fmt.Errorf("native %s input rejected (sent=%d, error=%d)", label, sent, code)
	}
	return nil
}
