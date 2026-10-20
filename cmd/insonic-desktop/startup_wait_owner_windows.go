//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"errors"
	"io"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

type nativeWaitOwner interface {
	Live() bool
	Close()
}
type nativeWaitParent struct{ handle windows.Handle }

var errNativeWaitOwnership = errors.New("native wait parent channel unavailable")
var nativeWaitPipeServer = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetNamedPipeServerProcessId")

func (parent *nativeWaitParent) Live() bool {
	state, err := windows.WaitForSingleObject(parent.handle, 0)
	return err == nil && state == uint32(windows.WAIT_TIMEOUT)
}
func (parent *nativeWaitParent) Close() { windows.CloseHandle(parent.handle) }

func nativeWaitBirth(handle windows.Handle) (uint64, error) {
	var created, exited, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &created, &exited, &kernel, &user); err != nil {
		return 0, err
	}
	return uint64(created.HighDateTime)<<32 | uint64(created.LowDateTime), nil
}

func nativeWaitSameExecutable(handle windows.Handle) bool {
	var path [32768]uint16
	size := uint32(len(path))
	if windows.QueryFullProcessImageName(handle, 0, &path[0], &size) != nil || size == 0 || size >= uint32(len(path)) {
		return false
	}
	parent, err := os.Stat(windows.UTF16ToString(path[:size]))
	if err != nil {
		return false
	}
	executable, err := os.Executable()
	if err != nil {
		return false
	}
	self, err := os.Stat(executable)
	return err == nil && os.SameFile(parent, self)
}

func openNativeWaitParent(input io.Reader) (nativeWaitOwner, error) {
	// Capture supplies a private inherited pipe created by this same executable.
	// Authenticate its kernel-reported creator, not a caller-supplied PID or flag.
	// CreatePipe uses a uniquely named pipe; the real cross-process test covers
	// this existing os/exec boundary. No pipe name, image path or token is output.
	channel, ok := input.(*os.File)
	if !ok {
		return nil, errNativeWaitOwnership
	}
	var creator uint32
	var pin runtime.Pinner
	pin.Pin(&creator)
	defer pin.Unpin()
	success, _, _ := nativeWaitPipeServer.Call(channel.Fd(), uintptr(unsafe.Pointer(&creator)))
	if success == 0 || creator == 0 || creator != uint32(os.Getppid()) {
		return nil, errNativeWaitOwnership
	}
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, creator)
	if err != nil {
		return nil, errNativeWaitOwnership
	}
	parent := &nativeWaitParent{handle: handle}
	parentBirth, birthErr := nativeWaitBirth(handle)
	selfBirth, selfErr := nativeWaitBirth(windows.CurrentProcess())
	if birthErr != nil || selfErr != nil || parentBirth == 0 || parentBirth > selfBirth || !parent.Live() || !nativeWaitSameExecutable(handle) {
		parent.Close()
		return nil, errNativeWaitOwnership
	}
	return parent, nil
}
