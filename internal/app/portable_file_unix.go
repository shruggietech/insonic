//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package app

import (
	"golang.org/x/sys/unix"
	"os"
)

func openPortablePin(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}
