// SPDX-License-Identifier: Apache-2.0
package workers

import "golang.org/x/sys/unix"

func FreeSpace(path string) (uint64, error) {
	var s unix.Statfs_t
	e := unix.Statfs(path, &s)
	return s.Bavail * uint64(s.Bsize), e
}
