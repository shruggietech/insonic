// SPDX-License-Identifier: Apache-2.0
package app

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/shruggietech/insonic/internal/contracts"
	"github.com/shruggietech/insonic/internal/library"
)

func verifyPortablePin(pin library.PinnedFile) error {
	before, e := os.Lstat(pin.Path)
	if e != nil || !before.Mode().IsRegular() {
		return contracts.Fail("unavailable")
	}
	f, e := openPortablePin(pin.Path)
	if e != nil {
		return contracts.Fail("unavailable")
	}
	defer f.Close()
	actual, e := f.Stat()
	if e != nil || !actual.Mode().IsRegular() || !os.SameFile(before, actual) {
		return contracts.Fail("unavailable")
	}
	digest := sha256.New()
	if _, e = io.Copy(digest, f); e != nil {
		return contracts.Fail("unavailable")
	}
	if hex.EncodeToString(digest.Sum(nil)) != pin.SHA256 {
		return contracts.Fail("incompatible_version")
	}
	return nil
}
