// SPDX-License-Identifier: Apache-2.0
package process

import "golang.org/x/sys/windows"

func fixtureProcessStopped(pid int) (bool, error) {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err == windows.ERROR_INVALID_PARAMETER {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(handle)
	state, err := windows.WaitForSingleObject(handle, 0)
	return state == windows.WAIT_OBJECT_0, err
}
