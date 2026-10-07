//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package secrets

import "os"

func persist(root *os.Root, _ string) error {
	f, e := root.Open(".")
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
