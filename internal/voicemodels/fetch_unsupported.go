//go:build !windows && !linux

// SPDX-License-Identifier: Apache-2.0
package voicemodels

import (
	"github.com/shruggietech/insonic/internal/contracts"
	"os"
)

func renameFetched(*os.Root, string, string) error { return contracts.Fail("unsupported_platform") }
