// SPDX-License-Identifier: Apache-2.0
package backup

import "golang.org/x/sys/windows"

func publishDirectory(stage, destination string) error {
	from, e := windows.UTF16PtrFromString(stage)
	if e != nil {
		return e
	}
	to, e := windows.UTF16PtrFromString(destination)
	if e != nil {
		return e
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_WRITE_THROUGH)
}
