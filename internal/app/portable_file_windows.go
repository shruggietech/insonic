//go:build windows

// SPDX-License-Identifier: Apache-2.0
package app

import "os"

func openPortablePin(path string) (*os.File, error) { return os.Open(path) }
