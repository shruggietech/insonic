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
		if err := openNativeExitMenu(hwnd, root); err != nil {
			return err
		}
		// Cancellation must return input to the browser before another native menu.
		if err := sendNativeKeyboard(hwnd, "File menu cancellation", []exitKeyboardInput{{Kind: 1, Key: 0x1b}, {Kind: 1, Key: 0x1b, Flags: 2}, {Kind: 1, Key: 0x1b}, {Kind: 1, Key: 0x1b, Flags: 2}}); err != nil {
			return err
		}
		deadline := time.Now().Add(2 * time.Second)
		restored := false
		for time.Now().Before(deadline) {
			state := observeExitMenu(hwnd, root, threadID)
			if state.queried && state.flags&4 == 0 && !state.owner && state.focus != 0 && exitWinCall("IsChild", hwnd, state.focus) != 0 {
				restored = true
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		if !restored {
			return fmt.Errorf("native menu cancellation did not restore browser focus")
		}
		fmt.Fprintln(os.Stderr, "native-exit-stage=menu-cancel-restored")
		if err := openNativeExitMenu(hwnd, root); err != nil {
			return err
		}
		if err := sendNativeKeyboard(hwnd, "File/Exit X", []exitKeyboardInput{{Kind: 1, Key: 0x58}, {Kind: 1, Key: 0x58, Flags: 2}}); err != nil {
			return err
		}
		fmt.Fprintln(os.Stderr, "native-exit-stage=menu-selected")
		return nil
	}
	return fmt.Errorf("unknown native Exit action")
}

func openNativeExitMenu(hwnd, root uintptr) error {
	nativeSystemMenuRequested.Store(false)
	if err := sendNativeKeyboard(hwnd, "Alt+F", []exitKeyboardInput{{Kind: 1, Key: 0x12}, {Kind: 1, Key: 0x46}, {Kind: 1, Key: 0x46, Flags: 2}, {Kind: 1, Key: 0x12, Flags: 2}}); err != nil {
		return err
	}
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	deadline := time.Now().Add(2 * time.Second)
	var state exitMenuState
	for time.Now().Before(deadline) {
		state = observeExitMenu(hwnd, root, thread)
		if state.ready() && state.focus == hwnd {
			if err := verifyNativeMenuRoute("menu"); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "native-menu-popup %s\n", state)
			fmt.Fprintln(os.Stderr, "native-exit-stage=menu-open")
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("Alt+F did not open the owned keyboard-focused File submenu (%s)", state)
}

// MENUBARINFO uses the native pointer alignment of the supported Windows amd64 ABI.
type exitMenuBarInfo struct {
	Size         uint32
	Rect         struct{ Left, Top, Right, Bottom int32 }
	Menu, Window uintptr
	Flags        uint32
}

type exitGUIThreadInfo struct {
	Size, Flags                                        uint32
	Active, Focus, Capture, MenuOwner, MoveSize, Caret uintptr
	CaretRect                                          struct{ Left, Top, Right, Bottom int32 }
}

type exitMenuState struct {
	queried, active, owner, barQueried, barMatches, popupVisible, popupOwned bool
	flags                                                                    uint32
	popup, focus                                                             uintptr
	barFlags, exitFlags                                                      uint32
	focusOwned, altDown                                                      bool
}

func (s exitMenuState) ready() bool {
	return s.queried && s.active && s.owner && s.flags&4 != 0 && s.barQueried && s.barMatches && s.popupVisible && s.popupOwned
}

func (s exitMenuState) String() string {
	return fmt.Sprintf("query=%t flags=%d active=%t owner=%t bar-query=%t bar-match=%t popup=%x visible=%t owned=%t bar-flags=%d exit-flags=%d focus=%x focus-owned=%t alt-down=%t", s.queried, s.flags, s.active, s.owner, s.barQueried, s.barMatches, s.popup, s.popupVisible, s.popupOwned, s.barFlags, s.exitFlags, s.focus, s.focusOwned, s.altDown)
}

func observeExitMenu(hwnd, root, thread uintptr) exitMenuState {
	info := exitGUIThreadInfo{Size: uint32(unsafe.Sizeof(exitGUIThreadInfo{}))}
	queried := exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) != 0
	bar := exitMenuBarInfo{Size: uint32(unsafe.Sizeof(exitMenuBarInfo{}))}
	// OBJID_MENU, first item: hwndMenu identifies File's actual submenu window.
	// GUI_POPUPMENUMODE alone does not describe an attached menu bar's popup.
	barQueried := exitWinCall("GetMenuBarInfo", hwnd, 0xfffffffd, 1, uintptr(unsafe.Pointer(&bar))) != 0
	var pid uint32
	popupThread := exitWinCall("GetWindowThreadProcessId", bar.Window, uintptr(unsafe.Pointer(&pid)))
	var focusPID uint32
	exitWinCall("GetWindowThreadProcessId", info.Focus, uintptr(unsafe.Pointer(&focusPID)))
	file := exitWinCall("GetSubMenu", root, 0)
	exitFlags := uint32(exitWinCall("GetMenuState", file, 0, 0x400))
	return exitMenuState{queried: queried, active: info.Active == hwnd, owner: info.MenuOwner == hwnd, flags: info.Flags, barQueried: barQueried, barMatches: bar.Menu == root, popup: bar.Window, popupVisible: bar.Window != 0 && exitWinCall("IsWindowVisible", bar.Window) != 0, popupOwned: popupThread == thread && pid == uint32(os.Getpid()), barFlags: bar.Flags, exitFlags: exitFlags, focus: info.Focus, focusOwned: info.Focus != 0 && focusPID == uint32(os.Getpid()), altDown: exitWinCall("GetAsyncKeyState", 0x12)&0x8000 != 0}
}

// Failure-only dismissal lets the normal Quit path leave a modal menu loop.
// It never selects Exit, invokes a callback, or changes the failed outcome.
func dismissFailedNativeMenu() {
	hwnd := exitWinCall("GetForegroundWindow")
	if requireNativeInputTarget(hwnd) != nil {
		fmt.Fprintln(os.Stderr, "native-menu-dismiss owned-foreground=false")
		return
	}
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	info := exitGUIThreadInfo{Size: uint32(unsafe.Sizeof(exitGUIThreadInfo{}))}
	fmt.Fprintln(os.Stderr, "native-menu-dismiss owned-foreground=true")
	if exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) != 0 && info.MenuOwner == hwnd && info.Flags&4 != 0 {
		if err := sendNativeKeyboard(hwnd, "failed menu Escape", []exitKeyboardInput{{Kind: 1, Key: 0x1b}, {Kind: 1, Key: 0x1b, Flags: 2}}); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
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

// Failure-only snapshot distinguishes a modal menu wait from lost focus or an
// unprocessed accelerator. The original watchdog and failed outcome remain.
func logNativeExitMenuState() {
	hwnd := exitWinCall("GetForegroundWindow")
	if requireNativeInputTarget(hwnd) != nil {
		fmt.Fprintln(os.Stderr, "native-menu-timeout owned-foreground=false")
		return
	}
	root := exitWinCall("GetMenu", hwnd)
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	fmt.Fprintf(os.Stderr, "native-menu-timeout %s\n", observeExitMenu(hwnd, root, thread))
}
