# Tasks: Desktop startup diagnostics and recovery

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [startup contract](contracts/startup-failure.md).

## Phase 1: Setup

- [x] T001 Confirm final accepted S041 main, available S042 scope/branch and #154/#115 mapping in specs/042-startup-diagnostics-recovery/spec.md.
- [x] T002 Complete design, issue/milestone/project mapping and blocking Spec Kit analysis in specs/042-startup-diagnostics-recovery/plan.md (FR-012).

## Phase 2: US1 actionable startup reporting (P1)

**Independent test**: Inject a returned cause and denied diagnostics; error identity and native feedback remain, while persisted events omit arbitrary/private inputs.

- [x] T003 [US1] Add returned-error wrapping and owned-transport cleanup regressions in cmd/insonic-desktop/startup_failure_test.go and application_options_test.go (FR-001/003, SC-001).
- [x] T004 [P] [US1] Add denied-storage, unavailable-presentation and persisted metadata privacy regressions in cmd/insonic-desktop/startup_failure_test.go and internal/diagnostics/logger_test.go (FR-002-004, SC-001/003).
- [x] T005 [US1] Wrap playback/window/close errors without losing original causes through cmd/insonic-desktop/application.go and main.go (FR-001).
- [x] T006 [US1] Add bounded failure feedback, per-user private event persistence and native Windows/Linux presentation in cmd/insonic-desktop/startup_failure*.go; keep explicit qualification noninteractive (FR-002/003).
- [x] T007 [US1] Allowlist startup stage and unsigned numeric system-error metadata only for desktop.startup in internal/diagnostics/logger.go (FR-004, SC-003).

## Phase 3: US2 usable private diagnostic files (P1)

**Independent test**: A fixture directory with inherited Modify creates current-user private log/lock files; denied access, ownership/path mismatch, rotation and contention remain correctly handled.

- [x] T008 [US2] Add a failing inherited-Modify reproduction and existing/rotation/lock/denied-access Windows regressions in internal/diagnostics/private_windows_test.go (FR-005/006, SC-002).
- [x] T009 [US2] Add foreign-owner, original-handle identity and reparse refusal coverage in internal/diagnostics/private_windows_test.go, plus fail-before-effects coverage in existing CLI diagnostics tests where affected (FR-005/006).
- [x] T010 [US2] Request only READ_CONTROL/WRITE_DAC/FILE_READ_ATTRIBUTES initially; request WRITE_OWNER only for necessary new-file owner normalization and revalidate identity in internal/diagnostics/private_windows.go (FR-005).
- [x] T011 [US2] Preserve restrictive protected ACLs and current owner checks; distinguish creation/path, privacy and lock opening corrections in internal/diagnostics/logger.go (FR-005/006).

## Phase 4: First independently mergeable increment

- [x] T012 Update docs/v1.0.1/desktop.md, cli.md, development.md and CHANGELOG.md for reporting/private diagnostic behavior and dated decisions (FR-001-006/012).
- [x] T013 Run required root, affected Go, site and file-integrity checks; record exact local evidence in specs/042-startup-diagnostics-recovery/verification.md (SC-001-003/005).
- [x] T014 Publish the frozen reporting/diagnostics PR, verify complete CI, squash merge and retain exact-main acceptance externally before the next increment (FR-011/012, SC-005).

## Phase 5: US3 concrete startup termination and recovery (P1)

**Independent test**: Inject synchronous/asynchronous and partial-initialization failures; one failed outcome skips the ordinary loop, cleanup balances on the owner thread after callback unwinding, and a later valid launch succeeds.

- [x] T015 [US3] Add focused fake-COM/message-only failure, late-completion, partial-cleanup and fresh-instance regressions in third_party/go-webview2/pkg/edge startup tests (FR-007-009, SC-004).
- [x] T016 [US3] Add an error-returning Embed path and first-failure outcome in third_party/go-webview2/pkg/edge/chromium.go; reject failed/null outputs and interrupted initialization without success fall-through (FR-007).
- [x] T017 [US3] Reuse owner-thread destruction/release after callback unwinding and preserve late-callback refusal in third_party/go-webview2/pkg/edge/window_lifetime_windows.go (FR-008/009).
- [x] T018 [US3] Preserve a checksum-verified pinned Wails baseline and correct only startup return/setup and dev/production loop entry in third_party/wails, with focused failure/success lifecycle regressions and caller-owned settings release (FR-007/008/010).
- [x] T019 [US3] Extend exact maintained source, Go replacement and notice qualification to exactly the two approved modules through scripts/webview-source.mjs, scripts/check.mjs, scripts/go_module_notices.py, qualification metadata and affected tests (FR-010, SC-006).
- [x] T020 [US3] Add a bounded controlled failed-launch/subsequent-valid-process qualification to existing cmd/insonic-desktop native/package fixtures without changing any original case or operation allowance (FR-009/011, SC-004/005).

## Phase 6: Recovery increment verification

- [x] T021 Update current recovery/provenance contracts and CHANGELOG.md, reconcile Spec Kit consistency, and run required affected/root/site/source-authority checks in specs/042-startup-diagnostics-recovery/verification.md (FR-007-012, SC-004-006).
- [ ] T022 Publish the frozen recovery PR, resolve concrete failures, qualify real Windows packages/native behavior, squash merge and retain exact-main acceptance externally (FR-011/012, SC-005/006).
- [ ] T023 Reconcile #154/#115 criteria and milestone/project state individually, preserve unresolved historical causes and S041 revision-bound campaign evidence in specs/042-startup-diagnostics-recovery/verification.md (FR-012).

## Dependencies and execution

T001 -> T002 -> US1/US2 -> T012 -> T013 -> T014 -> US3 -> T021 -> T022 -> T023.

US1 and US2 tests in separate files may be authored independently; shared logger edits remain sequential. T004 privacy tests can run alongside the Windows ACL reproduction once fixtures exist. The reporting/diagnostics increment is the first useful delivery and does not depend on US3. Native startup and framework changes remain in the second increment. S041 campaign bookkeeping is not a dependency of either product merge.

## Phase 7: Demonstrated first-candidate corrections

- [x] T024 Preserve the failed candidate, correct stale diagnostic/qualification fixtures, return an existing-file profile error before native creation, and report bounded observed CI input identity fields without weakening checks; verify regressions before a new candidate (FR-007/009/011/012).
- [x] T025 Reproduce the second candidate's queued workflow summary with matching identity and successful readiness; accept that coarse active state only alongside exact job authority, preserving running-aggregate, all-other-terminal and strict terminal-audit requirements in scripts/ci-acceptance.mjs (FR-011/012).
- [x] T026 Retain the exact controlled-child diagnostic failure category and generated Windows access/owner-equality observations; add a hidden owned-child Modify-directory regression without weakening private-event acceptance (FR-003-006/009/012).
- [x] T027 Reproduce and correct new-file default-owner normalization under inherited Modify: install the already-required protected current-user DACL before requesting WRITE_OWNER through an identity-validated handle, retaining existing foreign-owner refusal and all private-file regressions (FR-005/006/009/012).
