//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"errors"
	"syscall"
)

// CapacityFailure recognizes native capacity causes through wrapped errors.
func CapacityFailure(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, syscall.EDQUOT)
}
