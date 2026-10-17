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

const (
	seeMaskNoAsync         = 0x00000100
	seeMaskFlagNoUI        = 0x00000400
	seeMaskNoConsole       = 0x00008000
	documentationShellMask = seeMaskNoAsync | seeMaskFlagNoUI | seeMaskNoConsole
)

func documentationProcedure(library, name string) (*windows.LazyProc, error) {
	procedure := windows.NewLazySystemDLL(library).NewProc(name)
	if procedure.Find() != nil {
		return nil, documentationCorrection
	}
	return procedure, nil
}

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
	initialize, err := documentationProcedure("ole32.dll", "CoInitializeEx")
	if err != nil {
		return err
	}
	uninitialize, err := documentationProcedure("ole32.dll", "CoUninitialize")
	if err != nil {
		return err
	}
	dispatch, err := documentationProcedure("shell32.dll", "ShellExecuteExW")
	if err != nil {
		return err
	}
	result, _, _ := initialize.Call(0, 2)
	if int32(result) < 0 {
		return documentationCorrection
	}
	defer uninitialize.Call()
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
	// provide the fixed correction. NO_CONSOLE prevents a console window for a
	// console-based association handler. Zone checking remains enabled.
	// The filename is not a shell command.
	info := shellExecuteInfo{Mask: documentationShellMask, Verb: verb, File: file, Show: 1}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, _ := dispatch.Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(file)
	if ok == 0 {
		return documentationCorrection
	}
	return nil
}

func ReportDocumentationError() {
	report, err := documentationProcedure("user32.dll", "MessageBoxW")
	if err != nil {
		return
	}
	message, _ := windows.UTF16PtrFromString(documentationCorrection.Error())
	title, _ := windows.UTF16PtrFromString("Open insonic documentation")
	report.Call(0, uintptr(unsafe.Pointer(message)), uintptr(unsafe.Pointer(title)), 0x30)
}
