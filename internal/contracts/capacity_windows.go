// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"errors"
	"syscall"

	"golang.org/x/sys/windows"
)

// CapacityFailure recognizes native capacity causes through wrapped errors.
func CapacityFailure(err error) bool {
	return errors.Is(err, syscall.ENOSPC) || errors.Is(err, windows.ERROR_DISK_FULL) || errors.Is(err, windows.ERROR_HANDLE_DISK_FULL) || errors.Is(err, windows.ERROR_DISK_QUOTA_EXCEEDED)
}
