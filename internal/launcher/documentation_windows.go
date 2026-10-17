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

func openDocumentation(directory, target string) error {
	if !filepath.IsAbs(directory) {
		return documentationCorrection
	}
	helper := filepath.Join(directory, "insonic-launcher.exe")
	command := syscall.EscapeArg(helper) + " --documentation --interactive"
	if target != "" {
		command += " --reference " + syscall.EscapeArg(target)
	}
	if handled, err := launchLimited(Plan{Mode: "documentation", Executable: helper, CommandLine: command}); handled || err != nil {
		return err
	}
	if target == "" {
		var err error
		target, err = DocumentationEntry(directory)
		if err != nil {
			return err
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	result, _, _ := ole.NewProc("CoInitializeEx").Call(0, 2)
	if int32(result) < 0 {
		return documentationCorrection
	}
	defer ole.NewProc("CoUninitialize").Call()
	file, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return documentationCorrection
	}
	verb, _ := windows.UTF16PtrFromString("open")
	type shellExecuteInfo struct {
		Size       uint32
		Mask       uint32
		Window     windows.Handle
		Verb       *uint16
		File       *uint16
		Parameters *uint16
		Directory  *uint16
		Show       int32
		Instance   windows.Handle
		IDList     uintptr
		Class      *uint16
		ClassKey   windows.Handle
		HotKey     uint32
		Icon       windows.Handle
		Process    windows.Handle
	}
	// NOASYNC retains COM state until dispatch completes; NO_UI lets the caller
	// provide the fixed correction. The filename is not a shell command.
	info := shellExecuteInfo{Mask: 0x100 | 0x400 | 0x8000, Verb: verb, File: file, Show: 1}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, _ := windows.NewLazySystemDLL("shell32.dll").NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(file)
	if ok == 0 {
		return documentationCorrection
	}
	return nil
}

func ReportDocumentationError() {
	message, _ := windows.UTF16PtrFromString(documentationCorrection.Error())
	title, _ := windows.UTF16PtrFromString("Open insonic documentation")
	windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)), 0x30)
}
