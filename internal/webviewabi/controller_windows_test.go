//go:build windows

// SPDX-License-Identifier: Apache-2.0
package webviewabi

import (
	"errors"
	"runtime"
	"syscall"
	"testing"

	"github.com/wailsapp/go-webview2/pkg/edge"
)

// Call the actual production wrapper through a native vtable, without creating
// a browser/window. A pointer to a bool is not a canonical native BOOL value.
func TestControllerMonitorScalePassesBOOLByValue(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for _, value := range []bool{false, true} {
		name, want := "false", uintptr(0)
		if value {
			name, want = "true", 1
		}
		t.Run(name, func(t *testing.T) {
			calls := 0
			var got uintptr
			callback := syscall.NewCallback(func(this, argument uintptr) uintptr { calls++; got = argument; return 0 })
			controller := &edge.ICoreWebView2Controller3{Vtbl: &edge.ICoreWebView2Controller3Vtbl{PutShouldDetectMonitorScaleChanges: edge.ComProc(callback)}}
			err := controller.PutShouldDetectMonitorScaleChanges(value)
			runtime.KeepAlive(controller)
			if err != nil || calls != 1 || got != want {
				t.Fatalf("BOOL argument canonical=%t expected=%d calls=%d error=%v", got == want, want, calls, err)
			}
		})
	}
}

func TestControllerMonitorScalePreservesHRESULT(t *testing.T) {
	const failure = uintptr(0x80004005)
	calls := 0
	callback := syscall.NewCallback(func(this, argument uintptr) uintptr { calls++; return failure })
	controller := &edge.ICoreWebView2Controller3{Vtbl: &edge.ICoreWebView2Controller3Vtbl{PutShouldDetectMonitorScaleChanges: edge.ComProc(callback)}}
	err := controller.PutShouldDetectMonitorScaleChanges(false)
	runtime.KeepAlive(controller)
	if calls != 1 || !errors.Is(err, syscall.Errno(failure)) {
		t.Fatalf("HRESULT propagation changed: calls=%d error=%v", calls, err)
	}
}
