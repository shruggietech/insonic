// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
)

func TestDesktopMenuUsesNativePlatformLabelsAndOneOrdinaryQuit(t *testing.T) {
	for _, test := range []struct{ host, file, exit string }{{"windows", "&File", "E&xit"}, {"linux", "File", "Exit"}} {
		t.Run(test.host, func(t *testing.T) {
			calls := 0
			m := ApplicationMenu(test.host, func() { calls++ })
			if len(m.Items) != 1 || m.Items[0].Label != test.file || m.Items[0].SubMenu == nil {
				t.Fatal("native File menu absent")
			}
			items := m.Items[0].SubMenu.Items
			if len(items) != 1 || items[0].Label != test.exit || items[0].Disabled || items[0].Hidden || items[0].Accelerator != nil {
				t.Fatal("Exit unavailable or intercepts a global shortcut")
			}
			items[0].Click(nil)
			if calls != 1 {
				t.Fatal("native callback did not request ordinary quit")
			}
		})
	}
}

func TestQuitLifecycleQueuesUntilStartupAndDispatchesOnce(t *testing.T) {
	ctx := context.WithValue(context.Background(), struct{}{}, "lifecycle")
	var calls atomic.Int32
	q := NewQuitLifecycle(func(received context.Context) {
		if received != ctx {
			t.Error("quit lost native context")
		}
		calls.Add(1)
	})
	q.Request()
	q.Request()
	q.Ready(nil)
	if calls.Load() != 0 {
		t.Fatal("early request dispatched without a lifecycle context")
	}
	q.Ready(ctx)
	q.Request()
	q.Ready(ctx)
	if calls.Load() != 1 {
		t.Fatal("quit must dispatch exactly once")
	}
}

func TestQuitLifecycleConcurrentStartupAndRepeatedRequests(t *testing.T) {
	var calls atomic.Int32
	q := NewQuitLifecycle(func(context.Context) { calls.Add(1) })
	var group sync.WaitGroup
	for i := 0; i < 100; i++ {
		group.Add(2)
		go func() { defer group.Done(); q.Request() }()
		go func() { defer group.Done(); q.Ready(context.Background()) }()
	}
	group.Wait()
	if calls.Load() != 1 {
		t.Fatal("concurrent native requests dispatched more than once")
	}
}
