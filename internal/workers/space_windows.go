// SPDX-License-Identifier: Apache-2.0
package workers

import "golang.org/x/sys/windows"

func FreeSpace(path string) (uint64, error) {
	p, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return 0, e
	}
	var available, total, free uint64
	e = windows.GetDiskFreeSpaceEx(p, &available, &total, &free)
	return available, e
}
