//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/sys/unix"
	"os"
)

func PrivateDirectory(dir string) error {
	stat, err := os.Lstat(dir)
	if err != nil || !stat.IsDir() || stat.Mode().Perm()&0077 != 0 {
		return contracts.Fail("unavailable")
	}
	var info unix.Stat_t
	if err := unix.Lstat(dir, &info); err != nil || info.Uid != uint32(os.Getuid()) {
		return contracts.Fail("unavailable")
	}
	return nil
}

func SecureDirectory(dir string) error { return PrivateDirectory(dir) }
