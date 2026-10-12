//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package diagnostics

import (
	"os"
	"syscall"
)

func privateFile(file *os.File, _ bool) error {
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() {
		return logError()
	}
	system, ok := stat.Sys().(*syscall.Stat_t)
	if !ok || int(system.Uid) != os.Getuid() {
		return logError()
	}
	return file.Chmod(0600)
}
