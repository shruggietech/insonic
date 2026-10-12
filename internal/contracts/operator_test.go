// SPDX-License-Identifier: Apache-2.0
package contracts

import (
	"strings"
	"testing"
)

func TestGenericConflictDoesNotInventAnOwner(t *testing.T) {
	if strings.Contains(strings.ToLower(Fail("conflict").Message), "owner") {
		t.Fatal("generic conflict falsely asserts ownership")
	}
}
