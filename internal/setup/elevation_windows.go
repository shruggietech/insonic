//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"errors"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Before reading per-user sources, hand elevated starts to the same account's
// linked limited token. Wait for that process and propagate its actual result.
// Never fall back to running setup with administrator privileges.
func limitedStart(args []string) (bool, int, error) {
	token := windows.GetCurrentProcessToken()
	var elevation, size uint32
	if err := windows.GetTokenInformation(token, windows.TokenElevation, (*byte)(unsafe.Pointer(&elevation)), 4, &size); err != nil {
		return true, 1603, err
	}
	if elevation == 0 {
		return false, 0, nil
	}
	fail := errors.New("open setup normally from the invoking account; a same-user limited token is unavailable")
	linked, err := token.GetLinkedToken()
	if err != nil {
		return true, 1603, fail
	}
	defer linked.Close()
	var limited windows.Token
	if windows.DuplicateTokenEx(linked, windows.TOKEN_QUERY|windows.TOKEN_DUPLICATE|windows.TOKEN_ASSIGN_PRIMARY, nil, windows.SecurityImpersonation, windows.TokenPrimary, &limited) != nil {
		return true, 1603, fail
	}
	defer limited.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return true, 1603, fail
	}
	other, err := limited.GetTokenUser()
	if err != nil || !user.User.Sid.Equals(other.User.Sid) || limited.IsElevated() {
		return true, 1603, fail
	}
	exe, err := os.Executable()
	if err != nil {
		return true, 1603, fail
	}
	application, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return true, 1603, fail
	}
	commandLine := syscall.EscapeArg(exe)
	for _, a := range args {
		commandLine += " " + syscall.EscapeArg(a)
	}
	command, err := windows.UTF16FromString(commandLine)
	if err != nil || len(command) > 32767 {
		return true, 1603, fail
	}
	base, err := limited.Environ(false)
	if err != nil {
		return true, 1603, fail
	}
	environment := []uint16{}
	for _, entry := range base {
		if strings.ContainsRune(entry, 0) {
			return true, 1603, fail
		}
		v, err := windows.UTF16FromString(entry)
		if err != nil {
			return true, 1603, fail
		}
		environment = append(environment, v...)
	}
	environment = append(environment, 0)
	startup := windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfo{}))}
	var process windows.ProcessInformation
	proc := windows.NewLazySystemDLL("advapi32.dll").NewProc("CreateProcessWithTokenW")
	if proc.Find() != nil {
		return true, 1603, fail
	}
	ok, _, _ := proc.Call(uintptr(limited), 1, uintptr(unsafe.Pointer(application)), uintptr(unsafe.Pointer(&command[0])), windows.CREATE_UNICODE_ENVIRONMENT|windows.CREATE_NO_WINDOW, uintptr(unsafe.Pointer(&environment[0])), 0, uintptr(unsafe.Pointer(&startup)), uintptr(unsafe.Pointer(&process)))
	runtime.KeepAlive(application)
	runtime.KeepAlive(command)
	runtime.KeepAlive(environment)
	if ok == 0 {
		return true, 1603, fail
	}
	defer windows.CloseHandle(process.Process)
	windows.CloseHandle(process.Thread)
	if _, err := windows.WaitForSingleObject(process.Process, windows.INFINITE); err != nil {
		return true, 1603, fail
	}
	var result uint32
	if windows.GetExitCodeProcess(process.Process, &result) != nil {
		return true, 1603, fail
	}
	return true, int(result), nil
}
