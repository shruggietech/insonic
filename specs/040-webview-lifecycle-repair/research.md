# Research and decisions

## 2026-10-19: renewed investigation after S039 merge

The owner directed full deconstruction and repair after squash merge c9ed48d682e255f9e03f9d4ffd1ebbb9801f9c20. Main has identical product bytes to S039 final head 8d2f1c7. Eleven issue closures and all project Done states are verified; milestone 23 is closed with zero open items. #154 is open under milestone 24/S040, In progress.

### Decision: reproduce the upstream native BOOL ABI defect first

The exact v1.0.22-to-v1.0.23 comparison contains only two modified source files. Both PutShouldDetectMonitorScaleChanges wrappers pass a pointer to the Go bool where the native ABI requires BOOL by value. Chromium controller initialization calls the edge method with false. The upstream correction uses canonical 0/1 and cites Wails #4701/#4694 for initialization/blank-window problems. Verify the actual vtable argument locally before adopting the correction. This is a concrete violation, not yet proof of every incident signature.

Primary upstream: [comparison](https://github.com/wailsapp/go-webview2/compare/v1.0.22...v1.0.23), [fix](https://github.com/wailsapp/go-webview2/commit/9a5551d5d93209f1446f778a757862b23219a121), [release](https://github.com/wailsapp/go-webview2/releases/tag/v1.0.23).

### Independent audit: response publication

Application document counters return before Wails HTML injection, responseWriter.Finish, UI invoke, native content publication and deferral.Complete. The native resource callback returns after scheduling asynchronous processing. The retained cold028 profile has only three Wails stacks (UI, signal, second-instance); no response worker appears, which disfavors a stranded Finish waiter in that occurrence but does not observe native completion. Wails replaces Content-Length after HTML rewriting, and help CSP does not apply to the initial page. The initial-response stream-shadow bug is separate because Wails creates the placeholder with nil content and fills it later.

### Independent audit: harness/environment

The real Exit parent assigns the suspended child to an owned job before resuming it and uses startup-hide. The final capped diagnostic reproduced that flag. Native INPUT/MENUBARINFO/MSG layouts have existing actual ABI checks. Each cold case owns a fresh profile; warm reuse follows observed natural release. The release failure retains actual kernel-observed live WebView processes, not ambient counts. The 18-second budget covers startup, action, root exit and descendants together; stage timings are missing. Cold028's seven members after nonzero root exit are not a separate natural-release timeout.

Available runtime 131.0.2903.86 is only a loader availability lookup, not the created environment's independently verified version. The default Go loader clears WEBVIEW2 environment overrides, so a future runtime control must use explicit Wails WebviewBrowserPath and verify actual environment identity.

### Independent audit: native lifetime

Owned controller Close, retained main interfaces and the COM apartment already have corrections and focused regressions. Additional queried settings/controller interface references and legacy callback QueryInterface are source defects to assess independently. Legacy callback QueryInterface reports S_OK without writing the output pointer; its generated sibling repeats that pattern. This is not an established explanation of the retained failure. The default loader wraps the environment callback with combridge, so not every legacy callback reaches native code on this build. Avoid broad migration before assessing the known startup ABI correction.

### Primary framework contracts

- [WebView2 threading and deferrals](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/threading-model): UI-thread access, no unsupported nested callbacks, explicit asynchronous completion.
- [Controller.Close](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2controller): closes the control and breaks event cycles; this does not itself record full browser-process exit.
- [BrowserProcessExited](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2environment5): reports associated processes/resources released; event observation requires retaining its environment and a live message pump.
- [Process model](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/process-model): user-data-folder/process-group ownership and multiple-environment interactions.
- [SHCreateMemStream](https://learn.microsoft.com/en-us/windows/win32/api/shlwapi/nf-shlwapi-shcreatememstream): copying stream creation and supported Windows thread-safety.

### Alternatives considered

A broad Wails upgrade to v2.17.0 has security/tooling changes but no directly identified incident repair in its release notes. Callback migration and extra-interface release could improve correctness but would add independent behavior variables. Timeout increases and weakened natural-release gates would hide the failure. A full packaging rerun supplies less discriminating evidence than the targeted native source experiment. None is selected as the first repair.

### Loader deconstruction while E002 runs

The default Go loader resolves the installed client DLL from registry channel/version data and calls CreateWebViewEnvironmentWithOptionsInternal with checkRunningInstance=true. Microsoft documents this internal export but warns that it can change or disappear and recommends CreateCoreWebView2EnvironmentWithOptions instead. This is a potential comparison boundary, not an established defect in the incident. [Primary API contract](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/webview2-idl?preserve-view=true&view=webview2-1.0.2420.47).

The native_webview2loader build tag selects the embedded Microsoft loader through go-winloader and also changes the environment callback from the default combridge wrapper to the legacy handler. A tag-only comparison therefore changes more than a single implementation detail and must record that confound. The native path also documents a fixed-folder/CGO limitation in its availability lookup. Neither path is changed or sampled while E002 is pending.

### Decision after E002: repair the demonstrated callback COM contract

The unchanged historical-ABI and corrected idle arms both passed120fresh launches, so the native BOOL correction is not established as the cause of #154. An independent unshown native-vtable probe then demonstrated that all9active Chromium callback objects claim successful QueryInterface without returning an interface, claim unsupported IAgileObject/IMarshal/arbitrary IIDs without clearing output, return success for a null output, and do not retain their Go implementation after a native AddRef. A forced-GC probe collected the actual implementation despite that native reference. These are independently demonstrated native contracts requiring correction; the active-window registry already roots Chromium, so the GC probe alone does not explain cold028.

Use the existing combridge (already used by the default environment loader) for those9callback interfaces only. Allocate callbacks after successful owned-window attachment in Embed, avoiding a registry-root cycle for NewChromium instances that never embed. Close the creator leases once after native controller Close and retained-interface release. Capture the Go environment completion target before scheduling the default loader, so a late completion can reject a destroyed owner without dereferencing a released inner callback. Guard late event delivery and close a late supplied controller instead of installing it. Do not change runtime selection or generated pkg/webview2.

The tests must validate canonical IUnknown identity via roundtrip, not assume every interface has the same pointer; reject unsupported interfaces; balance actual AddRef/Release; invoke native callback vtables; prove sole-native-reference retention followed by final-release collection; and cover canceled/late completion. Official Microsoft.Web.WebView2 SDK1.0.3537.50 header confirms all9IIDs (NuGet SHA2565ea526bbd728adda0da4d31219267e96460494a427e4894c4e09d9f320f4b9aa, WebView2.h SHA256387a0524c1a4498db32756a6d3b1f4d43a90ac80cb921527f15d3dbf4a857459). Source-contract correction is not an assertion of the browser failure's private cause.

### Active accelerator output ABI (2026-10-19)

The native AddAcceleratorKeyPressed signature requires EventRegistrationToken*, while the existing Go parameter already has that pointer type. Passing its address creates the wrong indirection and prevents native output reaching the caller. A guarded actual-vtable regression reproduces the mismatch before writing; a one-line argument correction passes, including unchanged failed HRESULT behavior. No causal attribution to #154 is made. The immutable E004 source excludes this later local correction. [Primary signature](https://learn.microsoft.com/en-us/microsoft-edge/webview2/reference/win32/icorewebview2controller?view=webview2-1.0.3537.50#add_acceleratorkeypressed).

The callback design follows Microsoft's [QueryInterface identity rules](https://learn.microsoft.com/en-us/windows/win32/com/rules-for-implementing-queryinterface) and [native reference ownership](https://learn.microsoft.com/en-us/windows/win32/com/managing-object-lifetimes-through-reference-counting). A nonmoving Go heap does not make a native uintptr a Go GC root; the native-only AddRef/GC regression establishes that separate defect.

### E004 diagnostic containment decision (2026-10-19)

The startup watchdog precedes a new native access violation in deferred WCT Close, so the latter is not an explanation for the original startup stall. SDK10.0.26100.0wct.h and Microsoft Open/Get/Close contracts match current pointer width, synchronous flags,7query arguments,16nodes and280-byte node layout. No same-thread contract or deterministic session-sharing violation was identified. Do not claim a thread-lock or pin-order tweak repairs this native fault.

The demonstrated product defect is containment and evidence ordering: startup_diagnostics.go collects optional waits before serializing already-collected primary counters. Emit primary progress first and isolate WCT in one hidden self-executable query process using the existing500ms context, bounded stdin/output and owned process capture. Validate only fixed requested roles/threads and sanitized nodes; native failure, timeout or invalid output produces an unavailable reason, never raw child traces. Actual WCT tests run inside that same isolation boundary. [WCT API](https://learn.microsoft.com/en-us/windows/win32/api/wct/nf-wct-getthreadwaitchain), [synchronous uncancellability](https://learn.microsoft.com/en-us/windows/win32/debug/wait-chain-traversal).

After E004 recurrence, the selected response observation extends the existing hooks with8fixed active-request slots, unique aggregate publication/completion/failure/overflow/retirement counts and an exact runtime:ready receipt. Existing first-response evidence stays intact; no URL/body, extra COM call/reference or unbounded identity map is elected. Later successful calls cannot erase an earlier failed response or make duplicate completion count as another request.

## Adjacent ABI sweep and active-path exclusion (2026-10-19)

A read-only sweep of edge Boolean arguments and event-token outputs confirms AddNavigationCompleted, AddPermissionRequested, AddProcessFailed, AddWebMessageReceived and AddWebResourceRequested already pass the caller token pointer directly. Wails frontend.go:565-599 uses ICoreWebViewSettings setters, whose active accelerator/pinch Boolean arguments are canonical values. The separate Settings3/Settings5 wrappers contain pointer-to-Boolean setters, and several Boolean getters allocate Go bool rather than Win32 BOOL; these are independent source defects to test/correct when their paths are elected. They are not executed by this initial Wails settings path, so they do not justify another speculative startup change. GetContainsFullScreenElement is event-driven and absent from the observed initial-navigation path. Keep this debt explicit rather than broadening the incident patch or claiming an unshown execution.

## Browser-side observation after E006 (2026-10-19)

E006 establishes successful observed host response/deferral completion for5requests but not browser consumption, script execution or actual DOM parsing. Wails's OnDomReady derives from NavigationCompleted. Its two blocking head scripts are injected before the original document, and served before product middleware; app script0 therefore does not mean no JavaScript resource requests. Native event markers alone would still leave fetch completion and early script exceptions ambiguous.

Select an explicitly elected disposable CDP observer, rather than a loader/runtime change. Minimal audited edge wrappers use the base vtable and existing COM bridge; generated pkg/webview2 is unsuitable because its JSON output pointer and completion-string ABI have independent defects. Register fixed Network/Runtime/Page receivers and require enable acknowledgements before diagnostic Navigate. This deliberately changes diagnostic scheduling and adds observation calls; keep it out of ordinary startup and final product qualification. Preserve the original watchdog so missing acknowledgements fail with explicit setup coverage. Bound all JSON/IDs/slots/events, retain only fixed asset classes/numeric statuses/lengths/MIME categories and exception/page counts. Capture the actual environment version once on its UI thread, separately from availability and capabilities.

A native-loader build comparison also changes the memory loader, option transport, comparator and outer completion adapter. The embedded DLL is verified official Microsoft.Web.WebView2 SDK1.0.992.28 (x64SHA256c4674acf95f0800793a4a6d61132adf5dfa694c218e482d86093494c4e84100a), but tag-only execution remains a multivariable comparison. The current131availability lookup is not an environment identity or an established runtime defect; no age-based upgrade/minimum-runtime restriction is elected.

### Concurrent native-host audit

No shared launch globals, process-name kill, global environment mutation, sibling-profile deletion or cold GUI runtime-endpoint join were found. Each ProcessTree job is per-instance, unnamed, assigned while suspended and then resumed. Detached CLI owners have a30-second idle policy checked on a one-second ticker and gated by connections/work, so describing an exact synchronized31-second teardown during GUI sampling would be inaccurate. The final31-second image cleanup runs only after GUI observation and the joined CLI future. E006's27passing cases each report zero remaining owned members before job close, with natural release0.032-1.094s; the failed nonzero-exit child's7members are failure-cleanup evidence. The new timeline measures actual worker lifetime and contract main-thread phases, not CPU/memory or every parallel worker.
