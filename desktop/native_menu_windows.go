// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
)

// SetNativeFileMenu binds only the application's native window operation.
func SetNativeFileMenu(b *Bridge, open func() error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nativeMenuOpener = open
	if b.nativeMenuReady == nil {
		b.nativeMenuReady = make(chan struct{})
	}
}

func (b *Bridge) OpenFileMenu() contracts.Response {
	b.mu.RLock()
	open := b.nativeMenuOpener
	b.mu.RUnlock()
	if open == nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.Fail("unavailable")}
	}
	if err := open(); err != nil {
		return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Error: contracts.SafeError(err)}
	}
	return contracts.Response{Kind: "runtime-response", Version: contracts.Version, Result: map[string]any{"native_file_menu": "requested"}}
}

// NativeMenuReady acknowledges an installed frontend listener, not Exit.
func (b *Bridge) NativeMenuReady() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.nativeMenuOpener == nil {
		return false
	}
	if b.nativeMenuReady == nil {
		b.nativeMenuReady = make(chan struct{})
	}
	if !b.nativeMenuBound {
		b.nativeMenuBound = true
		close(b.nativeMenuReady)
	}
	return true
}

func WaitNativeFileMenuReady(ctx context.Context, b *Bridge) error {
	b.mu.Lock()
	if b.nativeMenuReady == nil {
		b.nativeMenuReady = make(chan struct{})
	}
	ready := b.nativeMenuReady
	b.mu.Unlock()
	select {
	case <-ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
