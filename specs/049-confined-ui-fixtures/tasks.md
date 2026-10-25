# Tasks: S049 Confined UI Fixtures

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md).

## Phase 1: Setup

- [x] T001 Establish aligned S049 branch, feature selector and bounded requirements in specs/049-confined-ui-fixtures/spec.md; link #176 and milestone 33.

## Phase 2: Foundation

- [x] T002 Resolve ownership/platform unknowns and audit adjacent allocation sites in specs/049-confined-ui-fixtures/research.md; pass constitution and read-only cross-artifact analysis before implementation.

## Phase 3: US1 Confine temporary state (P1)

**Independent test**: Both tool allocators create real unique children beneath a nested spaces/Unicode parent; Windows drive/UNC and canonical root inputs fail without allocation.

- [x] T003 [US1] Add failing allocation/path tests in tests/capture-application-ui.test.mjs and tests/docs-ui.test.mjs (FR-001, FR-002, FR-006).
- [x] T004 [US1] Implement canonical non-root parent validation and retained parent/fixture identities in scripts/owned-ui-fixture.mjs; integrate both tool factories and source inventory (FR-001, FR-002, FR-009).

## Phase 4: US2 Release all owned state (P1)

**Independent test**: Real marker/profile failures remove owned state; changed markers/replacements preserve unrelated sentinels; forced removal failure names residual and cause.

- [x] T005 [US2] Add failing initialization, ownership, removal and handled-failure regressions in tests/capture-application-ui.test.mjs and tests/docs-ui.test.mjs (FR-003 through FR-006).
- [x] T006 [US2] Implement immediate cleanup responsibility, committed-marker verification, idempotent removal and residual diagnostics in scripts/owned-ui-fixture.mjs (FR-003 through FR-005).
- [x] T007 [US2] Integrate owned context/helper release, settled launch, cleanup publication gates and qualification-only location evidence in scripts/capture-application-ui.mjs and scripts/check-docs-ui.mjs (FR-004, FR-005, FR-009).

## Phase 5: Verification and delivery

- [x] T008 Extend actual Linux app/Windows docs browser coverage and workflow regressions in .github/workflows/native-qualification.yml, scripts/qualify-ui-capture.py, scripts/qualify-docs-ui.py and tests/ci-scope.test.mjs; retain source/capture/location/removal evidence (FR-007, SC-001, SC-004).
- [ ] T009 Update CHANGELOG.md and run root/affected gates; record audit and platform evidence in specs/049-confined-ui-fixtures/verification.md (FR-006 through FR-009, SC-002).
- [ ] T010 Commit, automatically push/open PR, require terminal exact-head CI, resolve every thread from at most two Codex review rounds, converge artifacts and hand off unmerged using specs/049-confined-ui-fixtures/verification.md (SC-003).

## Dependencies and Execution

T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008 -> T009 -> T010. US1 is the first testable increment; US2 builds on its allocation identity. Separate test files can be prepared/read in parallel, but shared helper/consumer edits are serialized. No parallel mutation is required. Implementation follows failing tests, minimal repair, green affected tests, then full CI and bounded external review. All ten tasks carry paths and map to requirements; no merge/release task exists.
