// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// VerifyAssetFile checks the already-open elected file, bounded by its expected
// size. Callers retain authority over opening and copying the checked bytes.
func VerifyAssetFile(ctx context.Context, file *os.File, role, name, digest string, size int64) error {
	if size < 0 {
		return Fail("invalid_request")
	}
	if ctx.Err() != nil {
		return Fail("cancelled")
	}
	st, err := file.Stat()
	if err != nil {
		return AssetFailure("asset_unreadable", role, name)
	}
	if !st.Mode().IsRegular() {
		return AssetFailure("asset_not_regular", role, name)
	}
	if st.Size() != size {
		return AssetFailure("asset_size", role, name)
	}
	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return AssetFailure("asset_unreadable", role, name)
	}
	h := sha256.New()
	buffer := make([]byte, 128<<10)
	var total int64
	for total <= size {
		if ctx.Err() != nil {
			return Fail("cancelled")
		}
		want := len(buffer)
		if remaining := size - total; remaining < int64(want) {
			want = int(remaining) + 1
		}
		n, e := file.Read(buffer[:want])
		h.Write(buffer[:n])
		total += int64(n)
		if e == io.EOF {
			break
		}
		if e != nil {
			return AssetFailure("asset_unreadable", role, name)
		}
	}
	if total != size {
		return AssetFailure("asset_size", role, name)
	}
	if hex.EncodeToString(h.Sum(nil)) != digest {
		return AssetFailure("asset_hash", role, name)
	}
	return nil
}

func AssetOpenFailure(ctx context.Context, err error, role, name string) *Error {
	if ctx.Err() != nil {
		return Fail("cancelled")
	}
	reason := "asset_unreadable"
	if os.IsNotExist(err) {
		reason = "asset_missing"
	}
	return AssetFailure(reason, role, name)
}
