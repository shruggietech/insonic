// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"os"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
	"golang.org/x/sys/windows"
)

func TestWindowsStorageCapacityErrorsRetainActionableCause(t *testing.T) {
	for _, cause := range []error{windows.ERROR_DISK_FULL, windows.ERROR_HANDLE_DISK_FULL, windows.ERROR_DISK_QUOTA_EXCEEDED} {
		err := &os.PathError{Op: "write", Path: "private-owner-path", Err: cause}
		if got := contracts.SafeError(redact(context.Background(), err)); got.Code != "insufficient_space" {
			t.Fatal(got)
		}
	}
	if got := contracts.SafeError(redact(context.Background(), windows.ERROR_ACCESS_DENIED)); got.Code != "unavailable" {
		t.Fatal("access refusal mislabeled as capacity", got)
	}
}
