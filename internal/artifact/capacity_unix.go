//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"errors"
	"syscall"
)

func capacityFailure(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT)
}
