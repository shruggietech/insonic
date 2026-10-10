//go:build !windows && !linux && !darwin

// SPDX-License-Identifier: Apache-2.0
package backup

import "github.com/shruggietech/insonic/internal/contracts"

func publishDirectory(string, string) error { return contracts.Fail("unsupported_capability") }
