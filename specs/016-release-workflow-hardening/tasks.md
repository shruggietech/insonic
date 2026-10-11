# Tasks: Release workflow hardening and v1 launch preparation

**Input**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md), [data-model.md](data-model.md), [launch contract](contracts/launch.md).

**Tests**: Autopilot test-driven discipline applies; security/authority regressions precede implementation.

## Phase 1: Setup

- [x] T001 Confirm S016/branch/issue #44 and complete intent/design/checklists in specs/016-release-workflow-hardening/.
- [x] T002 Preserve baseline/kit identity inventory and establish validation/evidence outputs under specs/016-release-workflow-hardening/evidence/.

## Phase 2: Foundation

- [x] T003 Extract native CLI build helper and test standalone desktop sibling preparation in scripts/qualify.py and tests/qualify_test.py (FR-004).

## Phase 3: US1 timely complete release evidence

**Goal**: Complete six-package release evidence with full checks and warm turnaround below 600s.

**Independent Test**: Two warmed PR runs and one publication-disabled release dispatch retain all complete acceptance/inventory/source receipts.

- [x] T004 [US1] Add workflow regression assertions for independent engine/desktop/cache lanes, source/attempt patterns and release target in tests/release-versions.test.mjs (AC-001-004, FR-001-004).
- [x] T005 [US1] Split engine and desktop/credential runtime jobs, distinct immutable cache identities and aggregate current native check in .github/workflows/native-qualification.yml (D01-D03, FR-001, FR-003).
- [x] T006 [US1] Add elected documentation target and configured signing inputs confined to package lanes in .github/workflows/native-qualification.yml (D04, FR-002, FR-008).
- [x] T007 [US1] Replace serialized release native matrix with reusable independent lanes/docs and exact revision/attempt collection in .github/workflows/release.yml (AC-001, AC-003, FR-002).
- [x] T008 [US1] Measure two full warm PR runs and a complete publication-disabled release dispatch; retain exact six-package/source/docs/timing receipts in specs/016-release-workflow-hardening/evidence/ (AC-001-004, SC-001-002).

## Phase 4: US2 existing destination preparation

**Goal**: Prepare exact published-archive Pages promotion while retaining generic hosting and truthful signing state.

**Independent Test**: Isolated valid/invalid Pages scenarios, read-only existing destination preflight and existing generic/signing regressions.

- [x] T009 [US2] Add destination/domain/archive/publication/race/build identity and credential-confinement regressions in tests/pages_test.py and tests/release_test.py (AC-005-008, FR-005-008).
- [x] T010 [US2] Implement read-only existing Pages preflight and confined verified archive tree preparation in scripts/pages.py (D05-D06, FR-005-006).
- [x] T011 [US2] Implement nonforced parented ref advancement, uncertain-operation reconciliation and bounded exact-commit Pages build completion in scripts/pages.py (AC-005-006, FR-005-006).
- [x] T012 [US2] Add explicit Pages/generic release CLI/workflow election with trusted-main published-source verifier and confined permissions in scripts/release.py and .github/workflows/release.yml (AC-007, FR-006-007).
- [x] T013 [US2] Document existing Pages route, configured/absent signing and unexecuted live deployment in docs/v1.0.0/releases.md and specs/016-release-workflow-hardening/verification.md (AC-008, FR-008, FR-011).

## Phase 5: US3 packaged connected qualification

**Goal**: Extracted binaries and matching companion worker perform connected current-authority recovery.

**Independent Test**: Controlled and elected Windows CPU package journeys with exact package/model/runtime receipts.

- [x] T014 [US3] Add required worker inventory and package selector/tamper/override/isolation regressions in tests/package_desktop_test.py and tests/qualify_journey_test.py (AC-009-011, FR-009).
- [x] T015 [US3] Include fixed matching processing worker in package inventory/required files in scripts/package-desktop.py (D08, AC-009, FR-009).
- [x] T016 [US3] Add archive/receipt/revision election and package-only runtime resolution with private extraction/home in scripts/qualify-journey.py (D09, AC-009-011, FR-009-010).
- [x] T017 [US3] Execute controlled and real Windows CPU extracted-package journeys and retain precise identities/current recovery/limitations in specs/016-release-workflow-hardening/evidence/ (AC-010, SC-004).
- [x] T018 [US3] Prepare concise highlights and exact post-squash launch commands in specs/016-release-workflow-hardening/launch.md and docs/v1.0.0/development.md (D07-D10, AC-012, FR-010-011).

## Phase 6: Polish and handoff

- [x] T019 Update dated architecture decisions/change effects and frozen current references in CHANGELOG.md and docs/v1.0.0/references/CHANGELOG.md; verify baseline/upstream integrity (FR-011).
- [x] T020 Run npm run check/npm test and affected Python/product/site checks, then Spec Kit convergence against current implementation; record actual results in specs/016-release-workflow-hardening/verification.md and convergence.md.
- [x] T021 Push/publish official PR, attach it, address every received finding in at most two reviews, require green exact final head and update specs/016-release-workflow-hardening/verification.md (FR-012, SC-005).
- [x] T022 Record complete acceptance/signing/acoustic/deployment/timing evidence and hand off owner review/merge through specs/016-release-workflow-hardening/launch.md (AC-012, SC-005).

## Dependencies and execution order

T001-T003 precede story implementation. Each story writes failing meaningful regressions before code. US1 provides actual reviewed package assets used by T017; US2 API tests can execute independently of live publication; US3 selector tests can use confined fixture packages. All stories precede final convergence and publication/review disposition. No live product tag/release/deployment occurs.

## Parallel opportunities

Read-only research for the three independent design boundaries ran concurrently. Independent Node/Python reads and verification may batch; file mutations and dependent checks remain ordered. Examples: US1 workflow versus native-helper checks; US2 Pages versus generic-host regressions; US3 inventory versus selector checks.

## Implementation strategy

Deliver shared helper and runtime lanes first, release-path reuse next, then Pages transport and extracted-package journey. Validate each boundary independently and run full native/package CI before the final owner handoff. No partial story or unrelated passing check satisfies completion.


## Phase 7: Measured source-build correction

- [x] T023 Preserve the full decoder/source/deadline contract while testing bounded Windows FFmpeg launch parallelism in scripts/media_source_build.py and tests/media_source_build_test.py; validate the actual cold result with T008 (FR-001, FR-003).

## Phase 8: Convergence

- [x] T024 Stage the complete Windows codec archive before final FFmpeg linking in scripts/media_source_build.py, scripts/media-transfer.py and .github/workflows/media-source.yml; bind unchanged configuration, compiler, source/recipe/launcher, dependency prefix, exact files and integer timestamps, reject tampering and resume without re-extraction or reconfiguration per FR-001/FR-002 (partial, HIGH).
- [x] T025 Validate the staged actual cold build and repeat full warm PR/release qualification with current source and exact receipts in specs/016-release-workflow-hardening/evidence/ per AC-001/AC-002 and SC-001/SC-002 (partial, HIGH).

## Phase 9: Convergence

- [x] T026 Preserve missing historical Next.js segment URLs only through exact verified flat-file replacements in scripts/pages.py and tests/pages_test.py; retain strict rejection of missing public pages/schemas, unmatched segments and ambiguous aliases, then prepare the actual release archive read-only per AC-005/FR-005 (partial, HIGH).
