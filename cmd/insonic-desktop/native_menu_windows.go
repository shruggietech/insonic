//go:build desktop && windows

// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shruggietech/insonic/desktop"
	"github.com/wailsapp/go-webview2/pkg/edge"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

var exitUser32 = windows.NewLazySystemDLL("user32.dll")

// Go can relocate a growing stack. Preserve pointer-valued uintptr arguments
// across this wrapper, including work performed before LazyProc.Call.
//
//go:uintptrescapes
func exitWinCall(name string, args ...uintptr) uintptr {
	value, _, _ := exitUser32.NewProc(name).Call(args...)
	return value
}
func exitWinLabel(handle uintptr, index uintptr) string {
	var label [64]uint16
	length := exitWinCall("GetMenuStringW", handle, index, uintptr(unsafe.Pointer(&label[0])), uintptr(len(label)), 0x400)
	return windows.UTF16ToString(label[:min(int(length), len(label))])
}
func exitOwnedWindow() (uintptr, error) { return findNativeExitWindow(true) }

func prepareNativeExitWindow(ctx context.Context) error {
	hwnd, err := findNativeExitWindow(false)
	if err != nil {
		return err
	}
	err = awaitNativeWindowOperation(ctx, hwnd, func() {
		before := exitWinCall("GetFocus")
		// A root that already owns Win32 focus would receive no new WM_SETFOCUS,
		// leaving Wails' controller focus unchanged after navigation Hide/Show.
		// This is only the fixture's owned UI thread, before any input.
		exitWinCall("SetFocus", 0)
		// The first show may consume STARTUPINFO's hide flag. Keep Wails' own
		// visibility/focus/controller behavior and complete both calls inline.
		wailsruntime.WindowShow(ctx)
		wailsruntime.WindowShow(ctx)
		fmt.Fprintf(os.Stderr, "native-exit-focus before=%x after=%x root=%x\n", before, exitWinCall("GetFocus"), hwnd)
	})
	if err != nil {
		return err
	}
	if exitWinCall("IsWindowVisible", hwnd) == 0 {
		return fmt.Errorf("owned native fixture could not be shown before readiness")
	}
	fmt.Fprintln(os.Stderr, "native-exit-stage=window-prepared")
	return nil
}

// MSG is borrowed only while Windows invokes the owned thread's queue hook.
type nativeWindowMessage struct {
	Window         uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Point          struct{ X, Y int32 }
	Private        uint32
}

var nativeWindowJoinSequence atomic.Uint32

// Execute the fixture operation on its owning UI thread. Wails Invoke then
// executes inline, including its controller focus, before readiness is signaled.
// The short-lived hook is removed before any native keyboard input begins.
func awaitNativeWindowOperation(ctx context.Context, hwnd uintptr, operation func()) (err error) {
	if err = ctx.Err(); err != nil {
		return err
	}
	var pid uint32
	thread := exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || thread == 0 || pid != uint32(os.Getpid()) || operation == nil {
		return fmt.Errorf("native UI join requires this process's owned window")
	}
	started := time.Now()
	deadline, stop := context.WithTimeout(ctx, 2*time.Second)
	defer stop()
	var entered, returned atomic.Int64
	defer func() {
		if err != nil {
			at, _ := deadline.Deadline()
			begin, end := entered.Load(), returned.Load()
			raw, _ := json.Marshal(map[string]any{"kind": "native-ui-join-failure", "budget_ns": at.Sub(started).Nanoseconds(), "entered_ns": begin, "returned_ns": end, "observed_ns": time.Since(started).Nanoseconds()})
			fmt.Fprintln(os.Stderr, string(raw))
			logNativeUIFunctionStacks()
			logNativeStartupProgress()
		}
	}()
	ready := make(chan struct{})
	var once sync.Once
	token := uintptr(0x5330333900000000) | uintptr(nativeWindowJoinSequence.Add(1))
	callback := syscall.NewCallback(func(code, kind uintptr, message *nativeWindowMessage) uintptr {
		result := exitWinCall("CallNextHookEx", 0, code, kind, uintptr(unsafe.Pointer(message)))
		if int32(code) >= 0 && kind == 1 && message != nil && message.Window == hwnd && message.Message == 0 && message.WParam == token {
			once.Do(func() {
				if deadline.Err() == nil {
					entered.Store(time.Since(started).Nanoseconds())
					operation()
					returned.Store(time.Since(started).Nanoseconds())
				}
				close(ready)
			})
		}
		return result
	})
	// WH_GETMESSAGE is confined to this process's specific owned UI thread.
	// Only our unique marker runs an operation; every message stays unchanged.
	hook := exitWinCall("SetWindowsHookExW", 3, callback, 0, thread)
	if hook == 0 {
		return fmt.Errorf("owned native UI completion observation unavailable")
	}
	defer func() {
		if exitWinCall("UnhookWindowsHookEx", hook) == 0 && err == nil {
			err = fmt.Errorf("owned native UI completion observer was not removed")
		}
	}()
	if exitWinCall("PostMessageW", hwnd, 0, token, 0) == 0 {
		return fmt.Errorf("owned native UI completion marker rejected")
	}
	select {
	case <-ready:
		if err = deadline.Err(); err != nil {
			return fmt.Errorf("owned native UI operation did not finish: %w", err)
		}
		return nil
	case <-deadline.Done():
		return fmt.Errorf("owned native UI operation did not finish: %w", deadline.Err())
	}
}

func findNativeExitWindow(requireVisible bool) (uintptr, error) {
	deadline := time.Now().Add(5 * time.Second)
	var handles []uintptr
	var owned, visible, active int
	callback := syscall.NewCallback(func(hwnd, unused uintptr) uintptr {
		var pid uint32
		exitWinCall("GetWindowThreadProcessId", hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid != uint32(os.Getpid()) || exitWinCall("GetMenu", hwnd) == 0 {
			return 1
		}
		var title [64]uint16
		exitWinCall("GetWindowTextW", hwnd, uintptr(unsafe.Pointer(&title[0])), uintptr(len(title)))
		if windows.UTF16ToString(title[:]) == "insonic" {
			owned++
			shown := exitWinCall("IsWindowVisible", hwnd) != 0
			foreground := exitWinCall("GetForegroundWindow") == hwnd
			if shown {
				visible++
			}
			if foreground {
				active++
			}
			if (!requireVisible || shown) && exitWinCall("IsWindowEnabled", hwnd) != 0 {
				handles = append(handles, hwnd)
			}
		}
		return 1
	})
	for time.Now().Before(deadline) {
		handles = nil
		owned, visible, active = 0, 0, 0
		exitWinCall("EnumWindows", callback, 0)
		if len(handles) > 1 {
			return 0, fmt.Errorf("ambiguous owned native window")
		}
		if len(handles) == 1 {
			return handles[0], nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return 0, fmt.Errorf("owned native window unavailable (owned=%d, visible=%d, active=%d)", owned, visible, active)
}

var nativeSystemMenuRequested atomic.Bool

func validateNativeFileMenu(hwnd uintptr) error {
	if err := requireNativeInputTarget(hwnd); err != nil {
		return err
	}
	root := exitWinCall("GetMenu", hwnd)
	file := exitWinCall("GetSubMenu", root, 0)
	if root == 0 || file == 0 || exitWinLabel(root, 0) != "&File" || exitWinLabel(file, 0) != "E&xit" {
		return fmt.Errorf("owned native File menu unavailable")
	}
	return nil
}

func configureNativeMenu(bridge *desktop.Bridge, diagnostic bool) func() {
	edge.EnableWindowBootstrapDiagnostics(diagnostic)
	nativeSystemMenuRequested.Store(false)
	edge.SetNativeSystemKeyHandler(func(hwnd uintptr, event edge.NativeSystemKeyEvent) bool {
		if event.VirtualKey != 0x46 || !event.PhysicalKeyStatus.IsMenuKeyDown || event.LParam&(1<<29) == 0 ||
			exitWinCall("GetKeyState", 0x11)&0x8000 != 0 || exitWinCall("GetKeyState", 0x5b)&0x8000 != 0 || exitWinCall("GetKeyState", 0x5c)&0x8000 != 0 {
			return false
		}
		if validateNativeFileMenu(hwnd) != nil {
			return false
		}
		// Post the actual WM_SYSKEYDOWN after this synchronous WebView callback
		// returns. Preserve Alt/scan/repeat context and avoid a DOM/Go modal request.
		if exitWinCall("PostMessageW", hwnd, 0x104, uintptr(event.VirtualKey), uintptr(event.LParam)) == 0 {
			return false
		}
		nativeSystemMenuRequested.Store(true)
		if diagnostic {
			fmt.Fprintln(os.Stderr, "native-exit-stage=menu-native-system-key")
		}
		return true
	})
	desktop.SetNativeFileMenu(bridge, func() error {
		// The focused document supplies the intent; select only its owned
		// foreground native menu, without allocating an EnumWindows callback.
		hwnd := exitWinCall("GetForegroundWindow")
		if err := validateNativeFileMenu(hwnd); err != nil {
			return err
		}
		if exitWinCall("PostMessageW", hwnd, 0x112, 0xf100, 'f') == 0 {
			return fmt.Errorf("native File menu request rejected")
		}
		if diagnostic {
			fmt.Fprintln(os.Stderr, "native-exit-stage=menu-mnemonic")
		}
		return nil
	})
	return func() { edge.SetNativeSystemKeyHandler(nil); edge.EnableWindowBootstrapDiagnostics(false) }
}

func verifyNativeMenuRoute(action string) error {
	if action == "menu" && !nativeSystemMenuRequested.Load() {
		return fmt.Errorf("native system-key File menu route was not observed")
	}
	return nil
}

func waitNativeMenuReady(ctx context.Context, bridge *desktop.Bridge) error {
	return desktop.WaitNativeFileMenuReady(ctx, bridge)
}
