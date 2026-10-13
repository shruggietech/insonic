// SPDX-License-Identifier: Apache-2.0
package workers

import "golang.org/x/sys/unix"

func FreeSpace(path string) (uint64, error) {
	path, e := spaceDirectory(path)
	if e != nil {
		return 0, e
	}
	var s unix.Statfs_t
	e = unix.Statfs(path, &s)
	return s.Bavail * uint64(s.Bsize), e
}

func SameVolume(left, right string) (bool, error) {
	a, e := spaceDirectory(left)
	if e != nil {
		return false, e
	}
	b, e := spaceDirectory(right)
	if e != nil {
		return false, e
	}
	var x, y unix.Stat_t
	if e = unix.Stat(a, &x); e != nil {
		return false, e
	}
	if e = unix.Stat(b, &y); e != nil {
		return false, e
	}
	return x.Dev == y.Dev, nil
}
