//go:build desktop && !windows && !linux

// SPDX-License-Identifier: Apache-2.0
package main

import "fmt"

func presentStartupFailure(string) error { return fmt.Errorf("desktop host is unsupported") }
