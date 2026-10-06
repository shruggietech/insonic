//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"os"
	"path"
)

// Flush the leaf entry and every newly created ancestor before admission.
// Open directories relative to the anchored root, never reconstruct host paths.
func persistObject(root *os.Root, key string) error {
	for dir := path.Dir(key); ; dir = path.Dir(dir) {
		f, e := root.Open(dir)
		if e != nil {
			return e
		}
		e = f.Sync()
		closed := f.Close()
		if e != nil {
			return e
		}
		if closed != nil {
			return closed
		}
		if dir == "." {
			return nil
		}
	}
}
