//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"fmt"
	"golang.org/x/sys/windows"
	"runtime"
	"testing"
)

//go:noinline
func recordStringOnNewStack(depth int, record uint32) (string, error) {
	var frame [127]byte
	frame[depth%len(frame)] = byte(depth)
	if depth > 0 {
		value, err := recordStringOnNewStack(depth-1, record)
		runtime.KeepAlive(frame)
		return value, err
	}
	var buffer [64]uint16
	size := uint32(len(buffer))
	err := checked("MsiRecordGetStringW", record, 1, &buffer[0], &size)
	return windows.UTF16ToString(buffer[:]), err
}

func TestNativeMSIOutputSurvivesGoStackGrowth(t *testing.T) {
	for depth := 0; depth < 128; depth++ {
		result := make(chan error, 1)
		go func(depth int) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			record, err := native("MsiCreateRecord", 1)
			if err != nil || record == 0 {
				result <- fmt.Errorf("fixture record creation: %v", err)
				return
			}
			defer closeMSI(record)
			if err := checked("MsiRecordSetStringW", record, 1, "retained-source-fixture"); err != nil {
				result <- err
				return
			}
			value, err := recordStringOnNewStack(depth, record)
			if err == nil && value != "retained-source-fixture" {
				err = fmt.Errorf("native MSI value corrupted across stack growth: %q", value)
			}
			result <- err
		}(depth)
		if err := <-result; err != nil {
			t.Errorf("depth %d: %v", depth, err)
		}
	}
}
