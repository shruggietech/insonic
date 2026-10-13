// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"golang.org/x/sys/windows"
	"strings"
	"syscall"
	"unsafe"
)

func FreeSpace(path string) (uint64, error) {
	path, e := spaceDirectory(path)
	if e != nil {
		return 0, e
	}
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return 0, e
	}
	var available, total, free uint64
	e = windows.GetDiskFreeSpaceEx(p, &available, &total, &free)
	return available, e
}

var volumeKernel = windows.NewLazySystemDLL("kernel32.dll")

func volumeIdentity(path string) (string, error) {
	path, e := spaceDirectory(path)
	if e != nil {
		return "", e
	}
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return "", e
	}
	buffer := make([]uint16, 32768)
	r, _, e := volumeKernel.NewProc("GetVolumePathNameW").Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if r == 0 {
		return "", e
	}
	root := windows.UTF16ToString(buffer)
	p, e = windows.UTF16PtrFromString(root)
	if e != nil {
		return "", e
	}
	r, _, e = volumeKernel.NewProc("GetVolumeNameForVolumeMountPointW").Call(uintptr(unsafe.Pointer(p)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if r == 0 {
		return "", e
	}
	id := windows.UTF16ToString(buffer)
	if id == "" {
		return "", syscall.EINVAL
	}
	return id, nil
}

func SameVolume(left, right string) (bool, error) {
	a, e := volumeIdentity(left)
	if e != nil {
		return false, e
	}
	b, e := volumeIdentity(right)
	if e != nil {
		return false, e
	}
	return strings.EqualFold(a, b), nil
}
