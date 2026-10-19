//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func TestOwnedWindowDestructionClosesControllerOnceOnItsUIThread(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	thread := windows.GetCurrentThreadId()
	before := ClosedControllerCount()
	defer func() { ownedWindows.Lock(); ownedWindows.closedControllers = before; ownedWindows.Unlock() }()
	user := windows.NewLazySystemDLL("user32.dll")
	class, _ := windows.UTF16PtrFromString("STATIC")
	title, _ := windows.UTF16PtrFromString("owned WebView lifetime fixture")
	// A message-only window is never shown or activated.
	hwnd, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("message-only fixture window unavailable")
	}
	defer user.NewProc("DestroyWindow").Call(hwnd)
	if err := (&Chromium{}).attachWindowLifetime(0); err == nil {
		t.Fatal("invalid native window acquired a lifetime owner")
	}
	foreignThread := make(chan error, 1)
	go func() { foreignThread <- (&Chromium{}).attachWindowLifetime(hwnd) }()
	if err := <-foreignThread; err == nil {
		t.Fatal("another thread acquired the native window lifetime")
	}
	calls := 0
	var releases []string
	release := func(name string) ComProc {
		return NewComProc(func(uintptr) uintptr {
			if windows.GetCurrentThreadId() != thread {
				t.Error("COM Release ran outside its UI thread")
			}
			releases = append(releases, name)
			return 0
		})
	}
	vtbl := &_ICoreWebView2ControllerVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("controller")}, Close: NewComProc(func(this uintptr) uintptr {
		calls++
		if windows.GetCurrentThreadId() != thread {
			t.Error("controller Close ran outside its UI thread")
		}
		return 0
	})}
	controller := &ICoreWebView2Controller{vtbl: vtbl}
	chromium := &Chromium{controller: controller, webview: &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("webview")}}}, environment: &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: release("environment")}}}, globalErrorCallback: func(err error) { t.Error(err) }}
	if err := chromium.attachWindowLifetime(hwnd); err != nil {
		t.Fatal(err)
	}
	if accepted, _, _ := user.NewProc("DestroyWindow").Call(hwnd); accepted == 0 {
		t.Fatal("normal native destruction failed")
	}
	if len(releases) != 3 || releases[0] != "webview" || releases[1] != "controller" || releases[2] != "environment" || chromium.webview != nil || chromium.controller != nil || chromium.environment != nil {
		t.Fatalf("owned COM references survived close: %v", releases)
	}
	if calls != 1 || !chromium.shuttingDown || ClosedControllerCount() != before+1 {
		t.Fatalf("controller Close calls=%d, shuttingDown=%t", calls, chromium.shuttingDown)
	}
	ownedWindows.Lock()
	retained := ownedWindows.values[hwnd] != nil
	ownedWindows.Unlock()
	if retained {
		t.Fatal("destroyed window retained its Go lifetime owner")
	}
	if alive, _, _ := user.NewProc("IsWindow").Call(hwnd); alive != 0 {
		t.Fatal("subclass suppressed the normal native window procedure")
	}
	if WindowCloseError() != nil {
		t.Fatal("successful Close retained an error")
	}
	t.Run("native menu focus does not reenter browser focus", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		handle, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
		if handle == 0 {
			t.Fatal("message-only menu fixture unavailable")
		}
		defer user.NewProc("DestroyWindow").Call(handle)
		forwarded := 0
		observer := syscall.NewCallback(func(hwnd, message, wparam, lparam, id, data uintptr) uintptr {
			if message == 7 {
				forwarded++
			}
			value, _, _ := lifetimeDefSubclass.Call(hwnd, message, wparam, lparam)
			return value
		})
		if ok, _, _ := lifetimeSetSubclass.Call(handle, observer, observer, 0); ok == 0 {
			t.Fatal("native focus observer unavailable")
		}
		view := &Chromium{hwnd: handle, globalErrorCallback: func(err error) { t.Error(err) }}
		if err := view.attachWindowLifetime(handle); err != nil {
			t.Fatal(err)
		}
		user.NewProc("SendMessageW").Call(handle, 7, 0, 0)
		if forwarded != 1 {
			t.Fatal("ordinary native focus was suppressed")
		}
		ownedWindows.Lock()
		ownedWindows.values[handle].menuDepth = 2
		ownedWindows.Unlock()
		user.NewProc("SendMessageW").Call(handle, 7, 0, 0)
		if forwarded != 1 {
			t.Fatal("native menu focus was forwarded back into the browser")
		}
		user.NewProc("SendMessageW").Call(handle, 0x212, 0, 0)
		user.NewProc("SendMessageW").Call(handle, 7, 0, 0)
		if forwarded != 1 {
			t.Fatal("nested native menu released browser focus early")
		}
		user.NewProc("SendMessageW").Call(handle, 0x212, 0, 0)
		user.NewProc("SendMessageW").Call(handle, 7, 0, 0)
		if forwarded != 2 {
			t.Fatal("ordinary focus forwarding was not restored after menu exit")
		}
	})
	t.Run("bootstrap diagnostics retain only owned fixed progress", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		previous := bootstrapDiagnosticsEnabled.Load()
		EnableWindowBootstrapDiagnostics(true)
		defer EnableWindowBootstrapDiagnostics(previous)
		handle, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
		if handle == 0 {
			t.Fatal("bootstrap message-only fixture unavailable")
		}
		defer user.NewProc("DestroyWindow").Call(handle)
		view := &Chromium{hwnd: handle, webview2RuntimeVersion: "152.0.4191.66", globalErrorCallback: func(err error) { t.Error(err) }}
		if err := view.attachWindowLifetime(handle); err != nil {
			t.Fatal(err)
		}
		recordWindowBootstrap(view, bootstrapNavigateRequested)
		recordWindowBootstrap(view, bootstrapNavigateAccepted)
		recordWindowBootstrap(view, bootstrapResourceEntered)
		recordWindowBootstrap(view, bootstrapNavigationCompleted)
		other := make(chan struct{})
		go func() { recordWindowBootstrap(view, bootstrapResourceReturned); close(other) }()
		<-other
		rows := OwnedWindowBootstrapSnapshots()
		if len(rows) != 1 || rows[0].RuntimeVersion != "152.0.4191.66" || rows[0].NavigationRequested != 1 || rows[0].NavigationAccepted != 1 || rows[0].ResourceEntered != 1 || rows[0].ResourceReturned != 0 || rows[0].NavigationCompleted != 1 || rows[0].CompletionWithoutHost != 1 || rows[0].OffThreadEvents != 1 {
			t.Fatal("bootstrap boundary lost ownership or fixed progress", rows)
		}
		recordWindowBootstrap(view, bootstrapResourceReturned)
		EnableWindowBootstrapDiagnostics(false)
		recordWindowBootstrap(view, bootstrapNavigateRequested)
		rows = OwnedWindowBootstrapSnapshots()
		if rows[0].ResourceReturned != 1 || rows[0].NavigationRequested != 1 {
			t.Fatal("disabled bootstrap observation changed state", rows)
		}
	})
	t.Run("a claimed accelerator stays handled", func(t *testing.T) {
		var handled []uintptr
		args := &ICoreWebView2AcceleratorKeyPressedEventArgs{vtbl: &_ICoreWebView2AcceleratorKeyPressedEventArgsVtbl{
			GetKeyEventKind: NewComProc(func(this uintptr, output *COREWEBVIEW2_KEY_EVENT_KIND) uintptr {
				*output = COREWEBVIEW2_KEY_EVENT_KIND_KEY_DOWN
				return 0
			}),
			GetVirtualKey: NewComProc(func(this uintptr, output *uint32) uintptr { *output = 0x46; return 0 }),
			GetPhysicalKeyStatus: NewComProc(func(this uintptr, output *internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS) uintptr {
				*output = internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS{}
				return 0
			}),
			PutHandled: NewComProc(func(this, value uintptr) uintptr { handled = append(handled, value); return 0 }),
		}}
		view := &Chromium{AcceleratorKeyCallback: func(key uint) bool { return key == 0x46 }, globalErrorCallback: func(err error) { t.Error(err) }}
		view.AcceleratorKeyPressed(nil, args)
		if len(handled) != 1 || handled[0] != 1 {
			t.Fatalf("claimed native accelerator was reset: %v", handled)
		}
		runtime.KeepAlive(args)
	})
	t.Run("native system key keeps its owned UI context and copied flags", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		owner := windows.GetCurrentThreadId()
		handle, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
		if handle == 0 {
			t.Fatal("system-key message-only fixture unavailable")
		}
		defer user.NewProc("DestroyWindow").Call(handle)
		view := &Chromium{hwnd: handle, globalErrorCallback: func(err error) { t.Error(err) }}
		if err := view.attachWindowLifetime(handle); err != nil {
			t.Fatal(err)
		}
		ownedWindows.Lock()
		previous := ownedWindows.systemKeyHandler
		ownedWindows.Unlock()
		defer SetNativeSystemKeyHandler(previous)
		const flags = 0x20210001
		var dispositions []uintptr
		args := &ICoreWebView2AcceleratorKeyPressedEventArgs{vtbl: &_ICoreWebView2AcceleratorKeyPressedEventArgsVtbl{
			GetKeyEventKind: NewComProc(func(this uintptr, output *COREWEBVIEW2_KEY_EVENT_KIND) uintptr {
				*output = COREWEBVIEW2_KEY_EVENT_KIND_SYSTEM_KEY_DOWN
				return 0
			}),
			GetVirtualKey:     NewComProc(func(this uintptr, output *uint32) uintptr { *output = 0x46; return 0 }),
			GetKeyEventLParam: NewComProc(func(this uintptr, output *uint32) uintptr { *output = flags; return 0 }),
			GetPhysicalKeyStatus: NewComProc(func(this uintptr, output *internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS) uintptr {
				*output = internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS{IsMenuKeyDown: 1}
				return 0
			}),
			PutHandled: NewComProc(func(this, value uintptr) uintptr { dispositions = append(dispositions, value); return 0 }),
		}}
		nativeCalls, legacyCalls := 0, 0
		view.AcceleratorKeyCallback = func(uint) bool { legacyCalls++; return false }
		SetNativeSystemKeyHandler(func(hwnd uintptr, event NativeSystemKeyEvent) bool {
			nativeCalls++
			if hwnd != handle || windows.GetCurrentThreadId() != owner || event.VirtualKey != 0x46 || event.LParam != flags || !event.PhysicalKeyStatus.IsMenuKeyDown {
				t.Error("native system-key metadata or UI ownership changed", event)
			}
			return true
		})
		view.AcceleratorKeyPressed(nil, args)
		if nativeCalls != 1 || legacyCalls != 0 || len(dispositions) != 1 || dispositions[0] != 1 {
			t.Fatal("native claimed event was duplicated or reset", nativeCalls, legacyCalls, dispositions)
		}
		SetNativeSystemKeyHandler(func(uintptr, NativeSystemKeyEvent) bool { return false })
		dispositions = nil
		view.AcceleratorKeyPressed(nil, args)
		if legacyCalls != 1 || len(dispositions) != 1 || dispositions[0] != 0 {
			t.Fatal("unclaimed event lost its existing handler", legacyCalls, dispositions)
		}
		other := make(chan bool, 1)
		go func() { other <- view.nativeSystemKeyHandler() == nil }()
		if !<-other {
			t.Fatal("another thread acquired the native system-key handler")
		}
		view.hwnd = 0
		if view.nativeSystemKeyHandler() != nil {
			t.Fatal("unowned native window acquired the system-key handler")
		}
		view.hwnd = handle
		runtime.KeepAlive(args)
	})
	t.Run("failed Close preserves destruction and error", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		ownedWindows.Lock()
		original := ownedWindows.failure
		ownedWindows.Unlock()
		defer func() { ownedWindows.Lock(); ownedWindows.failure = original; ownedWindows.Unlock() }()
		failed, _, _ := user.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, 0, 0)
		if failed == 0 {
			t.Fatal("message-only failure fixture unavailable")
		}
		defer user.NewProc("DestroyWindow").Call(failed)
		attempts := 0
		reported := false
		failedReleases := 0
		failedTable := &_ICoreWebView2ControllerVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { failedReleases++; return 0 })}, Close: NewComProc(func(uintptr) uintptr { attempts++; return 0x80004005 })}
		failedView := &Chromium{controller: &ICoreWebView2Controller{vtbl: failedTable}, globalErrorCallback: func(error) { reported = true }}
		if err := failedView.attachWindowLifetime(failed); err != nil {
			t.Fatal(err)
		}
		if accepted, _, _ := user.NewProc("DestroyWindow").Call(failed); accepted == 0 {
			t.Fatal("failed Close suppressed destruction")
		}
		if failedReleases != 1 {
			t.Fatal("failed Close retained its owned interface")
		}
		if attempts != 1 || !reported || WindowCloseError() == nil {
			t.Fatal("failed Close lost its result", attempts, reported, WindowCloseError())
		}
		ownedWindows.Lock()
		retained := ownedWindows.values[failed] != nil
		ownedWindows.Unlock()
		if retained {
			t.Fatal("failed Close stranded the window owner")
		}
	})
	t.Run("resource deferral release balances its owned reference", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		owner := windows.GetCurrentThreadId()
		refs, addCalls, releaseCalls, completeCalls := 1, 0, 0, 0
		checkThread := func() {
			if windows.GetCurrentThreadId() != owner {
				t.Error("resource deferral ran outside its owning UI thread")
			}
		}
		table := &ICoreWebView2DeferralVtbl{
			_IUnknownVtbl: _IUnknownVtbl{
				AddRef:  NewComProc(func(uintptr) uintptr { checkThread(); addCalls++; refs++; return uintptr(refs) }),
				Release: NewComProc(func(uintptr) uintptr { checkThread(); releaseCalls++; refs--; return uintptr(refs) }),
			},
			Complete: NewComProc(func(uintptr) uintptr { checkThread(); completeCalls++; return 0 }),
		}
		deferral := &ICoreWebView2Deferral{Vtbl: table}
		if err := deferral.AddRef(); err != nil {
			t.Fatal(err)
		}
		if err := deferral.Complete(); err != nil {
			t.Fatal(err)
		}
		if err := deferral.Release(); err != nil {
			t.Fatal(err)
		}
		if err := deferral.Release(); err != nil {
			t.Fatal(err)
		}
		if refs != 0 || addCalls != 1 || releaseCalls != 2 || completeCalls != 1 {
			t.Fatalf("completed deferral retained references: refs=%d add=%d release=%d complete=%d", refs, addCalls, releaseCalls, completeCalls)
		}
		runtime.KeepAlive(deferral)
	})
	t.Run("owned COM apartment is balanced on its original thread", func(t *testing.T) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		query := windows.NewLazySystemDLL("ole32.dll").NewProc("CoGetApartmentType")
		apartment := func() uintptr {
			var kind, qualifier uint32
			hr, _, _ := query.Call(uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&qualifier)))
			return uintptr(uint32(hr))
		}
		if apartment() != 0x800401f0 {
			t.Fatal("fixture needs a fresh uninitialized worker apartment")
		}
		ownedWindows.Lock()
		priorThread, priorOwned, priorFinalized := ownedWindows.apartmentThread, ownedWindows.apartmentOwned, ownedWindows.apartmentFinalized
		ownedWindows.Unlock()
		if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err != nil {
			t.Fatal(err)
		}
		defer func() {
			if apartment() == 0 {
				windows.CoUninitialize()
			}
			ownedWindows.Lock()
			ownedWindows.apartmentThread, ownedWindows.apartmentOwned, ownedWindows.apartmentFinalized = priorThread, priorOwned, priorFinalized
			ownedWindows.Unlock()
		}()
		recordOwnedCOMApartment()
		other := make(chan error, 1)
		go func() { other <- FinalizeCOMApartment() }()
		if err := <-other; err == nil {
			t.Fatal("another thread finalized the COM apartment")
		}
		if err := FinalizeCOMApartment(); err != nil {
			t.Fatal(err)
		}
		if apartment() != 0x800401f0 {
			t.Fatal("the binding's COM initialization survived finalization")
		}
		if err := FinalizeCOMApartment(); err != nil {
			t.Fatal("idempotent finalization failed", err)
		}
	})

}

// Only fake WebView interfaces and a native memory stream are used here. No
// browser, actual controller, visible window or input destination is created.
type firstResponseFixture struct {
	t        *testing.T
	entry    *windowLifetime
	view     *Chromium
	args     *ICoreWebView2WebResourceRequestedEventArgs
	response *ICoreWebView2WebResourceResponse
	deferral *ICoreWebView2Deferral
	headers  *ICoreWebView2HttpResponseHeaders
	stream   *IStream
	fail     string
	calls    []string
	threads  []uint32
}

func newFirstResponseFixture(t *testing.T, enabled bool, fail string) *firstResponseFixture {
	t.Helper()
	previous := bootstrapDiagnosticsEnabled.Load()
	EnableWindowBootstrapDiagnostics(enabled)
	f := &firstResponseFixture{t: t, fail: fail}
	f.view = &Chromium{hwnd: 0x534040}
	f.entry = &windowLifetime{chromium: f.view, thread: windows.GetCurrentThreadId()}
	ownedWindows.Lock()
	prior := ownedWindows.values[f.view.hwnd]
	ownedWindows.values[f.view.hwnd] = f.entry
	ownedWindows.Unlock()
	t.Cleanup(func() {
		if f.stream != nil {
			f.stream.Release()
		}
		ownedWindows.Lock()
		if prior == nil {
			delete(ownedWindows.values, f.view.hwnd)
		} else {
			ownedWindows.values[f.view.hwnd] = prior
		}
		ownedWindows.Unlock()
		EnableWindowBootstrapDiagnostics(previous)
		runtime.KeepAlive(f)
	})
	record := func(name string) uintptr {
		f.calls = append(f.calls, name)
		f.threads = append(f.threads, windows.GetCurrentThreadId())
		if fail == name {
			return 0x80004005
		}
		return 0
	}
	f.headers = &ICoreWebView2HttpResponseHeaders{vtbl: &_ICoreWebView2HttpResponseHeadersVtbl{
		_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { return record("headers.release") })},
	}}
	f.response = &ICoreWebView2WebResourceResponse{vtbl: &_ICoreWebView2WebResourceResponseVtbl{
		_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { return record("response.release") })},
		GetHeaders: NewComProc(func(this uintptr, output **ICoreWebView2HttpResponseHeaders) uintptr {
			hr := record("headers")
			if hr == 0 {
				*output = f.headers
			}
			return hr
		}),
		PutStatusCode: NewComProc(func(this, status uintptr) uintptr {
			if status != 200 {
				t.Errorf("native status changed: %d", status)
			}
			return record("status")
		}),
		PutReasonPhrase: NewComProc(func(this uintptr, reason *uint16) uintptr {
			if windows.UTF16PtrToString(reason) != "OK" {
				t.Error("native reason changed")
			}
			return record("reason")
		}),
		PutContent: NewComProc(func(this uintptr, stream *IStream) uintptr {
			hr := record("content")
			if hr == 0 && stream != nil {
				if err := stream.AddRef(); err != nil {
					t.Error(err)
				}
				f.stream = stream
			}
			return hr
		}),
	}}
	f.deferral = &ICoreWebView2Deferral{Vtbl: &ICoreWebView2DeferralVtbl{
		_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { return record("deferral.release") })},
		Complete: NewComProc(func(uintptr) uintptr {
			// Snapshotting inside a native callback also proves observation does not
			// hold the owner mutex across a reentrant COM boundary.
			snapshot := f.snapshot()
			if enabled && (snapshot.Completion.Entered != 1 || snapshot.Completion.Returned != 0) {
				t.Error("native completion entry was not distinguished from return")
			}
			return record("complete")
		}),
	}}
	f.args = &ICoreWebView2WebResourceRequestedEventArgs{vtbl: &_ICoreWebView2WebResourceRequestedEventArgsVtbl{
		PutResponse: NewComProc(func(this uintptr, response *ICoreWebView2WebResourceResponse) uintptr {
			if response != f.response {
				t.Error("native response identity changed")
			}
			return record("assign")
		}),
		GetDeferral: NewComProc(func(this uintptr, output **ICoreWebView2Deferral) uintptr {
			hr := record("deferral")
			if hr == 0 {
				*output = f.deferral
			}
			return hr
		}),
	}}
	recordWindowBootstrapRequest(f.view, f.args)
	return f
}

func (f *firstResponseFixture) snapshot() WindowBootstrapResponseSnapshot {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	return f.entry.bootstrap.FirstResponse
}

func (f *firstResponseFixture) assign() {
	f.t.Helper()
	if err := f.args.PutResponse(f.response); err != nil {
		f.t.Fatal(err)
	}
	if got, err := f.args.GetDeferral(); err != nil || got != f.deferral {
		f.t.Fatal("native deferral changed", err)
	}
}

func TestFirstNativeResponseObservationPreservesPublicationContract(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			f := newFirstResponseFixture(t, enabled, "")
			f.assign()
			headers, err := f.response.GetHeaders()
			if err != nil || headers != f.headers {
				t.Fatal("native headers changed", err)
			}
			headers.Release()
			if err := f.response.PutStatusCode(200); err != nil {
				t.Fatal(err)
			}
			body := []byte("<html>private fixture body: \u03c0</html>")
			want := bytes.Clone(body)
			if err := f.response.PutByteContent(body); err != nil {
				t.Fatal(err)
			}
			for i := range body {
				body[i] = 'x'
			}
			runtime.GC()
			got := make([]byte, len(want))
			if f.stream == nil {
				t.Fatal("native response did not retain its content stream")
			}
			if n, err := f.stream.Read(got); n != len(want) || err != nil || !bytes.Equal(got, want) {
				t.Fatal("published content was not independently retained", n, err)
			}
			if n, err := f.stream.Read(make([]byte, 1)); n != 0 || !errors.Is(err, io.EOF) {
				t.Fatal("native stream EOF changed", n, err)
			}
			if err := f.response.Release(); err != nil {
				t.Fatal(err)
			}
			if err := f.deferral.Complete(); err != nil {
				t.Fatal(err)
			}
			if err := f.deferral.Release(); err != nil {
				t.Fatal(err)
			}
			wantCalls := []string{"assign", "deferral", "headers", "headers.release", "status", "reason", "content", "response.release", "complete", "deferral.release"}
			if !reflect.DeepEqual(f.calls, wantCalls) {
				t.Fatal("observation changed native call order/count", f.calls)
			}
			for _, thread := range f.threads {
				if thread != f.entry.thread {
					t.Fatal("observation rerouted a native call")
				}
			}
			snapshot := f.snapshot()
			if !enabled {
				if snapshot != (WindowBootstrapResponseSnapshot{}) || f.entry.firstRequest != (windowBootstrapResponse{}) {
					t.Fatal("disabled observation retained response state")
				}
				return
			}
			one := WindowBootstrapCallSnapshot{Entered: 1, Returned: 1}
			for _, got := range []WindowBootstrapCallSnapshot{snapshot.Assignment, snapshot.Deferral, snapshot.Headers, snapshot.Status, snapshot.Reason, snapshot.Content, snapshot.Completion} {
				if got != one {
					t.Fatal("native publication boundary changed", got)
				}
			}
			if snapshot.StatusCode != 200 || snapshot.ContentAttempts != 1 || snapshot.ContentBytes != uint64(len(want)) || snapshot.StreamsCreated != 1 || snapshot.StreamFailures != 0 {
				t.Fatal("content preparation was not distinguished", snapshot)
			}
			if f.entry.firstRequest.args != 0 || f.entry.firstRequest.response != 0 || f.entry.firstRequest.deferral != 0 || !f.entry.firstRequest.seen {
				t.Fatal("retired native identities survived")
			}
			// A later request and recycled native identity cannot overwrite the first response.
			recordWindowBootstrapRequest(f.view, f.args)
			if err := f.response.PutStatusCode(200); err != nil {
				t.Fatal(err)
			}
			if f.snapshot() != snapshot {
				t.Fatal("a retired response identity changed the first receipt")
			}
			raw, err := json.Marshal(f.entry.bootstrap)
			if err != nil || strings.Contains(string(raw), "private") || strings.Contains(string(raw), "fixture") || strings.Contains(string(raw), "args") || strings.Contains(string(raw), "pointer") {
				t.Fatal("response observation retained content or identities")
			}
			runtime.KeepAlive(f)
		})
	}
}

func TestFirstNativeResponseObservationPreservesFailedHRESULTs(t *testing.T) {
	for _, point := range []string{"assign", "deferral", "headers", "status", "reason", "content", "complete"} {
		t.Run(point, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			f := newFirstResponseFixture(t, true, point)
			var err error
			switch point {
			case "assign":
				err = f.args.PutResponse(f.response)
			case "deferral":
				if err = f.args.PutResponse(f.response); err != nil {
					t.Fatal(err)
				}
				_, err = f.args.GetDeferral()
			default:
				f.assign()
				switch point {
				case "headers":
					_, err = f.response.GetHeaders()
				case "status", "reason":
					err = f.response.PutStatusCode(200)
				case "content":
					err = f.response.PutByteContent([]byte("private failure body"))
				case "complete":
					err = f.deferral.Complete()
				}
			}
			if !errors.Is(err, syscall.Errno(0x80004005)) {
				t.Fatal("native HRESULT was replaced", err)
			}
			snapshot := f.snapshot()
			calls := map[string]WindowBootstrapCallSnapshot{"assign": snapshot.Assignment, "deferral": snapshot.Deferral, "headers": snapshot.Headers, "status": snapshot.Status, "reason": snapshot.Reason, "content": snapshot.Content, "complete": snapshot.Completion}
			if got := calls[point]; got.Entered != 1 || got.Returned != 1 || got.LastHRESULT != 0x80004005 || got.OffThreadCalls != 0 {
				t.Fatal("failed native boundary was not retained", got)
			}
			if point == "status" && snapshot.Reason.Entered != 0 {
				t.Fatal("a failed status setter gained a reason call")
			}
			if (point == "assign" || point == "deferral") && f.entry.firstRequest.args != 0 {
				t.Fatal("failed preparation retained its borrowed event identity")
			}
			if point == "content" && f.stream != nil {
				t.Fatal("failed setter acquired a native stream reference")
			}
			runtime.KeepAlive(f)
		})
	}
}

func TestFirstNativeResponseObservationFlagsThreadWithoutRerouting(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	f := newFirstResponseFixture(t, true, "")
	f.assign()
	completed := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		if err := f.response.PutStatusCode(200); err != nil {
			completed <- err
			return
		}
		completed <- f.deferral.Complete()
	}()
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
	snapshot := f.snapshot()
	for _, call := range []WindowBootstrapCallSnapshot{snapshot.Status, snapshot.Reason, snapshot.Completion} {
		if call.Entered != 1 || call.Returned != 1 || call.OffThreadCalls != 1 || call.LastHRESULT != 0 {
			t.Fatal("foreign-thread call was hidden or altered", call)
		}
	}
	if len(f.calls) != 5 {
		t.Fatal("foreign-thread call count changed", f.calls)
	}
	for i, thread := range f.threads {
		if i >= 2 && thread == f.entry.thread {
			t.Fatal("observation rerouted a foreign-thread native call")
		}
	}
	runtime.KeepAlive(f)
}
