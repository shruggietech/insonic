// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"golang.org/x/sys/unix"
	"os"
)

func renameFetched(parent *os.Root, from, to string) error {
	dir, err := parent.Open(".")
	if err != nil {
		return err
	}
	defer dir.Close()
	return unix.RenameatxNp(int(dir.Fd()), from, int(dir.Fd()), to, unix.RENAME_EXCL)
}
