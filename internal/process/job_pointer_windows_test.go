// SPDX-License-Identifier: Apache-2.0
package process

import (
	"golang.org/x/sys/windows"
	"runtime"
	"testing"
	"unsafe"
)

//go:noinline
func jobFlagsOnNewStack(depth int, job windows.Handle) (uint32, error) {
	var frame [127]byte
	frame[depth%len(frame)] = byte(depth)
	if depth > 0 {
		flags, err := jobFlagsOnNewStack(depth-1, job)
		runtime.KeepAlive(frame)
		return flags, err
	}
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	err := queryJobInformation(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits)))
	return limits.BasicLimitInformation.LimitFlags, err
}

func TestNativeJobInformationSurvivesGoStackGrowth(t *testing.T) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(job)
	var limits windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	limits.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := setJobInformation(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&limits)), uint32(unsafe.Sizeof(limits))); err != nil {
		t.Fatal(err)
	}
	for depth := 0; depth < 128; depth++ {
		result := make(chan struct {
			flags uint32
			err   error
		}, 1)
		go func(depth int) {
			flags, err := jobFlagsOnNewStack(depth, job)
			result <- struct {
				flags uint32
				err   error
			}{flags, err}
		}(depth)
		value := <-result
		if value.err != nil || value.flags != limits.BasicLimitInformation.LimitFlags {
			t.Fatalf("depth %d lost actual job limits: flags=%x error=%v", depth, value.flags, value.err)
		}
	}
}
