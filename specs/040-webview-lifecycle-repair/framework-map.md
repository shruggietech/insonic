# Framework deconstruction and failure inventory

## 2026-10-19 source identities

Product baseline: S039 squash c9ed48d682e255f9e03f9d4ffd1ebbb9801f9c20, tree identical to accepted 8d2f1c7. Go 1.27.1, Wails v2.16.0, maintained go-webview2 v1.0.22. Proposed source correction: authoritative v1.0.23, whose complete upstream delta is two BOOL wrappers. Do not equate the loader's available runtime string with the created environment's identity.

## Responsibility map

| Stage | Source authority | Ownership and observed boundary |
| --- | --- | --- |
| Launch | internal/process/hide_windows.go:9-14; tree_windows.go:12-65 | Hidden suspended root, assigned owned job before resume. |
| COM/UI thread | edge/corewebview2.go:18-26; Wails internal/frontend/desktop/windows/winc/app.go:22-23 | Locked initialization/UI thread, STA apartment. |
| Host creation | Wails windows/frontend.go:161-184; window.go:50-152 | One owned native host; controller setup precedes normal main loop. |
| Environment | edge/chromium.go:161-215,315-334 | Default loader uses installed runtime; retained borrowed environment. |
| Controller | edge/chromium.go:337-399 | Retain controller, query Controller3, configure raw bounds and monitor scale, get WebView, register callbacks. |
| Native BOOL defect | edge/ICoreWebView2Controller3.go:111-120 | v1.0.22 passes address of bool to BOOL-by-value setter. Initialization invokes false at chromium.go:356. Upstream v1.0.23 replaces it with 0/1. |
| Navigate | Wails windows/frontend.go:603-614; edge/chromium.go:259-268 | Register all-resource filter, request initial navigation; acceptance is not completion. |
| Deferred resource | Wails windows/frontend.go:635-690; pkg/assetserver/webview/request_windows.go:17-56 | On UI thread, create empty response, assign it, acquire deferral, snapshot metadata and dispatch asynchronous work. |
| Handler | Wails pkg/assetserver/assetserver_webview.go:26-58,65-149; app startup_diagnostics.go:25-40 | Middleware return is before HTML transformation and native Finish. |
| HTML | Wails pkg/assetserver/assetserver.go:140-172,187-220; common.go:45-55 | Inject IPC/runtime scripts and replace Content-Length with final byte length. |
| Publish | Wails pkg/assetserver/webview/responsewriter_windows.go:56-104 | UI invoke sets headers, status/reason and copied stream content. |
| Complete | Wails pkg/assetserver/webview/request_windows.go:123-141 | Release caller response reference, Complete deferral, release acquired deferral reference. |
| Ready | edge/chromium.go NavigationCompleted; Wails windows/frontend.go:952-1004 | Navigation callback starts OnDomReady, controller Hide/Show and application showing. Frontend listener is another readiness boundary. |
| Native input | cmd/insonic-desktop/exit_qualification_windows.go; native_menu_windows.go | Actual owned foreground target, original native key route, menu cancellation/focus restoration, reopen and callback. |
| Close | Wails internal/app/app_production.go:17-24; edge/window_lifetime_windows.go:220-248 | Loop exit followed by native parent destruction; subclass invokes Close once and releases retained interfaces before forwarding destruction. |
| Apartment | cmd/insonic-desktop/webview_lifetime_windows.go; edge/window_lifetime_windows.go:334-355 | Balance original thread's COM initialization after owned windows disappear. |
| Process group | internal/process/capture.go:73-85; descendants_windows.go:22-100 | After root exit, observe owned job active count and actual process handles; forced cleanup never passes. |

Wails paths above are relative to the exact pinned Wails v2.16.0 module; edge paths are relative to third_party/go-webview2/pkg/. This map is version-bound.

## Retained signatures

| Signature | Established evidence | Current status/limit |
| --- | --- | --- |
| Controller creation | Earlier failure before native input, recovered HRESULT800700aa | Private runtime condition unproved. The upstream BOOL error is executed during controller completion, distinct from a failing CreateController result. |
| Pre-DOM | cold028 and final-head CI attempt1: navigation1/1, resource1/1, document1/1, scripts0/completion0, live Go UI pump | Final response publication and browser consumption unobserved. Upstream ABI correction is not yet a demonstrated explanation. |
| UI join | Earlier two-second owned-UI operation timeout | Not captured by cold028; do not conflate with its prepared window and idle GetMessage. |
| Menu consumption | Captured missing X despite visible popup; owned queue responded | Native focus correction has actual cancellation/reopen regressions and two 144-case samples. Historical browser internals unproved; retain those limits. |
| Browser release | Final-head CI attempt2: action passed, host exit0, two actual owned live WebView processes at deadline | No stage durations reveal remaining release budget; Close is not a full process-exit receipt. |

## Rejected or separated explanations

- Initial CSP is not applied to the initial document; help policy is path-specific.
- Final Content-Length is rewritten after framework HTML injection.
- Cold028 already mirrors startup-hide and reaches window-prepared; initial show/keyboard handling cannot explain that pre-DOM occurrence.
- Cold028 has no selected blocked Wails response-worker stack. This disfavors a stranded Invoke waiter for that captured instant; the existing four-stack cap still needs honest limit accounting when more stacks exist.
- Warm profiles are reused only after observed natural release. No evidence establishes profile overlap as the cause.
- The nonempty CreateWebResourceResponse stream-shadow defect is outside this Wails path, which initially passes nil content.
- Existing process observations inspect owned handles; they cannot be dismissed as ambient PID counts.
- Additional settings/controller query leaks and invalid legacy callback QueryInterface are real independent source concerns. Explicit successful Close should release the WebView and event cycles; those concerns do not by themselves explain surviving browser processes.

## Experiment decision boundary

First prove/correct the native BOOL argument independently. Then observe the first unresolved response/release boundary in a bounded source experiment. If status/content/Complete succeed on the owning thread before a pre-DOM stall, move to browser consumption/runtime identity rather than adding another host response theory. If release fails, use stage timing and actual process roles before changing shutdown integration. Keep each result separate and retain every failed occurrence.
