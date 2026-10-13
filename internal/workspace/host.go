// SPDX-License-Identifier: Apache-2.0
package workspace

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"runtime"
)

// ValidateHost describes the owned runtime support boundary without side effects.
func ValidateHost(system string) error {
	if system != "windows" && system != "linux" {
		return contracts.Fail("unsupported_platform")
	}
	return nil
}

func RequireSupportedHost() error { return ValidateHost(runtime.GOOS) }
