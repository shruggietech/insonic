//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package firstsetup

import "os"

func replaceDraft(source, destination string) error { return os.Rename(source, destination) }

// File fsync establishes bytes; directory fsync establishes the renamed entry.
func syncPublication(directory string) error {
	f, e := os.Open(directory)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
