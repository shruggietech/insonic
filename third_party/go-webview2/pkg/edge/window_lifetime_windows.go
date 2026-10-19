//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var lifetimeComctl = windows.NewLazySystemDLL("comctl32.dll")
var lifetimeSetSubclass = lifetimeComctl.NewProc("SetWindowSubclass")
var lifetimeRemoveSubclass = lifetimeComctl.NewProc("RemoveWindowSubclass")
var lifetimeDefSubclass = lifetimeComctl.NewProc("DefSubclassProc")
var lifetimeGetOwner = windows.NewLazySystemDLL("user32.dll").NewProc("GetWindowThreadProcessId")
var lifetimeGetFocus = windows.NewLazySystemDLL("user32.dll").NewProc("GetFocus")
var lifetimeSetFocus = windows.NewLazySystemDLL("user32.dll").NewProc("SetFocus")
var lifetimeGetForeground = windows.NewLazySystemDLL("user32.dll").NewProc("GetForegroundWindow")
var lifetimeIsChild = windows.NewLazySystemDLL("user32.dll").NewProc("IsChild")

// NativeSystemKeyEvent copies accelerator metadata; no borrowed COM pointer escapes.
type NativeSystemKeyEvent struct {
	VirtualKey        uint
	LParam            uint32
	PhysicalKeyStatus COREWEBVIEW2_PHYSICAL_KEY_STATUS
}

// SetNativeSystemKeyHandler configures the single host application's native
// system-key route. Call before running the UI and clear after its run returns.
// Handlers must only post native work, never enter modal UI inside a COM callback.
func SetNativeSystemKeyHandler(handler func(uintptr, NativeSystemKeyEvent) bool) {
	ownedWindows.Lock()
	ownedWindows.systemKeyHandler = handler
	ownedWindows.Unlock()
}

func (e *Chromium) nativeSystemKeyHandler() func(uintptr, NativeSystemKeyEvent) bool {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[e.hwnd]
	if entry == nil || entry.chromium != e || entry.closed || entry.thread != windows.GetCurrentThreadId() {
		return nil
	}
	return ownedWindows.systemKeyHandler
}

// WindowBootstrapSnapshot contains only fixed counters and the browser version,
// never request URLs, script contents or application data.
type WindowBootstrapSnapshot struct {
	OffThreadEvents       uint64                           `json:"off_thread_events"`
	RuntimeVersion        string                           `json:"runtime_version"` // Available-loader lookup, not created-environment identity.
	NavigationRequested   uint64                           `json:"navigation_requested"`
	NavigationAccepted    uint64                           `json:"navigation_accepted"`
	ResourceEntered       uint64                           `json:"resource_entered"`
	ResourceReturned      uint64                           `json:"resource_returned"`
	NavigationCompleted   uint64                           `json:"navigation_completed"`
	CompletionWithoutHost uint64                           `json:"completion_without_host"`
	FirstResponse         WindowBootstrapResponseSnapshot  `json:"first_response"`
	Responses             WindowBootstrapResponsesSnapshot `json:"responses"`
	RuntimeReadyReceived  uint64                           `json:"runtime_ready_received"`
}

// WindowBootstrapCallSnapshot records actual native call boundaries. A zero
// HRESULT is meaningful only when Returned is nonzero. No native address escapes.
type WindowBootstrapCallSnapshot struct {
	Entered        uint64 `json:"entered"`
	Returned       uint64 `json:"returned"`
	Failed         uint64 `json:"failed"`
	OffThreadCalls uint64 `json:"off_thread_calls"`
	LastHRESULT    uint32 `json:"last_hresult"`
}

// WindowBootstrapResponseCalls contains fixed native response-call boundaries.
// Headers observes GetHeaders acquisition, not individual AppendHeader calls.
type WindowBootstrapResponseCalls struct {
	Assignment WindowBootstrapCallSnapshot `json:"assignment"`
	Headers    WindowBootstrapCallSnapshot `json:"headers"`
	Status     WindowBootstrapCallSnapshot `json:"status"`
	Reason     WindowBootstrapCallSnapshot `json:"reason"`
	Content    WindowBootstrapCallSnapshot `json:"content"`
	Deferral   WindowBootstrapCallSnapshot `json:"deferral"`
	Completion WindowBootstrapCallSnapshot `json:"completion"`
}

// WindowBootstrapResponseSnapshot preserves detail for the first request only.
// These counters establish publication calls, not browser consumption or rendering.
type WindowBootstrapResponseSnapshot struct {
	WindowBootstrapResponseCalls
	StatusCode      int    `json:"status_code"`
	ContentAttempts uint64 `json:"content_attempts"`
	ContentBytes    uint64 `json:"content_bytes"`
	StreamsCreated  uint64 `json:"streams_created"`
	StreamFailures  uint64 `json:"stream_failures"`
}

// WindowBootstrapResponsesSnapshot aggregates requests using eight active slots.
// Overflow and requests without deferrals make coverage limits explicit. Published and
// Completed count unique requests, so repeated setters/Complete cannot hide one
// unfinished request. Published means assignment, GetHeaders, status, reason and
// content all returned successfully; it does not attest individual header writes.
// Retired means metadata retired, not a native Release result.
type WindowBootstrapResponsesSnapshot struct {
	WindowBootstrapResponseCalls
	Tracked           uint64 `json:"tracked"`
	Active            uint64 `json:"active"`
	Overflow          uint64 `json:"overflow"`
	WithoutDeferral   uint64 `json:"without_deferral"`
	Published         uint64 `json:"published"`
	Completed         uint64 `json:"completed"`
	Retired           uint64 `json:"retired"`
	RetiredIncomplete uint64 `json:"retired_incomplete"`
	StreamFailures    uint64 `json:"stream_failures"`
}

const (
	bootstrapResponseAssignment uint8 = iota
	bootstrapResponseHeaders
	bootstrapResponseStatus
	bootstrapResponseReason
	bootstrapResponseContent
	bootstrapResponseDeferral
	bootstrapResponseCompletion
	bootstrapResponseContentAttempt
	bootstrapResponseStreamCreated
	bootstrapResponseStreamFailed
)

// Borrowed native identities are equality tokens only, never dereferenced or
// retained as COM references. The fixed slots disappear with their window owner.
const bootstrapResponseCapacity = 8

type windowBootstrapResponse struct {
	ordinal                                        uint64
	callback, args, response, deferral             uintptr
	assignmentOK, headers, status, reason, content bool
	deferred                                       bool
	published, completed                           bool
}

func recordWindowBootstrapRequest(e *Chromium, args *ICoreWebView2WebResourceRequestedEventArgs) {
	if !bootstrapDiagnosticsEnabled.Load() {
		return
	}
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[e.hwnd]
	if entry == nil || entry.chromium != e || entry.closed || entry.thread != windows.GetCurrentThreadId() {
		return
	}
	for index := range entry.requests {
		request := &entry.requests[index]
		if request.ordinal != 0 {
			continue
		}
		entry.bootstrap.Responses.Tracked++
		entry.bootstrap.Responses.Active++
		identity := uintptr(unsafe.Pointer(args))
		*request = windowBootstrapResponse{ordinal: entry.bootstrap.Responses.Tracked, callback: identity, args: identity}
		return
	}
	entry.bootstrap.Responses.Overflow++
}

func recordWindowBootstrapRequestReturned(e *Chromium, args *ICoreWebView2WebResourceRequestedEventArgs) {
	if !bootstrapDiagnosticsEnabled.Load() {
		return
	}
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[e.hwnd]
	if entry == nil || entry.chromium != e || entry.closed {
		return
	}
	identity := uintptr(unsafe.Pointer(args))
	for index := range entry.requests {
		request := &entry.requests[index]
		if request.ordinal == 0 || request.callback != identity {
			continue
		}
		request.callback = 0
		// Synchronous responses and default native handling need no deferral.
		// Record that coverage distinction without treating it as failure.
		if !request.deferred {
			entry.bootstrap.Responses.WithoutDeferral++
		}
		request.args = 0
		retireBootstrapRequest(entry, request)
		return
	}
}

func retireBootstrapRequest(entry *windowLifetime, request *windowBootstrapResponse) {
	if request.ordinal == 0 || request.callback != 0 || request.args != 0 || request.response != 0 || request.deferral != 0 {
		return
	}
	entry.bootstrap.Responses.Active--
	entry.bootstrap.Responses.Retired++
	if request.deferred && !request.completed {
		entry.bootstrap.Responses.RetiredIncomplete++
	}
	*request = windowBootstrapResponse{}
}

func bootstrapResponseStage(calls *WindowBootstrapResponseCalls, event uint8) *WindowBootstrapCallSnapshot {
	switch event {
	case bootstrapResponseAssignment:
		return &calls.Assignment
	case bootstrapResponseHeaders:
		return &calls.Headers
	case bootstrapResponseStatus:
		return &calls.Status
	case bootstrapResponseReason:
		return &calls.Reason
	case bootstrapResponseContent:
		return &calls.Content
	case bootstrapResponseDeferral:
		return &calls.Deferral
	case bootstrapResponseCompletion:
		return &calls.Completion
	}
	return nil
}

func observeBootstrapCall(call *WindowBootstrapCallSnapshot, returned bool, hr uintptr, offThread bool) {
	if returned {
		call.Returned++
		call.LastHRESULT = uint32(hr)
		if hr != 0 {
			call.Failed++
		}
	} else {
		call.Entered++
		if offThread {
			call.OffThreadCalls++
		}
	}
}

// Never hold the ownership lock across a native call: COM may pump callbacks.
// value is an internal identity, HTTP status, or content byte count. The wrappers
// retain their original native calls and HRESULT handling.
func recordBootstrapResponseCall(identity uintptr, event uint8, returned bool, hr, value uintptr) {
	if !bootstrapDiagnosticsEnabled.Load() || identity == 0 {
		return
	}
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	for _, entry := range ownedWindows.values {
		if entry.closed {
			continue
		}
		for index := range entry.requests {
			request := &entry.requests[index]
			if request.ordinal == 0 {
				continue
			}
			var expected uintptr
			switch event {
			case bootstrapResponseAssignment, bootstrapResponseDeferral:
				expected = request.args
			case bootstrapResponseCompletion:
				expected = request.deferral
			default:
				expected = request.response
			}
			if identity != expected {
				continue
			}
			first := &entry.bootstrap.FirstResponse
			aggregate := &entry.bootstrap.Responses
			isFirst := request.ordinal == 1
			switch event {
			case bootstrapResponseAssignment:
				if !returned {
					request.response = value
				} else {
					request.assignmentOK = hr == 0
					if hr != 0 {
						request.args = 0
					}
				}
			case bootstrapResponseHeaders:
				if returned {
					request.headers = hr == 0
				}
			case bootstrapResponseStatus:
				if !returned && isFirst {
					first.StatusCode = int(value)
				}
				if returned {
					request.status = hr == 0
				}
			case bootstrapResponseReason:
				if returned {
					request.reason = hr == 0
				}
			case bootstrapResponseContent:
				if returned {
					request.content = hr == 0
				}
			case bootstrapResponseDeferral:
				if returned {
					if hr == 0 && value != 0 {
						request.deferral = value
						request.deferred = true
					}
					request.args = 0
				}
			case bootstrapResponseCompletion:
				if returned && hr == 0 && !request.completed {
					request.completed = true
					aggregate.Completed++
				}
			case bootstrapResponseContentAttempt:
				if isFirst {
					first.ContentAttempts++
					first.ContentBytes = uint64(value)
				}
				return
			case bootstrapResponseStreamCreated:
				if isFirst {
					first.StreamsCreated++
				}
				return
			case bootstrapResponseStreamFailed:
				aggregate.StreamFailures++
				if isFirst {
					first.StreamFailures++
				}
				return
			default:
				return
			}
			offThread := entry.thread != windows.GetCurrentThreadId()
			observeBootstrapCall(bootstrapResponseStage(&aggregate.WindowBootstrapResponseCalls, event), returned, hr, offThread)
			if isFirst {
				observeBootstrapCall(bootstrapResponseStage(&first.WindowBootstrapResponseCalls, event), returned, hr, offThread)
			}
			if !request.published && request.assignmentOK && request.headers && request.status && request.reason && request.content {
				request.published = true
				aggregate.Published++
			}
			return
		}
	}
}

func retireBootstrapResponse(identity uintptr, deferral bool) {
	if !bootstrapDiagnosticsEnabled.Load() || identity == 0 {
		return
	}
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	for _, entry := range ownedWindows.values {
		for index := range entry.requests {
			request := &entry.requests[index]
			if request.ordinal == 0 {
				continue
			}
			if deferral && request.deferral == identity {
				request.deferral = 0
			}
			if !deferral && request.response == identity {
				request.response = 0
			}
			retireBootstrapRequest(entry, request)
		}
	}
}

var bootstrapDiagnosticsEnabled atomic.Bool

func EnableWindowBootstrapDiagnostics(enabled bool) { bootstrapDiagnosticsEnabled.Store(enabled) }

const (
	bootstrapNavigateRequested uint8 = iota
	bootstrapNavigateAccepted
	bootstrapResourceEntered
	bootstrapResourceReturned
	bootstrapNavigationCompleted
	bootstrapRuntimeReadyReceived
)

func recordWindowBootstrap(e *Chromium, event uint8) {
	if !bootstrapDiagnosticsEnabled.Load() {
		return
	}
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[e.hwnd]
	if entry == nil || entry.chromium != e || entry.closed {
		return
	}
	value := &entry.bootstrap
	if entry.thread != windows.GetCurrentThreadId() {
		value.OffThreadEvents++
		return
	}
	value.RuntimeVersion = e.webview2RuntimeVersion
	if len(value.RuntimeVersion) > 64 {
		value.RuntimeVersion = value.RuntimeVersion[:64]
	}
	switch event {
	case bootstrapNavigateRequested:
		value.NavigationRequested++
	case bootstrapNavigateAccepted:
		value.NavigationAccepted++
	case bootstrapResourceEntered:
		value.ResourceEntered++
	case bootstrapResourceReturned:
		value.ResourceReturned++
	case bootstrapRuntimeReadyReceived:
		value.RuntimeReadyReceived++
	case bootstrapNavigationCompleted:
		value.NavigationCompleted++
		if e.NavigationCompletedCallback == nil {
			value.CompletionWithoutHost++
		}
	}
}
func OwnedWindowBootstrapSnapshots() []WindowBootstrapSnapshot {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	var result []WindowBootstrapSnapshot
	for _, entry := range ownedWindows.values {
		if len(result) < 4 {
			result = append(result, entry.bootstrap)
		}
	}
	return result
}

type windowLifetime struct {
	bootstrap WindowBootstrapSnapshot
	requests  [bootstrapResponseCapacity]windowBootstrapResponse
	chromium  *Chromium
	thread    uint32
	closed    bool
	menuDepth uint32
	menuFocus uintptr
}

// Native code retains only HWNDs and a static callback, never a Go object pointer.
var ownedWindows = struct {
	sync.Mutex
	values                             map[uintptr]*windowLifetime
	failure                            error
	closedControllers                  uint32
	apartmentThread                    uint32
	apartmentOwned, apartmentFinalized bool
	systemKeyHandler                   func(uintptr, NativeSystemKeyEvent) bool
}{values: make(map[uintptr]*windowLifetime)}

var lifetimeSubclass uintptr

func init() { lifetimeSubclass = syscall.NewCallback(lifetimeWindowProc) }

// Native menus must own the host thread's keyboard input while the out-of-process
// WebView is focused. Do not let the framework's WM_SETFOCUS handler immediately
// move it back into the browser. Restore the actual child after the menu closes.
func beginNativeMenuFocus(hwnd uintptr) {
	foreground, _, _ := lifetimeGetForeground.Call()
	if foreground != hwnd {
		return
	}
	focus, _, _ := lifetimeGetFocus.Call()
	child, _, _ := lifetimeIsChild.Call(hwnd, focus)
	if focus != 0 && focus != hwnd && child == 0 {
		return
	}
	ownedWindows.Lock()
	entry := ownedWindows.values[hwnd]
	if entry == nil || entry.closed || entry.thread != windows.GetCurrentThreadId() {
		ownedWindows.Unlock()
		return
	}
	entry.menuDepth++
	first := entry.menuDepth == 1
	if first {
		entry.menuFocus = focus
	}
	ownedWindows.Unlock()
	if first {
		lifetimeSetFocus.Call(hwnd)
	}
}
func endNativeMenuFocus(hwnd uintptr) {
	ownedWindows.Lock()
	entry := ownedWindows.values[hwnd]
	if entry == nil || entry.closed || entry.thread != windows.GetCurrentThreadId() || entry.menuDepth == 0 {
		ownedWindows.Unlock()
		return
	}
	entry.menuDepth--
	if entry.menuDepth != 0 {
		ownedWindows.Unlock()
		return
	}
	previous := entry.menuFocus
	entry.menuFocus = 0
	ownedWindows.Unlock()
	foreground, _, _ := lifetimeGetForeground.Call()
	focus, _, _ := lifetimeGetFocus.Call()
	if foreground != hwnd || focus != hwnd || previous == hwnd {
		return
	}
	child, _, _ := lifetimeIsChild.Call(hwnd, previous)
	if previous == 0 || child != 0 {
		lifetimeSetFocus.Call(previous)
	}
}
func nativeMenuOwnsFocus(hwnd uintptr) bool {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[hwnd]
	return entry != nil && entry.thread == windows.GetCurrentThreadId() && (entry.closed || entry.menuDepth > 0)
}

func lifetimeWindowProc(hwnd, message, wparam, lparam, id, data uintptr) uintptr {
	if message == 0x211 {
		beginNativeMenuFocus(hwnd)
	}
	if message == 0x212 {
		endNativeMenuFocus(hwnd)
	}
	if message == 7 && nativeMenuOwnsFocus(hwnd) {
		return 0
	}
	if message == 0x0002 || message == 0x0082 { // WM_DESTROY / WM_NCDESTROY
		ownedWindows.Lock()
		entry := ownedWindows.values[hwnd]
		closeController := entry != nil && !entry.closed
		if closeController {
			entry.closed = true
		}
		if message == 0x0082 {
			delete(ownedWindows.values, hwnd)
		}
		ownedWindows.Unlock()
		if closeController {
			entry.chromium.ShuttingDown()
			if controller := entry.chromium.controller; controller != nil {
				if err := controller.Close(); err != nil {
					entry.chromium.recordControllerCloseFailure(err)
				} else {
					ownedWindows.Lock()
					ownedWindows.closedControllers++
					ownedWindows.Unlock()
				}
			}
			entry.chromium.releaseOwnedInterfaces()
		}
		if message == 0x0082 {
			lifetimeRemoveSubclass.Call(hwnd, lifetimeSubclass, id)
		}
	}
	result, _, _ := lifetimeDefSubclass.Call(hwnd, message, wparam, lparam)
	return result
}

// WindowCloseError preserves a failed Close without interrupting ordinary
// native destruction or the application's remaining shutdown callbacks.
func WindowCloseError() error {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	return ownedWindows.failure
}

// ClosedControllerCount reports completed Close operations, not merely requests.
func ClosedControllerCount() uint32 {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	return ownedWindows.closedControllers
}

func (e *Chromium) attachWindowLifetime(hwnd uintptr) error {
	for _, procedure := range []*windows.LazyProc{lifetimeSetSubclass, lifetimeRemoveSubclass, lifetimeDefSubclass, lifetimeGetOwner} {
		if err := procedure.Find(); err != nil {
			return err
		}
	}
	var pid uint32
	thread, _, _ := lifetimeGetOwner.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if hwnd == 0 || pid != uint32(os.Getpid()) || uint32(thread) != windows.GetCurrentThreadId() {
		return fmt.Errorf("WebView lifetime requires its owned UI-thread window")
	}
	ownedWindows.Lock()
	if ownedWindows.values[hwnd] != nil {
		ownedWindows.Unlock()
		return fmt.Errorf("WebView window lifetime already registered")
	}
	ownedWindows.values[hwnd] = &windowLifetime{chromium: e, thread: uint32(thread)}
	ownedWindows.Unlock()
	if accepted, _, _ := lifetimeSetSubclass.Call(hwnd, lifetimeSubclass, lifetimeSubclass, 0); accepted == 0 {
		ownedWindows.Lock()
		delete(ownedWindows.values, hwnd)
		ownedWindows.Unlock()
		return fmt.Errorf("WebView window lifetime registration failed")
	}
	return nil
}

// Close synchronously releases the WebView's browser resources and event cycles.
// It must run on the owning UI thread before the parent window is destroyed.
func (i *ICoreWebView2Controller) Close() error {
	hr, _, _ := i.vtbl.Close.Call(uintptr(unsafe.Pointer(i)))
	if windows.Handle(hr) != windows.S_OK {
		return windows.Errno(hr)
	}
	return nil
}

// Close stops the WebView; Release balances the interfaces retained by Embed.
func (e *Chromium) releaseOwnedInterfaces() {
	if value := e.webview; value != nil {
		e.webview = nil
		value.vtbl.Release.Call(uintptr(unsafe.Pointer(value)))
	}
	if value := e.controller; value != nil {
		e.controller = nil
		value.Release()
	}
	if value := e.environment; value != nil {
		e.environment = nil
		value.vtbl.Release.Call(uintptr(unsafe.Pointer(value)))
	}
	e.releaseOwnedCallbacks()
}

func (e *Chromium) recordControllerCloseFailure(err error) {
	failure := fmt.Errorf("close WebView controller: %w", err)
	ownedWindows.Lock()
	if ownedWindows.failure == nil {
		ownedWindows.failure = failure
	}
	ownedWindows.Unlock()
	e.globalErrorCallback(failure)
}

func recordOwnedCOMApartment() {
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	ownedWindows.apartmentThread = windows.GetCurrentThreadId()
	ownedWindows.apartmentOwned = true
	ownedWindows.apartmentFinalized = false
}

// FinalizeCOMApartment balances the binding's existing successful CoInitializeEx
// after all owned windows are destroyed. It must run on that original UI thread.
// Close errors remain separate; failure cleanup must still release its apartment.
func FinalizeCOMApartment() error {
	ownedWindows.Lock()
	if len(ownedWindows.values) != 0 {
		ownedWindows.Unlock()
		return fmt.Errorf("WebView windows remain before COM apartment cleanup")
	}
	if ownedWindows.apartmentFinalized {
		ownedWindows.Unlock()
		return nil
	}
	if !ownedWindows.apartmentOwned || ownedWindows.apartmentThread != windows.GetCurrentThreadId() {
		ownedWindows.Unlock()
		return fmt.Errorf("WebView COM cleanup requires its initialized owning UI thread")
	}
	ownedWindows.apartmentOwned = false
	ownedWindows.apartmentFinalized = true
	ownedWindows.Unlock()
	windows.CoUninitialize()
	return nil
}
