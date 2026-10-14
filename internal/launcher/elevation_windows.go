//go:build windows

// SPDX-License-Identifier: Apache-2.0
package launcher

import (
	"golang.org/x/sys/windows"
	"path/filepath"
	"runtime"
	"syscall"
	"unsafe"
)

var createProcessWithToken = windows.NewLazySystemDLL("advapi32.dll").NewProc("CreateProcessWithTokenW")

func elevated(token windows.Token) (bool, error) {
	var value, size uint32
	e := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&value)), uint32(unsafe.Sizeof(value)), &size)
	return value != 0, e
}

// launchLimited handles every elevated launch, including failures. Its caller
// must never retry that path through ordinary exec, which would inherit elevation.
func launchLimited(p Plan) (handled bool, err error) {
	var current windows.Token
	if windows.OpenProcessToken(windows.CurrentProcess(), windows.TOKEN_QUERY, &current) != nil {
		return true, normalLaunchAction
	}
	defer current.Close()
	isElevated, e := elevated(current)
	if e != nil {
		return true, normalLaunchAction
	}
	if !isElevated {
		return false, nil
	}
	linked, e := current.GetLinkedToken()
	if e != nil {
		return true, normalLaunchAction
	}
	defer linked.Close()
	var limited windows.Token
	if windows.DuplicateTokenEx(linked, windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY, nil, windows.SecurityImpersonation, windows.TokenPrimary, &limited) != nil {
		return true, normalLaunchAction
	}
	defer limited.Close()
	user, e := current.GetTokenUser()
	if e != nil {
		return true, normalLaunchAction
	}
	other, e := limited.GetTokenUser()
	if e != nil {
		return true, normalLaunchAction
	}
	stillElevated, e := elevated(limited)
	if e != nil || limitedAccount(user.User.Sid.String(), other.User.Sid.String(), stillElevated) != nil {
		return true, normalLaunchAction
	}
	base, e := limited.Environ(false)
	if e != nil {
		return true, normalLaunchAction
	}
	environment, e := unicodeEnvironment(base, p.Environment)
	if e != nil {
		return true, e
	}
	application, e := windows.UTF16PtrFromString(p.Executable)
	if e != nil {
		return true, normalLaunchAction
	}
	commandLine := p.CommandLine
	directory := filepath.Dir(p.Environment["INSONIC_LAUNCH_CLI"])
	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_NEW_CONSOLE)
	if p.Mode == "desktop" {
		commandLine = syscall.EscapeArg(p.Executable)
		directory = filepath.Dir(p.Executable)
		flags = windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_NO_WINDOW
	} else if p.Mode != "cli" {
		return true, normalLaunchAction
	}
	command, e := windows.UTF16FromString(commandLine)
	if e != nil || len(command) > 1024 {
		return true, normalLaunchAction
	}
	working, e := windows.UTF16PtrFromString(directory)
	if e != nil {
		return true, normalLaunchAction
	}
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var process windows.ProcessInformation
	// LOGON_WITH_PROFILE retains the limited account's HKCU and private config.
	if createProcessWithToken.Find() != nil {
		return true, normalLaunchAction
	}
	ok, _, _ := createProcessWithToken.Call(uintptr(limited), 1, uintptr(unsafe.Pointer(application)), uintptr(unsafe.Pointer(&command[0])), uintptr(flags), uintptr(unsafe.Pointer(&environment[0])), uintptr(unsafe.Pointer(working)), uintptr(unsafe.Pointer(&startup)), uintptr(unsafe.Pointer(&process)))
	runtime.KeepAlive(environment)
	runtime.KeepAlive(command)
	runtime.KeepAlive(application)
	runtime.KeepAlive(working)
	if ok == 0 {
		return true, normalLaunchAction
	}
	windows.CloseHandle(process.Thread)
	windows.CloseHandle(process.Process)
	return true, nil
}
