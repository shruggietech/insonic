# Tasks: Media lifecycle correctness

**Input**: [spec.md](spec.md), [plan.md](plan.md), research, data model and contracts.
**Tests**: Regression-first checks are required by the issue acceptance and autopilot workflow.

## Phase 1: Setup

- [x] T001 Confirm S033, five selected issues, branch and directory identity in spec.md and plan.md.
- [x] T002 Read constitution/current contracts, inspect existing fixture/cleanup systems and complete research.md.

## Phase 2: Foundation

- [x] T003 Review requirements/checklists and run cross-artifact analysis before product edits in specs/033-media-lifecycle-correctness/.
- [x] T004 Inspect current .gitignore and prepared native tools; establish hidden test execution using scripts/run-hidden.ps1 or equivalent.

## Phase 3: US1 accepted refresh (P1)

**Goal**: Committed updates succeed under normal retained-report leases.
**Independent test**: Pending cleanup survives retry/reopen and completes after release/expiry without another acceptance.

- [x] T005 [US1] Add held-lease, retry/reload and real-failure-precedence regressions in internal/library/cleanup_test.go and backend parity tests.
- [x] T006 [US1] Share accepted cleanup in internal/library/service.go and internal/library/admission.go; genuine errors outrank deferred conflicts.
- [x] T007 [US1] Exercise refresh cleanup recovery through existing runtime sweeps in internal/app tests or installed CLI qualification.

## Phase 4: US2 embedded admission and playback (P1)

**Goal**: Supported subtitle admission executes and returned MIME matches supplied bytes.
**Independent test**: Exact companions admit embedded tracks; canonical and preview descriptors agree with probed output.

- [x] T008 [P] [US2] Add canonical/legacy/preview MIME regressions in internal/app/playback_mime_test.go and playback_preview_test.go.
- [x] T009 [P] [US2] Add real packaged-format SubRip and supported-format coverage in internal/library/admission_test.go.
- [x] T010 [US2] Select canonical format before source MIME in internal/app/desktop_ops.go while preserving preview overrides.
- [x] T011 [US2] Select the shipped SubRip encoder in internal/library/embedded.go without changing subtitle output format.

## Phase 5: US3 transcript batch and remote causes (P2)

**Goal**: Manifest targets reach admission and bounded failures retain actionable causes.
**Independent test**: Two-record manifest import and loopback timeout/limit/cancel cases preserve prior accepted state and durable errors.

- [x] T012 [P] [US3] Add conditional-target and incompatible-input regressions in cmd/insonic command/domain tests.
- [x] T013 [P] [US3] Add transport deadline, declared/streamed/exact byte bounds, cancellation, cleanup and redaction tests in internal/library/acquisition_test.go.
- [x] T014 [US3] Correct transcript parser required-target registration/help in cmd/insonic/commands.go.
- [x] T015 [US3] Preserve classified acquisition causes in internal/library/acquisition.go with existing public errors and durable correction notices in internal/library/admission.go.
- [x] T016 [US3] Cover durable error reload and failed-replacement preservation in library/app or installed-package qualification tests.

## Phase 6: Qualification and publication

- [x] T017 Extend scripts/cli_contract_qualification.py and reusable fixture support with all five installed-binary regressions using spaces/Unicode paths; document partial #114 coverage.
- [x] T018 Update docs/v1.0.1 contracts and CHANGELOG.md with resulting behavior and decisions.
- [x] T019 Run focused/full relevant checks, package qualification, UTF-8/LF/mojibake sanity and record exact evidence in verification.md.
- [x] T020 Commit, push and open/attach the PR closing the five issues; update GitHub tracking and record publication/review evidence in verification.md. Maintain the remaining capped-review and final-head CI gates directly on the PR, stopping before owner merge.

## Dependencies and Execution Order

Setup precedes foundation and analysis. Each story's regression tests precede its production fix. US1 is first priority; US2 and US3 are independently testable after foundation. Shared service/admission/acquisition edits stay sequential. Qualification and publication follow all stories.

## Parallel Examples

US1 backend research can run alongside US2 playback research. US2 MIME and embedded regression design use different files. US3 parser and transport regression design use different files. Research agents supply findings; implementation coordinates shared files in one checkout.

## Implementation Strategy

Complete and verify US1, then US2 and US3, and finally connected package regressions. Deliver the full authorized five-issue slice before publication; no partial story is presented as completion.

## Phase 7: Convergence

- [x] T021 Extend internal/library/cleanup_test.go with expired-report-lease retirement coverage per FR-002 and SC-002 (partial verification coverage).

- [x] T022 Strengthen embedded subtitle regressions to assert parsed cue text and exact start/end times in internal/library/admission_test.go and scripts/cli_media_qualification.py.

- [x] T023 Resolve round-one review: reject media intent in transcript manifests before dispatch, preserve transcript aliases/presets, and verify the mixed-batch rejection through the installed CLI.
