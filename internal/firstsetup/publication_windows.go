// SPDX-License-Identifier: Apache-2.0
package firstsetup

import (
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
)

func replaceDraft(source, destination string) error {
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

// Windows has no POSIX directory fsync. Write-through replacement and a final
// writable handle flush publish the file's metadata, including replay retries.
// https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw
func syncPublication(directory string) error {
	f, e := os.OpenFile(filepath.Join(directory, "setup-draft.json"), os.O_RDWR, 0)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
