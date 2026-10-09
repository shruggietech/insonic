// SPDX-License-Identifier: Apache-2.0
package voicemodels

import "os"

func renameFetched(parent *os.Root, from, to string) error {
	// Windows rename cannot replace an existing directory, including an empty one.
	return parent.Rename(from, to)
}
