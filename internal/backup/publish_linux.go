// SPDX-License-Identifier: Apache-2.0
package backup

import "golang.org/x/sys/unix"

func publishDirectory(stage, destination string) error {
	return unix.Renameat2(unix.AT_FDCWD, stage, unix.AT_FDCWD, destination, unix.RENAME_NOREPLACE)
}
