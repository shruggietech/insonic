//go:build windows && !native_webview2loader

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/combridge"
)

func useCapturedEnvironmentTarget(t *testing.T, finalized chan struct{}) {
	t.Helper()
	e := NewChromium()
	e.prepareOwnedCallbacks()
	runtime.SetFinalizer(e, func(*Chromium) { finalized <- struct{}{} })
	pointer := uintptr(unsafe.Pointer(e.envCompleted))
	captured := captureEnvironmentHandler(e.envCompleted)
	e.ShuttingDown()
	e.releaseOwnedInterfaces()
	if combridge.Resolve[*Chromium](pointer) != nil {
		t.Fatal("native environment callback creator lease remained rooted")
	}
	// Only the default loader's captured Go adapter keeps the target alive.
	e = nil
	for i := 0; i < 3; i++ {
		runtime.GC()
		runtime.Gosched()
		select {
		case <-finalized:
			t.Fatal("default loader lost its captured environment target")
		default:
		}
	}
	// The former native callback allocation has already been freed. The actual
	// default-loader adapter must invoke its captured Go target without it.
	if captured.EnvironmentCompleted(0, nil) != 0 {
		t.Fatal("late default-loader completion failed")
	}
	runtime.KeepAlive(captured)
}
func TestNativeCallbackBridgeDefaultLoaderCapturesGoTarget(t *testing.T) {
	finalized := make(chan struct{}, 1)
	useCapturedEnvironmentTarget(t, finalized)
	collectCallbackTarget(t, finalized)
}
