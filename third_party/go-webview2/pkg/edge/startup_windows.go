//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/wailsapp/go-webview2/internal/w32"
	"github.com/wailsapp/go-webview2/webviewloader"
	"golang.org/x/sys/windows"
)

// StartupError retains the generated boundary and original native cause.
type StartupError struct {
	Stage string
	Err   error
}

func (e *StartupError) Error() string        { return fmt.Sprintf("WebView2 %s: %v", e.Stage, e.Err) }
func (e *StartupError) Unwrap() error        { return e.Err }
func (e *StartupError) StartupStage() string { return e.Stage }

type startupOutcome struct {
	failure   error
	succeeded bool
}
type startupPlatform struct {
	createEnvironment func(string, string, *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler, string) error
	runtimeVersion    func(string) (string, error)
	nextMessage       func(*w32.Msg) (int32, error)
	dispatch          func(*w32.Msg)
}

func nativeStartupPlatform() startupPlatform {
	return startupPlatform{
		createEnvironment: createCoreWebView2EnvironmentWithOptions,
		runtimeVersion:    webviewloader.GetAvailableCoreWebView2BrowserVersionString,
		nextMessage: func(msg *w32.Msg) (int32, error) {
			result, _, err := w32.User32GetMessageW.Call(uintptr(unsafe.Pointer(msg)), 0, 0, 0)
			return int32(result), err
		},
		dispatch: func(msg *w32.Msg) {
			w32.User32TranslateMessage.Call(uintptr(unsafe.Pointer(msg)))
			w32.User32DispatchMessageW.Call(uintptr(unsafe.Pointer(msg)))
		},
	}
}
func (e *Chromium) failStartup(stage string, cause error) error {
	if e.startup == nil {
		e.startup = &startupOutcome{}
	}
	if e.startup.failure == nil && !e.startup.succeeded {
		e.startup.failure = wrapStartupError(stage, cause)
	}
	return e.startup.failure
}
func startupCompletionError(kind string, result uintptr, missing bool) error {
	if int32(result) < 0 {
		return fmt.Errorf("%s failed (HRESULT 0x%08x): %w", kind, uint32(result), syscall.Errno(uint32(result)))
	}
	if missing {
		return fmt.Errorf("%s returned no object (HRESULT 0x%08x)", kind, uint32(result))
	}
	return nil
}

// EmbedWithError waits for controller configuration and bootstrap request acceptance.
// Its caller owns the window and must retain its UI thread for the full lifetime.
func (e *Chromium) EmbedWithError(hwnd uintptr) error {
	return e.EmbedWithConfiguration(hwnd, nil)
}

// EmbedWithConfiguration keeps one pending startup outcome through the host
// configuration, before the caller may publish frontend readiness.
func (e *Chromium) EmbedWithConfiguration(hwnd uintptr, configure func() error) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	return e.embedWithConfiguration(hwnd, nativeStartupPlatform(), configure)
}
func (e *Chromium) embedWithError(hwnd uintptr, platform startupPlatform) error {
	return e.embedWithConfiguration(hwnd, platform, nil)
}
func (e *Chromium) embedWithConfiguration(hwnd uintptr, platform startupPlatform, configure func() error) (err error) {
	if e.startup != nil || e.shuttingDown {
		return &StartupError{Stage: "window-lifetime", Err: fmt.Errorf("Chromium startup is single-use; create a fresh instance")}
	}
	e.startup = &startupOutcome{}
	e.hwnd = hwnd
	defer func() {
		if err != nil {
			e.closeOwnedBrowser()
		}
	}()
	if err = e.attachWindowLifetime(hwnd); err != nil {
		return e.failStartup("window-lifetime", err)
	}
	e.prepareOwnedCallbacks()
	dataPath := e.DataPath
	if dataPath == "" {
		config, pathErr := os.UserConfigDir()
		if pathErr != nil {
			return e.failStartup("profile-directory", pathErr)
		}
		executable := make([]uint16, windows.MAX_PATH)
		if _, pathErr = windows.GetModuleFileName(0, &executable[0], uint32(len(executable))); pathErr != nil {
			return e.failStartup("profile-directory", pathErr)
		}
		dataPath = filepath.Join(config, filepath.Base(windows.UTF16ToString(executable)))
	}
	// An existing non-directory cannot be a browser profile. Return this
	// concrete path failure before handing it to native initialization.
	if info, pathErr := os.Stat(dataPath); pathErr == nil && !info.IsDir() {
		return e.failStartup("profile-directory", &os.PathError{Op: "open browser profile directory", Path: dataPath, Err: windows.ERROR_DIRECTORY})
	}
	if e.BrowserPath != "" {
		if _, pathErr := os.Stat(e.BrowserPath); pathErr != nil {
			return e.failStartup("browser-path", pathErr)
		}
	}
	if err = e.callWithError(func() {
		if nativeErr := platform.createEnvironment(e.BrowserPath, dataPath, e.envCompleted, strings.Join(e.AdditionalBrowserArgs, " ")); nativeErr != nil {
			e.errorCallback(nativeErr)
		}
	}); err != nil {
		return e.failStartup("environment-create", err)
	}
	if e.startup.failure != nil {
		return e.startup.failure
	}
	if err = e.callWithError(func() {
		var nativeErr error
		e.webview2RuntimeVersion, nativeErr = platform.runtimeVersion(e.BrowserPath)
		if nativeErr != nil {
			e.errorCallback(nativeErr)
		}
	}); err != nil {
		return e.failStartup("runtime-version", err)
	}
	var msg w32.Msg
	for atomic.LoadUintptr(&e.inited) == 0 && e.startup.failure == nil {
		result, messageErr := platform.nextMessage(&msg)
		if result <= 0 {
			if result == 0 {
				messageErr = fmt.Errorf("startup message loop received WM_QUIT")
			} else if messageErr == nil || messageErr == syscall.Errno(0) {
				messageErr = fmt.Errorf("GetMessage failed")
			}
			return e.failStartup("message-loop", messageErr)
		}
		platform.dispatch(&msg)
	}
	if e.startup.failure != nil {
		return e.startup.failure
	}
	if e.shuttingDown || e.webview == nil {
		return e.failStartup("webview", fmt.Errorf("startup ended without an owned webview"))
	}
	if err = e.InitWithError("window.external={invoke:s=>window.chrome.webview.postMessage(s)}"); err != nil {
		return e.failStartup("bootstrap-script", err)
	}
	if e.startup.failure != nil {
		return e.startup.failure
	}
	if configure != nil {
		var configurationError error
		callbackError := e.callWithError(func() { configurationError = configure() })
		if e.startup.failure != nil {
			return e.startup.failure
		}
		if callbackError != nil {
			return e.failStartup("controller-configuration", callbackError)
		}
		if configurationError != nil {
			return e.failStartup("controller-configuration", configurationError)
		}
	}
	e.startup.succeeded = true
	return nil
}

// Existing void SDK methods keep their architecture-specific implementations.
// These bounded synchronous facades return their native error on the owner UI
// thread; callback capture is scoped to the call and restored on every return.
func (e *Chromium) callWithError(operation func()) (err error) {
	previous := e.callError
	e.callError = &err
	defer func() {
		e.callError = previous
		if previous == nil && e.shuttingDown && !e.browserClosed {
			e.closeOwnedBrowser()
		}
	}()
	operation()
	return err
}
func (e *Chromium) ResizeWithError() error {
	return e.callWithStartupError("controller-configuration", e.Resize)
}
func (e *Chromium) NavigateWithError(url string) error {
	return e.callWithStartupError("webview", func() { e.Navigate(url) })
}
func (e *Chromium) PutZoomFactorWithError(factor float64) error {
	return e.callWithStartupError("controller-configuration", func() { e.PutZoomFactor(factor) })
}
func (e *Chromium) OpenDevToolsWindowWithError() error {
	return e.callWithStartupError("controller-configuration", e.OpenDevToolsWindow)
}
func (e *Chromium) AddWebResourceRequestedFilterWithError(filter string, context COREWEBVIEW2_WEB_RESOURCE_CONTEXT) error {
	return e.callWithStartupError("controller-configuration", func() { e.AddWebResourceRequestedFilter(filter, context) })
}
func (e *Chromium) SetBackgroundColourWithError(r, g, b, a uint8) error {
	return e.callWithStartupError("controller-configuration", func() { e.SetBackgroundColour(r, g, b, a) })
}

func (e *Chromium) startupInterrupted() bool {
	return e.shuttingDown || e.startup != nil && e.startup.failure != nil
}

func (e *Chromium) InitWithError(script string) error {
	return e.callWithStartupError("bootstrap-script", func() { e.Init(script) })
}

func wrapStartupError(stage string, cause error) error {
	var native *StartupError
	if errors.As(cause, &native) {
		return cause
	}
	return &StartupError{Stage: stage, Err: cause}
}
func (e *Chromium) callWithStartupError(stage string, operation func()) error {
	err := e.callWithError(operation)
	if err == nil {
		return nil
	}
	return wrapStartupError(stage, err)
}
