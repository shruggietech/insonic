# Native investigation reassessment

## 2026-10-19: owner-directed cost and hypothesis audit

S039 and #154 remain incomplete. PR #156 is open and unmerged. Both external review rounds have been used. No additional bot review is requested.

| Boundary | Established evidence | Limit |
| --- | --- | --- |
| Controller startup | Original main failed before native input; S039 recovered HRESULT 800700aa. | The internal WebView condition producing that HRESULT is not proved. |
| Owned process drain | A real zero-exit root previously killed a surviving child at job close; controlled regression failed before the drain correction and passed afterward. | This does not prove every browser failure has the same cause. |
| Owned WebView/COM teardown | Missing Close/reference/apartment balancing has focused native regressions; an archive also retained three live browser processes after Close. | Actual browser release and all package lanes still require final verification. |
| Native menu invocation | Earlier actual Alt+F did not open File; the pinned framework drops system-key context. The original accelerator also reset a handled key to unhandled. | Correcting those demonstrated defects did not eliminate the intermittent menu stall. |
| Latest menu stall | bfaf303 diagnostic 39284836144: two cold-workspace menu stalls, 24 successful natural releases and one complete 12-case parent. Both failures had the native key route, visible owned popup, Exit flags 128, physical Alt released and accepted X injection. Neither reached menu-request. | SendInput acceptance and highlighting do not establish native menu consumption. Warm-profile-only and missing-native-route explanations are insufficient. |
| GitHub producer status | Completed producer logs/artifacts coexist with queued API snapshots; bfaf303 archive assembly reached its unchanged source-status deadline before native qualification. | This is a separate CI authority failure. Logs/artifacts cannot replace required successful statuses. |

The audit covered input senders, foreground guards, framework accelerator forwarding, main and modal dispatch, startup/frontend readiness, owned profiles, teardown, and source/package workflow authority. Repository input injection is confined to the disposable Exit harness; browser automation elsewhere uses its own browser API. No evidence currently establishes interference from another repository SendInput sender.

### Corrected inference

The old failure profile uses runtime.GoroutineProfile, whose public StackRecord stores 32 PCs, and limits emitted names to 32. The outer caller is missing. A DefWindowProc wait at the top can coexist with an enclosing COM call or callback. The retained profile cannot exclude that case. Earlier exclusion of COM reentrancy was too strong. Two default-window-procedure frames also do not establish duplicate menu requests, because normal system-character handling can nest through a system-command message.

A controlled 48-frame call regression fails against the extracted old capture (outer marker absent) and passes with runtime/pprof's deeper goroutine capture. The replacement runs only after the existing watchdog has already marked failure. Its raw text remains in bounded memory, and only parsed function names enter the receipt. Address, path, label and argument-like text rejection and memory bounds have dedicated checks. The output identifies its depth/budget limit explicitly.

References: [Go runtime profile contracts](https://pkg.go.dev/runtime/pprof) and [WebView2 threading/reentrancy](https://learn.microsoft.com/en-us/microsoft-edge/webview2/concepts/threading-model).

### Next decision and stopping rule

One focused hosted diagnostic on the new capture should distinguish a normal main-loop menu wait from a menu nested within a WebView callback/COM call. Inspect the complete retained caller chain before another behavioral patch. If no failure is captured, passing repetitions do not establish a cause or justify another identical run. Do not repeat full package CI merely to diagnose this wait while producer-status authority is stale. Keep actual input/callback/cleanup proofs, original timeouts and all final gates unchanged.

### Deep capture result and input counterfactual

Focused diagnostic 39287503108 (artifact 12205045823) at f0a6263 completed nine 12-case parents and 109 natural-release children, then failed on warm no-workspace menu. The failure profile has 65 function frames and limited=false. Its outer caller is Wails RunMainLoop/DispatchMessage, with no enclosing Go WebView callback or COM method. This rejects that specific callback-nesting hypothesis for the captured failure. It does not expose native browser stacks. Playback/workspace loading is not necessary for this signature.

Resume T046 with a bounded, temporary same-owned-thread GETMESSAGE/MSGFILTER observer, recording only qualification key metadata in memory and emitting only after failure. Its real unshown-window regression checks unchanged message retrieval and observer removal. After an already failed watchdog, a joined UI operation may release focus only when it is an actual child of the owned fixture. Existing Escape/Quit cleanup follows; the timeout remains failure. If this releases the menu, distinguish focus loss canceling the popup from subsequent key consumption using the post-release menu state and recorded keys. If no failure is captured, do not treat the observed passing runs as proof or repeat an identical diagnostic. Remove temporary observers before final qualification.
