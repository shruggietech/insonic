//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// Temporary investigation under T046. Observe only this disposable child's UI
// thread and its qualification keys. Do not modify messages or print before failure.
type nativeInputEvent struct {
	Hook    int     `json:"hook"`
	Window  uintptr `json:"window"`
	Message uint32  `json:"message"`
	Key     uintptr `json:"key"`
	Context uintptr `json:"context"`
	Time    uint32  `json:"time"`
}
type nativeInputTrace struct {
	mu      sync.Mutex
	events  []nativeInputEvent
	dropped int
	stopped bool
	hooks   []uintptr
}

var nativeExitInputTrace atomic.Pointer[nativeInputTrace]

func (t *nativeInputTrace) observe(hook int, m *nativeWindowMessage) {
	if m == nil || m.Message < 0x100 || m.Message > 0x107 {
		return
	}
	switch m.WParam {
	case 0x12, 0x1b, 0x46, 0x66, 0x58, 0x78, 0x73:
	default:
		return
	}
	event := nativeInputEvent{hook, m.Window, m.Message, m.WParam, m.LParam, m.Time}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return
	}
	if len(t.events) >= 96 {
		t.dropped++
		return
	}
	t.events = append(t.events, event)
}
func startNativeExitInputTrace(hwnd uintptr) error {
	var pid uint32
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || thread == 0 || pid != uint32(os.Getpid()) {
		return fmt.Errorf("input observation requires owned UI window")
	}
	if err := stopNativeExitInputTrace(); err != nil {
		return err
	}
	trace := &nativeInputTrace{}
	nativeExitInputTrace.Store(trace)
	for _, hook := range []int{3, -1} {
		callback := syscall.NewCallback(func(code, kind uintptr, message *nativeWindowMessage) uintptr {
			result := exitWinCall("CallNextHookEx", 0, code, kind, uintptr(unsafe.Pointer(message)))
			if (hook == 3 && int32(code) >= 0 && kind == 1) || (hook == -1 && int32(code) == 2) {
				trace.observe(hook, message)
			}
			return result
		})
		handle := exitWinCall("SetWindowsHookExW", uintptr(hook), callback, 0, thread)
		if handle == 0 {
			stopNativeExitInputTrace()
			return fmt.Errorf("owned input observation unavailable")
		}
		trace.mu.Lock()
		stopped := trace.stopped
		if !stopped {
			trace.hooks = append(trace.hooks, handle)
		}
		trace.mu.Unlock()
		if stopped {
			exitWinCall("UnhookWindowsHookEx", handle)
			return fmt.Errorf("input observer stopped during setup")
		}
	}
	return nil
}
func stopNativeExitInputTrace() error {
	var cleanupError error
	trace := nativeExitInputTrace.Swap(nil)
	if trace != nil {
		trace.mu.Lock()
		trace.stopped = true
		hooks := append([]uintptr(nil), trace.hooks...)
		trace.mu.Unlock()
		for _, hook := range hooks {
			if exitWinCall("UnhookWindowsHookEx", hook) == 0 {
				cleanupError = fmt.Errorf("owned input observer removal failed")
			}
		}
	}
	return cleanupError
}
func logNativeExitInputTrace(phase string) {
	trace := nativeExitInputTrace.Load()
	if trace == nil {
		return
	}
	trace.mu.Lock()
	events := append([]nativeInputEvent(nil), trace.events...)
	dropped := trace.dropped
	trace.mu.Unlock()
	raw, _ := json.Marshal(map[string]any{"kind": "native-input-observation", "phase": phase, "events": events, "dropped": dropped})
	fmt.Fprintln(os.Stderr, string(raw))
}

// Counterfactual cleanup after the watchdog has already marked failure. Releasing
// only a descendant focus on this owned UI thread cannot confer qualification.
func probeNativeExitInputAfterFailure(ctx context.Context) {
	if nativeExitInputTrace.Load() == nil {
		return
	}
	logNativeExitInputTrace("before-focus-release")
	hwnd := exitWinCall("GetForegroundWindow")
	if requireNativeInputTarget(hwnd) != nil {
		return
	}
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, 0)
	info := exitGUIThreadInfo{Size: uint32(unsafe.Sizeof(exitGUIThreadInfo{}))}
	if exitWinCall("GetGUIThreadInfo", thread, uintptr(unsafe.Pointer(&info))) == 0 || info.MenuOwner != hwnd || info.Flags&4 == 0 {
		return
	}
	probe, stop := context.WithTimeout(ctx, time.Second)
	defer stop()
	err := awaitNativeWindowOperation(probe, hwnd, func() {
		focus := exitWinCall("GetFocus")
		child := focus != 0 && exitWinCall("IsChild", hwnd, focus) != 0
		if child {
			exitWinCall("SetFocus", 0)
		}
		fmt.Fprintf(os.Stderr, "native-input-focus-release previous=%x child=%t after=%x\n", focus, child, exitWinCall("GetFocus"))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "native-input-focus-release ui-join-failed")
	}
	logNativeExitMenuState()
	time.AfterFunc(200*time.Millisecond, func() { logNativeExitInputTrace("after-focus-release") })
}
