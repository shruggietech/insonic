//go:build (darwin && !cgo) || (!windows && !linux && !darwin)

// SPDX-License-Identifier: Apache-2.0
package secrets

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
)

type unavailableNative struct{}

func newNative(string) nativeBackend                      { return unavailableNative{} }
func (unavailableNative) available(context.Context) error { return contracts.Fail("unavailable") }
func (unavailableNative) get(context.Context, string) ([]byte, error) {
	return nil, contracts.Fail("unavailable")
}
func (unavailableNative) put(context.Context, string, []byte) error {
	return contracts.Fail("unavailable")
}
func (unavailableNative) remove(context.Context, string) error { return contracts.Fail("unavailable") }
