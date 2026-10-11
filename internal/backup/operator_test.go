// SPDX-License-Identifier: Apache-2.0
package backup

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"io"
	"strings"
	"testing"
)

func TestCorruptBackupDescribesBytesRatherThanOwnership(t *testing.T) {
	for _, test := range []struct {
		body           string
		size           int64
		digest, reason string
	}{{"bad", 3, strings.Repeat("0", 64), "backup_checksum"}, {"too long", 3, strings.Repeat("0", 64), "backup_size"}} {
		err := hashStream(context.Background(), io.Discard, strings.NewReader(test.body), test.size, test.digest)
		typed, ok := err.(*contracts.Error)
		if !ok || typed.Reason != test.reason || strings.Contains(typed.Message, "owner") {
			t.Fatal(err)
		}
	}
}
