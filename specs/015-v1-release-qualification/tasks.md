# Tasks: v1 release candidate and final qualification

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contracts](contracts/qualification.md).

## Phase 1: Setup

- [x] T001 Establish S015 branch/feature resolver and tracking issue #42/milestone/Project in `.specify/feature.json` and `specs/015-v1-release-qualification/spec.md` (FR-012).
- [x] T002 Complete clarify, requirements-quality review and independent Phase 0 research in `specs/015-v1-release-qualification/research.md` (FR-001–012).
- [x] T003 Run read-only blocking analysis and retain its coverage/constitution disposition in `specs/015-v1-release-qualification/analysis.md` outside the analyze command (FR-011).

## Phase 2: Foundational version and source authority

- [x] T004 Add meaningful candidate-status, new-doc examples and baseline-preservation failures in `tests/release_test.py` before correcting `scripts/release.py` (FR-005, FR-007).
- [x] T005 Correct candidate preparation/current-binding scope and source-closure verification in `scripts/release.py`, `scripts/package-desktop.py`, `tests/package_desktop_test.py`; reject self-consistent stale recipe/pins and incomplete dependency groups (FR-005, FR-007).
- [x] T006 Integrate PR #41 exact pins in `go.mod`/`go.sum` and preserve runtime/credential checks (FR-009).

## Phase 3: User Story 1, connected library and recovery

**Independent test**: Start empty, complete elected operations, make owned source store unavailable and compare restored current authority/query results.

- [x] T007 [US1] Add election, bounded execution, source-isolation and receipt failure tests in `tests/qualify_journey_test.py` (FR-001–004).
- [x] T008 [US1] Implement `scripts/qualify-journey.py` using actual shared CLI operations, owned scratch, bounded IPC waits and exact current authority/digest comparisons (FR-001, FR-002).
- [x] T009 [US1] Execute controlled connected qualification with real media/Cueson/graph plus deterministic acoustic adapters; retain evidence in `specs/015-v1-release-qualification/verification.md` (FR-001, FR-002, FR-004).
- [x] T010 [US1] Execute current explicit Windows CPU recognition/diarization/enrollment/matching using installed exact models; retain resource/quality diagnostics and unexecuted claims in `specs/015-v1-release-qualification/verification.md` (FR-003, SC-002).

## Phase 4: User Story 2, internally consistent candidate

**Independent test**: Six real qualified packages and matching docs collect against one clean actual build revision; incomplete/corrupt/stale inputs fail.

- [x] T011 [US2] Add same-run/attempt producer-consumer, read-only collection and retained-full-check regressions in `tests/ci-scope.test.mjs` or `tests/release-versions.test.mjs` (FR-006, FR-007, FR-011).
- [x] T012 [US2] Add independent complete package/runtime lanes, successful archive uploads and read-only candidate collection in `.github/workflows/native-qualification.yml` and `.github/workflows/ci.yml` (FR-006, FR-007, FR-011).
- [x] T013 [US2] Prepare 1.0.0 through corrected `scripts/release.py`, updating current owned bindings and creating `docs/v1.0.0/`/`schemas/v1.0.0/` without changing baseline bytes (FR-005).
- [ ] T014 [US2] Qualify all six native variants and validate real exact-source collection, recording signing/deployment disposition and actual complete cached timing in `specs/015-v1-release-qualification/verification.md` (FR-006, FR-007, FR-010, FR-011, SC-003).

## Phase 5: User Story 3, major-version documentation

**Independent test**: Current candidate and old baseline routes/export links remain coherent; technical terms and release status are explicit.

- [x] T015 [US3] Derive current-version schema/frontend checks and public status dynamically in `tests/schemas.test.mjs`, `desktop/frontend/tests/contracts.mjs`, `site/app/page.jsx`, `site/app/docs/page.jsx` (FR-005, FR-008).
- [x] T016 [US3] Audit all docs vocabulary, refresh `docs/v1.0.0/glossary.md`, current root docs and matching frozen references, and record classifications in `specs/015-v1-release-qualification/verification.md` (FR-008, SC-004).
- [x] T017 [US3] Maintain truthful candidate capability/status, concise highlights and major-version changelog in `docs/v1.0.0/`, `CHANGELOG.md` and `specs/015-v1-release-qualification/verification.md` (FR-008, FR-010).

## Phase 6: Integration and owner handoff

- [x] T018 Run repository/schema/Node/Python/Go/vet/affected-race/frontend/site parity and retain results in `specs/015-v1-release-qualification/verification.md` (FR-011, SC-005).
- [ ] T019 Run current-code converge and implement any appended remaining work, recording assessment in `specs/015-v1-release-qualification/convergence.md` (FR-011).
- [ ] T020 Push/open the official PR, complete at most two review rounds, answer/resolve every received finding and verify final-head full CI in `specs/015-v1-release-qualification/verification.md` and the PR (FR-012, SC-005).

## Dependencies and implementation strategy

Setup precedes all implementation. T004 precedes T005; T007 precedes T008–010; T011 precedes T012. T005 and T015 precede T013; version preparation has one writer. T013 precedes final docs snapshots/candidate builds. Full review qualification follows integrated checks and convergence. Independent read-only plan research ran in parallel; implementation edits and shared qualification are sequential. Independent CI package/runtime checks run in parallel with read-only candidate collection after packages finish.

Deliver the whole approved slice, not only the first independently testable story. No product release/deployment or owner merge is performed.

## Phase 7: Convergence

- [ ] T021 Verify all six real native packages and the exact same-run candidate, retain actual complete cached timing and resolve any failures in `specs/015-v1-release-qualification/verification.md` per FR-006, FR-007, FR-011 and SC-003 (partial).
- [ ] T022 Complete official PR publication, every received review disposition within two rounds and final-head green checks in `specs/015-v1-release-qualification/verification.md` per FR-012 and SC-005 (missing).
