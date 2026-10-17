//go:build !windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"errors"
	"fmt"
	"os"
)

func Run([]string) int {
	fmt.Fprintln(os.Stderr, "insonic setup EXE requires Windows x86-64")
	return 1
}
func CleanupUninstalled(string) error { return errors.New("MSI cleanup requires Windows") }
