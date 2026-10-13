// SPDX-License-Identifier: Apache-2.0
package workers

import (
	"os"
	"path/filepath"
)

// Read capacity at the nearest existing directory without creating a preview
// destination. Resolve links so mount points and relocated roots stay accurate.
func spaceDirectory(path string) (string, error) {
	path, e := filepath.Abs(path)
	if e != nil {
		return "", e
	}
	for {
		st, e := os.Stat(path)
		if e == nil && st.IsDir() {
			return filepath.EvalSymlinks(path)
		}
		if e != nil && !os.IsNotExist(e) {
			return "", e
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", os.ErrNotExist
		}
		path = parent
	}
}
