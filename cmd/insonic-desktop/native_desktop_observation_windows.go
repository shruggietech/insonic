//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type nativeUserObject struct {
	Name          string            `json:"name,omitempty"`
	Flags         *uint32           `json:"flags,omitempty"`
	Visible       *bool             `json:"visible,omitempty"`
	ReceivesInput *bool             `json:"receives_input,omitempty"`
	Errors        map[string]string `json:"errors,omitempty"`
}

// Borrowed station/thread desktop handles stay borrowed. Only OpenInputDesktop
// creates a handle that this observer owns and closes.
//
//go:uintptrescapes
func nativeDesktopCall(name string, args ...uintptr) (uintptr, error) {
	value, _, err := exitUser32.NewProc(name).Call(args...)
	if value == 0 {
		if err == syscall.Errno(0) {
			err = fmt.Errorf("%s returned zero without extended error", name)
		}
		return 0, err
	}
	return value, nil
}

func observeNativeUserObject(handle uintptr, station bool) nativeUserObject {
	value := nativeUserObject{Errors: map[string]string{}}
	var name [128]uint16
	var needed uint32
	if _, err := nativeDesktopCall("GetUserObjectInformationW", handle, 2, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Sizeof(name)), uintptr(unsafe.Pointer(&needed))); err != nil {
		value.Errors["name"] = err.Error()
	} else {
		value.Name = windows.UTF16ToString(name[:])
	}
	if station {
		var flags struct {
			Inherit, Reserved int32
			Flags             uint32
		}
		if _, err := nativeDesktopCall("GetUserObjectInformationW", handle, 1, uintptr(unsafe.Pointer(&flags)), uintptr(unsafe.Sizeof(flags)), uintptr(unsafe.Pointer(&needed))); err != nil {
			value.Errors["flags"] = err.Error()
		} else {
			value.Flags = &flags.Flags
			visible := flags.Flags&1 != 0
			value.Visible = &visible
		}
	} else {
		var input int32
		if _, err := nativeDesktopCall("GetUserObjectInformationW", handle, 6, uintptr(unsafe.Pointer(&input)), uintptr(unsafe.Sizeof(input)), uintptr(unsafe.Pointer(&needed))); err != nil {
			value.Errors["receives_input"] = err.Error()
		} else {
			receives := input != 0
			value.ReceivesInput = &receives
		}
	}
	return value
}

func nativeDesktopSnapshot(hwnd uintptr, stage string) map[string]any {
	errors := map[string]string{}
	var owner uint32
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&owner)))
	result := map[string]any{
		"kind": "native-desktop-observation", "stage": stage, "pid": os.Getpid(),
		"window_pid": owner, "window_thread": thread, "owned": owner == uint32(os.Getpid()) && thread != 0,
		"visible": exitWinCall("IsWindowVisible", hwnd) != 0, "enabled": exitWinCall("IsWindowEnabled", hwnd) != 0,
		"errors": errors,
	}
	var session uint32
	value, _, err := windows.NewLazySystemDLL("kernel32.dll").NewProc("ProcessIdToSessionId").Call(uintptr(os.Getpid()), uintptr(unsafe.Pointer(&session)))
	if value == 0 {
		errors["process_session"] = err.Error()
	} else {
		result["process_session"] = session
	}
	station, err := nativeDesktopCall("GetProcessWindowStation")
	if err != nil {
		errors["window_station"] = err.Error()
	} else {
		result["window_station"] = observeNativeUserObject(station, true)
	}
	if thread != 0 {
		desktop, err := nativeDesktopCall("GetThreadDesktop", thread)
		if err != nil {
			errors["ui_desktop"] = err.Error()
		} else {
			result["ui_desktop"] = observeNativeUserObject(desktop, false)
		}
	} else {
		errors["ui_desktop"] = "owned window thread unavailable"
	}
	input, err := nativeDesktopCall("OpenInputDesktop", 0, 0, 1)
	if err != nil {
		errors["input_desktop"] = err.Error()
	} else {
		result["input_desktop"] = observeNativeUserObject(input, false)
		if _, err := nativeDesktopCall("CloseDesktop", input); err != nil {
			errors["close_input_desktop"] = err.Error()
		}
	}
	foreground := exitWinCall("GetForegroundWindow")
	var foregroundPID uint32
	exitWinCall("GetWindowThreadProcessId", foreground, uintptr(unsafe.Pointer(&foregroundPID)))
	result["foreground_owned"] = foregroundPID == uint32(os.Getpid()) && foreground != 0
	result["foreground_is_window"] = foreground == hwnd && hwnd != 0
	result["foreground_pid"] = foregroundPID
	return result
}

func logNativeDesktopObservation(hwnd uintptr, stage string) {
	raw, _ := json.Marshal(nativeDesktopSnapshot(hwnd, stage))
	fmt.Fprintln(os.Stderr, string(raw))
}
