// SPDX-License-Identifier: Apache-2.0
package secrets

import "os"

func persist(root *os.Root, name string) error {
	f, e := root.OpenFile(name, os.O_RDWR, 0)
	if e != nil {
		return e
	}
	defer f.Close()
	return f.Sync()
}
