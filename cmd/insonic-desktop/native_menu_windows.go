//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"fmt"
	"github.com/shruggietech/insonic/desktop"
	"golang.org/x/sys/windows"
	"os"
	"syscall"
	"time"
	"unsafe"
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

func configureNativeMenu(bridge *desktop.Bridge, diagnostic bool) {
	desktop.SetNativeFileMenu(bridge, func() error {
		hwnd, err := exitOwnedWindow()
		if err != nil {
			return err
		}
		if err = requireNativeInputTarget(hwnd); err != nil {
			return err
		}
		if exitWinCall("PostMessageW", hwnd, 0x112, 0xf100, 'f') == 0 {
			return fmt.Errorf("native File menu request rejected")
		}
		if diagnostic {
			fmt.Fprintln(os.Stderr, "native-exit-stage=menu-mnemonic")
		}
		return nil
	})
}

func waitNativeMenuReady(ctx context.Context, bridge *desktop.Bridge) error {
	return desktop.WaitNativeFileMenuReady(ctx, bridge)
}
