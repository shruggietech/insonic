//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/combridge"
	"golang.org/x/sys/windows"
)

type nativeCallbackView struct{ table *nativeCallbackTable }
type nativeCallbackTable struct {
	_IUnknownVtbl
	Invoke ComProc
}
type callbackFixture struct {
	name, iid string
	pointer   unsafe.Pointer
}

func actualCallbackFixtures(e *Chromium) []callbackFixture {
	return []callbackFixture{
		{"environment", "{4e8a3389-c9d8-4bd2-b6b5-124fee6cc14d}", unsafe.Pointer(e.envCompleted)},
		{"controller", "{6c4819f3-c9b7-4260-8127-c9f5bde7f68c}", unsafe.Pointer(e.controllerCompleted)},
		{"webMessage", "{57213f19-00e6-49fa-8e07-898ea01ecbd2}", unsafe.Pointer(e.webMessageReceived)},
		{"permission", "{15e1c6a3-c72a-4df3-91d7-d097fbec6bfd}", unsafe.Pointer(e.permissionRequested)},
		{"resource", "{ab00b74c-15f1-4646-80e8-e76341d25d71}", unsafe.Pointer(e.webResourceRequested)},
		{"accelerator", "{b29c7e28-fa79-41a8-8e44-65811c76dcb2}", unsafe.Pointer(e.acceleratorKeyPressed)},
		{"navigation", "{d33a35bf-1c49-4f98-93ab-006e0533fe1c}", unsafe.Pointer(e.navigationCompleted)},
		{"processFailed", "{79e0aea4-990b-42d9-aa1d-0fcc2e5bc7f1}", unsafe.Pointer(e.processFailed)},
		{"fullscreen", "{e45d98b1-afef-45be-8baf-6c7728867f73}", unsafe.Pointer(e.containsFullScreenElementChanged)},
	}
}
func callbackTable(pointer unsafe.Pointer) *nativeCallbackTable {
	return (*nativeCallbackView)(pointer).table
}
func queryCallback(pointer unsafe.Pointer, iid *windows.GUID, out *uintptr) uint32 {
	value, _, _ := callbackTable(pointer).QueryInterface.Call(uintptr(pointer), uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(out)))
	return uint32(value)
}
func releaseCallback(pointer unsafe.Pointer) uintptr {
	value, _, _ := callbackTable(pointer).Release.Call(uintptr(pointer))
	return value
}

func TestNativeCallbackBridgeQueryInterfaceAndIdentity(t *testing.T) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	defer e.releaseOwnedCallbacks()
	if len(e.callbackLeases) != 9 {
		t.Fatal("not all callback creator leases were retained")
	}
	unknown, _ := windows.GUIDFromString("{00000000-0000-0000-c000-000000000046}")
	for _, callback := range actualCallbackFixtures(e) {
		t.Run(callback.name, func(t *testing.T) {
			iid, _ := windows.GUIDFromString(callback.iid)
			var own, canonical, roundtrip, canonicalAgain uintptr
			if hr := queryCallback(callback.pointer, &iid, &own); hr != uint32(windows.S_OK) || own != uintptr(callback.pointer) {
				t.Fatalf("own IID failed: HRESULT=%08x valid_identity=%v", hr, own == uintptr(callback.pointer))
			}
			defer releaseCallback(unsafe.Pointer(own))
			if hr := queryCallback(callback.pointer, &unknown, &canonical); hr != uint32(windows.S_OK) || canonical == 0 {
				t.Fatalf("IUnknown failed: HRESULT=%08x", hr)
			}
			defer releaseCallback(unsafe.Pointer(canonical))
			if hr := queryCallback(unsafe.Pointer(canonical), &iid, &roundtrip); hr != uint32(windows.S_OK) || roundtrip != own {
				t.Fatalf("IUnknown to own IID roundtrip failed: HRESULT=%08x", hr)
			}
			defer releaseCallback(unsafe.Pointer(roundtrip))
			if hr := queryCallback(unsafe.Pointer(roundtrip), &unknown, &canonicalAgain); hr != uint32(windows.S_OK) || canonicalAgain != canonical {
				t.Fatalf("canonical IUnknown identity changed: HRESULT=%08x", hr)
			}
			defer releaseCallback(unsafe.Pointer(canonicalAgain))
			for _, unsupported := range []string{"{94ea2b94-e9cc-49e0-c0ff-ee64ca8f5b90}", "{00000003-0000-0000-c000-000000000046}", "{cb80e614-4539-4d83-a4a7-46b937ea5a27}"} {
				bad, _ := windows.GUIDFromString(unsupported)
				out := uintptr(1)
				if hr := queryCallback(callback.pointer, &bad, &out); hr != uint32(windows.E_NOINTERFACE) || out != 0 {
					t.Errorf("unsupported IID accepted or output not cleared: HRESULT=%08x", hr)
				}
			}
			if hr := queryCallback(callback.pointer, &iid, nil); hr != uint32(windows.E_POINTER) {
				t.Errorf("null output accepted: HRESULT=%08x", hr)
			}
			out := uintptr(1)
			if hr := queryCallback(callback.pointer, nil, &out); hr != uint32(windows.E_INVALIDARG) || out != 0 {
				t.Errorf("null IID accepted or output not cleared: HRESULT=%08x", hr)
			}
		})
	}
}

func collectCallbackTarget(t *testing.T, finalized <-chan struct{}) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		runtime.GC()
		select {
		case <-finalized:
			return
		default:
		}
		runtime.Gosched()
		time.Sleep(time.Millisecond)
	}
	t.Fatal("callback target remained rooted after its last owner released it")
}
func neverEmbeddedTarget(finalized chan<- struct{}) {
	e := NewChromium()
	runtime.SetFinalizer(e, func(*Chromium) { finalized <- struct{}{} })
	runtime.KeepAlive(e)
}
func TestNativeCallbackBridgeNeverEmbeddedCollects(t *testing.T) {
	e := NewChromium()
	if len(e.callbackLeases) != 0 || e.envCompleted != nil {
		t.Fatal("NewChromium allocated native callback leases before Embed")
	}
	finalized := make(chan struct{}, 1)
	neverEmbeddedTarget(finalized)
	collectCallbackTarget(t, finalized)
}
func nativeOnlyCallbackTarget(finalized chan<- struct{}) (uintptr, ComProc) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	runtime.SetFinalizer(e, func(*Chromium) { finalized <- struct{}{} })
	pointer := uintptr(unsafe.Pointer(e.controllerCompleted))
	table := e.controllerCompleted.vtbl
	table.AddRef.Call(pointer)
	e.releaseOwnedCallbacks() // Release all 9 creator leases; one native ref remains.
	runtime.KeepAlive(e)
	return pointer, table.Release
}
func TestNativeCallbackBridgeNativeReferenceRetainsUntilFinalRelease(t *testing.T) {
	finalized := make(chan struct{}, 1)
	pointer, release := nativeOnlyCallbackTarget(finalized)
	for i := 0; i < 3; i++ {
		runtime.GC()
		runtime.Gosched()
		select {
		case <-finalized:
			t.Fatal("native AddRef did not retain its target after all creator leases closed")
		default:
		}
	}
	if got := combridge.Resolve[*Chromium](pointer); got == nil {
		t.Fatal("native callback lost its bridge registry target")
	}
	remaining, _, _ := release.Call(pointer)
	if remaining != 0 {
		t.Fatal("final native Release left references", remaining)
	}
	if combridge.Resolve[*Chromium](pointer) != nil {
		t.Fatal("final native Release retained a registry root")
	}
	collectCallbackTarget(t, finalized)
}
func partialCallbackTarget(finalized chan<- struct{}) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	runtime.SetFinalizer(e, func(*Chromium) { finalized <- struct{}{} })
	values := actualCallbackFixtures(e)
	e.ShuttingDown()
	e.releaseOwnedInterfaces()
	e.releaseOwnedInterfaces()
	for _, value := range values {
		if combridge.Resolve[*Chromium](uintptr(value.pointer)) != nil {
			panic("partial initialization retained a callback lease")
		}
	}
	runtime.KeepAlive(e)
}
func TestNativeCallbackBridgePartialInitializationAndRepeatedCleanup(t *testing.T) {
	finalized := make(chan struct{}, 1)
	partialCallbackTarget(finalized)
	collectCallbackTarget(t, finalized)
}

func TestNativeCallbackBridgeLateCompletionsAndNotifications(t *testing.T) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	values := actualCallbackFixtures(e)
	// Stand in for native WebView2 retaining every callback before owner teardown.
	for _, value := range values {
		callbackTable(value.pointer).AddRef.Call(uintptr(value.pointer))
		defer releaseCallback(value.pointer)
	}
	captured := captureEnvironmentCallback(e.envCompleted)
	e.ShuttingDown()
	e.releaseOwnedInterfaces()
	environmentCalls, controllerCloses, controllerReferences, notifications := 0, 0, 0, 0
	env := &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{
		_IUnknownVtbl:                _IUnknownVtbl{AddRef: NewComProc(func(uintptr) uintptr { environmentCalls++; return 1 })},
		CreateCoreWebView2Controller: NewComProc(func(uintptr, uintptr, uintptr) uintptr { environmentCalls++; return 0 }),
	}}
	controller := &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		_IUnknownVtbl: _IUnknownVtbl{AddRef: NewComProc(func(uintptr) uintptr { controllerReferences++; return 1 }), Release: NewComProc(func(uintptr) uintptr { controllerReferences--; return 0 })},
		Close:         NewComProc(func(uintptr) uintptr { controllerCloses++; return 0 }),
	}}
	e.NavigationCompletedCallback = func(*ICoreWebView2, *ICoreWebView2NavigationCompletedEventArgs) { notifications++ }
	e.ProcessFailedCallback = func(*ICoreWebView2, *ICoreWebView2ProcessFailedEventArgs) { notifications++ }
	e.ContainsFullScreenElementChangedCallback = func(*ICoreWebView2, *ICoreWebView2ContainsFullScreenElementChangedEventArgs) { notifications++ }
	for _, value := range values {
		first, second := uintptr(0), uintptr(0)
		if value.name == "environment" {
			second = uintptr(unsafe.Pointer(env))
		}
		if value.name == "controller" {
			second = uintptr(unsafe.Pointer(controller))
		}
		hr, _, _ := callbackTable(value.pointer).Invoke.Call(uintptr(value.pointer), first, second)
		if hr != 0 {
			t.Errorf("late %s callback failed: %08x", value.name, hr)
		}
	}
	// The default Go loader captures a Go implementation, not an expiring native allocation.
	if captured.EnvironmentCompleted(0, env) != 0 {
		t.Fatal("captured default-loader target failed after cancellation")
	}
	if environmentCalls != 0 || controllerCloses != 1 || controllerReferences != 0 || notifications != 0 || e.controller != nil || e.environment != nil {
		t.Fatal("late completion resurrected or called a destroyed WebView", environmentCalls, controllerCloses, controllerReferences, notifications)
	}
	runtime.KeepAlive(env)
	runtime.KeepAlive(controller)
	runtime.KeepAlive(e)
}

func TestNativeCallbackBridgeDispatchesLiveNotifications(t *testing.T) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	defer e.releaseOwnedCallbacks()
	delivered := map[string]int{}
	e.MessageCallback = func(message string, _ *ICoreWebView2, _ *ICoreWebView2WebMessageReceivedEventArgs) {
		if message != "" {
			t.Error("unexpected message")
		}
		delivered["webMessage"]++
	}
	messageSender := &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{PostWebMessageAsString: NewComProc(func(uintptr, *uint16) uintptr { return 0 })}}
	message := &ICoreWebView2WebMessageReceivedEventArgs{vtbl: &iCoreWebView2WebMessageReceivedEventArgsVtbl{TryGetWebMessageAsString: NewComProc(func(_ uintptr, out **uint16) uintptr { *out = nil; return 0 }), GetAdditionalObjects: NewComProc(func(_ uintptr, out **ICoreWebView2ObjectCollectionView) uintptr { *out = nil; return 0 })}}
	permission := &iCoreWebView2PermissionRequestedEventArgs{vtbl: &iCoreWebView2PermissionRequestedEventArgsVtbl{
		GetPermissionKind: NewComProc(func(_ uintptr, out *CoreWebView2PermissionKind) uintptr {
			*out = CoreWebView2PermissionKindMicrophone
			return 0
		}),
		PutState: NewComProc(func(uintptr, uintptr) uintptr { delivered["permission"]++; return 0 }),
	}}
	requestReleases := 0
	request := &ICoreWebView2WebResourceRequest{vtbl: &_ICoreWebView2WebResourceRequestVtbl{_IUnknownVtbl: _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { requestReleases++; return 0 })}}}
	resource := &ICoreWebView2WebResourceRequestedEventArgs{vtbl: &_ICoreWebView2WebResourceRequestedEventArgsVtbl{GetRequest: NewComProc(func(_ uintptr, out **ICoreWebView2WebResourceRequest) uintptr { *out = request; return 0 })}}
	e.WebResourceRequestedCallback = func(got *ICoreWebView2WebResourceRequest, _ *ICoreWebView2WebResourceRequestedEventArgs) {
		if got != request {
			t.Error("request pointer changed")
		}
		delivered["resource"]++
	}
	accelerator := &ICoreWebView2AcceleratorKeyPressedEventArgs{vtbl: &_ICoreWebView2AcceleratorKeyPressedEventArgsVtbl{
		GetKeyEventKind: NewComProc(func(_ uintptr, out *COREWEBVIEW2_KEY_EVENT_KIND) uintptr {
			*out = COREWEBVIEW2_KEY_EVENT_KIND_KEY_DOWN
			return 0
		}),
		GetVirtualKey: NewComProc(func(_ uintptr, out *uint) uintptr { *out = 65; return 0 }),
		GetPhysicalKeyStatus: NewComProc(func(_ uintptr, out *internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS) uintptr {
			*out = internal_COREWEBVIEW2_PHYSICAL_KEY_STATUS{}
			return 0
		}),
		PutHandled: NewComProc(func(_ uintptr, handled uintptr) uintptr {
			if handled != 1 {
				t.Error("accelerator disposition lost")
			}
			return 0
		}),
	}}
	e.AcceleratorKeyCallback = func(key uint) bool {
		if key != 65 {
			t.Error("accelerator key changed")
		}
		delivered["accelerator"]++
		return true
	}
	e.NavigationCompletedCallback = func(*ICoreWebView2, *ICoreWebView2NavigationCompletedEventArgs) { delivered["navigation"]++ }
	e.ProcessFailedCallback = func(*ICoreWebView2, *ICoreWebView2ProcessFailedEventArgs) { delivered["processFailed"]++ }
	e.ContainsFullScreenElementChangedCallback = func(*ICoreWebView2, *ICoreWebView2ContainsFullScreenElementChangedEventArgs) {
		delivered["fullscreen"]++
	}
	arguments := map[string]unsafe.Pointer{"webMessage": unsafe.Pointer(message), "permission": unsafe.Pointer(permission), "resource": unsafe.Pointer(resource), "accelerator": unsafe.Pointer(accelerator)}
	for _, value := range actualCallbackFixtures(e) {
		if value.name == "environment" || value.name == "controller" {
			continue
		}
		sender := uintptr(0)
		if value.name == "webMessage" {
			sender = uintptr(unsafe.Pointer(messageSender))
		}
		hr, _, _ := callbackTable(value.pointer).Invoke.Call(uintptr(value.pointer), sender, uintptr(arguments[value.name]))
		if hr != 0 || delivered[value.name] != 1 {
			t.Errorf("live %s callback did not dispatch exactly once", value.name)
		}
	}
	if requestReleases != 1 {
		t.Error("resource callback did not release its owned request")
	}
	runtime.KeepAlive([]any{messageSender, message, permission, request, resource, accelerator, e})
}

func TestNativeCallbackBridgeDispatchesLiveCreation(t *testing.T) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	envReferences, controllerReferences, webviewReferences := 1, 1, 1
	nativeHandlers := []unsafe.Pointer{}
	register := NewComProc(func(_ uintptr, handler unsafe.Pointer, _ uintptr) uintptr {
		callbackTable(handler).AddRef.Call(uintptr(handler))
		nativeHandlers = append(nativeHandlers, handler)
		return 0
	})
	view := &ICoreWebView2{vtbl: &iCoreWebView2Vtbl{
		_IUnknownVtbl:         _IUnknownVtbl{Release: NewComProc(func(uintptr) uintptr { webviewReferences--; return uintptr(webviewReferences) })},
		AddWebMessageReceived: register, AddPermissionRequested: register, AddWebResourceRequested: register, AddNavigationCompleted: register, AddProcessFailed: register, AddContainsFullScreenElementChanged: register,
	}}
	closes := 0
	controller := &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		_IUnknownVtbl: _IUnknownVtbl{
			AddRef:         NewComProc(func(uintptr) uintptr { controllerReferences++; return uintptr(controllerReferences) }),
			Release:        NewComProc(func(uintptr) uintptr { controllerReferences--; return uintptr(controllerReferences) }),
			QueryInterface: NewComProc(func(_ uintptr, _ uintptr, out *uintptr) uintptr { *out = 0; return uintptr(windows.E_NOINTERFACE) }),
		},
		GetCoreWebView2: NewComProc(func(_ uintptr, out **ICoreWebView2) uintptr { *out = view; return 0 }), AddAcceleratorKeyPressed: register,
		Close: NewComProc(func(uintptr) uintptr {
			closes++
			for _, handler := range nativeHandlers {
				releaseCallback(handler)
			}
			nativeHandlers = nil
			return 0
		}),
	}}
	created := 0
	environment := &ICoreWebView2Environment{vtbl: &iCoreWebView2EnvironmentVtbl{
		_IUnknownVtbl: _IUnknownVtbl{AddRef: NewComProc(func(uintptr) uintptr { envReferences++; return uintptr(envReferences) }), Release: NewComProc(func(uintptr) uintptr { envReferences--; return uintptr(envReferences) })},
		CreateCoreWebView2Controller: NewComProc(func(_ uintptr, _ uintptr, handler unsafe.Pointer) uintptr {
			created++
			callbackTable(handler).Invoke.Call(uintptr(handler), 0, uintptr(unsafe.Pointer(controller)))
			return 0
		}),
	}}
	hr, _, _ := e.envCompleted.vtbl.Invoke.Call(uintptr(unsafe.Pointer(e.envCompleted)), 0, uintptr(unsafe.Pointer(environment)))
	if hr != 0 || created != 1 || atomic.LoadUintptr(&e.inited) != 1 || len(nativeHandlers) != 7 || e.environment != environment || e.controller != controller || e.webview != view {
		t.Fatal("actual creation callbacks failed to install native state", created, len(nativeHandlers))
	}
	e.ShuttingDown()
	if err := e.controller.Close(); err != nil {
		t.Fatal(err)
	}
	e.releaseOwnedInterfaces()
	e.releaseOwnedInterfaces()
	if closes != 1 || envReferences != 1 || controllerReferences != 1 || webviewReferences != 0 || len(e.callbackLeases) != 0 {
		t.Fatal("native Close/interface/creator cleanup did not balance ownership", closes, envReferences, controllerReferences, webviewReferences)
	}
	runtime.KeepAlive([]any{environment, controller, view, e})
}

func TestNativeCallbackBridgeAcceleratorRegistrationOutput(t *testing.T) {
	e := NewChromium()
	e.prepareOwnedCallbacks()
	defer e.releaseOwnedCallbacks()
	token := &_EventRegistrationToken{value: 17}
	handler := e.acceleratorKeyPressed
	handlerIdentity := uintptr(unsafe.Pointer(handler))
	const nativeToken int64 = 0x1234567890abcdef
	calls, writes := 0, 0
	wrongOutput, wrongHandler, wrongController, fail := false, false, false, false
	var controller *ICoreWebView2Controller
	controller = &ICoreWebView2Controller{vtbl: &_ICoreWebView2ControllerVtbl{
		AddAcceleratorKeyPressed: NewComProc(func(receiver, receivedHandler uintptr, output *_EventRegistrationToken) uintptr {
			calls++
			wrongController = wrongController || receiver != uintptr(unsafe.Pointer(controller))
			wrongHandler = wrongHandler || receivedHandler != handlerIdentity
			if uintptr(unsafe.Pointer(output)) != uintptr(unsafe.Pointer(token)) {
				// The old wrapper passes the address of its own pointer variable.
				// Do not write through that incorrect address on the red run.
				wrongOutput = true
				return 0
			}
			if fail {
				return uintptr(windows.E_FAIL)
			}
			*output = _EventRegistrationToken{value: nativeToken}
			writes++
			return 0
		}),
	}}
	if err := controller.AddAcceleratorKeyPressed(handler, token); err != nil {
		t.Fatal(err)
	}
	if wrongOutput {
		t.Fatal("native token output address differs from the caller's token; no write attempted")
	}
	if wrongHandler || wrongController || calls != 1 || writes != 1 || token.value != nativeToken {
		t.Fatal("native registration changed its receiver/handler or failed to return the token", wrongController, wrongHandler, calls, writes, token.value)
	}
	fail = true
	token.value = 23
	err := controller.AddAcceleratorKeyPressed(handler, token)
	if !errors.Is(err, windows.Errno(windows.E_FAIL)) || calls != 2 || writes != 1 || token.value != 23 {
		t.Fatal("native registration HRESULT or failed-call output was changed", err, calls, writes, token.value)
	}
	runtime.KeepAlive(controller)
	runtime.KeepAlive(token)
	runtime.KeepAlive(handler)
}
