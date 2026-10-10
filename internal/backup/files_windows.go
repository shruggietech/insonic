// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"golang.org/x/sys/windows"
	"os"
)

func openRegular(root *os.Root, key string) (*os.File, error) { return root.Open(key) }

// Windows publication uses a write-through nonreplacing move after every file
// is individually flushed. Directory fsync is not a Windows file operation.
func syncDirectory(string) error { return nil }

func replaceConfiguration(source, destination string) error {
	from, e := windows.UTF16PtrFromString(source)
	if e != nil {
		return e
	}
	to, e := windows.UTF16PtrFromString(destination)
	if e != nil {
		return e
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
