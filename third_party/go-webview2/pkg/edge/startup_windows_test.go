//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"errors"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/wailsapp/go-webview2/internal/w32"
	"golang.org/x/sys/windows"
)

type startupFixture struct {
	t                                            *testing.T
	view                                         *Chromium
	hwnd                                         uintptr
	thread                                       uint32
	failure                                      string
	queue                                        []func()
	callback                                     bool
	environment                                  *ICoreWebView2Environment
	controller                                   *ICoreWebView2Controller
	webview                                      *ICoreWebView2
	environmentRefs, controllerRefs, webviewRefs int
	closes, requests, registrations, scripts     int
	profile                                      string
}

func newStartupFixture(t *testing.T, failure string) *startupFixture {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)
	f := &startupFixture{t: t, thread: windows.GetCurrentThreadId(), failure: failure}
	f.view = NewChromium()
	f.view.DataPath = filepath.Join(t.TempDir(), "intended-profile")
	f.profile = f.view.DataPath
	class, _ := windows.UTF16PtrFromString("STATIC")
	title, _ := windows.UTF16PtrFromString("unshown startup fixture")
	user := windows.NewLazySystemDLL("user32.dll")
	f.hwnd, _, _ = user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if f.hwnd == 0 {
		t.Fatal("unshown message-only window unavailable")
	}
	before := ClosedControllerCount()
	t.Cleanup(func() {
		user.NewProc("DestroyWindow").Call(f.hwnd)
		ownedWindows.Lock()
		ownedWindows.closedControllers = before
		ownedWindows.Unlock()
		runtime.KeepAlive(f)
	})
	release := func(refs *int) ComProc {
		return NewComProc(func(uintptr) uintptr { f.assertOwnerOutsideCallback(); *refs--; return 0 })
	}
	register := NewComProc(func(this, handler, token uintptr) uintptr {
		f.registrations++
		if failure == "registration" {
			return 0x80070005
		}
		return 0
	})
	f.webview = &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{
		_IUnknownVtbl:         _IUnknownVtbl{Release: release(&f.webviewRefs)},
		AddWebMessageReceived: register, AddPermissionRequested: register,
		AddWebResourceRequested: register, AddNavigationCompleted: register,
		AddProcessFailed: register, AddContainsFullScreenElementChanged: register,
		AddScriptToExecuteOnDocumentCreated: NewComProc(func(this, script, callback uintptr) uintptr {
			f.scripts++
			if failure == "bootstrap" {
				return 0x80070005
			}
			return 0
		}),
	}}
	f.controller = &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		_IUnknownVtbl: _IUnknownVtbl{
			QueryInterface: NewComProc(func(this, iid uintptr, output *uintptr) uintptr { *output = 0; return 0x80004002 }),
			AddRef:         NewComProc(func(uintptr) uintptr { f.controllerRefs++; return uintptr(f.controllerRefs) }),
			Release:        release(&f.controllerRefs),
		},
		GetCoreWebView2: NewComProc(func(this uintptr, output **ICoreWebView2) uintptr {
			if failure == "webview" {
				return 0x80070005
			}
			if failure != "null-webview" {
				*output = f.webview
				f.webviewRefs++
			}
			return 0
		}),
		AddAcceleratorKeyPressed: register,
		Close:                    NewComProc(func(uintptr) uintptr { f.assertOwnerOutsideCallback(); f.closes++; return 0 }),
	}}
	f.environment = &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{
		_IUnknownVtbl: _IUnknownVtbl{
			AddRef:  NewComProc(func(uintptr) uintptr { f.environmentRefs++; return uintptr(f.environmentRefs) }),
			Release: release(&f.environmentRefs),
		},
		CreateCoreWebView2Controller: NewComProc(func(this, parent, handler uintptr) uintptr {
			f.requests++
			if failure == "controller-request" {
				return 0x80070005
			}
			f.queue = append(f.queue, func() {
				switch failure {
				case "failed-controller-object":
					f.view.CreateCoreWebView2ControllerCompleted(0x80070005, f.controller)
				case "controller-completion":
					f.view.CreateCoreWebView2ControllerCompleted(0x80070005, nil)
				case "null-controller":
					f.view.CreateCoreWebView2ControllerCompleted(0, nil)
				default:
					f.view.CreateCoreWebView2ControllerCompleted(0, f.controller)
				}
			})
			return 0
		}),
	}}
	return f
}
func (f *startupFixture) assertOwnerOutsideCallback() {
	f.t.Helper()
	if windows.GetCurrentThreadId() != f.thread || f.callback {
		f.t.Error("destructive COM cleanup ran off-owner or before callback unwinding")
	}
}
func (f *startupFixture) platform() startupPlatform {
	return startupPlatform{
		createEnvironment: func(browser, profile string, handler *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler, args string) error {
			if profile != f.profile {
				f.t.Fatal("intended profile changed")
			}
			if f.failure == "environment-create" {
				return syscall.Errno(5)
			}
			f.queue = append(f.queue, func() {
				switch f.failure {
				case "failed-environment-object":
					f.view.EnvironmentCompleted(0x80070005, f.environment)
				case "environment-completion":
					f.view.EnvironmentCompleted(0x80070005, nil)
				case "null-environment":
					f.view.EnvironmentCompleted(0, nil)
				default:
					f.view.EnvironmentCompleted(0, f.environment)
				}
			})
			return nil
		},
		runtimeVersion: func(string) (string, error) {
			if f.failure == "version" {
				return "", syscall.Errno(5)
			}
			return "152.0.4191.66", nil
		},
		nextMessage: func(*w32.Msg) (int32, error) {
			if f.failure == "message-error" {
				return -1, syscall.Errno(87)
			}
			if f.failure == "message-quit" || len(f.queue) == 0 {
				return 0, nil
			}
			return 1, nil
		},
		dispatch: func(*w32.Msg) { fn := f.queue[0]; f.queue = f.queue[1:]; f.callback = true; fn(); f.callback = false },
	}
}
func TestStartupFailuresCompleteAndReleaseAfterCallbackUnwinds(t *testing.T) {
	for _, c := range []struct {
		name, stage               string
		closes, requests, scripts int
	}{
		{"environment-create", "environment-create", 0, 0, 0},
		{"version", "runtime-version", 0, 0, 0},
		{"environment-completion", "environment-completion", 0, 0, 0},
		{"null-environment", "environment-completion", 0, 0, 0},
		{"failed-environment-object", "environment-completion", 0, 0, 0},
		{"controller-request", "controller-request", 0, 1, 0},
		{"controller-completion", "controller-completion", 0, 1, 0},
		{"null-controller", "controller-completion", 0, 1, 0},
		{"failed-controller-object", "controller-completion", 1, 1, 0},
		{"webview", "webview", 1, 1, 0},
		{"null-webview", "webview", 1, 1, 0},
		{"registration", "controller-configuration", 1, 1, 0},
		{"bootstrap", "bootstrap-script", 1, 1, 1},
		{"message-error", "message-loop", 0, 0, 0},
		{"message-quit", "message-loop", 0, 0, 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newStartupFixture(t, c.name)
			err := f.view.embedWithError(f.hwnd, f.platform())
			var startup *StartupError
			if !errors.As(err, &startup) || startup.Stage != c.stage {
				t.Fatalf("startup outcome missing: %v", err)
			}
			if c.name == "environment-completion" || c.name == "controller-completion" || c.name == "registration" {
				if !errors.Is(err, syscall.Errno(0x80070005)) {
					t.Fatal("original HRESULT lost", err)
				}
			}
			if f.closes != c.closes || f.requests != c.requests || f.scripts != c.scripts ||
				f.environmentRefs != 0 || f.controllerRefs != 0 || f.webviewRefs != 0 || len(f.view.callbackLeases) != 0 || !f.view.shuttingDown {
				t.Fatalf("failure fell through or leaked owned state: closes=%d requests=%d scripts=%d refs=%d/%d/%d leases=%d", f.closes, f.requests, f.scripts, f.environmentRefs, f.controllerRefs, f.webviewRefs, len(f.view.callbackLeases))
			}
			if c.name == "registration" && f.registrations != 1 {
				t.Fatal("registration continued after failure")
			}
			f.view.closeOwnedBrowser()
			if f.closes != c.closes {
				t.Fatal("cleanup closed the controller twice")
			}
		})
	}
}
func TestStartupSuccessOwnsResourcesUntilNormalWindowDestruction(t *testing.T) {
	f := newStartupFixture(t, "")
	if err := f.view.embedWithError(f.hwnd, f.platform()); err != nil {
		t.Fatal(err)
	}
	if f.closes != 0 || f.scripts != 1 || f.registrations != 7 || f.environmentRefs != 1 || f.controllerRefs != 1 || f.webviewRefs != 1 {
		t.Fatal("startup success was published before complete initialization")
	}
	windows.NewLazySystemDLL("user32.dll").NewProc("DestroyWindow").Call(f.hwnd)
	if f.closes != 1 || f.environmentRefs != 0 || f.controllerRefs != 0 || f.webviewRefs != 0 {
		t.Fatal("normal window cleanup failed")
	}
}
func TestStartupFirstFailureRefusesLateEnvironmentAndFreshInstanceSucceeds(t *testing.T) {
	failed := newStartupFixture(t, "version")
	err := failed.view.embedWithError(failed.hwnd, failed.platform())
	if !errors.Is(err, syscall.Errno(5)) {
		t.Fatal(err)
	}
	failed.view.EnvironmentCompleted(0, failed.environment)
	if failed.requests != 0 || failed.environmentRefs != 0 || failed.view.environment != nil {
		t.Fatal("late callback reopened failed startup")
	}
	if second := failed.view.failStartup("controller-completion", syscall.Errno(87)); second != err {
		t.Fatal("later failure replaced original outcome")
	}
	valid := newStartupFixture(t, "")
	if err = valid.view.embedWithError(valid.hwnd, valid.platform()); err != nil {
		t.Fatal("fresh instance could not start", err)
	}
}

func TestStartupCallbackFailurePrecedesLaterSynchronousError(t *testing.T) {
	f := newStartupFixture(t, "")
	platform := f.platform()
	platform.createEnvironment = func(string, string, *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler, string) error {
		f.callback = true
		f.view.EnvironmentCompleted(0x80070005, nil)
		f.callback = false
		return syscall.Errno(87)
	}
	err := f.view.embedWithError(f.hwnd, platform)
	if !errors.Is(err, syscall.Errno(0x80070005)) || errors.Is(err, syscall.Errno(87)) {
		t.Fatal("first callback failure was replaced", err)
	}
}
func TestStartupOwningWindowDestructionCompletesPendingWait(t *testing.T) {
	f := newStartupFixture(t, "")
	platform := f.platform()
	platform.dispatch = func(*w32.Msg) { windows.NewLazySystemDLL("user32.dll").NewProc("DestroyWindow").Call(f.hwnd) }
	err := f.view.embedWithError(f.hwnd, platform)
	var failure *StartupError
	if !errors.As(err, &failure) || failure.Stage != "window-destroyed" || len(f.view.callbackLeases) != 0 {
		t.Fatal("destroyed owner left startup pending", err)
	}
}
func TestStartupCleanupRemainsSingleDuringNativeReentrantDestruction(t *testing.T) {
	f := newStartupFixture(t, "bootstrap")
	f.controller.vtbl.Close = NewComProc(func(uintptr) uintptr {
		f.assertOwnerOutsideCallback()
		f.closes++
		windows.NewLazySystemDLL("user32.dll").NewProc("DestroyWindow").Call(f.hwnd)
		return 0
	})
	err := f.view.embedWithError(f.hwnd, f.platform())
	if !errors.Is(err, syscall.Errno(0x80070005)) || f.closes != 1 || f.controllerRefs != 0 || f.webviewRefs != 0 || f.environmentRefs != 0 {
		t.Fatal("reentrant cleanup replaced failure or duplicated owned cleanup", err)
	}
}

func TestStartupConfigurationReturnsNativeErrorsWithoutFatalCallbacks(t *testing.T) {
	f := newStartupFixture(t, "")
	f.view.hwnd = f.hwnd
	f.view.controller = f.controller
	f.view.webview = f.webview
	notified := 0
	f.view.SetErrorCallback(func(error) { notified++ })
	f.controller.vtbl.PutBounds = NewComProc(func(this, bounds uintptr) uintptr { return 0x80070005 })
	f.controller.vtbl.PutZoomFactor = NewComProc(func(this, value uintptr) uintptr { return 0x80070005 })
	f.webview.vtbl.Navigate = NewComProc(func(this, url uintptr) uintptr { return 0x80070005 })
	f.webview.vtbl.OpenDevToolsWindow = NewComProc(func(uintptr) uintptr { return 0x80070005 })
	f.webview.vtbl.AddWebResourceRequestedFilter = NewComProc(func(this, filter, kind uintptr) uintptr { return 0x80070005 })
	for _, operation := range []func() error{
		f.view.ResizeWithError,
		func() error { return f.view.NavigateWithError("about:blank") },
		func() error { return f.view.PutZoomFactorWithError(1) },
		f.view.OpenDevToolsWindowWithError,
		func() error {
			return f.view.AddWebResourceRequestedFilterWithError("*", COREWEBVIEW2_WEB_RESOURCE_CONTEXT_ALL)
		},
	} {
		if err := operation(); !errors.Is(err, syscall.Errno(0x80070005)) {
			t.Fatal("native setup cause lost", err)
		}
		if f.view.callError != nil || notified != 0 {
			t.Fatal("startup error escaped synchronous reporting")
		}
	}
	if err := f.view.SetBackgroundColourWithError(0, 0, 0, 255); !errors.Is(err, syscall.Errno(0x80004002)) {
		t.Fatal("background interface cause lost", err)
	}
	if f.view.callError != nil || notified != 0 {
		t.Fatal("background error escaped synchronous reporting")
	}
}

func TestStartupReentrantOwnerDestructionWaitsForControllerCallback(t *testing.T) {
	f := newStartupFixture(t, "")
	f.controller.vtbl.GetCoreWebView2 = NewComProc(func(this uintptr, output **ICoreWebView2) uintptr {
		windows.NewLazySystemDLL("user32.dll").NewProc("DestroyWindow").Call(f.hwnd)
		if f.closes != 0 {
			t.Error("controller closed before its completion callback unwound")
		}
		*output = f.webview
		f.webviewRefs++
		return 0
	})
	err := f.view.embedWithError(f.hwnd, f.platform())
	var failure *StartupError
	if !errors.As(err, &failure) || failure.Stage != "window-destroyed" || f.closes != 1 || f.registrations != 0 || f.scripts != 0 || f.controllerRefs != 0 || f.webviewRefs != 0 || f.environmentRefs != 0 {
		t.Fatal("reentrant owner destruction escaped startup cleanup", err)
	}
}
func TestStartupConfigurationWindowDestructionWaitsForNativeCall(t *testing.T) {
	f := newStartupFixture(t, "")
	if err := f.view.embedWithError(f.hwnd, f.platform()); err != nil {
		t.Fatal(err)
	}
	f.controller.vtbl.PutBounds = NewComProc(func(this, bounds uintptr) uintptr {
		f.callback = true
		windows.NewLazySystemDLL("user32.dll").NewProc("DestroyWindow").Call(f.hwnd)
		if f.closes != 0 {
			t.Error("controller closed inside native configuration call")
		}
		f.callback = false
		return 0
	})
	err := f.view.ResizeWithError()
	var failure *StartupError
	if !errors.As(err, &failure) || failure.Stage != "window-destroyed" || f.closes != 1 || f.controllerRefs != 0 || f.webviewRefs != 0 || f.environmentRefs != 0 {
		t.Fatal("native configuration lost owner destruction", err)
	}
}

func TestStartupBrowserProcessFailureReturnsBeforeFrameworkFailureHandler(t *testing.T) {
	for _, missing := range []bool{false, true} {
		t.Run(map[bool]string{false: "native-kind", true: "missing-details"}[missing], func(t *testing.T) {
			f := newStartupFixture(t, "")
			delegated := 0
			f.view.ProcessFailedCallback = func(*ICoreWebView2, *ICoreWebView2ProcessFailedEventArgs) { delegated++ }
			args := &ICoreWebView2ProcessFailedEventArgs{vtbl: &_ICoreWebView2ProcessFailedEventArgsVtbl{GetProcessFailedKind: NewComProc(func(this uintptr, kind *COREWEBVIEW2_PROCESS_FAILED_KIND) uintptr {
				*kind = COREWEBVIEW2_PROCESS_FAILED_KIND_BROWSER_PROCESS_EXITED
				return 0
			})}}
			if missing {
				args = nil
			}
			f.webview.vtbl.AddScriptToExecuteOnDocumentCreated = NewComProc(func(this, script, callback uintptr) uintptr {
				f.callback = true
				f.view.ProcessFailed(f.webview, args)
				if f.closes != 0 {
					t.Error("startup process failure closed inside native callback")
				}
				f.callback = false
				return 0
			})
			err := f.view.embedWithError(f.hwnd, f.platform())
			var failure *StartupError
			if !errors.As(err, &failure) || failure.Stage != "browser-process" || delegated != 0 || f.closes != 1 || f.controllerRefs != 0 || f.webviewRefs != 0 || f.environmentRefs != 0 {
				t.Fatal("startup process failure escaped returned-error cleanup", err, delegated)
			}
			f.view.ProcessFailed(nil, nil)
			if delegated != 0 {
				t.Fatal("late event invoked framework failure UI")
			}
		})
	}
}
func TestStartupConfigurationProcessFailureReturnsAndRuntimePolicyRemains(t *testing.T) {
	f := newStartupFixture(t, "")
	delegated := 0
	f.view.ProcessFailedCallback = func(*ICoreWebView2, *ICoreWebView2ProcessFailedEventArgs) { delegated++ }
	if err := f.view.embedWithError(f.hwnd, f.platform()); err != nil {
		t.Fatal(err)
	}
	f.controller.vtbl.PutBounds = NewComProc(func(this, bounds uintptr) uintptr { f.view.ProcessFailed(f.webview, nil); return 0 })
	err := f.view.ResizeWithError()
	var failure *StartupError
	if !errors.As(err, &failure) || failure.Stage != "browser-process" || delegated != 0 {
		t.Fatal("configuration process failure escaped synchronous reporting", err, delegated)
	}
	f.view.ProcessFailed(f.webview, nil)
	if delegated != 1 {
		t.Fatal("ordinary post-startup process callback policy changed")
	}
}

func TestStartupNativeCallsHoldResourcesUntilTheyReturn(t *testing.T) {
	for _, boundary := range []string{"factory", "version", "bootstrap"} {
		t.Run(boundary, func(t *testing.T) {
			f := newStartupFixture(t, "")
			platform := f.platform()
			destroy := func() {
				f.callback = true
				windows.NewLazySystemDLL("user32.dll").NewProc("DestroyWindow").Call(f.hwnd)
				if len(f.view.callbackLeases) == 0 || f.closes != 0 {
					t.Error("native startup call released resources before returning")
				}
				f.callback = false
			}
			switch boundary {
			case "factory":
				platform.createEnvironment = func(string, string, *iCoreWebView2CreateCoreWebView2EnvironmentCompletedHandler, string) error {
					destroy()
					return nil
				}
			case "version":
				platform.runtimeVersion = func(string) (string, error) { destroy(); return "152.0.4191.66", nil }
			case "bootstrap":
				f.webview.vtbl.AddScriptToExecuteOnDocumentCreated = NewComProc(func(this, script, callback uintptr) uintptr { destroy(); return 0 })
			}
			err := f.view.embedWithError(f.hwnd, platform)
			var failure *StartupError
			if !errors.As(err, &failure) || failure.Stage != "window-destroyed" || len(f.view.callbackLeases) != 0 || f.controllerRefs != 0 || f.webviewRefs != 0 || f.environmentRefs != 0 {
				t.Fatal("native call lifetime failure was not returned and balanced", err)
			}
		})
	}
}
