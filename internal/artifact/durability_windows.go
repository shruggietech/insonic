// SPDX-License-Identifier: Apache-2.0
package artifact

import "os"

// Windows flushes file metadata through FlushFileBuffers rather than Unix
// directory fsync. Reopen the final link with write access and flush AFTER link
// creation; a flush of the earlier stage cannot establish final-link durability.
// https://learn.microsoft.com/en-us/windows/win32/fileio/file-caching
func persistObject(root *os.Root, key string) error {
	f, e := root.OpenFile(key, os.O_RDWR, 0)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
