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
