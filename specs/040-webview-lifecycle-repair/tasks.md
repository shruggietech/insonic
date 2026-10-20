# Tasks: Windows desktop lifecycle repair

**Input**: [spec](spec.md), [plan](plan.md), [research](research.md), [contract](contracts/native-lifecycle.md).
**Tests**: Native ABI, ownership, privacy and exact-source regressions are required. Run failing regressions before corrections.

## Phase 1: Setup

- [x] T001 Verify merged source/tree and finish branches, milestone and project housekeeping, recorded in specs/040-webview-lifecycle-repair/research.md. (FR-001)
- [x] T002 Create the S040 specification, requirements review, design and dedicated issue tracking in specs/040-webview-lifecycle-repair/. (FR-012, FR-015, FR-016)

## Phase 2: Foundation

- [x] T003 Consolidate the independent framework audits into specs/040-webview-lifecycle-repair/framework-map.md with all five failure signatures, exact source lines and rejected hypotheses. (FR-002, FR-003, SC-001)
- [x] T004 Complete the read-only analysis gate and coverage record in specs/040-webview-lifecycle-repair/analysis.md before implementation. (FR-004, FR-005, FR-007, FR-009, FR-013, FR-014)

## Phase 3: US1 - Reach a usable desktop consistently

**Independent test**: Native vtable arguments fail before the upstream correction and pass after it; actual cold/warm startup is then observed under original deadlines.

- [x] T005 [US1] Add the native false/true argument and HRESULT propagation regression for the actual production wrapper in internal/webviewabi/controller_windows_test.go (the unused generated API adopts verified upstream bytes without an execution claim); record unchanged v1.0.22 failures in specs/040-webview-lifecycle-repair/verification.md. (FR-005, SC-002)
- [x] T006 [US1] Acquire authoritative v1.0.23, apply its two-file upstream delta and retain the existing maintained lifetime corrections in third_party/go-webview2/, go.mod and go.sum. (FR-003, FR-005, FR-013)
- [x] T007 [US1] Update version/tree authority and notices in third_party/go-webview2.source.json, third_party/README.md, third_party/go-webview2/NOTICE.insonic, scripts/webview-source.mjs and scripts/go_module_notices.py. (FR-013)
- [x] T008 [US1] Run the native ABI regression and existing unshown lifetime normal/race tests; verify version/source rejection tests in tests/webview-source.test.mjs and tests/go_module_notices_test.py. (FR-005, FR-008, FR-013, SC-002)
- [x] T009 [US1] Cover successful/failed final native response observation with actual fake vtables and content ownership, if the unresolved-boundary experiment requires it, in internal/webviewabi/ and third_party/go-webview2/pkg/edge/window_lifetime_windows_test.go. Record any demonstrated non-need in specs/040-webview-lifecycle-repair/experiments.md instead of adding unused machinery. (FR-005, FR-006, FR-007)
- [x] T010 [US1] Implement the minimum needed qualified response-completion/actual-runtime observations in third_party/go-webview2/pkg/edge/ and cmd/insonic-desktop/startup_diagnostics.go, preserving calls, HRESULTs, disabled behavior and bounds; update authority if bytes change. (FR-006, FR-007, FR-008, FR-013)
- [x] T011 [US1] Execute the written bounded source experiment through scripts/diagnose-desktop-startup.py and the disposable workflow; classify the first failed boundary and record native startup outcomes in specs/040-webview-lifecycle-repair/experiments.md. (FR-004, FR-009, SC-003)
- [ ] T012 [US1] If a distinct startup failure persists, reproduce and repair its demonstrated boundary with a failing native/product regression in cmd/insonic-desktop/ or third_party/go-webview2/pkg/edge/ before qualification. (FR-005, FR-006, SC-002, SC-003)

## Phase 4: US2 - Close and release owned resources

**Independent test**: All three actual native actions reach their callbacks and natural owned process release while shared work survives.

- [x] T013 [US2] Add and validate bounded launch/root-exit/release timing in internal/process/capture.go and internal/process/capture_windows_test.go if required to classify the combined-deadline signature, preserving its original deadline. (FR-005, FR-007, FR-011)
- [ ] T014 [US2] Classify shutdown against owning-thread Close, interface/apartment lifetime and actual process group release in specs/040-webview-lifecycle-repair/framework-map.md and experiments.md. (FR-002, FR-003, FR-010, FR-011)
- [ ] T015 [US2] Repair any reproduced remaining teardown violation in cmd/insonic-desktop/webview_lifetime_windows.go or third_party/go-webview2/pkg/edge/window_lifetime_windows.go with a failing-before/passing-after native regression. If no additional behavior change is demonstrated, retain and validate the existing corrected contract. (FR-005, FR-010, FR-011, SC-002)
- [ ] T016 [US2] Validate File/Exit cancellation/reopen/action, window close and actual Alt+F4 for cold/warm, with/without-library cases, natural owned-browser release, playback release and durable shared-work survival through cmd/insonic-desktop/exit_qualification.go. (FR-004, FR-008, FR-010, FR-011, SC-003)

## Phase 5: US3 - Evidence-backed repair

**Independent test**: The record links every conclusion to immutable source, a violated contract/regression or a bounded observation, without overstating finite samples.

- [x] T017 [US3] Record each experiment's prediction, control, bounds, result and next decision before another dispatch in specs/040-webview-lifecycle-repair/experiments.md. (FR-009, SC-005)
- [ ] T018 [US3] Update issue #154 with the upstream ABI finding, new observations, proven/rejected/unresolved distinctions and exact verification identities, mirrored in specs/040-webview-lifecycle-repair/verification.md. (FR-002, FR-012, FR-016, SC-001, SC-005)
- [x] T019 [US3] Run root npm run check/npm test, affected Python/native/product and site checks; retain results in specs/040-webview-lifecycle-repair/verification.md. (FR-007, FR-008, FR-013, FR-014)
- [ ] T020 [US3] Run ordinary final exact-head source/package/candidate CI within 660 seconds and retain complete check/run/attempt/artifact identities in specs/040-webview-lifecycle-repair/verification.md. (FR-004, FR-013, FR-014, SC-004)

## Phase 6: Polish and convergence

- [ ] T021 Update CHANGELOG.md with demonstrated behavior and the dated upstream dependency decision; update affected authoritative docs/v1.0.1/ without internal planning links or changed release version. (FR-015)
- [ ] T022 Verify UTF-8 without BOM, LF, no mojibake, source/package notices and git diff integrity for the complete repair. (FR-007, FR-013, FR-014, FR-015)
- [ ] T023 Converge specs/040-webview-lifecycle-repair/tasks.md with implementation/evidence, preserve unresolved failures explicitly and prepare the separate repair review handoff without merge/release or a third #156 review. (FR-012, FR-016, SC-005)

## Dependencies and execution order

T001/T002 -> T003/T004 -> T005 -> T006/T007 -> T008. T009/T010 follow the source audit, and T011 follows T017's written experiment. T012 is conditional on observed startup failure, with explicit evidence required for disposition. US2 is independently auditable after foundation; T013/T014 precede any T015 correction and T016 final native qualification. T018/T019 precede final T020; T021/T022 and convergence reflect the final behavior and exact source.

## Parallel opportunities

Read-only response/lifetime/harness audits execute independently under the plan skill. After source settles, root/Python/native/site checks can execute independently. Do not mutate the SDK, its manifest, module versions or checksum tests in parallel. Source/native experiments run sequentially to preserve ownership and causal variables.

## Implementation strategy

The first increment proves and corrects the exact upstream native BOOL violation. Response and release observations are added only to answer remaining incident questions; they are not substitutes for cause repair. All three stories and final gates are required for a completed repair. Failed cases remain failed after cleanup; inherited S039 exceptions do not satisfy S040.

## Phase 7: Demonstrated callback ownership repair

- [x] T024 Reproduce the actual9callback QueryInterface/native-reference violations without a browser and verify their IIDs against the official SDK; record the independent causality limit in research.md and verification.md. (FR-003, FR-005, SC-002)
- [x] T025 Adopt existing combridge for the actual callbacks, lazy owned creation, exactly-once creator cleanup and late-completion/event rejection in third_party/go-webview2/pkg/edge/callback_bridge_windows.go, chromium.go, create_env_go.go and window_lifetime_windows.go; cover meaningful native identity/reference/GC/late-callback cases. (FR-005, FR-008, FR-010, SC-002)
- [x] T026 Reconcile precise source/notice authority for callback changes and run selected unshown native normal/race/product checks before actual final qualification. (FR-007, FR-008, FR-013, FR-014)

## Phase 8: Active registration output ABI

- [x] T027 Reproduce the caller-output pointer violation in actual AddAcceleratorKeyPressed native-vtable calls without writing through the wrong pointer, preserving handler/HRESULT evidence in verification.md. (FR-003, FR-005, SC-002)
- [x] T028 Correct only the active registration output argument in third_party/go-webview2/pkg/edge/ICoreWebView2Controller.go; verify native output/HRESULT regressions, exact authority/notices and affected source/product checks. Keep E004's immutable comparison separate. (FR-005, FR-008, FR-013, FR-014)

## Phase 9: Failed-boundary observation and containment

- [x] T029 Preserve primary progress before optional wait observation and isolate WCT in one hidden bounded query process in cmd/insonic-desktop/startup_diagnostics.go, startup_wait_windows.go and early helper routing; retain native ABI/redaction and prove native-fault/timeout/invalid-output containment without a GUI. (FR-005, FR-007, FR-008, FR-009)
- [x] T030 Observe uniquely completed native responses with8fixed active slots and a fixed runtime-ready receipt through existing hooks in third_party/go-webview2/pkg/edge/window_lifetime_windows.go and chromium.go; verify interleaving/duplicates/overflow/failures/retirement/privacy/disabled behavior without extra native COM calls. (FR-006, FR-007, FR-008)
- [x] T031 Reconcile exact source authority/notices and write the next discriminating experiment after T029/T030 verification, before any further hosted dispatch. (FR-003, FR-009, FR-013, FR-014)

## Phase 10: Browser delivery and concurrent-work boundary

- [x] T032 Add an explicit disposable-only browser observation election, actual created-environment version and audited native CDP callbacks in third_party/go-webview2/pkg/edge/; preserve ordinary startup, bound/redact browser fetch/header/script/page summaries and prove native ABI/ownership/late/disabled behavior with unshown regressions. Setup acknowledgement precedes diagnostic navigation and its scheduling difference stays explicit. (FR-003, FR-005, FR-006, FR-007, FR-008)
- [x] T033 Retain bounded monotonic worker/phase/case timelines in scripts/diagnose-desktop-startup.py and cli_contract_qualification.py without changing workload/commands/deadlines; test actual interval/phase/privacy/failure disposition. Use this to classify overlap rather than inferred election. (FR-004, FR-007, FR-009)
- [x] T034 Reconcile precise source/notice authority and validate the browser observation checkpoint; write its prediction/control/bounds before any single next dispatch, retaining the scheduling and old-runtime identity limits. (FR-003, FR-009, FR-013, FR-014)

## Phase 11: Separate resource size from script processing

- [x] T035 Add and validate the explicit disposable CSS size control in scripts/diagnose-desktop-startup.py and its workflow, with exact original/padded asset authority, restoration on compilation failure, unchanged scripts/deadlines and a pre-dispatch experiment record. (FR-003, FR-004, FR-007, FR-009, FR-013)

## Phase 12: Fresh-child workload control

- [x] T036 Add and verify a single fresh-GUI media-lifecycle gate and bounded endpoint resource observation in scripts/diagnose-desktop-startup.py and scripts/windows_native_resources.py, preserving E008 assets, nonblocking CLI work, exact native deadlines and failure disposition; write E009 before dispatch. (FR-004, FR-007, FR-008, FR-009)

## Phase 13: Startup timing and live ownership observation

- [x] T037 Verify fixed existing-stage monotonic timing and unchanged watchdog-arm brackets in cmd/insonic-desktop/, plus one off-path live owned-resource endpoint in scripts/windows_native_resources.py and the single-child harness; preserve duplicate-handle cleanup, original deadlines and unavailable semantics, and record E010 before dispatch. (FR-004, FR-007, FR-008, FR-009, FR-011)

## Phase 14: Verify a launch-priority contribution

- [x] T038 Prove the actual hidden Windows inheritance/control mechanism without a GUI, then verify the explicit disposable Normal-host-only treatment and effective class receipt in scripts/diagnose-native-launch.py; preserve default behavior, ownership, source/assets/deadlines, native failures and unavailable classification, and write E011 before dispatch. (FR-003, FR-004, FR-007, FR-008, FR-009, FR-011)

## Phase 15: Separate scheduled, ready and waiting intervals

- [ ] T039 Prepare and validate an explicitly elected disposable-only scheduler observer with a locked stable Microsoft decoder, small owned WPR session, real console preflight, verified process/thread lifetime projection, bounded fixed output, loss/coverage/unknown semantics and raw scratch cleanup; retain E011 controls and native deadlines, and complete E012 prediction/review before dispatch. (FR-003, FR-004, FR-007, FR-008, FR-009, FR-011)

## Phase16: Resolve decoder preflight unavailability

- [ ] T040 Add fixed-code decoder rejection evidence and a bounded console-only disposable validation path after E012's zero-GUI failure; retain all gates and raw scratch deletion, and document E013 before dispatch. (FR-004, FR-007, FR-008, FR-009, FR-011)
