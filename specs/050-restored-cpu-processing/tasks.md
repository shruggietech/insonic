# Tasks: S050 Restored-library CPU processing acceptance

Input: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md). TDD applies. Tracking: #178 under #147.

## Phase 1: Setup

- [x] T001 Align slice/branch/selector and tracking, resolve public retention/setup semantics in specs/050-restored-cpu-processing/spec.md and plan.md (FR-001..010).

## Phase 2: Foundation

- [x] T002 Add failing strict destination receipt and fresh-work lifecycle cases in tests/journey_cpu_test.py and tests/journey_restored_test.py, retaining legacy source receipt meaning (FR-003..009).

## Phase 3: US1 - Fresh destination processing (P1, MVP)

Independent test: destination setup plus fresh held-out/sequence generate/run/match produces new work/revisions from restored media with source absent.

- [x] T003 [US1] Refactor worker prep and processing/matching observations into separate evidence sinks in scripts/journey_cpu.py without model reacquisition or duplicated validation (FR-002..004).
- [x] T004 [US1] Add destination preparation/processing/matching/retention stages, work identities, current supersession and canonical/profile/history comparison in scripts/journey_cpu.py (FR-001..005,010).

## Phase 4: US2 - Recovery and failure contracts (P2)

Independent test: focused existing portable successor/pin/cancellation suites and new source-withdrawal/retained-history negative cases reject invalid authority or promotion.

- [x] T005 [US2] Exercise missing/changed pins, stable local-model successor, cancellation/deadline and retained state in internal/app/portable_work_test.go, internal/app/workers_test.go and internal/workers/probe_diagnostics_test.go; add regression/repair only for demonstrated gaps (FR-006,007).

## Phase 5: US3 - Auditable acceptance (P2)

Independent test: receipt mutations reject absent/contradictory destination evidence; one complete exact-source real CPU campaign passes.

- [x] T006 [US3] Implement versioned strict destination evidence with common engine/reference validators in scripts/journey_cpu.py and consistent failure kind in scripts/qualify-journey.py (FR-008,009).
- [ ] T007 [US3] Qualify exact clean complete Windows CLI package, independently validate source/destination receipt and record compact audited provenance/timing/cleanup in specs/050-restored-cpu-processing/verification.md and GitHub (FR-001..010).

## Phase 6: Cross-cutting and delivery

- [ ] T008 Run root/affected Python/Go/site checks, encoding/link/schema checks and Spec Kit convergence; update CHANGELOG.md and concise specs/050-restored-cpu-processing/verification.md (SC-001..004).
- [ ] T009 Commit/push/open official PR, monitor elected final-head CI and resolve every review thread within at most two Codex rounds; retain compact failures before full reruns (SC-004).
- [ ] T010 Verify final SHA, terminal checks, actual CPU evidence, resolved reviews and project tracking; remove owned scratch and hand off unmerged using specs/050-restored-cpu-processing/tasks.md (SC-004).

## Dependencies and execution

T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> local-check portion of T008 -> initial commit/push/PR portion of T009 -> T007 -> final convergence portion of T008 -> terminal CI/review portion of T009 -> T010. T006's mutations begin at T002 and final validation follows all code changes. T007 uses the clean committed exact-source package produced by initial PR CI; final CI follows any campaign-driven repair. T007 may overlap independent CI jobs; inference never runs inside CI. No parallel edits to shared harness/test files. Existing independent Go negative suites may run alongside root/site checks after implementation. US1 is the MVP, but all tasks are required for the authorized handoff.
