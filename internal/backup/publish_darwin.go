// SPDX-License-Identifier: Apache-2.0
package backup

import "golang.org/x/sys/unix"

func publishDirectory(stage, destination string) error {
	return unix.RenamexNp(stage, destination, unix.RENAME_EXCL)
}
