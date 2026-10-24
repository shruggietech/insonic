//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package process

import "syscall"

func fixtureProcessStopped(pid int) (bool, error) {
	err := syscall.Kill(pid, 0)
	if err == syscall.ESRCH {
		return true, nil
	}
	return false, err
}
