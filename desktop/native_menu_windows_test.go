// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"github.com/shruggietech/insonic/internal/contracts"
	"testing"
	"time"
)

func TestNativeFileMenuBridgeRequiresListenerAndOwnedOpener(t *testing.T) {
	bridge := &Bridge{}
	if bridge.OpenFileMenu().Error == nil || bridge.NativeMenuReady() {
		t.Fatal("unconfigured menu accepted")
	}
	calls := 0
	SetNativeFileMenu(bridge, func() error { calls++; return nil })
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if WaitNativeFileMenuReady(ctx, bridge) == nil {
		t.Fatal("unbound frontend considered ready")
	}
	if !bridge.NativeMenuReady() || WaitNativeFileMenuReady(context.Background(), bridge) != nil {
		t.Fatal("registered listener not observed")
	}
	if bridge.OpenFileMenu().Error != nil || calls != 1 {
		t.Fatal("native opener not called")
	}
	SetNativeFileMenu(bridge, func() error { return contracts.Fail("unavailable") })
	if bridge.OpenFileMenu().Error == nil {
		t.Fatal("native open failure became success")
	}
}
