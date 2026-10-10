//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"golang.org/x/sys/unix"
	"os"
)

func openRegular(root *os.Root, key string) (*os.File, error) {
	return root.OpenFile(key, os.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
}
func syncDirectory(directory string) error {
	f, e := os.Open(directory)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}

func replaceConfiguration(source, destination string) error {
	return os.Rename(source, destination)
}
