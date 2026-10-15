# Tasks: Release pipeline reliability

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/.

## Phase 1: Setup

- [x] T001 Define scope and requirements in specs/030-release-pipeline-reliability/spec.md and link issues #102/#103 to milestone and Project.
- [x] T002 Resolve bootstrap and publication trust decisions in specs/030-release-pipeline-reliability/research.md and plan.md.

## Phase 2: Foundations

- [x] T003 Write failure-first bootstrap fixtures in tests/bootstrap_actions_test.py and workflow contracts in tests/bootstrap-actions.test.mjs.
- [x] T004 Write failure-first public-content and read-only API fixtures in tests/public_verification_test.py.

## Phase 3: US1 - Recover bootstrap downloads

**Independent test**: retry, deadline, integrity and extraction fixtures, plus
real local-action/cache main/post execution on both hosted qualification OSes.

- [x] T005 [US1] Implement bounded locked acquisition in scripts/bootstrap-actions.py and scripts/bootstrap-actions.lock.json (FR-001/002).
- [x] T006 [US1] Replace setup action acquisition in .github/workflows/ci.yml, native-qualification.yml, release.yml and other consumers, preserving inputs/conditions and workflow-revision historical tooling (FR-003/004).
- [x] T007 [US1] Verify archive acquisitions and focused contracts in tests/bootstrap_actions_test.py and tests/bootstrap-actions.test.mjs.

## Phase 4: US2 - Verify public publication

**Independent test**: original archive fixture and GET-only fake API; stale
bytes, alias, inventory, redirect, deployment race and identity failures.

- [x] T008 [US2] Implement bounded original-archive/live-content verification in scripts/public-verification.py, reusing scripts/pages.py archive checks (FR-005/007/009).
- [x] T009 [US2] Add verify-public command and post-Pages integration in scripts/release.py, retaining all candidate/tag/asset validation (FR-006/008).
- [x] T010 [US2] Run focused tests and read-only existing-publication verification; record exact proof levels in specs/030-release-pipeline-reliability/verification.md.

## Phase 5: Polish and delivery

- [x] T011 Document operational limits, proof levels, lock maintenance and recovery in docs/v1.0.1/releases.md and dated decisions in CHANGELOG.md (FR-011).
- [x] T012 Run full local checks, site build and Spec Kit convergence; record coverage in specs/030-release-pipeline-reliability/verification.md (FR-010).
- [ ] T013 Push the official PR, verify exact-head green CI within 660 seconds and resolve every review thread in at most two rounds; track final evidence on the PR and issues (FR-010).

## Dependencies and parallel opportunities

Setup -> foundations -> US1/US2 -> polish/delivery. US1 and US2 have independent
implementation files; their failure fixtures can be developed separately after
design. Workflow integration follows bootstrap tests; Pages integration follows
public-verifier tests. This execution uses sequential edits to shared files.

## Implementation strategy

Deliver bounded bootstrap first, then reusable publication proof. Preserve every
existing gate and immutable release. Stop only for final owner merge after PR
CI and reviews satisfy the done gate.

## Phase 6: Convergence

- [x] T014 [US2] Close FR-007/009 URL-selection gap in scripts/public-verification.py and tests/public_verification_test.py: generic destinations with a path prefix must verify actual resolved navigation/resource URLs, including root-relative resources, rather than only reconstructing URLs from archive filenames. Keep all requests on the elected origin.
