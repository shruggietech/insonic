//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/combridge"
	"golang.org/x/sys/windows"
)

func browserTestString(t *testing.T, text string) *uint16 {
	t.Helper()
	chars, err := windows.UTF16FromString(text)
	if err != nil {
		t.Fatal(err)
	}
	address, _, _ := windows.NewLazySystemDLL("ole32.dll").NewProc("CoTaskMemAlloc").Call(uintptr(len(chars) * 2))
	if address == 0 {
		t.Fatal("COM test string allocation failed")
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(address)), len(chars)), chars)
	return (*uint16)(unsafe.Pointer(address))
}

type browserNativeFixture struct {
	t                 *testing.T
	view              *Chromium
	thread            uint32
	fail              string
	calls             []string
	receivers         [len(browserEventNames)]*browserCDPReceiver
	handlers          [len(browserEventNames)]uintptr
	commands          [len(browserEnableMethods)]uintptr
	removed, released int
	args              *browserCDPArgs
	payload           string
	eventHR           uintptr
}

func newBrowserNativeFixture(t *testing.T, enabled bool, fail string) *browserNativeFixture {
	t.Helper()
	runtime.LockOSThread()
	previous := browserDiagnosticsEnabled.Load()
	EnableWindowBrowserDiagnostics(enabled)
	f := &browserNativeFixture{t: t, thread: windows.GetCurrentThreadId(), fail: fail}
	f.view = &Chromium{hwnd: 0x534047, globalErrorCallback: func(err error) { t.Error(err) }}
	ownedWindows.Lock()
	prior := ownedWindows.values[f.view.hwnd]
	ownedWindows.values[f.view.hwnd] = &windowLifetime{chromium: f.view, thread: f.thread}
	ownedWindows.Unlock()
	for i := range f.receivers {
		index := i
		f.receivers[i] = &browserCDPReceiver{vtbl: &browserCDPReceiverVtbl{
			_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { f.checkThread(); f.released++; return 0 })},
			Add: NewComProc(func(this, handler uintptr, token *_EventRegistrationToken) uintptr {
				f.checkThread()
				f.calls = append(f.calls, fmt.Sprintf("add:%d", index))
				if this != uintptr(unsafe.Pointer(f.receivers[index])) || handler == 0 || token == nil {
					t.Error("receiver Add arguments changed")
				}
				if f.fail == fmt.Sprintf("add:%d", index) {
					return 0x80004005
				}
				callbackTable(unsafe.Pointer(handler)).AddRef.Call(handler)
				f.handlers[index] = handler
				token.value = int64(0x3141592600000000 + index)
				return 0
			}),
			Remove: NewComProc(func(this, token uintptr) uintptr {
				f.checkThread()
				f.removed++
				if this != uintptr(unsafe.Pointer(f.receivers[index])) || token != uintptr(0x3141592600000000+index) {
					t.Error("receiver Remove token was not passed by value")
				}
				if f.handlers[index] != 0 {
					releaseCallback(unsafe.Pointer(f.handlers[index]))
					f.handlers[index] = 0
				}
				if f.fail == "remove" {
					return 0x80004005
				}
				return 0
			}),
		}}
	}
	f.args = &browserCDPArgs{vtbl: &browserCDPArgsVtbl{GetJSON: NewComProc(func(_ uintptr, out **uint16) uintptr {
		if out == nil {
			t.Error("JSON getter did not receive the output address")
			return 0x80004005
		}
		*out = browserTestString(t, f.payload)
		return f.eventHR
	})}}
	f.view.webview = &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{
		Navigate: NewComProc(func(_ uintptr, target *uint16) uintptr {
			f.checkThread()
			f.calls = append(f.calls, "navigate:"+windows.UTF16PtrToString(target))
			return 0
		}),
		GetDevToolsProtocolEventReceiver: NewComProc(func(this uintptr, name *uint16, out **browserCDPReceiver) uintptr {
			f.checkThread()
			value := windows.UTF16PtrToString(name)
			index := -1
			for i, n := range browserEventNames {
				if n == value {
					index = i
				}
			}
			if this != uintptr(unsafe.Pointer(f.view.webview)) || index < 0 || out == nil {
				t.Error("GetReceiver arguments changed")
				return 0x80004005
			}
			f.calls = append(f.calls, fmt.Sprintf("get:%d", index))
			if f.fail == fmt.Sprintf("get:%d", index) {
				return 0x80004005
			}
			*out = f.receivers[index]
			return 0
		}),
		CallDevToolsProtocolMethod: NewComProc(func(this uintptr, method, parameters *uint16, handler uintptr) uintptr {
			f.checkThread()
			value := windows.UTF16PtrToString(method)
			index := -1
			for i, n := range browserEnableMethods {
				if n == value {
					index = i
				}
			}
			if this != uintptr(unsafe.Pointer(f.view.webview)) || index < 0 || handler == 0 {
				t.Error("CallDevTools arguments changed")
				return 0x80004005
			}
			expected := "{}"
			if index == 3 {
				expected = `{"enabled":true}`
			}
			if windows.UTF16PtrToString(parameters) != expected {
				t.Error("enable parameters changed")
			}
			f.calls = append(f.calls, fmt.Sprintf("enable:%d", index))
			if f.fail == fmt.Sprintf("enable:%d", index) {
				return 0x80004005
			}
			callbackTable(unsafe.Pointer(handler)).AddRef.Call(handler)
			f.commands[index] = handler
			return 0
		}),
	}}
	t.Cleanup(func() {
		f.view.closeBrowserDiagnostics()
		for i, pointer := range f.commands {
			if pointer != 0 {
				releaseCallback(unsafe.Pointer(pointer))
				f.commands[i] = 0
			}
		}
		ownedWindows.Lock()
		if prior == nil {
			delete(ownedWindows.values, f.view.hwnd)
		} else {
			ownedWindows.values[f.view.hwnd] = prior
		}
		ownedWindows.Unlock()
		EnableWindowBrowserDiagnostics(previous)
		runtime.KeepAlive(f)
		runtime.UnlockOSThread()
	})
	return f
}
func (f *browserNativeFixture) checkThread() {
	if windows.GetCurrentThreadId() != f.thread {
		f.t.Error("diagnostic native call changed thread")
	}
}
func (f *browserNativeFixture) ack(index int, hr uintptr) {
	f.t.Helper()
	pointer := f.commands[index]
	if pointer == 0 {
		f.t.Fatal("method callback not registered", index)
	}
	// The result is borrowed LPCWSTR. A Go buffer must never be CoTaskMemFree'd.
	result, _ := windows.UTF16PtrFromString(`{"private":"not retained"}`)
	value, _, _ := callbackTable(unsafe.Pointer(pointer)).Invoke.Call(pointer, hr, uintptr(unsafe.Pointer(result)))
	if value != 0 {
		f.t.Fatal("completion callback changed HRESULT")
	}
	runtime.KeepAlive(result)
}
func (f *browserNativeFixture) arm() {
	f.view.Navigate("http://wails.localhost/")
	for i := range browserEnableMethods {
		f.ack(i, 0)
	}
}
func (f *browserNativeFixture) emit(index int, payload string, hr uintptr) {
	f.t.Helper()
	pointer := f.handlers[index]
	if pointer == 0 {
		f.t.Fatal("event callback not registered", index)
	}
	f.payload = payload
	f.eventHR = hr
	args := f.args
	value, _, _ := callbackTable(unsafe.Pointer(pointer)).Invoke.Call(pointer, uintptr(unsafe.Pointer(f.view.webview)), uintptr(unsafe.Pointer(args)))
	if value != 0 {
		f.t.Fatal("event callback changed HRESULT")
	}
	runtime.KeepAlive(args)
}
func (f *browserNativeFixture) snapshot() WindowBrowserSnapshot {
	state := f.view.browserDiagnostics
	state.Lock()
	defer state.Unlock()
	return state.snapshot
}
func browserRequestJSON(id, asset string) string {
	return fmt.Sprintf(`{"requestId":%q,"loaderId":"target-loader","frameId":"target-frame","request":{"url":%q}}`, id, "http://wails.localhost/"+asset)
}

func TestBrowserDiagnosticsDisabledAddsNoNativeCalls(t *testing.T) {
	f := newBrowserNativeFixture(t, false, "")
	environment := &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{GetBrowserVersionString: NewComProc(func(uintptr, uintptr) uintptr { t.Error("disabled version getter called"); return 0 })}}
	f.view.observeBrowserEnvironment(0, environment)
	f.view.Navigate("http://wails.localhost/")
	f.view.ShuttingDown()
	if len(f.calls) != 1 || f.calls[0] != "navigate:http://wails.localhost/" || f.view.browserDiagnostics != nil || f.removed != 0 || f.released != 0 {
		t.Fatal("disabled diagnostics changed ordinary call flow", f.calls)
	}
}
func TestBrowserDiagnosticsNativeSetupOrderingIdentityAndCleanup(t *testing.T) {
	f := newBrowserNativeFixture(t, true, "")
	f.view.Navigate("http://wails.localhost/")
	if s := f.snapshot(); s.Registered != 127 || s.EnabledDomains != 0 || s.NavigationStarted || s.Armed {
		t.Fatal("registration/ack coverage incorrect", s)
	}
	f.emit(6, `{"name":"load","loaderId":"blank","frameId":"blank"}`, 0)
	for i := range browserEnableMethods {
		if f.commands[i] == 0 {
			t.Fatal("missing sequential command", i)
		}
		for _, call := range f.calls {
			if strings.HasPrefix(call, "navigate:") {
				t.Fatal("navigation preceded enable acknowledgements")
			}
		}
		f.ack(i, 0)
	}
	s := f.snapshot()
	if !s.Armed || !s.NavigationStarted || s.EnabledDomains != 15 || s.EarlyEvents != 1 || f.calls[len(f.calls)-1] != "navigate:http://wails.localhost/" {
		t.Fatal("complete setup did not release exactly one navigation", s, f.calls)
	}
	fixtures := []struct {
		pointer uintptr
		iid     string
	}{{f.handlers[0], "{e2fda4be-5456-406c-a261-3d452138362c}"}, {f.commands[0], "{5c4889f0-5ef6-4c5a-952c-d8f1b92d0574}"}}
	for _, fixture := range fixtures {
		iid, _ := windows.GUIDFromString(fixture.iid)
		var out uintptr
		if hr := queryCallback(unsafe.Pointer(fixture.pointer), &iid, &out); hr != 0 || out != fixture.pointer {
			t.Fatal("official callback IID/identity missing", hr)
		}
		releaseCallback(unsafe.Pointer(out))
		other, _ := windows.GUIDFromString("{00000003-0000-0000-c000-000000000046}")
		out = 1
		if hr := queryCallback(unsafe.Pointer(fixture.pointer), &other, &out); hr != uint32(windows.E_NOINTERFACE) || out != 0 {
			t.Fatal("callback advertised unsupported marshalling", hr)
		}
	}
	retained := f.handlers[0]
	callbackTable(unsafe.Pointer(retained)).AddRef.Call(retained)
	f.view.ShuttingDown()
	f.view.ShuttingDown()
	runtime.GC()
	if f.removed != 7 || f.released != 7 || !f.snapshot().Closed {
		t.Fatal("Close did not remove handlers and release receivers once", f.removed, f.released)
	}
	if combridge.Resolve[browserCDPEventCallback](retained) == nil {
		t.Fatal("native reference lost its target after creator cleanup")
	}
	callbackTable(unsafe.Pointer(retained)).Invoke.Call(retained, 0, 0)
	if f.snapshot().LateCallbacks != 1 {
		t.Fatal("late event was not ignored")
	}
	releaseCallback(unsafe.Pointer(retained))
	if combridge.Resolve[browserCDPEventCallback](retained) != nil {
		t.Fatal("final native release retained callback target")
	}
	command := f.commands[0]
	f.ack(0, 0)
	if f.snapshot().LateCallbacks != 2 {
		t.Fatal("late command callback was not ignored")
	}
	releaseCallback(unsafe.Pointer(command))
	f.commands[0] = 0
	if combridge.Resolve[browserCDPMethodCallback](command) != nil {
		t.Fatal("command creator lease survived Close")
	}
}
func TestBrowserDiagnosticsSetupFailuresKeepOriginalNavigation(t *testing.T) {
	for _, failure := range []string{"get:2", "add:2", "enable:1", "ack:1", "remove"} {
		t.Run(failure, func(t *testing.T) {
			f := newBrowserNativeFixture(t, true, failure)
			f.view.Navigate("http://wails.localhost/")
			for i := range browserEnableMethods {
				if f.commands[i] != 0 {
					hr := uintptr(0)
					if failure == fmt.Sprintf("ack:%d", i) {
						hr = 0x80004005
					}
					f.ack(i, hr)
				}
			}
			navigations := 0
			for _, call := range f.calls {
				if strings.HasPrefix(call, "navigate:") {
					navigations++
				}
			}
			s := f.snapshot()
			if navigations != 1 {
				t.Fatal("setup changed original navigation count", navigations)
			}
			if failure != "remove" && (s.Armed || s.SetupFailed != 1 || s.SetupHRESULT != 0x80004005 || s.SetupStage != "setup-failed") {
				t.Fatal("setup failure coverage lost", s)
			}
			f.view.closeBrowserDiagnostics()
			if failure == "remove" && f.snapshot().CleanupFailures != 7 {
				t.Fatal("remove failure was not retained")
			}
		})
	}
}
func TestBrowserDiagnosticsVersionOwnershipBoundsAndAllowlist(t *testing.T) {
	for _, sample := range []struct {
		value string
		hr    uintptr
		valid bool
	}{{"152.0.4191.66", 0, true}, {"152.0.4191.66 beta", 0, true}, {"152.0.4191.66 canary", 0, true}, {"152.0.4191.66 dev", 0, true}, {"152.0.4191.66 private", 0, false}, {"152.0.4191.66\nsecret", 0, false}, {strings.Repeat("1", 65), 0, false}, {"152.0.4191.66", 0x80004005, false}} {
		t.Run(fmt.Sprint(len(sample.value), sample.valid, sample.hr), func(t *testing.T) {
			f := newBrowserNativeFixture(t, true, "")
			calls := 0
			env := &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{GetBrowserVersionString: NewComProc(func(_ uintptr, out **uint16) uintptr {
				f.checkThread()
				calls++
				*out = browserTestString(t, sample.value)
				return sample.hr
			})}}
			f.view.observeBrowserEnvironment(0, env)
			f.view.observeBrowserEnvironment(0, env)
			s := f.snapshot()
			if calls != 1 || s.VersionCalls != 1 || s.VersionHRESULT != uint32(sample.hr) || s.VersionInvalid == sample.valid || (sample.valid && s.ActualRuntimeVersion != sample.value) || (!sample.valid && s.ActualRuntimeVersion != "") {
				t.Fatal("actual version coverage or allowlist incorrect", s)
			}
		})
	}
	for _, sample := range []string{"", "abc", "1.2.3", "1.2.3.4 stable", "1.2.3.4 ", "1.2.3.4  beta", "1.2.-3.4"} {
		if browserVersionOnly(sample) {
			t.Error("private/malformed version accepted", sample)
		}
	}
	for _, size := range []int{4, 5} {
		frees := 0
		pointer := browserTestString(t, strings.Repeat("x", size))
		value, ok := readBrowserOwnedString(pointer, 4, func(p unsafe.Pointer) { frees++; windows.CoTaskMemFree(p) })
		if frees != 1 || ok != (size == 4) || (ok && len(value) != 4) {
			t.Fatal("bounded COM output ownership changed", frees, ok)
		}
	}
}
func TestBrowserDiagnosticsClassesRejectPrivateURLVariants(t *testing.T) {
	for _, value := range []string{"http://private@wails.localhost/", "http://wails.localhost/wails/%69pc.js", "http://wails.localhost/?", "https://wails.localhost/", "http://wails.localhost/?secret=x", "http://other/", "http://wails.localhost/#private"} {
		if browserAssetClass(value) != -1 {
			t.Error("nonliteral application URL accepted", value)
		}
	}
}
func TestBrowserDiagnosticsSixAssetsBytesExceptionsAndLoaderMilestones(t *testing.T) {
	f := newBrowserNativeFixture(t, true, "")
	f.arm()
	f.emit(5, `{ "exceptionDetails": {"url":"about:blank","lineNumber":0,"columnNumber":0,"text":"private-before-document"}}`, 0)
	paths := []string{"", "wails/ipc.js", "wails/runtime.js", "assets/bundle/app.css", "assets/bundle/product.css", "assets/bundle/app.js"}
	for i, path := range paths {
		id := fmt.Sprint("private-id-", i)
		f.emit(0, browserRequestJSON(id, path), 0)
		mime := "application/javascript"
		if i == 0 {
			mime = "text/html"
		}
		if i == 3 || i == 4 {
			mime = "text/css"
		}
		response := fmt.Sprintf(`{"requestId":%q,"response":{"url":%q,"status":200,"mimeType":%q,"headers":{"Content-Length":"4","Content-Type":%q,"Set-Cookie":"private-cookie","Authorization":"private-token"}}}`, id, "http://wails.localhost/"+path, mime, mime+"; private=header")
		f.emit(1, response, 0)
		f.emit(2, fmt.Sprintf(`{"requestId":%q,"dataLength":4,"data":"private-body"}`, id), 0)
		f.emit(3, fmt.Sprintf(`{"requestId":%q,"encodedDataLength":900}`, id), 0)
	}
	f.emit(6, `{"name":"load","loaderId":"about-blank","frameId":"target-frame"}`, 0)
	f.emit(6, `{"name":"networkIdle","loaderId":"target-loader","frameId":"target-frame"}`, 0)
	f.emit(6, `{"name":"DOMContentLoaded","loaderId":"target-loader","frameId":"target-frame"}`, 0)
	f.emit(6, `{"name":"load","loaderId":"target-loader","frameId":"target-frame"}`, 0)
	f.emit(5, `{"exceptionDetails":{"url":"http://wails.localhost/wails/runtime.js","lineNumber":220,"columnNumber":3,"text":"private-exception","exception":{"description":"private-stack"}}}`, 0)
	s := f.snapshot()
	for i, a := range s.Assets {
		if a.Requested != 1 || a.Responses != 1 || a.Finished != 1 || a.DecodedBytes != 4 || a.EncodedBytes != 900 || a.LengthChecks != 1 || a.LengthMismatches != 0 || a.Status != 200 || a.MIMEUnexpected != 0 {
			t.Fatal("asset delivery evidence incorrect", i, a)
		}
	}
	if !s.LifecycleAssociationAvailable || s.UnassociatedExceptions != 1 || s.DOMContentLoaded != 1 || s.LoadEvent != 1 || s.IgnoredLifecycle != 2 || s.Exceptions != 1 || s.ExceptionClass != "runtime" || !s.ExceptionLocationKnown || s.ExceptionLine != 220 || s.ExceptionColumn != 3 {
		t.Fatal("loader milestone/exception summary incorrect", s)
	}
	encoded, err := json.Marshal(OwnedWindowBrowserSnapshots())
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{"private-", "http:", "target-loader", "target-frame", "Set-Cookie", "Authorization", "exceptionDetails"} {
		if strings.Contains(string(encoded), private) {
			t.Fatal("snapshot exposed private transient data", private)
		}
	}
}
func TestBrowserDiagnosticsUnknownLengthsFailuresMalformedAndBounds(t *testing.T) {
	f := newBrowserNativeFixture(t, true, "")
	f.arm()
	f.emit(6, `{"name":"load","loaderId":"blank","frameId":"blank"}`, 0)
	f.emit(0, `{"requestId":"missing-association","request":{"url":"http://wails.localhost/"}}`, 0)
	f.emit(6, `{"name":"DOMContentLoaded","loaderId":"target-loader","frameId":"target-frame"}`, 0)
	for i, encoding := range []string{"identity", "gzip", "identity"} {
		id := fmt.Sprint("r", i)
		f.emit(0, browserRequestJSON(id, "wails/ipc.js"), 0)
		f.emit(1, fmt.Sprintf(`{"requestId":%q,"response":{"url":"http://wails.localhost/wails/ipc.js","status":200,"mimeType":"text/plain","headers":{"Content-Length":"5","Content-Encoding":%q}}}`, id, encoding), 0)
		if i != 2 {
			f.emit(2, fmt.Sprintf(`{"requestId":%q,"dataLength":3}`, id), 0)
		}
		f.emit(3, fmt.Sprintf(`{"requestId":%q,"encodedDataLength":500}`, id), 0)
	}
	f.emit(0, browserRequestJSON("failed", "assets/bundle/app.js"), 0)
	f.emit(4, `{"requestId":"failed","canceled":true,"blockedReason":"private-reason","errorText":"private-error"}`, 0)
	f.emit(3, `{"requestId":"failed","encodedDataLength":1}`, 0)
	f.emit(1, "{broken", 0)
	f.emit(1, "{}", 0x80004005)
	f.emit(1, strings.Repeat("x", browserJSONLimit), 0)
	f.emit(0, browserRequestJSON(strings.Repeat("x", browserIDLimit+1), ""), 0)
	for i := 0; i < browserRequestLimit+2; i++ {
		f.emit(0, browserRequestJSON(fmt.Sprint("bounded", i), "wails/runtime.js"), 0)
	}
	s := f.snapshot()
	a := s.Assets[1]
	if a.LengthChecks != 1 || a.LengthMismatches != 1 || a.MIMEUnexpected != 3 || s.Assets[5].Failed != 1 || s.Assets[5].Canceled != 1 || s.Assets[5].Blocked != 1 || s.Invalid < 5 || s.Overflow < 1 || s.AssociationMissing != 3 || s.DOMContentLoaded != 0 || s.LoadEvent != 0 {
		t.Fatal("unknown/failure/bounds evidence incorrect", s)
	}
	before := s.Events
	for i := 0; i < browserEventLimit+1; i++ {
		f.emit(6, `{"name":"networkIdle"}`, 0)
	}
	if s = f.snapshot(); s.Events != browserEventLimit || s.Overflow == 0 || s.Events < before {
		t.Fatal("event storage budget exceeded", s)
	}
}
func TestBrowserDiagnosticsRejectsOffThreadCallbacksWithoutNativeReads(t *testing.T) {
	f := newBrowserNativeFixture(t, true, "")
	f.arm()
	pointer := f.handlers[0]
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		callbackTable(unsafe.Pointer(pointer)).Invoke.Call(pointer, 0, 0)
		f.view.closeBrowserDiagnostics()
		close(done)
	}()
	<-done
	if s := f.snapshot(); s.OffThreadCallbacks != 2 || s.Closed || s.Events != 0 {
		t.Fatal("off-thread observation called native getter or closed UI ownership", s)
	}
}

func TestBrowserDiagnosticsIntegralNumbersPreserveExactBounds(t *testing.T) {
	for _, value := range []string{"200", "200.0", "2e2", "2E+2", "0.200e3"} {
		if got, ok := browserNumber(json.Number(value), 999); !ok || got != 200 {
			t.Error("valid integral CDP number rejected", value, got, ok)
		}
	}
	for _, value := range []string{"200.5", "200.000000000000000001", "1e999999999", "1e-999999999", "NaN", "Inf", "-1", "1000", "0.0000001"} {
		if _, ok := browserNumber(json.Number(value), 999); ok {
			t.Error("invalid/fractional/unbounded CDP number accepted", value)
		}
	}
}

func TestBrowserDiagnosticsMalformedHeaderNeverClaimsBodyMismatch(t *testing.T) {
	for _, headers := range []string{`{"Content-Length":"5","Content-Encoding":3}`, `{"Content-Length":"5","content-length":"6"}`, `{"Content-Length":"+5"}`} {
		t.Run(headers, func(t *testing.T) {
			f := newBrowserNativeFixture(t, true, "")
			f.arm()
			f.emit(0, browserRequestJSON("doc", ""), 0)
			f.emit(1, `{"requestId":"doc","response":{"url":"http://wails.localhost/","status":200.0,"mimeType":"text/html","headers":`+headers+`}}`, 0)
			f.emit(2, `{"requestId":"doc","dataLength":2e0}`, 0)
			f.emit(3, `{"requestId":"doc","encodedDataLength":90}`, 0)
			s := f.snapshot()
			if s.Invalid == 0 || s.Assets[0].LengthChecks != 0 || s.Assets[0].LengthMismatches != 0 || s.Assets[0].Status != 200 {
				t.Fatal("invalid length metadata produced a byte mismatch claim", s)
			}
		})
	}
}
func TestBrowserDiagnosticsEnvironmentHookKeepsNativeCreationOrder(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprint(enabled), func(t *testing.T) {
			f := newBrowserNativeFixture(t, enabled, "")
			var calls []string
			environment := &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{
				_IUnknownVtbl: _IUnknownVtbl{AddRef: NewComProc(func(uintptr) uintptr { f.checkThread(); calls = append(calls, "addref"); return 2 })},
				GetBrowserVersionString: NewComProc(func(_ uintptr, out **uint16) uintptr {
					f.checkThread()
					calls = append(calls, "version")
					*out = browserTestString(t, "152.0.4191.66")
					return 0
				}),
				CreateCoreWebView2Controller: NewComProc(func(uintptr, uintptr, uintptr) uintptr {
					f.checkThread()
					calls = append(calls, "controller")
					return 0
				}),
			}}
			f.view.EnvironmentCompleted(0, environment)
			expected := "addref,controller"
			if enabled {
				expected = "addref,version,controller"
			}
			if strings.Join(calls, ",") != expected {
				t.Fatal("environment hook changed native creation sequence", calls)
			}
			runtime.KeepAlive(environment)
		})
	}
}

func TestBrowserDiagnosticsHTTPStatusFailureRemainsSticky(t *testing.T) {
	f := newBrowserNativeFixture(t, true, "")
	f.arm()
	for i, status := range []int{500, 200} {
		id := fmt.Sprint("doc", i)
		f.emit(0, browserRequestJSON(id, ""), 0)
		f.emit(1, fmt.Sprintf(`{"requestId":%q,"response":{"url":"http://wails.localhost/","status":%d,"mimeType":"text/html","headers":{}}}`, id, status), 0)
	}
	a := f.snapshot().Assets[0]
	if a.Status != 200 || a.Responses != 2 || a.UnexpectedStatus != 1 {
		t.Fatal("later successful status erased the earlier HTTP failure", a)
	}
}
