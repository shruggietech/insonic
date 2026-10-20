//go:build windows

// SPDX-License-Identifier: Apache-2.0
package edge

import (
	"encoding/json"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf16"
	"unsafe"

	"github.com/wailsapp/go-webview2/pkg/combridge"
	"golang.org/x/sys/windows"
)

const browserJSONLimit = 64 << 10
const browserEventLimit = 256
const browserRequestLimit = 16
const browserIDLimit = 128
const browserByteLimit = 1 << 30

var browserDiagnosticsEnabled atomic.Bool

// EnableWindowBrowserDiagnostics elects only disposable source diagnostics.
// Domain setup delays the initial navigation until its four acknowledgements;
// it does not change the original watchdog or ordinary launch behavior.
func EnableWindowBrowserDiagnostics(enabled bool) { browserDiagnosticsEnabled.Store(enabled) }

var browserEventNames = [...]string{"Network.requestWillBeSent", "Network.responseReceived", "Network.dataReceived", "Network.loadingFinished", "Network.loadingFailed", "Runtime.exceptionThrown", "Page.lifecycleEvent"}
var browserEnableMethods = [...]string{"Network.enable", "Runtime.enable", "Page.enable", "Page.setLifecycleEventsEnabled"}
var browserAssetNames = [...]string{"document", "ipc", "runtime", "app-css", "product-css", "app-script"}

type WindowBrowserAssetSnapshot struct {
	Class        string `json:"class"`
	Requested    uint64 `json:"requested"`
	Responses    uint64 `json:"responses"`
	Finished     uint64 `json:"finished"`
	Failed       uint64 `json:"failed"`
	Canceled     uint64 `json:"canceled"`
	Blocked      uint64 `json:"blocked"`
	DataEvents   uint64 `json:"data_events"`
	DecodedBytes uint64 `json:"decoded_bytes"`
	EncodedBytes uint64 `json:"encoded_bytes"`
	// Status is the last observed response status; the failure count is sticky.
	Status           uint16 `json:"status"`
	UnexpectedStatus uint64 `json:"unexpected_status"`
	MIME             string `json:"mime"`
	HeaderMIME       string `json:"header_mime"`
	ContentLength    uint64 `json:"content_length"`
	LengthKnown      bool   `json:"length_known"`
	LengthChecks     uint64 `json:"length_checks"`
	LengthMismatches uint64 `json:"length_mismatches"`
	MIMEUnexpected   uint64 `json:"mime_unexpected"`
}

// Snapshot fields are fixed classes/counters only. CDP payloads, URLs, headers,
// request IDs, exceptions and native addresses never leave bounded transient state.
type WindowBrowserSnapshot struct {
	ActualRuntimeVersion          string `json:"actual_runtime_version"`
	VersionCalls                  uint64 `json:"version_calls"`
	VersionHRESULT                uint32 `json:"version_hresult"`
	VersionInvalid                bool   `json:"version_invalid"`
	Registered                    uint16 `json:"registered_mask"`
	EnabledDomains                uint8  `json:"enabled_domains_mask"`
	SetupStage                    string `json:"setup_stage"`
	SetupFailed                   uint64 `json:"setup_failed"`
	SetupHRESULT                  uint32 `json:"setup_hresult"`
	Armed                         bool   `json:"armed"`
	NavigationDeferred            bool   `json:"navigation_deferred"`
	NavigationStarted             bool   `json:"navigation_started"`
	DocumentObserved              bool   `json:"document_observed"`
	LifecycleAssociationAvailable bool   `json:"lifecycle_association_available"`
	AssociationMissing            uint64 `json:"association_missing"`
	IgnoredLifecycle              uint64 `json:"ignored_lifecycle"`
	Closed                        bool   `json:"closed"`
	Events                        uint64 `json:"events"`
	Overflow                      uint64 `json:"overflow"`
	Invalid                       uint64 `json:"invalid"`
	Unclassified                  uint64 `json:"unclassified"`
	EarlyEvents                   uint64 `json:"early_events"`
	LateCallbacks                 uint64 `json:"late_callbacks"`
	OffThreadCallbacks            uint64 `json:"off_thread_callbacks"`
	CleanupFailures               uint64 `json:"cleanup_failures"`
	UnassociatedExceptions        uint64 `json:"unassociated_exceptions"`
	Exceptions                    uint64 `json:"exceptions"`
	ExceptionClass                string `json:"exception_class"`
	ExceptionLine                 uint32 `json:"exception_line"`
	ExceptionColumn               uint32 `json:"exception_column"`
	ExceptionLocationKnown        bool   `json:"exception_location_known"`
	// Milestones require the initial document loader and frame identity.
	DOMContentLoaded uint64                        `json:"dom_content_loaded"`
	LoadEvent        uint64                        `json:"load_event"`
	Assets           [6]WindowBrowserAssetSnapshot `json:"assets"`
}

type browserCDPReceiverVtbl struct {
	_IUnknownVtbl
	Add, Remove ComProc
}
type browserCDPReceiver struct{ vtbl *browserCDPReceiverVtbl }
type browserCDPArgsVtbl struct {
	_IUnknownVtbl
	GetJSON ComProc
}
type browserCDPArgs struct{ vtbl *browserCDPArgsVtbl }
type browserRequest struct {
	id                            string
	class                         int
	response, terminal            bool
	dataEvents, decoded           uint64
	length                        uint64
	lengthKnown, identityEncoding bool
	lengthMetadataValid           bool
}
type browserReceiverOwner struct {
	receiver   *browserCDPReceiver
	token      _EventRegistrationToken
	registered bool
	lease      nativeCallbackLease
}
type windowBrowserDiagnostics struct {
	sync.Mutex
	chromium        *Chromium
	thread          uint32
	started, closed bool
	pendingURL      string
	targetLoader    string
	targetFrame     string
	snapshot        WindowBrowserSnapshot
	requests        [browserRequestLimit]browserRequest
	receivers       [len(browserEventNames)]browserReceiverOwner
	commands        [len(browserEnableMethods)]nativeCallbackLease
}

type browserCDPEventCallback interface {
	EventReceived(*ICoreWebView2, *browserCDPArgs) uintptr
}
type browserCDPMethodCallback interface {
	MethodCompleted(uintptr, *uint16) uintptr
}
type browserEventTarget struct {
	state *windowBrowserDiagnostics
	event int
}
type browserMethodTarget struct {
	state  *windowBrowserDiagnostics
	method int
}

func init() {
	// Microsoft.Web.WebView2 1.0.3537.50 WebView2.h, native Invoke signatures.
	combridge.RegisterVTable[combridge.IUnknown, browserCDPEventCallback]("{e2fda4be-5456-406c-a261-3d452138362c}", invokeBrowserCDPEvent)
	combridge.RegisterVTable[combridge.IUnknown, browserCDPMethodCallback]("{5c4889f0-5ef6-4c5a-952c-d8f1b92d0574}", invokeBrowserCDPMethod)
}
func invokeBrowserCDPEvent(this uintptr, sender *ICoreWebView2, args *browserCDPArgs) uintptr {
	if target := combridge.Resolve[browserCDPEventCallback](this); target != nil {
		return target.EventReceived(sender, args)
	}
	return 0
}
func invokeBrowserCDPMethod(this uintptr, hr uintptr, result *uint16) uintptr {
	if target := combridge.Resolve[browserCDPMethodCallback](this); target != nil {
		return target.MethodCompleted(hr, result)
	}
	return 0
}
func (t *browserEventTarget) EventReceived(_ *ICoreWebView2, args *browserCDPArgs) uintptr {
	t.state.event(t.event, args)
	return 0
}
func (t *browserMethodTarget) MethodCompleted(hr uintptr, _ *uint16) uintptr {
	t.state.enabled(t.method, hr)
	return 0
}

func (e *Chromium) browserDiagnosticState() *windowBrowserDiagnostics {
	if !browserDiagnosticsEnabled.Load() {
		return nil
	}
	ownedWindows.Lock()
	defer ownedWindows.Unlock()
	entry := ownedWindows.values[e.hwnd]
	if entry == nil || entry.chromium != e || entry.closed || entry.thread != windows.GetCurrentThreadId() {
		return nil
	}
	if e.browserDiagnostics == nil {
		state := &windowBrowserDiagnostics{chromium: e, thread: entry.thread}
		state.snapshot.SetupStage = "idle"
		for i, name := range browserAssetNames {
			state.snapshot.Assets[i].Class = name
		}
		e.browserDiagnostics = state
	}
	return e.browserDiagnostics
}

// readBrowserOwnedString always frees the COM-owned output, even on bounds error.
func readBrowserOwnedString(value *uint16, limit int, free func(unsafe.Pointer)) (string, bool) {
	if value == nil {
		return "", false
	}
	defer free(unsafe.Pointer(value))
	capacity := limit
	if capacity > 128 {
		capacity = 128
	}
	data := make([]uint16, 0, capacity)
	for i := 0; i <= limit; i++ {
		c := *(*uint16)(unsafe.Add(unsafe.Pointer(value), uintptr(i)*2))
		if c == 0 {
			return string(utf16.Decode(data)), true
		}
		if i == limit {
			return "", false
		}
		data = append(data, c)
	}
	return "", false
}
func browserVersionOnly(value string) bool {
	fields := strings.Split(value, " ")
	if len(fields) > 2 {
		return false
	}
	if len(fields) == 2 && fields[1] != "beta" && fields[1] != "dev" && fields[1] != "canary" {
		return false
	}
	parts := strings.Split(fields[0], ".")
	if len(parts) != 4 {
		return false
	}
	for _, part := range parts {
		if len(part) == 0 || len(part) > 10 {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
func (e *Chromium) observeBrowserEnvironment(result uintptr, environment *ICoreWebView2Environment) {
	if result != 0 || environment == nil {
		return
	}
	state := e.browserDiagnosticState()
	if state == nil {
		return
	}
	state.Lock()
	if state.snapshot.VersionCalls != 0 {
		state.Unlock()
		return
	}
	state.snapshot.VersionCalls++
	state.Unlock()
	var version *uint16
	hr, _, _ := environment.vtbl.GetBrowserVersionString.Call(uintptr(unsafe.Pointer(environment)), uintptr(unsafe.Pointer(&version)))
	value, valid := readBrowserOwnedString(version, 64, windows.CoTaskMemFree)
	state.Lock()
	state.snapshot.VersionHRESULT = uint32(hr)
	if hr == 0 && valid && browserVersionOnly(value) {
		state.snapshot.ActualRuntimeVersion = value
	} else {
		state.snapshot.VersionInvalid = true
	}
	state.Unlock()
}

func (e *Chromium) deferBrowserDiagnosticNavigation(target string) bool {
	// Only the fixed initial application navigation is elected, never arbitrary URLs.
	if !browserDiagnosticsEnabled.Load() || target != "http://wails.localhost/" {
		return false
	}
	state := e.browserDiagnosticState()
	if state == nil {
		return false
	}
	state.Lock()
	if state.started || state.closed {
		state.Unlock()
		return false
	}
	state.started = true
	state.pendingURL = target
	state.snapshot.NavigationDeferred = true
	state.snapshot.SetupStage = "register"
	state.Unlock()
	for i, name := range browserEventNames {
		if hr := state.register(i, name); hr != 0 {
			state.setupFailure(hr)
			return true
		}
	}
	state.enable(0)
	return true
}
func (state *windowBrowserDiagnostics) register(index int, name string) uintptr {
	text, _ := windows.UTF16PtrFromString(name)
	var receiver *browserCDPReceiver
	hr, _, _ := state.chromium.webview.vtbl.GetDevToolsProtocolEventReceiver.Call(uintptr(unsafe.Pointer(state.chromium.webview)), uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(&receiver)))
	if receiver != nil {
		state.receivers[index].receiver = receiver
	}
	if hr != 0 {
		return hr
	}
	if receiver == nil {
		return uintptr(windows.E_POINTER)
	}
	lease := combridge.New[browserCDPEventCallback](&browserEventTarget{state: state, event: index})
	state.receivers[index].lease = lease
	var token _EventRegistrationToken
	hr, _, _ = receiver.vtbl.Add.Call(uintptr(unsafe.Pointer(receiver)), lease.Ref(), uintptr(unsafe.Pointer(&token)))
	if hr == 0 {
		state.receivers[index].token = token
		state.receivers[index].registered = true
		state.Lock()
		state.snapshot.Registered |= 1 << index
		state.Unlock()
	}
	return hr
}
func (state *windowBrowserDiagnostics) enable(index int) {
	state.Lock()
	if state.closed {
		state.Unlock()
		return
	}
	state.snapshot.SetupStage = [...]string{"enable-network", "enable-runtime", "enable-page", "enable-lifecycle"}[index]
	state.Unlock()
	lease := combridge.New[browserCDPMethodCallback](&browserMethodTarget{state: state, method: index})
	state.commands[index] = lease
	method, _ := windows.UTF16PtrFromString(browserEnableMethods[index])
	parameterJSON := "{}"
	if index == 3 {
		parameterJSON = `{"enabled":true}`
	}
	parameters, _ := windows.UTF16PtrFromString(parameterJSON)
	hr, _, _ := state.chromium.webview.vtbl.CallDevToolsProtocolMethod.Call(uintptr(unsafe.Pointer(state.chromium.webview)), uintptr(unsafe.Pointer(method)), uintptr(unsafe.Pointer(parameters)), lease.Ref())
	if hr != 0 {
		state.setupFailure(hr)
	}
}
func (state *windowBrowserDiagnostics) enabled(index int, hr uintptr) {
	state.Lock()
	if state.closed || state.pendingURL == "" {
		state.snapshot.LateCallbacks++
		state.Unlock()
		return
	}
	if state.thread != windows.GetCurrentThreadId() {
		state.snapshot.OffThreadCallbacks++
		state.Unlock()
		return
	}
	if index < 0 || index >= len(browserEnableMethods) || state.snapshot.EnabledDomains != uint8((1<<index)-1) {
		state.snapshot.Invalid++
		state.Unlock()
		return
	}
	if hr != 0 {
		state.Unlock()
		state.setupFailure(hr)
		return
	}
	state.snapshot.EnabledDomains |= 1 << index
	last := index == len(browserEnableMethods)-1
	if last {
		state.snapshot.Armed = true
		state.snapshot.SetupStage = "armed"
	}
	state.Unlock()
	if last {
		state.navigate()
	} else {
		state.enable(index + 1)
	}
}
func (state *windowBrowserDiagnostics) setupFailure(hr uintptr) {
	state.Lock()
	if state.closed || state.pendingURL == "" {
		state.Unlock()
		return
	}
	state.snapshot.SetupFailed++
	state.snapshot.SetupHRESULT = uint32(hr)
	state.snapshot.SetupStage = "setup-failed"
	state.Unlock()
	// A setup failure invalidates this observation, without inventing a new product gate.
	state.navigate()
}
func (state *windowBrowserDiagnostics) navigate() {
	state.Lock()
	target := state.pendingURL
	state.pendingURL = ""
	closed := state.closed
	if !closed && target != "" {
		state.snapshot.NavigationStarted = true
	}
	state.Unlock()
	if !closed && target != "" && !state.chromium.shuttingDown {
		state.chromium.navigateWithoutBrowserDiagnostics(target)
	}
}

func (e *Chromium) closeBrowserDiagnostics() {
	state := e.browserDiagnostics
	if state == nil {
		return
	}
	state.Lock()
	if state.closed {
		state.Unlock()
		return
	}
	if state.thread != windows.GetCurrentThreadId() {
		state.snapshot.OffThreadCallbacks++
		state.Unlock()
		return
	}
	state.closed = true
	state.pendingURL = ""
	state.targetLoader = ""
	state.targetFrame = ""
	state.snapshot.Closed = true
	state.snapshot.SetupStage = "closed"
	state.requests = [browserRequestLimit]browserRequest{}
	state.Unlock()
	for i := range state.receivers {
		owner := &state.receivers[i]
		if owner.receiver != nil {
			if owner.registered {
				// EventRegistrationToken is an eight-byte value, not a pointer to that value.
				var hr uintptr
				if unsafe.Sizeof(uintptr(0)) == 4 {
					hr, _, _ = owner.receiver.vtbl.Remove.Call(uintptr(unsafe.Pointer(owner.receiver)), uintptr(uint32(owner.token.value)), uintptr(uint32(uint64(owner.token.value)>>32)))
				} else {
					hr, _, _ = owner.receiver.vtbl.Remove.Call(uintptr(unsafe.Pointer(owner.receiver)), uintptr(owner.token.value))
				}
				if hr != 0 {
					state.Lock()
					state.snapshot.CleanupFailures++
					state.Unlock()
				}
			}
			owner.receiver.vtbl.Release.Call(uintptr(unsafe.Pointer(owner.receiver)))
		}
		if owner.lease != nil {
			_ = owner.lease.Close()
		}
		*owner = browserReceiverOwner{}
	}
	for i, lease := range state.commands {
		if lease != nil {
			_ = lease.Close()
			state.commands[i] = nil
		}
	}
}
func OwnedWindowBrowserSnapshots() []WindowBrowserSnapshot {
	ownedWindows.Lock()
	var states [4]*windowBrowserDiagnostics
	count := 0
	for _, entry := range ownedWindows.values {
		if count < len(states) && entry.chromium.browserDiagnostics != nil {
			states[count] = entry.chromium.browserDiagnostics
			count++
		}
	}
	ownedWindows.Unlock()
	result := make([]WindowBrowserSnapshot, 0, count)
	for _, state := range states[:count] {
		state.Lock()
		result = append(result, state.snapshot)
		state.Unlock()
	}
	return result
}

func browserAssetClass(raw string) int {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "http" || parsed.Host != "wails.localhost" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil || parsed.RawPath != "" || parsed.ForceQuery {
		return -1
	}
	switch parsed.Path {
	case "/":
		return 0
	case "/wails/ipc.js":
		return 1
	case "/wails/runtime.js":
		return 2
	case "/assets/bundle/app.css":
		return 3
	case "/assets/bundle/product.css":
		return 4
	case "/assets/bundle/app.js":
		return 5
	}
	return -1
}
func browserMIMEClass(value string) string {
	switch strings.ToLower(strings.TrimSpace(strings.SplitN(value, ";", 2)[0])) {
	case "text/html":
		return "html"
	case "application/javascript", "text/javascript", "application/ecmascript", "text/ecmascript":
		return "javascript"
	case "text/css":
		return "css"
	case "":
		return "missing"
	}
	return "other"
}
func browserNumber(value json.Number, limit uint64) (uint64, bool) {
	text := string(value)
	if len(text) == 0 || len(text) > 64 {
		return 0, false
	}
	// Bound exponent work before exact rational parsing of integral CDP numbers.
	if at := strings.IndexAny(text, "eE"); at >= 0 {
		exponent, err := strconv.Atoi(text[at+1:])
		if err != nil || exponent < -64 || exponent > 64 {
			return 0, false
		}
	}
	number, ok := new(big.Rat).SetString(text)
	if !ok || !number.IsInt() || !number.Num().IsUint64() {
		return 0, false
	}
	result := number.Num().Uint64()
	return result, result <= limit
}

type browserNetworkEvent struct {
	RequestID string `json:"requestId"`
	LoaderID  string `json:"loaderId"`
	FrameID   string `json:"frameId"`
	Name      string `json:"name"`
	Request   struct {
		URL string `json:"url"`
	} `json:"request"`
	Response struct {
		URL     string                     `json:"url"`
		Status  json.Number                `json:"status"`
		MIME    string                     `json:"mimeType"`
		Headers map[string]json.RawMessage `json:"headers"`
	} `json:"response"`
	DataLength    json.Number `json:"dataLength"`
	EncodedLength json.Number `json:"encodedDataLength"`
	Canceled      bool        `json:"canceled"`
	BlockedReason string      `json:"blockedReason"`
	Exception     struct {
		URL    string      `json:"url"`
		Line   json.Number `json:"lineNumber"`
		Column json.Number `json:"columnNumber"`
	} `json:"exceptionDetails"`
}

func (state *windowBrowserDiagnostics) event(index int, args *browserCDPArgs) {
	state.Lock()
	if state.closed {
		state.snapshot.LateCallbacks++
		state.Unlock()
		return
	}
	if state.thread != windows.GetCurrentThreadId() {
		state.snapshot.OffThreadCallbacks++
		state.Unlock()
		return
	}
	if !state.snapshot.Armed || !state.snapshot.NavigationStarted {
		state.snapshot.EarlyEvents++
		state.Unlock()
		return
	}
	if state.snapshot.Events >= browserEventLimit {
		state.snapshot.Overflow++
		state.Unlock()
		return
	}
	state.snapshot.Events++
	state.Unlock()
	if args == nil {
		state.invalidBrowserEvent()
		return
	}
	var raw *uint16
	hr, _, _ := args.vtbl.GetJSON.Call(uintptr(unsafe.Pointer(args)), uintptr(unsafe.Pointer(&raw)))
	value, valid := readBrowserOwnedString(raw, browserJSONLimit/2, windows.CoTaskMemFree)
	if hr != 0 || !valid || len(value) > browserJSONLimit {
		state.invalidBrowserEvent()
		return
	}
	var event browserNetworkEvent
	if json.Unmarshal([]byte(value), &event) != nil {
		state.invalidBrowserEvent()
		return
	}
	state.Lock()
	defer state.Unlock()
	if state.closed {
		state.snapshot.LateCallbacks++
		return
	}
	if index == 6 {
		if event.Name != "DOMContentLoaded" && event.Name != "load" {
			state.snapshot.IgnoredLifecycle++
			return
		}
		if state.targetLoader == "" || state.targetFrame == "" {
			state.snapshot.AssociationMissing++
			return
		}
		if event.LoaderID != state.targetLoader || event.FrameID != state.targetFrame {
			state.snapshot.IgnoredLifecycle++
			return
		}
		if event.Name == "DOMContentLoaded" {
			state.snapshot.DOMContentLoaded++
		} else {
			state.snapshot.LoadEvent++
		}
		return
	}
	if index == 5 {
		if !state.snapshot.DocumentObserved {
			state.snapshot.UnassociatedExceptions++
			return
		}
		state.snapshot.Exceptions++
		if state.snapshot.Exceptions == 1 {
			class := browserAssetClass(event.Exception.URL)
			line, lineOK := browserNumber(event.Exception.Line, 1<<24)
			column, columnOK := browserNumber(event.Exception.Column, 1<<24)
			state.snapshot.ExceptionClass = "other"
			if class >= 0 {
				state.snapshot.ExceptionClass = browserAssetNames[class]
			}
			if class >= 0 && lineOK && columnOK {
				state.snapshot.ExceptionLine = uint32(line)
				state.snapshot.ExceptionColumn = uint32(column)
				state.snapshot.ExceptionLocationKnown = true
			}
		}
		return
	}
	if event.RequestID == "" || len(event.RequestID) > browserIDLimit {
		state.snapshot.Invalid++
		return
	}
	var request *browserRequest
	for i := range state.requests {
		if state.requests[i].id == event.RequestID {
			request = &state.requests[i]
			break
		}
	}
	if index == 0 {
		class := browserAssetClass(event.Request.URL)

		if class == 0 && !state.snapshot.DocumentObserved {
			state.snapshot.DocumentObserved = true
			if event.LoaderID != "" && len(event.LoaderID) <= browserIDLimit && event.FrameID != "" && len(event.FrameID) <= browserIDLimit {
				state.targetLoader = event.LoaderID
				state.targetFrame = event.FrameID
				state.snapshot.LifecycleAssociationAvailable = true
			} else {
				state.snapshot.AssociationMissing++
			}
		}
		if class >= 0 && state.targetLoader != "" && (event.LoaderID != state.targetLoader || event.FrameID != state.targetFrame) {
			state.snapshot.Unclassified++
			return
		}
		if class < 0 {
			state.snapshot.Unclassified++
			return
		}
		if request != nil {
			state.snapshot.Invalid++
			return
		}
		for i := range state.requests {
			if state.requests[i].id == "" {
				request = &state.requests[i]
				*request = browserRequest{id: event.RequestID, class: class}
				break
			}
		}
		if request == nil {
			state.snapshot.Overflow++
			return
		}
		state.snapshot.Assets[class].Requested++
		return
	}
	if request == nil {
		state.snapshot.Unclassified++
		return
	}
	asset := &state.snapshot.Assets[request.class]
	switch index {
	case 1:
		if request.response || request.terminal || len(event.Response.Headers) > 64 || browserAssetClass(event.Response.URL) != request.class {
			state.snapshot.Invalid++
			return
		}
		status, ok := browserNumber(event.Response.Status, 999)
		if !ok {
			state.snapshot.Invalid++
			return
		}
		request.response = true
		asset.Responses++
		asset.Status = uint16(status)
		if status != 200 {
			asset.UnexpectedStatus++
		}
		asset.MIME = browserMIMEClass(event.Response.MIME)
		expected := "javascript"
		if request.class == 0 {
			expected = "html"
		}
		if request.class == 3 || request.class == 4 {
			expected = "css"
		}
		if asset.MIME != expected {
			asset.MIMEUnexpected++
		}
		request.identityEncoding = true
		request.lengthMetadataValid = true
		seenHeaders := uint8(0)
		for name, rawValue := range event.Response.Headers {
			var text string
			if !strings.EqualFold(name, "Content-Length") && !strings.EqualFold(name, "Content-Type") && !strings.EqualFold(name, "Content-Encoding") {
				continue
			}
			headerBit := uint8(1)
			if strings.EqualFold(name, "Content-Type") {
				headerBit = 2
			}
			if strings.EqualFold(name, "Content-Encoding") {
				headerBit = 4
			}
			if seenHeaders&headerBit != 0 {
				state.snapshot.Invalid++
				request.lengthMetadataValid = false
				continue
			}
			seenHeaders |= headerBit
			if json.Unmarshal(rawValue, &text) != nil || len(text) > 128 {
				request.lengthMetadataValid = false
				state.snapshot.Invalid++
				continue
			}
			switch strings.ToLower(name) {
			case "content-length":
				length, err := strconv.ParseUint(text, 10, 64)
				digitsOnly := text != ""
				for _, digit := range text {
					if digit < '0' || digit > '9' {
						digitsOnly = false
					}
				}
				if err != nil || !digitsOnly || length > browserByteLimit || request.lengthKnown {
					request.lengthMetadataValid = false
					state.snapshot.Invalid++
					continue
				}
				request.length = length
				request.lengthKnown = true
				asset.ContentLength = length
				asset.LengthKnown = true
			case "content-type":
				asset.HeaderMIME = browserMIMEClass(text)
			case "content-encoding":
				request.identityEncoding = text == "" || strings.EqualFold(text, "identity")
			}
		}
	case 2:
		size, ok := browserNumber(event.DataLength, browserByteLimit)
		if !ok || request.terminal || request.decoded > browserByteLimit-size {
			state.snapshot.Invalid++
			return
		}
		request.dataEvents++
		request.decoded += size
		asset.DataEvents++
		asset.DecodedBytes += size
	case 3:
		size, ok := browserNumber(event.EncodedLength, browserByteLimit)
		if !ok || request.terminal {
			state.snapshot.Invalid++
			return
		}
		request.terminal = true
		asset.Finished++
		asset.EncodedBytes += size
		if request.response && request.lengthKnown && request.identityEncoding && request.lengthMetadataValid && (request.dataEvents != 0 || request.length == 0) {
			asset.LengthChecks++
			if request.length != request.decoded {
				asset.LengthMismatches++
			}
		}
	case 4:
		if request.terminal {
			state.snapshot.Invalid++
			return
		}
		request.terminal = true
		asset.Failed++
		if event.Canceled {
			asset.Canceled++
		}
		if event.BlockedReason != "" {
			asset.Blocked++
		}
	default:
		state.snapshot.Invalid++
	}
}
func (state *windowBrowserDiagnostics) invalidBrowserEvent() {
	state.Lock()
	state.snapshot.Invalid++
	state.Unlock()
}
