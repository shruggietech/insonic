# S040 verification

## 2026-10-19: source and ABI baseline

Housekeeping: verified PR #156 squash merge c9ed48d, identical tree to 8d2f1c7; fast-forwarded main, deleted verified merged local S038/S039 branches, verified 11 closed issue/project Done states, closed milestone23 with zero open items. #154 remains open, assigned to milestone24/S040 In progress.

The native production-wrapper regression failed against v1.0.22 for both false and true: one native call each returned a noncanonical BOOL, with no wrapper error. The HRESULT propagation regression passed. Adopting authoritative v1.0.23 makes both exact ABI cases and the same HRESULT test pass. No browser or application window was created.

Go module authority: v1.0.23 tag e08e9397f04ad0b1050d7e5bdc8e27faddc87987, upstream sum h1:jmv8qhz1lHibCc79bMM/a/FqOnnzOGEisLav+a0b9P0=. All397 file hashes reconstruct that exact sum. The only upstream differences from v1.0.22 are the two Controller3 Boolean wrappers. The original maintained three-file modifications/four additions are preserved before further observation work.

The initially attempted import of unused pkg/webview2 fails at package initialization because its pre-existing callback signature exceeds uintptr. Wails imports pkg/edge and does not execute that generated API. The executable regression therefore tests the actual production edge wrapper; the generated counterpart adopts the checksum-verified upstream source but has no native execution claim. No unrelated generated API rewrite is elected.

An initial local lifetime-test command used the nested module filesystem path, which Go correctly rejected; the selected test is rerun through its actual module import path. This is a command-selection failure, not a product test failure.

Native incident causality and S040 final qualification remain unproved. E002 dispatch and its result are recorded in experiments.md.

The selected existing unshown native lifetime suite passes through its actual module import path, including Close/reference/apartment, menu focus, claimed accelerator, bootstrap observation and deferral release cases.

## 2026-10-19: original release-budget timing

T013 native helper regressions failed without timing evidence and passed after the additive report. One fixed2.5-second budget covers root plus descendants. Measured failed fast/slow roots left2.4241058s/1.6212023s and waited those remaining budgets; both retained live owned children and stayed failed. Natural counterparts, no-deadline compatibility and prior natural/cleanup/held-handle regressions pass (focused selection9.771s). Timing observes exec.Cmd.Wait completion including pipe joins, not precise kernel exit; it does not change deadlines or prove a WebView cause. Python targeted-capture equivalents preserve the original18-second budget for5s/17s synthetic root durations. All24 selected startup diagnostic tests pass.

The upstream ABI native regression also passes three race-detection repetitions.

## 2026-10-19: existing post-merge pipeline baseline

Read-only audit of already completed main CI39325568611 on c9ed48d:43jobs,20success,8failure,15skip. Windows assembly failed its same-run source-artifact deadline before native qualification; all four Windows lanes failed waiting for assembly. Media AV1 artifact upload and adapter fixtures also failed. This is not a post-merge native startup reproduction and does not establish a common browser cause. No retry was dispatched. The targeted source experiment remains independent of these package handoff failures.

An intermediate npm test invocation was made before SDK observation edits and source-manifest refresh finished; its stale-source rejection is expected and will be rerun after source settles. Copied module-cache read-only attributes on the two adopted source files were normalized to writable source. Temporary symlink permissions also require the authorized unsandboxed test path; no integrity tests are weakened.

## 2026-10-19: final native publication observation

T009/T010 add exact first-native-resource association and fixed entered/returned/HRESULT/thread counters for assignment, headers, status/reason, content publication and deferral completion. No WebView2 COM call is added/reordered and no address/body/URL is emitted. Borrowed identities retire after their acquisition/Release; owner state disappears on native destruction. Actual-environment version is not queried; the legacy runtime_version field remains explicitly an availability lookup.

Three actual native-vtable tests verify enabled/disabled behavior, operation ordering/count, all seven failed HRESULT boundaries, foreign-thread observation without rerouting, borrowed-identity retirement, reentrant snapshot access and real SHCreateMemStream ownership after buffer mutation and GC. Selected normal tests plus prior unshown ownership suite pass0.042s; the same selection with race count3 passes1.091s. No browser/controller/window is created by the three new tests. Maintained upstream scope is now exactly five modified originals and the same four additions, with all397 upstream hashes preserved against v1.0.23 and patched sum h1:JaxxLqAJMJO/4zi5VgV41qyJquTZeLfJONk87N9lDas=.

Current local checks: root check and145Node tests pass; full394Python tests pass before four additional paired-control cases, and all28 targeted startup-diagnostic tests pass afterward. Production-tagged desktop, host, process and ABI packages pass, with vet clean. Site17tests and build pass; a subsequent export check validates111pages. Initial sandbox-only workspace/directory creation errors disappear on the authorized unsandboxed path; no product logic changed for them.

The observer reads native thread identity; it adds no WebView2 COM calls. Both paired arms use the same observers.

## 2026-10-19: complete Python regression after paired-control additions

The complete Python discovery suite passes398tests in41.5seconds through the verified hidden launcher. No local GUI or native browser is launched. E002 remains the only hosted diagnostic dispatched.

## 2026-10-19: actual callback COM regressions before correction

An ignored test overlay calls the actual9NewChromium callback vtables at31f3959 with official own-interface, IUnknown, IAgileObject, IMarshal and unsupported IIDs. All9claim S_OK without supplying supported interfaces, fail to reject/null unsupported outputs, and claim success for a null output. Actual native AddRef/AddRef/Release/Release returns1/1/1/1. A separate forced-GC probe drops Go owner cycles after actual native AddRef and observes the controller callback implementation finalize, demonstrating absent native retention. Expected-failure test command exits1 in0.797seconds; test source SHA256d25e38d9e4236d7556415aa95d837be637b5e1f72824aa2111f722db415424fe. No window/browser is created and no tracked source is changed by that baseline probe.

The final regression will allow COM's separate canonical IUnknown pointer and test its roundtrip identity, then prove sole-native-reference survival followed by final-release collection. The initial probe's S_OK-with-null failure remains invalid regardless of that identity refinement. The active-window lifetime registry already roots Chromium, so this experiment does not by itself explain cold028.

## 2026-10-19: E002 and E003 native outcome

E00239329500568 retained120historical-ABI and120corrected idle passing fresh close cases; it provides no incident-cause differential. E00339330500115 reproduced watchdog failure on child029 at the identical corrected GUI hash, with all7first-response operations successful on the owning thread. Script middleware ran,6native resource callbacks returned, and navigation completion remained0. Later response completion/browser consumption/event delivery remain unresolved. Full immutable artifact/source/counter identities and comparison limits are in experiments.md. No retry was dispatched. The ABI fix is independently valid and demonstrably insufficient as the complete #154 repair.

## 2026-10-19: callback correction and loader compatibility

The9active callbacks now use the existing native-allocated combridge with real native references, supported-interface queries and canonical IUnknown roundtrips. Creation occurs after owned-window attachment, creator leases close once after native Close/interface cleanup, default-loader completion captures its Go target, and late callbacks cannot notify application code or install a controller after shutdown.

Final selected native suite passes3repetitions in all4configurations: default normal0.906s/race3.203s; native-loader normal0.719s/race2.656s. It covers all9actual callback identities, failed queries/null outputs, native-only retention after every creator lease closes, final-release collection, never-embedded collection, partial/repeated cleanup, live native dispatch and canceled/late completions. Existing unshown window/response tests are included. Initial test-fixture/channel-declaration mistakes were corrected and both configurations rerun; no product workaround was added.

Exact source scope is now7modified upstream originals/7additions,404maintained files, with397original hashes verified against v1.0.23. Maintained sum h1:Wm8ws8VPkT1XHXhf5T8qk0gsrJ3XF+QPeY3++ew840g=. Root check and145Node tests, full398Python tests, affected production Go tests/vet, site17tests/check/build all pass after this refresh. No local browser/application GUI is executed. Callback-native regressions establish contracts, not the private cause of E003.

The corrected GUI was compiled locally without execution and all50actual linked-module notice inventories verified against the refreshed source authority. The binary records v1.0.23 with the exact repository-local replacement; no checksum is falsely attributed to that replacement. The rebuilt site export verifies111pages.

## 2026-10-19: active accelerator registration output

E005 reproduces the original output-address mismatch without attempting a native write through the wrong pointer. The actual-vtable test then passes after exactly one product line changes &token to token, including fixed64-bit caller-visible output, receiver/active bridge identity and failed HRESULT propagation with no changed failure output. The entire native callback selection passes3times in default normal0.843s/race3.079s and native-loader normal0.859s/race3.219s.

Authority is now8modified originals/7additions,404maintained files against397official upstream hashes, maintained sum h1:/VJFPw1ywAs2zJxIfRvRCJUxMVvU+ZiWlpg7TkyiBY4=. Required root check/145Node tests, full398Python tests, affected production Go tests/vet, site17tests/check/build/111-page export and50actual linked-module notices pass after the final argument correction. The rebuilt GUI was compiled without execution. E004 remains at its immutable earlier9a57e90 source and does not qualify this later correction.

## 2026-10-19: E004 recurring startup and separate diagnostic crash

E00439332330692 at9a57e90 fails child027 after27passes, reaching the original watchdog before DOM readiness. The wait-chain collector then faults inside CloseThreadWaitChainSession, exits2 and loses the already-collected primary startup snapshot because the logger adds waits before serialization. These are sequential failures, not a new explanation of the earlier stall. Exact first/later response state is unavailable. Corrected callback contracts are valid but insufficient; the later token-output fix is not part of E004. Artifact/run/byte identities and next-boundary decision remain in experiments.md.

## 2026-10-19: primary evidence containment and all-response boundary

The primary-order regression fails under the old combined logger, then passes when progress JSON is emitted before optional wait observation. WCT now runs only in one hidden self-helper with500ms context/8KiB capture, strict fixed-target JSON and numeric failure metadata. Native-style0xc0000005 termination, timeout (0.51s), malformed/truncated/private/wrong-target/excess-node responses are contained; actual WCT executes only in a hidden test child. The focused suite passes1.194s. An initial agent full-host run fails3existing workspace.Init fixture prerequisites; the full affected production suite passes with the repository native environment and owned temporary root. No thread-affinity, pin-order or private WCT cause is claimed.

Eight fixed active-response slots now preserve unique observed publication/completion, sticky native failures, explicit overflow and acquired-deferral retirement. First-response detail remains, and exact runtime:ready receives only a fixed count while existing forwarding/echo remains unchanged. Headers means GetHeaders acquisition, not AppendHeader writes; Published attests the observed setters only. Tests cover six interleaved/out-of-order responses, duplicates, later failures, overflow/reuse, synchronous no-deferral handling, owning-thread/disabled/stream/privacy contracts and actual message forwarding. The final combined unshown native selection passes3repetitions in default normal0.054s/race1.096s and native-loader normal0.056s/race1.101s (launcher0.922s/3.266s/0.937s/3.36s).

Exact maintained source remains8modified originals/7additions,404files, with397official v1.0.23 hashes unchanged; maintained sum h1:eIa/vUbVD3JkciCJbnDxigl0flivs0IvI0gERjFowdQ=. Root check/145Node tests, full398Python tests, affected production Go tests/vet, site17tests/check/build and all50actual linked-module notices pass. The GUI compiles without execution. E006 is prewritten to distinguish all-response completion, runtime-ready and native navigation completion; it is diagnostic evidence, not final package qualification.

## 2026-10-19: E006 browser boundary

E00639334452671 atafcbee90 fails child027 after27passes. All5intercepted responses successfully finish the seven observed native operations and acquired deferrals, on the owning thread; all5retire with active/overflow/failures/incomplete0. Runtime-ready0, navigation-completed0, product app-script0. The five-resource set is consistent with document,2Wails blocking scripts and2CSS responses, but individual identities are not yet observed. No process/header/Finish error is logged. Primary evidence survives; optional isolated waits return numericmetadata normally. Native complete/retire does not prove browser consumed each stream. The app dom-ready marker comes from Wails NavigationCompleted, not Chromium DOMContentLoaded. A separate opt-in browser fetch/script/page observer plus actual environment version is selected before any loader/runtime change. Repeated failing ordinal27/29 under elected load also warrants actual worker/phase intervals instead of assumed overlap. Full result/identities are in experiments.md.

## 2026-10-19: concurrent-work timing and issue reconciliation

The timeline regressions fail before implementation and pass afterward: two focused Python modules38tests in1.584s, including actual worker entry/exit, pending/finished/failed overlap, bounded phases/cases, export before idle cleanup and native-failure precedence. Root's additional observer-election rejection test passes with the updated startup module34tests in1.709s. Workload arguments/commands/deadlines remain unchanged;11authored main-thread phase callbacks do not measure individual parallel workers or resource utilization.

Issue #154 now has a current-status section and broader startup/release title while preserving its original report. Only the2already-delivered stream/diagnostic acceptance criteria are checked; cause repair, final packaged matrix and exact-head/post-merge CI remain outstanding. E006 results and all counter/byte limits are retained rather than converted to a passing retry.

## 2026-10-19: elected browser observer validation

The native fake-vtable suite verifies actual environment string ownership/freeing and bounded version-only output, all7event receivers,4acknowledged commands, callback identity/native ownership, cleanup/late delivery, disabled zero-extra-call behavior, fixed resource classes/privacy, loader/frame association, finite event/request/JSON limits, exact integral numeric forms, sticky HTTP status failures and decoded/encoded byte distinctions. Agent final normal/race tests pass0.050s/1.084s.

Root's optional combined -count=3 invocation exceeds Go's permanent Windows callback-trampoline capacity in repeated fake-vtable fixtures (runtime: too many callback functions). This is retained as a test-invocation failure, not a WebView startup occurrence. One complete suite per fresh process preserves every selected case and passes default normal/race and native-loader normal/race. The product suite and each diagnostic child create different process lifetimes; no product callback-limit repair is claimed.

After source reconciliation, npm run check/145Node tests, full406Python tests, affected production Go tests/vet and all50actual linked-module notices pass. The GUI compiles without execution; no local browser/input occurs. The site remains byte-unchanged from its previously passing17tests/check/build/111-page export. Maintained source is8modified originals/9additions,406files, upstream397hashes unchanged; sum h1:ELA08j/sI5oMpkNpU4dxSi7y5TMJk0RPICYkMe8UWv0=.

## E007 browser/overlap result (2026-10-20 UTC)

Single owner-authorized run39345395087/attempt1/job118376652446 at92a1bab7a207cac305b1965ea878a2d8fa74dacc failscold-024 after24passes under originaldeadlines. Exactbinary/artifact identities and boundedbrowser/workload findings are retained in experiments.md. Browser observation valid in all25cases; all24passingreferences have6finishedassets and bothassociatedpage milestones plus naturalownedrelease. Failedcase has all6hostComplete paths and fullbrowserbodydata, but runtime/app-script unfinished and readiness0. Runtimeactual131.0.2903.86. OptionalWCT observationdeadline is contained. No ordinarynative/packagequalification or repairedincident claim.

## 2026-10-20 UTC: native stream and CSS control checks

One ignored, hidden, unshown native probe passes 36 size/read combinations on local Windows 11 build 26300, including the exact E007 sizes and the 16 KiB boundary. Actual MAINSTA-to-MTA COM marshal/unmarshal succeeds, IAgileObject/IMarshal are supported and the free-threaded marshaler is observed. Creator release precedes worker reading; original-buffer mutation does not change copied content. All cases reach S_FALSE/0 EOF with no E_PENDING, native failure or timeout; native work takes 0.094s. An initial assertion incorrectly required S_FALSE on a short nonzero read; its receipt is retained as a probe expectation correction, not a product failure. Final receipt: build/s040-native-memstream-probe.json. This does not qualify Server 2022 or the private WebView consumer.

E008's explicit CSS control preserves the exact 6,784-byte CSS prefix and unchanged 402,310-byte app script, appends only an inert comment to equalize size, and restores generated CSS after either success or compilation failure. Its dedicated regressions cover unchanged assets, exact padding/receipts/restoration and incompatible election before acquisition. Root npm run check and all 145 Node tests pass. The complete Python suite passes 409 tests in 37.367s (launcher 37.968s). The padded GUI compiles without execution; all 50 linked-module notices verify, and exact generated CSS restoration is confirmed. SDK/Go/product/site source bytes are unchanged from the prior checks; the CSS mutation exists only during the explicit disposable compile. No production feature flag, deadline or loader/runtime change is included.

Independent pre-dispatch review identified that the new control accepted startup_hide=false despite the written single-variable experiment. The new election regression fails at that field before correction. Requiring hidden startup resolves it; all37affected tests then pass in2.187s. No native/product behavior changes. E008 local build/notice and asset-restoration results remain applicable because the correction only rejects an incompatible election before acquisition.

## E008 result and E009 preparation (2026-10-20 UTC)

E00839347305532/attempt1/job118382906752 at d9ccbbac39565281e68f8e800916e6d0a21e7dea failscold-023after23passes. All6assets, including the402310B CSScontrol and unchangedappscript, report loadingFinished; runtime-ready1, ChromiumDOM/load and nativeNavigationCompleted0. This differs from E007's unfinishedscriptboundary and does not establish a sharedcause or paddingrepair. Exactidentities and overlap are retained in experiments.md and publicissuecomment6280447731. Neither final nativequalification nor an incidentrepair is claimed.

E009 retains E008assets and changes only the sampling schedule to onefreshGUI at the actualmedia-lifecycle callback, with two fixed systemresourceendpoints and one owned-job snapshot. Independentreview found two measurement/election defects. The injected pre-snapshotdelay regression and expiredgate regression both fail before correction; all45affectedtests pass afterward in1.839s. New native-resource tests cover actual Windows API/jobqueries with an unshown noninteractive Pythonchild, x64ABI, ownership recheck,16memberbound, unavailable semantics, CPUidle subtraction and redaction. No production SDK/Go/frontend/site bytes change. Endpointtiming does not establish peaks or threadstarvation. Finalreview confirms no remainingblockingfinding and preserves those limits.

An initial sandboxed localcheck attempt cannot exercise temporarysource-aliascleanup and Windowsinstallation ownership fixtures, and cannot download isolatedGo dependencies. These are retained as localenvironment failures, not WebView occurrences. The established hidden launcher is rerun with the required hostpermissions; no desktopGUI or ownerinput is used. Root npmcheck and145Node tests pass after that permission correction. FullPython result follows before dispatch.

The complete Python suite passes424tests in38.614s (launcher39.141s). Rootcheck and145Node tests pass; all45affected startup regressions pass. UTF-8withoutBOM/LF/mojibake checks cover all9edited/new files and gitdiffcheck passes. SDK/Go/frontend/site bytes remain unchanged, so their previously passing checks remain applicable to those bytes. No E009 dispatch precedes these results.

## E009 first-child failure (2026-10-20 UTC)

Run 39349661564 / attempt 1 / job 118390600839 fails the only fresh child at the original outer 18-second deadline. No prior GUI launches; gate and case timings confirm actual media-work overlap. System CPU averages 99.785% busy, while the owned GUI job accumulates 1.109375 CPU seconds and endpoint available memory remains above 12 GB. Browser and child-watchdog progress are unavailable, so no E007/E008 browser state is inferred. Immediate cleanup precedes process metrics, leaving no surviving member; process priority is unavailable. Exact control/binary/artifact identities and interpretation limits are in experiments.md. #154 remains open and its current status is refreshed. No extra diagnostic has been selected or dispatched.

## E010 observation validation (2026-10-20 UTC)

The existing full root check and all 145 Node tests pass for the new election/observer checkpoint. All 433 Python tests pass after final anchor sequencing in 36.789s (launcher 37.391s). The focused startup module passes 49 tests after correction in 1.979s. A deliberately blocked live native query cannot postpone an actual 0.1-second timeout or owned-job termination; failed exit23 and unexpected wait exceptions explicitly terminate before original-handle close, preserving their original failure. Resource tests cover actual duplicated ownership, root/member inclusion, cancellation, bounded queries, unavailable duplication, redaction and integral clock anchors. The installed Python can have an owned launcher child; an initial one-process-only fixture assumption was corrected without changing product behavior.

The added child clock is disabled without clock/output work. Timer creation is bracketed by two reads without output between them, using the unchanged 12-second timer, and optional output follows arming. Fixed child UTC-microsecond origins align through bracketed parent UTC/monotonic anchors under the recorded coherence/no-wall-step assumption; unrelated monotonic origins are never directly subtracted. Independent review found the parent anchor initially ran after the native budget reset. An injected 1-second snapshot plus 2-second anchor, 17-second root and 0.5-second natural release fails before moving every prelaunch observation ahead of the reset; the same regression passes afterward. Missing/crossing live snapshots and unavailable clock alignment remain explicit.

Affected production Go tests and vet pass in 18.969s and 0.688s after fixing a stale test invocation from the stage-observer refactor. The two new observer tests also pass under the race detector in 1.099s, without a GUI. SDK, module dependencies, original frontend assets and site bytes are unchanged. Prior source-authority, linked-module-notice and site results remain applicable to those bytes. No production priority, browser flag, loader, runtime or native deadline change is made. E010 remains diagnostic-only and has not yet been dispatched.

## E010 result (2026-10-20 UTC)

The only fresh child in run 39351706900 / attempt 1 / job 118397162060 fails the original outer deadline at 18.172s. Native navigation has not started: the diagnostic observer is registered but still awaiting Network.enable, with no domain enable acknowledgement or resource event. Actual runtime is 131.0.2903.86. This earlier observer boundary is retained separately from E007/E008. OnStartup arrives at child 5.710195s; the timer fires 12.003474s after arming, so no watchdog-starvation claim follows. Valid live collection completes before cleanup and finds 7 BelowNormal, 1 Idle and 1 AboveNormal process classes among 9 verified members, without role mapping. GUI-job CPU is 1.15625s at live collection and 1.1875s after cleanup, while system CPU averages 99.914% busy and memory retains substantial headroom. Exact records and identities are in experiments.md and [issue comment6281028311](https://github.com/shruggietech/insonic/issues/154#issuecomment-6281028311). No qualification or repaired-incident claim.

## E011 local control verification (2026-10-20 UTC)

Root check and all 145 Node tests pass. The complete Python suite passes 438 tests in 36.594s (launcher 37.047s). This includes actual hidden owned Windows inheritance: a BelowNormal parent produces a BelowNormal child through the original wrapper and a Normal child through the explicit control, with natural job release. No GUI or owner-process priority changes. Wrong-parent refusal precedes any GUI creation, invalid actual class cannot confer success, unavailable errors retain no private text, original nonzero child exits survive, and incompatible elections fail before artifact acquisition. The existing launch test module's scripts import path is now explicit for independent invocation. Go/SDK/dependencies/frontend/site bytes remain unchanged from E010, so their existing passing checks remain applicable.

The final independent review's unavailable-control classification finding is corrected with a failing-before/passing-after 12-case regression. Exactly one effective complete receipt is required; absent/duplicate/malformed/incorrect-class/PID/timing and incomplete capture remain explicit control-unavailable rather than an incident recurrence or success. Original process statuses survive. Final root check and145Node tests pass; all439Python tests pass in36.701s (launcher37.187s), including51affected startup tests. Independent final review reports no blocker. No E011 dispatch precedes this checkpoint.

## E011 result (2026-10-20 UTC)

The onlyfreshGUI in39353609715/attempt1/job118403277238 fails the original18-second limit despite a validNormalhostcontrol beforestartup andat15s. It advances throughCDPsetup/navigation, thenretains5completedhostresponses, fullIPC/runtimebodydata without loadingFinished, appscriptrequestedwithout hostentry, readiness/DOM/load0. Thewatchdogfires12.000167s afterarming. Exactidentities, bounds, timing/resourceinterpretation and [publicresult](https://github.com/shruggietech/insonic/issues/154#issuecomment-6281265089) are retained in experiments.md. This rejects an initialhostNormal change as a sufficientrepair; it does notresolve waitingversusrunnable scheduling. No productpriority change, qualification or incidentrepair is claimed.

### E018 console-only overlap witness (2026-10-20 UTC)

Exactsource7fa392557ae8d4138a4263f8a8551c88efb63542, run39362252300/attempt1/job118430877769, failed strict decoder in1m03s. Zero product build/GUI/CLI workload. Trace2614CSwitch/1383ReadyThread, zero loss, valid ownership/range/clock. PID7452/TID1932 scheduled[362484300,378039500)ns conflicts with ready[378039300,378042300)ns by200ns. Scheduler interpretation remains unavailable. Cleanup/raw-directory removal confirmed. Artifact12237788151,1205B,SHA25622b02365c6c46c4f690dbe80c49c157ab5b10cf53b52e57dbbfdec0fdc53d1b8. This is an offline-testable tool case, not product-cause evidence. No next run selected.


### S040 E019 result: console tool passes with explicit unknown time

[Run39362821471](https://github.com/shruggietech/insonic/actions/runs/39362821471), attempt1/job118432700679, exactsource649a90aded9785d720db86bc1e8ce8796cd5149c, passes in44s. Zero product build/GUI/CLI workload. Owned console PID2864/mainTID6384 meets unchanged known wait/work thresholds: observed522869300ns, waiting348271700ns(max200053200), scheduled174162300ns(max15745100), ready434300ns(max37000), unknown1000ns. Cleanup/raw-directory removal confirmed. Artifact12238271258,711B,SHA2569aac7ca3630af11c72d413ea4ce52d338df5850a95ea24a7d71cd48e47925330.

The compact success receipt retained the main summary but omitted its conflict counters/witness. Therefore the1000ns unknown cannot be attributed specifically to the overlap branch. The exact local E018 regression exercises that arithmetic; this integration pass validates only the real console known lower bounds. Preserve those existing fixed decoder fields in future compact receipts with an offline regression; no repeated console dispatch is needed for that receipt-only correction. #154 remains open; no next native run selected by this result alone.


### S040 E020 result: native failure retained; scheduling interpretation limited by a newly found observer error

[Run39363334310](https://github.com/shruggietech/insonic/actions/runs/39363334310), attempt1/job118434328306, exactsource644f141f04753dc5a43bf3cdc3a9c10600ae6d2a, fails in6m11s. Its plain console preflight passes with zero unknown, no conflicts and known wait/work. Only fresh GUI fails at18.219s (exit1/timeout, complete/untruncated). Normal control valid:GUI7464/class32, wrapper7288/class16384. OnStartup5.9267665s, arm5.9267665s, prepare6.0445127s; child origin is326112us after the parent clock anchor, so the outer deadline precedes a full post-controller12s watchdog interval. No browser/watchdog snapshot arrives. The kind4 log maps to UTILITY_PROCESS_EXITED, not renderer-unresponsive; it has no timestamp and cannot be separated from forced cleanup.

The15s live query finishes before cleanup (47.125 versus50.281 shared seconds), directly verifies all9 selected members, and independently identifies renderer5108 at Idle64, GUI7464/32, browser7152/32, GPU1288/32768, utilities2116 and6728/32, crashpad5144/32, wrapper7288 and other-owned6312/16384. Job CPU1.46875s live/1.484375s final; system interval99.871%busy, available physical>12.6GB. Full case32.094-50.328 overlaps media32.094-92.687; CLI passes146.578s, dropped phase events0.

Trace902064CSwitch/516404ReadyThread, zero loss and valid clock/range/native lifetimes. Conservative overlap accounting completes and removes disputed portions. However, a further audit finds that closed-interval projection unconditionally used ReadyDuration/WaitingDuration without checking provenance. The pinned library can derive initial ready from thread birth or prior waiting from incomplete state anchors. Those quantities are not established known lower bounds. RendererTID1388's reported5.4116983s ready(max5.356703)/6.4748692s waiting and utility6728/TID6996's reported12.4889671s ready therefore remain candidate categories, not proof of scheduling delay. The earlier chat statement about multi-second ready intervals is qualified by this observer error. Aggregates cannot be corrected offline because raw ETL is deliberately deleted. Independently valid process classes, native timeout and cleanup remain evidence; no cause is assigned.

Cleanup/raw-directory removal confirmed. GUI5b3bfa469b3c2bd64ed7c5dbbcde41413e5aefede8f56fbb10333e7d1d61ab7d, CLIbf1585f303afc8a541ce41cd85e7775e060d91ef8b6f2f9371e44cc7e5d5806e, exact CSS/script controls match. Artifact12238835917,17190B,SHA256a43e0d41de5d93c54f794b049cfeaa58e62dcdae8f4078c74017391e86975c35. Next: locally verify closed-interval provenance against actual pinned APIs and source, keeping unsupported state time unknown before selecting any further run. #154 remains open; no next dispatch selected.


### Offline state-provenance repair after E020 (2026-10-20 UTC)

Actual pinned CpuThreadActivity fixtures reproduce positive900ns ReadyDuration with no initial ready event and after a previousWaiting/no-ready-event gap. Corrected adapter returns unknown for both, while positive explicit-ready/wait/preemption cases retain known portions. Additional real-object fixtures cover mismatched canonical identity with reused numeric TID, missing endpoints, unknown timestamps, nonresident/unresolved events, unsupported states, R<P conflict, intervening event history, R==S witness and pre-window tail readiness. Exact overlap/clipping/maximum-source regressions also pass. Locked decoder build has zero warnings; independent final targeted source review has no blocker. Root check/145Node tests pass. This is an observer correction and supplies no product-cause or runtime qualification.

The final compact-receipt regression also retains closed-input disposition counts. All14focused Python scheduler tests pass in0.690s; the last full455Python suite was39.096s before this allowlist-only addition. Native deadlines, workload and failure preservation are unchanged. No further hosted run selected at this checkpoint.


### S040 E021 result: real console provenance passes, conflict branch exercised

[Run39365318063](https://github.com/shruggietech/insonic/actions/runs/39365318063), attempt1/job118440726012, exactsourcee779656db645585893f3e539bfa81a7f04484836, passes in38s. Zero product build/GUI/CLI workload. Owned consolePID5980/mainTID1992 meets unchanged thresholds: observed531207700ns, knownwaiting353479600ns(max200561400), scheduled176956500ns(max21161200), ready765100ns(max47000), unknown6500ns. The explicit Waiting→ready provenance and200.5614ms maximum match the deliberate200ms sleep; initial/unproved gaps remain unknown.

Closed-input dispositions:30waiting-and-explicit-ready,9ready-switch-out,6explicit-ready-only,2unproved-closed-gap. One actual200ns scheduled/explicit-ready conflict is retained with endpoints and excluded from both categories, exercising the corrected branch. Maxima and fixed sources remain in the receipt. Cleanup/raw-directory removal confirmed. Artifact12238719074,1247B,SHA2566271795093572dd7c6465e71106a8afe6724102b1fc153241ba3cca6c9b43d93. This validates the real console observer only and does not repair or retrospectively qualify E020's unsupported categories.

### S040 E022 result: validated ready intervals during the retained native failure

[Run 39365570539](https://github.com/shruggietech/insonic/actions/runs/39365570539), attempt 1, job 118441527276, exact source e779656db645585893f3e539bfa81a7f04484836, created 2026-10-20T03:40:57Z, fails in 5m49s. Its plain console preflight passes. The only fresh GUI fails at 18.157s (exit 1/timeout, complete/untruncated); Normal control is valid (GUI 3700 class 32, wrapper 2744 class 16384). OnStartup 8.0867045s, arm 8.0867486s, prepare 8.1884729s. The outer deadline precedes the full post-controller 12s watchdog interval; no browser/version snapshot arrives. Actual runtime version is unavailable. Untimestamped kind 4 utility-exit and kind 6 GPU-exit messages cannot be separated from forced cleanup.

The live query finishes at shared 40.188s before cleanup at 43.297s (case approximately 15.016s and 18.125s). All nine selected members match native and trace lifetimes: renderer 2204 Idle64, browser 788 Normal32, GUI 3700 Normal32, GPU 4788 AboveNormal32768, utilities 6172/7528 Normal32, crashpad 6476 Normal32, wrapper 2744 and other-owned 5664 BelowNormal16384. Job CPU is 1.1875s live / 1.21875s final; system interval 99.764% busy, available physical 12,547,817,472 bytes. These endpoint classes are not dynamic thread priorities or affinity measurements.

Scheduler observation is available: 1,000,772 context switches, 573,105 ReadyThread events, zero loss, strict clock/range/native-lifetime gates and cleanup/raw removal confirmed. The corrected canonical event provenance supports these case-relative intervals:

- Renderer oldest TID1964 is explicitly ready from 13.9590587s to the observation end 18.150304s (tail-ready-event). Approximately 4.166s is before forced cleanup begins at 18.125s. It has only 85.816ms scheduled during 9.854s observed, with a separate 5.114s validated waiting interval.
- Renderer worker TID1900 remains ready for 5.4508773s, 8.3555079-13.8063852s, from a validated Ready switch-out with no intervening event. This interval is wholly before cleanup.
- GUI oldest TID2796 has a maximum known ready interval of 1.6295ms, with 17.814s known waiting and 126.111ms scheduled. Its waiting dependency is unidentified.
- GPU worker TID544 has a 4.7529561s known ready interval, and Normal-class utility workers TID4588/TID2616 have 5.6911469s/5.6822632s known ready intervals. Several workers run around case13.8063s. A starvation-boost mechanism is a hypothesis, not demonstrated by retained events.

All thread time is computed before sparse retention of oldest plus one longest-ready and one longest-waiting thread per owned process; this does not prove a critical path. Unknown initial/unsupported/conflicting fragments remain explicit. This occurrence supports browser-side runnable scheduling delay rather than GUI-oldest starvation, but renderer process class alone does not explain the GPU/Normal-utility delays, and no root cause is assigned. Diagnostic browser-domain setup before navigation remains an observer limitation.

CLI passes in 112.750s, zero dropped events. Main media 25.157-70.875s fully covers the GUI case 25.157-43.329s. Original controls/assets match. GUI SHA256 070b64be98add4c22e17bd7d4118c7f2947e3244682b878603b8718145c679af; CLI 61175a1f02da2886d77dd6ec2991295df4c6f34d7561d03315a8cd36983ccbc5. Artifact 12239695563, 17,610 bytes, SHA256 f04f4fb5467656adb0c9eaf02b43ab47418eef7f22503a53969f9c4c0a955ea8.

Independent interpretation supports one narrow renderer-backgrounding policy comparison before a broad runtime change. The intervention may change OS priority and browser visibility/background policy; even a pass cannot isolate those effects or qualify the repair. #154 remains open. No next dispatch has occurred.

### E023 diagnostic overlay validation (2026-10-20 UTC)

**Final E023 pre-dispatch validation**: All458Python tests pass (38.718s test process,39.281s launcher); npm check/145Node tests and actual hidden Go1.27.1 overlay compile/fake-COM tests pass. Independent actual source review finds no remaining blocker after the checksum correction: exact single-expression binding reaches the maintained module replacement, only GUI compilation receives it, fixed receipt links source/replacement to derived GUI, all existing control gates/default isolation and first-failure/early-pass disposition are preserved. Source comparison confirms no committed product/frontend/SDK/dependency change since E022. UTF-8/no-BOM/LF/mojibake and diff checks pass.

### S040 E023 result: startup and close complete; natural browser release still times out

[Run 39367049914](https://github.com/shruggietech/insonic/actions/runs/39367049914), attempt 1, job 118446158498, exact source 5edfc3aba1f91017fb3b19c7545f0b67eb0ffbdc, created 2026-10-20T03:58:03Z, fails in 5m40s. The exact elected overlay receipt matches original546a92905b17890764af1daefa3a475b9182d397a92a60722d774f2bc355cf73 and replacement79b98d9dec01f3cdf1ad311f27404657583f605a29fca7f8583b18f176ffc50b. Plain console preflight passes.

The only fresh child returns exit0 and stdout native_exit=passed/action=close/playback=not-selected, but the original whole-case gate correctly remains failed: timed_out=true, owned_release=false,18.031s elapsed, complete/untruncated. Stage times relative to child origin are OnStartup4.8042963s, watchdog arm4.8042963s, preparation4.9207202s, DOM-ready5.7589117s and shutdown6.0242875s. The parent records root exit at6.547s, leaving11.453s for natural owned-process release; that entire remaining budget expires. Post-forced-kill zero members is not a natural release pass. Normal launch control is valid (GUI7736/32, wrapper6096/16384).

The actual created-environment runtime is131.0.2903.86. All six resource classes finish with expected bytes/status/MIME, associated DOMContentLoaded/load both1, setup armed, navigation accepted, no JavaScript exceptions. Renderer and GUI have exited before the15s live endpoint, so effective renderer class and renderer scheduling are unavailable. This observation is consistent with a startup contribution from the elected policy, but changed machine state and missing treatment measurement prevent causal attribution or a startup repair claim. The single native case as a whole still fails.

The live query completes at shared46.813s, before forced cleanup49.797s, with five directly verified members/job-active5 of9 total: browser4304 Normal32, GPU2112 AboveNormal32768, utilities4236/5832 Normal32, crashpad5448 Normal32. GUI/wrapper/renderer are absent. Job CPU1.890625s live/1.90625s final; system interval99.848% busy, physical available12,707,233,792 bytes at final snapshot. The live sample alone does not establish which members survive until the final deadline.

Scheduler observation is available with966,316 context switches,558,129 ReadyThread events, zero loss, strict native/trace lifetime, clock/range and cleanup/raw-removal gates. Supported times below are relative to child origin (298384us after the parent window anchor); maxima are translated from thread-observation-relative endpoints:

- Normal utility5832/worker2392 has11.5001475s known ready versus5.2124ms scheduled in11.5450406s observed. Its longest known ready interval is5.6945109s,10.1907605-15.8852714s, from Ready switch-out without an intervening event, wholly before cleanup at17.701616s. This overlaps the natural-release wait.
- Browser4304 oldest3400 has9.6787424s known waiting at6.213002-15.8917444s, maximum known ready2.5039ms and472.326ms scheduled. The waiting dependency remains unidentified.
- GPU2112 worker5904 has4.4956832s known ready at5.5510848-10.046768s. Additional browser/utility ready suffixes are retained, but no dynamic priority, affinity, task criticality or exact browser shutdown dependency is proved.

CLI passes in144.156s with zero dropped events. Media31.781-91.234s fully covers the case31.781-49.828s. Exact CSS/script controls match E022; CLI receives no overlay. GUI SHA2565eb85105c113ced3d20fbf67d2a44a5670db0ca1080d99dfd779ab58231d4703; CLIfab602c5317045bc88e93d39bf06b9b58a1b74e1f96bf0569378442dd952a27b. Artifact12240595013,16,253 bytes,SHA2569b9edce09150284f597ec209632d8e3fb41491ee087c1e91874f59da40c26da3.

The release-timeout signature now has direct surviving-role/state evidence, independent of a pre-DOM failure. Next inspect post-Run ownership and browser background work before selecting a further control. Three-thread retention remains sparse; initial/unsupported/conflicting fragments stay unknown. No shipping flag, new product correction, qualification, identical retry or next dispatch is selected. #154 remains open.

### Process lifetime bound correction after E023 (2026-10-20 UTC)

Independent review finds that Decode intersected case and thread lifetime but omitted the matched canonical process birth/exit. A thread without an observed exit could therefore receive inferred tail states after a known process exit. E023 did not retain process-exit metadata; no late-tail values can be repaired retrospectively, and which15s members survive at18s remains unknown. Its cited utility5.6945109s ready and browser9.6787424s waiting intervals are closed by actual later switch-ins, not inferred tails. Original native/release outcome and15s membership remain valid.

An actual pinned-interface fixture extracts the existing calculation and fails with a process200-1300ns, thread born100ns/no exit and case0-3000ns. Intersecting process/thread/case bounds returns200-1300ns and passes. Additional fixtures cover tighter case/thread bounds, missing thread metadata and disjoint lifetimes. The same effective end now bounds final-tail inference, partition and summary; retain only a fixed observed-process-exit flag and nullable relative-window timestamp. Locked build has zero warnings and all self-tests pass without recording or product execution. This diagnostic correction supplies no incident repair or new hosted result.

**Retrospective E022 qualification**: The uncapped process lifetime also prevents calling the full4.166s renderer tail established alive time. Its15s membership remains verified. The closed worker1900 ready interval5.4508773s remains supported by a later switch-in. Its8.3555079-13.8063852s endpoints were relative to the parent case window; child-origin-relative endpoints are8.0473979-13.4982752s (origin+0.308110s), with identical duration. No old inferred tail is retroactively corrected from deleted traces.

### E024 control validation (2026-10-20 UTC)

**Final E024 local validation**: All464Python tests pass in41.331s (41.875s launcher), including actual hidden normal/breakaway/nested affinity inheritance and unchanged parent mask. Root npm check/145Node pass. Locked decoder build/self-tests have zero warnings and include the failing-before/passing-after process bound. Independent final source review has no blocker. Default/product/GUI/frontend/SDK/dependency source is unchanged from E023; only elected diagnostic workload allocation and interpretation differ. UTF-8/no-BOM/LF/mojibake and git diff checks pass.

### S040 E024 result: native pass under verified workload CPU allocation

[Run 39369206466](https://github.com/shruggietech/insonic/actions/runs/39369206466), attempt1/job118453001255, exact source44da29bf85ba43f67f0bdc642328c43292a2eb55, created2026-10-20T04:22:13Z, finishes in6m47s. Its plain console preflight passes. The sole fresh native case passes: exit0, complete/untruncated capture, no timeout, owned_release=true, elapsed9.843s. Root exit is9.718s, leaving8.282s natural-release budget; release completes in0.125s with all nine owned processes naturally gone before job close. No forced cleanup earns this result. Normal GUI1016/class32 and wrapper3420/class16384 controls are valid.

Child-origin stages: OnStartup/watchdog6.228363s, preparation6.3440984s, DOM-ready8.4650956s and shutdown8.9639877s. Frontend readiness and actual close input complete. Actual created runtime is131.0.2903.86. Browser setup is armed, navigation accepted, all six resources finish, associated DOMContentLoaded/load both1 and JavaScript exceptions0. Shared case25.437-35.296s lies entirely in main media25.437-89.125s. CLI passes in132.843s with zero dropped timeline events.

The workload receipt verifies all1297 launched roots against mask7 before resume (four available CPUs/mask15, three workload CPUs, excluded bit8), with zero failed launches.703 roots permit breakaway; this is not a count of independently observed detached descendants. Explicit-root affinity and normal/nested/breakaway inheritance were proved locally, but every descendant's whole lifetime was not sampled. Shared parent/GUI affinity is unchanged; in-process fixture threads and ambient work remain unrestricted. Excluding bit8 does not guarantee an idle or exclusive CPU. System interval average is79.279% busy, final available physical memory13,099,323,392 bytes; job charged CPU1.5625s.

The workflow is red solely because the native case ends before the unchanged15s live query: scheduler actual-input is unavailable with reason not-started-before-capture-end. Cleanup/raw removal is confirmed. This is unavailable observation, not a native startup or release failure. No renderer-class or ready/wait latency improvement can be measured from this case. E023's diagnostic renderer flag remains enabled, so this supports CPU allocation as a contributor without isolating flag policy or proving a complete cause/repair.

GUI SHA2567aa95e1ee79c522d5c58a0a15232f066e099c4f5748662ce55d0a4b60a636ed7; CLId4a6ffa1a30d0ccaf0164a40706223784a21f31371a8758024b772267d38ec59. Exact CSS/scripts and original/replacement overlay authority match E023. Artifact12240937175,13,109bytes,SHA256415fb2b60e0c4a8399256b221cb316279fa37dba18702249f8bd61ab0a1773bd. Independent interpretation agrees: next separate the flag while preserving allocation and old SDK source. No matrix, product affinity/flag policy or qualification is selected. #154 remains open.

### E025 local validation

**Final E025 pre-dispatch validation**: All464Python tests pass in40.519s (41.125s launcher) and root npm check/145Node tests pass. The first restricted local attempt failed Windows fixture authorization/drive setup and dependency access, without a product GUI or trace; the same hidden suite passes with those local permissions. Independent actual gate/build/test review finds no blocker. Source comparison and checksum confirm unchanged E024 product/frontend/SDK/dependencies. UTF-8/no-BOM/LF/mojibake and diff checks pass.
