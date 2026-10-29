//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package artifact

import (
	"context"
	"os"
	"syscall"
	"testing"

	"github.com/shruggietech/insonic/internal/contracts"
)

func TestUnixStorageQuotaRetainsActionableCause(t *testing.T) {
	err := &os.PathError{Op: "write", Path: "private-owner-path", Err: syscall.EDQUOT}
	if got := contracts.SafeError(redact(context.Background(), err)); got.Code != "insufficient_space" {
		t.Fatal(got)
	}
	if got := contracts.SafeError(redact(context.Background(), syscall.EACCES)); got.Code != "unavailable" {
		t.Fatal("access refusal mislabeled as capacity", got)
	}
}
