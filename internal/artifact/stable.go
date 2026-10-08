// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"

	"github.com/shruggietech/insonic/internal/contracts"
)

// copyStable verifies both the opened source and its binding after staging.
// A second full read catches same-size rewrites and restored modification times.
func copyStable(ctx context.Context, path string, input *os.File, output io.Writer) (string, int64, error) {
	return copyStableBound(ctx, path, input, output, 1<<63-1)
}

func copyStableBound(ctx context.Context, path string, input *os.File, output io.Writer, maxBytes int64) (string, int64, error) {
	before, e := input.Stat()
	if e != nil {
		return "", 0, redact(ctx, e)
	}
	if before.Size() > maxBytes {
		return "", 0, contracts.Fail("output_limit")
	}
	bound := maxBytes
	if bound < 1<<63-1 {
		bound++
	}
	h := sha256.New()
	n, e := copyContext(ctx, io.MultiWriter(output, h), io.LimitReader(input, bound))
	if e != nil {
		return "", 0, redact(ctx, e)
	}
	if n > maxBytes {
		return "", 0, contracts.Fail("output_limit")
	}
	after, e := input.Stat()
	if e != nil {
		return "", 0, redact(ctx, e)
	}
	binding, e := os.Stat(path)
	if e != nil {
		return "", 0, contracts.Fail("conflict")
	}
	if before.Size() != n || after.Size() != n || !before.ModTime().Equal(after.ModTime()) || !os.SameFile(before, after) || !os.SameFile(after, binding) {
		return "", 0, contracts.Fail("conflict")
	}
	if _, e = input.Seek(0, io.SeekStart); e != nil {
		return "", 0, redact(ctx, e)
	}
	verify := sha256.New()
	vn, e := copyContext(ctx, verify, io.LimitReader(input, bound))
	if e != nil {
		return "", 0, redact(ctx, e)
	}
	final, e := input.Stat()
	if e != nil {
		return "", 0, redact(ctx, e)
	}
	binding, e = os.Stat(path)
	if e != nil {
		return "", 0, contracts.Fail("conflict")
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if vn != n || hex.EncodeToString(verify.Sum(nil)) != digest || !final.ModTime().Equal(after.ModTime()) || final.Size() != n || !os.SameFile(final, binding) {
		return "", 0, contracts.Fail("conflict")
	}
	return digest, n, nil
}
