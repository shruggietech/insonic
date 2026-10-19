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

### Captured input-ownership boundary and repair

Run 39290333232 at 4dbd8a1 (artifact 12204874232) completed five parents and 60 naturally released children, then failed on the first cold no-workspace menu. The original X never reached the native GETMESSAGE or menu MSGFILTER observer (no dropped records). At the failed watchdog the owned UI queue still executed the joined marker. Releasing the actual browser-child focus changed subsequent Escape delivery: both key transitions reached the native queue and menu filter. No shutdown completed, so clearing focus alone is not a repair and this remains a failed qualification. The complete UI profile again ends in normal main dispatch.

The parent window must hold keyboard focus while its native menu consumes input. Its normal WM_SETFOCUS forwarding otherwise immediately sends focus back to the browser. A real unshown native-subclass regression failed against the old forwarding boundary. The maintained ownership subclass now saves the prior focus on WM_ENTERMENULOOP, sets native host focus, suppresses browser refocus while that menu is active, and restores the previous child on WM_EXITMENULOOP only while the same parent owns foreground input. Native destruction also suppresses refocus into a closing controller. Nested menu depth keeps ordinary forwarding disabled until final exit. The same regression and selected complete SDK test pass, including three race-detection repetitions.

Temporary input observers, their flag and the failed-focus counterfactual are removed. Actual Windows qualification now cancels a native menu, verifies browser-child focus restoration, reopens through the original native key route, requires host focus while the popup is active, then injects real X and retains all callback/shutdown/job/runtime assertions. All original deadlines remain. Hosted behavior is still pending; this does not establish the private cause of every historical controller HRESULT.

Primary input contracts: [SetFocus](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setfocus), [WM_EXITMENULOOP](https://learn.microsoft.com/en-us/windows/win32/menurc/wm-exitmenuloop), [menu MessageProc](https://learn.microsoft.com/en-us/windows/win32/winmsg/messageproc).

## 2026-10-19: final owner-capped diagnostic and deferral reference correction

The owner allowed at most one further targeted diagnostic run. Run 39313195708 at fea2349e43c8583d842b61b3488d8b357e286676 used the existing current-source cold-close sampler with startup-hide and the maintained CLI workload enabled. It stopped on child 28 after 28 natural-release successes. The failed child exited 1 in 13.469 seconds after the original 12-second watchdog, with complete untruncated streams and seven owned members present before failed cleanup. No native close input occurred. Artifact 12215768901 has SHA256 ba0710e41c9e3a245bc854102b7db0428131d4db04b421a1e40882306f561e7b. This exhausts the diagnostic allowance; no further diagnostic dispatch or rerun is elected.

The failure counters show one accepted navigation, one entered/returned resource callback and one entered/returned document handler, with zero script requests and zero navigation completions. No off-thread callback was counted. The binding reports available runtime version 131.0.2903.86. The complete Go UI caller chain is idle GetMessage. Native wait observations return one host-thread node (status 3) and one browser-thread node (status 2), with no reported wait-chain cycle. These two thread observations do not inspect renderer/GPU internals or prove browser health.

The middleware counter ends before the framework's HTML injection and native response/deferral finishing. A returned document handler therefore does not prove the final response bytes were consumed or the deferral completed. The captured signature narrows the remaining response/navigation boundary; it does not establish a cause. The earlier two-second UI-join signature remains separate and was not captured in this run. The concurrently elected CLI contract workload completed successfully, including all 168 help examples; this does not establish load as causal.

Read-only inspection of that response path found a deterministic independent ownership defect: ICoreWebView2Deferral.Release dispatches to AddRef. The selected unshown native COM regression fails before correction with refs=4, add=3, release=0, complete=1. Dispatching Release to the actual Release entry balances references while preserving Complete and error behavior. The full selected ownership regression passes after correction. This does not establish that the leak caused the captured pre-DOM stall.

The maintained source allowlist now names exactly the three implicated upstream files (the deferral, chromium and corewebview2), retains all 397 upstream hashes and the same four additions, and binds the updated checksums and explicit notice. Required local checks and one ordinary complete CI validate this concrete SDK correction; they are not additional diagnostic sampling and cannot clear T053 merely by passing.

## 2026-10-19: owner-approved completion boundary

The owner explicitly directed S039 to finish with its proven fixes while #154 remains open for the unexplained startup failure, with no more diagnostics and PR #156 left unmerged for final review. This supersedes the earlier cause-repair completion condition. T053 is explicitly deferred outside the accepted slice; it is not marked implemented. The original eleven issue closures and proven native process/controller/COM/input/deferral corrections remain delivered.

The last product implementation head 8cb8dc0d6d2793cff074c01be08985ccfa2586ba passed CI 39315026478 attempt 1 in 649 seconds (43 jobs: 25 success, 18 conditional skips), including all four Windows scopes, exact-byte join and candidate. Its source/merge is 07bc391dd827d97075988edf6bff2783f15c85b1. Local root check/145 Node tests, 393 Python tests, affected production desktop tests/vet, selected unshown native normal/race checks, site check/17 tests/build and compiled-but-unexecuted 50-module notices pass. The closeout changes only internal planning; the configured CI must validate its final commit before handoff, with live evidence in PR #156.

The existing review thread is replied to and resolved; two external rounds are used and no third is requested. #154 stays open outside the S039 completion milestone. No cause repair is claimed for the retained pre-DOM or two-second UI-join signatures. No merge, release, site publication, owner installation or additional diagnostic is performed.
