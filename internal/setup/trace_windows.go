//go:build windows

// SPDX-License-Identifier: Apache-2.0
package setup

import (
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/windows"
)

type nativeTrace struct {
	Progress uint64   `json:"progress_messages"`
	Actions  []string `json:"actions"`
	UILevel  uint32   `json:"ui_level"`
	Passive  bool     `json:"passive"`
}

// Native progress receipts accompany the private verbose log. The observer
// never cancels, replaces UI, invents progress or delays the Installer callback.
func observeMSI(directory string, ui uint32) (func() error, error) {
	var mutex sync.Mutex
	trace := nativeTrace{UILevel: ui, Passive: true}
	callback := windows.NewCallback(func(_ uintptr, kind uint32, text *uint16) uintptr {
		mutex.Lock()
		defer mutex.Unlock()
		switch kind & 0xff000000 {
		case 0x0a000000:
			trace.Progress++
		case 0x08000000:
			if len(trace.Actions) < 128 {
				value := windows.UTF16PtrToString(text)
				value = strings.TrimSpace(value)
				if len(value) > 350 {
					value = value[:350]
				}
				trace.Actions = append(trace.Actions, value)
			}
		}
		return 0
	})
	proc := installer.NewProc("MsiSetExternalUIW")
	if err := proc.Find(); err != nil {
		return nil, err
	}
	previous, _, _ := proc.Call(callback, 1<<8|1<<10, 0)
	return func() error {
		proc.Call(previous, 0, 0)
		mutex.Lock()
		defer mutex.Unlock()
		return atomicJSON(filepath.Join(directory, "setup-events.json"), trace)
	}, nil
}
