//go:build !system_ladybug

// SPDX-License-Identifier: Apache-2.0
package graph

import "github.com/shruggietech/insonic/internal/contracts"

// Untagged source builds never impersonate the real native engine.
func OpenLadybug(string) (Adapter, error) { return nil, contracts.Fail("unavailable") }
