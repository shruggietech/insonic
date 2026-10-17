// SPDX-License-Identifier: Apache-2.0
package desktop

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/menu"
)

// ApplicationMenu uses host-native labels without reserving a global shortcut.
// GTK's pinned adapter does not interpret Windows ampersand mnemonics.
func ApplicationMenu(host string, requestQuit func()) *menu.Menu {
	file, exit := "File", "Exit"
	if host == "windows" {
		file, exit = "&File", "E&xit"
	}
	app := menu.NewMenu()
	app.AddSubmenu(file).AddText(exit, nil, func(*menu.CallbackData) { requestQuit() })
	return app
}

// QuitLifecycle queues a menu request until native startup provides its context.
// Wails creates menus before it invokes OnStartup asynchronously.
type QuitLifecycle struct {
	mu                    sync.Mutex
	ctx                   context.Context
	requested, dispatched bool
	quit                  func(context.Context)
}

func NewQuitLifecycle(quit func(context.Context)) *QuitLifecycle { return &QuitLifecycle{quit: quit} }

func (q *QuitLifecycle) Ready(ctx context.Context) {
	if ctx == nil {
		return
	}
	q.mu.Lock()
	if q.ctx == nil {
		q.ctx = ctx
	}
	dispatch := q.requested && !q.dispatched
	if dispatch {
		q.dispatched = true
	}
	selected := q.ctx
	q.mu.Unlock()
	if dispatch {
		q.quit(selected)
	}
}

func (q *QuitLifecycle) Request() {
	q.mu.Lock()
	q.requested = true
	dispatch := q.ctx != nil && !q.dispatched
	if dispatch {
		q.dispatched = true
	}
	selected := q.ctx
	q.mu.Unlock()
	if dispatch {
		q.quit(selected)
	}
}
