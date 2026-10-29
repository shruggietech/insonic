// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"errors"
	"io"
	"os"
)

type stagedSnapshot interface {
	io.Writer
	Sync() error
	Seek(int64, int) (int64, error)
	Close() error
}

// prepareStableStage requires durable, readable snapshot readiness. On failure,
// cleanup errors remain in the internal chain; callers redact owner paths.
func prepareStableStage(ctx context.Context, source string, input *os.File, stage stagedSnapshot, maxBytes int64, discard func() error) (digest string, size int64, err error) {
	defer func() {
		if err != nil {
			closed, removed := stage.Close(), discard()
			if closed != nil || removed != nil {
				err = errors.Join(err, closed, removed)
			}
			digest, size = "", 0
		}
	}()
	digest, size, err = copyStableBound(ctx, source, input, stage, maxBytes)
	if err == nil {
		err = stage.Sync()
	}
	if err == nil {
		_, err = stage.Seek(0, io.SeekStart)
	}
	return
}
