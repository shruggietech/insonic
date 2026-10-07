//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"golang.org/x/sys/unix"
	"os"
)

func openNoFollow(source string) (*os.File, error) {
	// NONBLOCK also prevents a swapped FIFO from blocking before fstat rejects it.
	fd, e := unix.Open(source, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if e != nil {
		return nil, e
	}
	return os.NewFile(uintptr(fd), source), nil
}
