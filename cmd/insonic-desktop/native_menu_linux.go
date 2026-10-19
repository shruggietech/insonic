//go:build desktop && linux

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"github.com/shruggietech/insonic/desktop"
)

func configureNativeMenu(*desktop.Bridge, bool)                  {}
func waitNativeMenuReady(context.Context, *desktop.Bridge) error { return nil }
