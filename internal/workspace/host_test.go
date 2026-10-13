// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
)

func TestSupportedHostBoundary(t *testing.T) {
	for _, system := range []string{"windows", "linux"} {
		if err := ValidateHost(system); err != nil {
			t.Fatal(system, err)
		}
	}
	for _, system := range []string{"darwin", "freebsd", ""} {
		err := ValidateHost(system)
		if err == nil {
			t.Fatal("unsupported host accepted", system)
		}
		if e, ok := err.(*contracts.Error); !ok || e.Code != "unsupported_platform" {
			t.Fatal("wrong diagnostic", err)
		}
	}
}
