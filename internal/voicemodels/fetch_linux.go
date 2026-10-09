// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"golang.org/x/sys/unix"
	"os"
)

// Both names are immediate children of the already-open destination parent.
func renameFetched(parent *os.Root, from, to string) error {
	dir, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	return unix.Renameat2(int(dir.Fd()), from, int(dir.Fd()), to, unix.RENAME_NOREPLACE)
}
