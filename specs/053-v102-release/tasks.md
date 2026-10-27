# Tasks: S053 v1.0.2 release preparation

## Phase 1: Setup

- [x] T001 Establish matching S053 branch, selector and planning directory in specs/053-v102-release/.
- [x] T002 Create issue #184, milestone 37 and Project 7 tracking for specs/053-v102-release/spec.md.

## Phase 2: Foundation

- [x] T003 Analyze spec.md, plan.md and tasks.md against .specify/memory/constitution.md before source edits.
- [x] T004 Record the original v1.0.1 package revalidation limitation separately from current acceptance in specs/053-v102-release/verification.md and GitHub tracking.

## Phase 3: US1 - Coherent candidate

Independent test: maintained version checks identify unpublished v1.0.2; original published schemas and historical navigation retain their authority.

- [x] T005 [US1] Prepare concise highlights in specs/053-v102-release/highlights.md and run scripts/release.py prepare for v1.0.2 dated 2026-10-27.
- [x] T006 [US1] Reconcile docs/versions.json and original published docs/v1.0.1/ and schemas/v1.0.1/ after preserving current corrections in v1.0.2.
- [x] T007 [US1] Verify preparation, historical schema authority and current owned bindings through tests/release_test.py and maintained root checks.

## Phase 4: US2 - Accurate release documentation

Independent test: current public/offline pages render matching verified captures and all six current product downloads; historical routes remain available.

- [x] T008 [US2] Cover the complete six-asset release link contract in tests/site-release.test.mjs, including setup EXE and MSI.
- [x] T009 [US2] Correct package-count and host-count descriptions in docs/v1.0.2/releases.md and roadmap.md; temporarily omit new-version screenshot sections for a clean capture bootstrap.
- [x] T010 [US2] Commit and push the clean bootstrap, open the official PR, and collect genuine application/density and disposable Windows installer captures via .github/workflows/native-qualification.yml.
- [x] T011 [US2] Validate capture hashes/source authority, import unchanged images/receipts into docs/screenshots/v1.0.2/ and restore all 20 embeds in docs/v1.0.2/desktop.md and windows-installation.md.
- [x] T012 [US2] Run site tests/build/export and inspect current and historical documentation through site/lib/docs.mjs and maintained validators.

## Phase 5: US3 - Qualified owner handoff

Independent test: the final PR head has terminal passing elected checks, zero unresolved actionable reviews and at most two Codex rounds; no merge/tag/release/deployment occurred.

- [x] T013 [US3] Complete root check/test, affected Python suites and final Windows/Linux source/native/package/installer CI; retain separate phase accounting in specs/053-v102-release/verification.md.
- [x] T014 [US3] Resolve every review comment, request at most one second Codex round, and retain review receipts through the official PR.
- [x] T015 [US3] Finalize executable post-merge exact-main CPU/restore/publication guidance in specs/053-v102-release/quickstart.md and concise highlights.md.

## Phase 6: Polish and delivery

- [x] T016 Converge actual implementation against all requirements and tasks in specs/053-v102-release/.
- [x] T017 Retain audited final-head CI/review receipts, update issue #184 and Project 7 to In review, and remove only .local/s053-work after readback.
- [x] T018 Verify clean final source, unmerged PR and no publication actions; hand off for owner final review with specs/053-v102-release/verification.md.

## Dependencies and execution

Setup -> foundation -> US1 -> US2 bootstrap/capture -> final US2 rendering -> US3 -> polish. The US3 release guide can be refined while CI runs. Within US1, read-only historical comparisons and current version checks can run together; within US2, link tests and prose inspection can run together; within US3, independent root/Python/site checks can run together after source readiness. No concurrent edits to the same files. The first independently reviewable increment is US1 plus the clean US2 capture bootstrap, followed by genuine captures and final qualification.
