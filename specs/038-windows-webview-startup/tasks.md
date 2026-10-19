# Tasks: Windows desktop startup and Exit reliability

**Issue**: #154 | **Slice**: S038 | **Branch**: codex/038-windows-webview-startup

## Phase 1: Setup

- [x] T001 Align branch, spec directory, issue/milestone/Project and owner stop boundary in specs/038-windows-webview-startup/spec.md.
- [x] T002 Complete clarify/checklist/plan/research/data model/contracts in specs/038-windows-webview-startup/.

## Phase 2: Foundation

- [x] T003 Pass blocking cross-artifact analysis and record traceability in specs/038-windows-webview-startup/verification.md.

## Phase 3: US1 - Retain actionable native evidence

Independent test: controlled failed output on both streams survives without converting failure to success.

- [x] T004 [US1] Add failing-first stdout/stderr, exit/cancellation/deadline/truncation and malformed-success regressions in cmd/insonic-desktop/exit_qualification_test.go (FR-001/FR-002, SC-001).
- [x] T005 [US1] Implement bounded separate 65536-byte stream evidence and result interpretation in cmd/insonic-desktop/exit_qualification.go (FR-001/FR-002).

## Phase 4: US2 - Demonstrate and correct startup cause

Independent test: incident-bound diagnostic observes actual native error; repeated native routes pass with owned cleanup after correction.

- [x] T006 [US2] Add identity/ownership/refusal regressions and focused disposable probe in tests/desktop_startup_diagnostic_test.py, scripts/diagnose-desktop-startup.py and .github/workflows/installer-invocation-diagnostic.yml (FR-003/FR-005, SC-002/SC-004).
- [ ] T007 [US2] Run focused Windows diagnosis before lifecycle changes and record actual native error/cause in specs/038-windows-webview-startup/verification.md (FR-003, SC-002).
- [ ] T008 [US2] Add cause-specific regression and proportional correction in cmd/insonic-desktop/exit_qualification*.go or demonstrated affected internal/process source; record why in research.md (FR-003/FR-005, SC-002/SC-004).
- [ ] T009 [US2] Extend repeated cold/warm menu/close/Alt+F4 and workspace/no-workspace evidence in cmd/insonic-desktop/exit_qualification*.go and qualification consumers scripts/qualify.py/scripts/package-desktop.py, preserving playback/durable work assertions (FR-004/FR-005, SC-003/SC-004).

## Phase 5: US3 - Preserve delivery authority

Independent test: failed producer retains its primary error and still blocks every exact-byte join/publication gate.

- [x] T010 [P] [US3] Add failed-producer evidence regressions and implementation in tests/ci-artifact.test.mjs, scripts/wait-ci-artifact.mjs and .github/workflows/windows-desktop-qualification.yml; retain existing tests/windows_package_pipeline_test.py refusal gates (FR-006, SC-004).
- [ ] T011 [US3] Run required root/affected suites and complete exact-head native/package delivery; record all gates and elapsed time at most 660 seconds in verification.md (FR-006/FR-007, SC-005).

## Phase 6: Convergence and delivery

- [ ] T012 Update CHANGELOG.md/docs/v1.0.1/development.md, converge requirements and record unverified post-merge boundary in specs/038-windows-webview-startup/verification.md (FR-008).
- [ ] T013 Push/open/attach PR, handle at most two review rounds, resolve all threads, verify final-head checks and hand off unmerged with S038/#154 metadata (FR-008, SC-005).

## Dependencies

T001-T003 precede implementation. T004 precedes T005; T006/T005 precede T007; T007 precedes T008/T009. T010 is independent of native diagnosis after foundation. T011 requires T008-T010; T012/T013 require final verification. US1 is the first independently testable delivery; diagnostic success alone does not complete US2 or US3. No parallel agents edit product code.

## Phase 7: Convergence

Assessment checked 20 requirement/success/acceptance criteria, six plan decisions and five core constitution principles. Three partial high-priority gaps remain under the original approved intent. No scope reduction or clean convergence is claimed.

- [ ] T014 Demonstrate the actual native startup error/cause and implement its supported correction with a relevant regression per FR-003, SC-002 and US2/AC1 (partial; HIGH; T007/T008 remain open).
- [ ] T015 Complete real packaged cold/warm native action and owned release/reuse proof per FR-004/FR-005, SC-003 and US2/AC2 (partial; HIGH; T009 remains open).
- [ ] T016 Complete exact-final-head delivery/660-second verification and the agreed review/scope boundary per SC-005, FR-008 and US3/AC2 (partial; HIGH; T011-T013 remain open).

## Phase 8: Owner-approved diagnostic checkpoint

On 2026-10-19 the owner selected "Deliver diagnostic checkpoint; keep #154 open." This supersedes the original cause-repair scope. T007-T009 and convergence T014-T015 remain unchecked as explicitly deferred #154 work. They are excluded from checkpoint convergence and must not be presented as implemented. T011-T013/T016 remain active checkpoint verification/delivery obligations.

- [x] T017 Align spec.md, plan.md, contracts, data model and quickstart with the approved diagnostic checkpoint; preserve the original deferred tasks and open #154.
- [ ] T018 Verify corrected historical/source identity, bounded streaming, compiler environment isolation and concurrent cleanup on disposable Windows; converge against the approved checkpoint.
- [ ] T019 Finish final exact-head delivery, resolve the remaining permitted review round, verify all threads and hand off unmerged for owner review.
