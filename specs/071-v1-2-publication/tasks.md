# Tasks: v1.2.0 publication and independent verification

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/publication.md.

## Phase 1: Setup

- [x] T001 Align S071 branch, specs/071-v1-2-publication and .specify/feature.json; verify ignore coverage.
- [x] T002 Complete research/design and blocking analysis in specs/071-v1-2-publication/verification.md.

## Phase 2: Foundational

- [x] T003 Run npm run check, npm test and existing release/Pages/public-verification/website/http-read guard regressions from tests/; retain .local/s071 logs before publication.
- [x] T004 Commit planning, switch to clean main, validate/trust exact product source, retain/audit main CI 41490507754 and run scripts/release.py pages-preflight into .local/s071 (FR-001, FR-003, FR-009).

## Phase 3: User Story 1 (P1), official downloads

Independent test: all ten original published files match accepted candidate names, nonnegative sizes and SHA-256 digests.

- [x] T005 [US1] Dispatch .github/workflows/release.yml preparation only with selected revision and release-target documentation, then await required terminal jobs (FR-002).
- [x] T006 [US1] Retain run/job/step/artifact metadata, verify original ZIP and confined extraction under .local/s071; compute truthful overlapping lane timing and hard-bound acceptance (FR-003).
- [x] T007 [US1] Run scripts/release.py exact_source/load_candidate plus require_release=True for both docs archives; inspect all four package answer/help receipts and exact MSI/setup joins in .local/s071 (FR-001, FR-002).
- [x] T008 [US1] Recheck main/checks/tag, tag selected source, and use scripts/release.py publish with retained bytes and specs/070-v1-2-release-preparation/highlights.md; retain actual publication readback (FR-004).
- [x] T009 [US1] Use scripts/release.py published into a fresh .local/s071 directory and independently compare all ten name/size/digest tuples against accepted bytes (FR-006).

## Phase 4: User Story 2 (P1), live documentation

Independent test: exact completed Pages deployment and all maintained public content/resource plus eight rendering checks pass.

- [x] T010 [US2] Use scripts/release.py pages-promote with explicit output receipt; retain exact staged/pushed/deployed identity, reconcile only exact uncertain state, and await the elected build (FR-005).
- [x] T011 [US2] Run scripts/release.py verify-public on independently downloaded candidate; materialize exact Pages tree as confined data and use scripts/website.py verify-deployed (FR-006).
- [x] T012 [US2] Run scripts/check-live-site-ui.mjs with installed full Chromium headlessly; inspect all eight screenshots and retain report/image hashes and cleanup proof under .local/s071 (FR-006).

## Phase 5: User Story 3 (P2), durable owner handoff

Independent test: evidence survives workflow retention, final checks/reviews agree with final head and PR remains for owner merge.

- [x] T013 [US3] Return to S071, retain compact original audited evidence in specs/071-v1-2-publication/evidence and verification.md; add truthful publication entry to CHANGELOG.md (FR-007).
- [x] T014 [US3] Converge spec/plan/tasks with delivered proof; run final root and affected site/regression checks and encoding/diff checks before final commit (FR-001 through FR-009).
- [ ] T015 [US3] Push, open official PR, attach to chat and Project7/S071/In review; keep review ledger in .local/s071 without creating an issue (FR-008, FR-009).
- [ ] T016 [US3] Resolve every actionable review, request at most one second round, and retain final applicable CI/terminal evidence in .local/s071 (FR-008).
- [ ] T017 [US3] Post compact final source-bound receipt in PR conversation, verify readback digest, final head/checks/reviews/project and clean checkout, then notify owner for merge (FR-007, FR-008).

## Dependencies and execution

T001-T004 -> US1 -> US2 -> evidence/convergence -> PR/reviews/CI -> retained owner handoff. Three read-only planning research agents run concurrently. Independent guard suites may run concurrently without shared outputs; mutations, candidate validation and deployment recovery remain sequential. Publication is the first independently testable outcome; all three stories are required. Final operational gates T015-T017 close in the PR conversation so committed artifacts do not assert their own future acceptance. No new issue or owner PR merge.
