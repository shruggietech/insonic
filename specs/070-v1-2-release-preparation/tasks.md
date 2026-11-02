# Tasks: v1.2.0 release preparation and dependency qualification

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/release-preparation.md.

## Phase 1: Setup

- [x] T001 Align S070 branch, .specify/feature.json and specs/070-v1-2-release-preparation artifacts; verify existing .gitignore coverage.
- [x] T002 Complete plan research and requirements coverage analysis in specs/070-v1-2-release-preparation/verification.md before implementation.

## Phase 2: Foundational

- [x] T003 Run existing release/history and checksum-security regression baselines in tests/release_test.py, tests/*release_history_test.py, tests/webview-source.test.mjs and tests/go_module_notices_test.py; retain logs under .local/s070/.

## Phase 3: User Story 1 (P1), consistent candidate

Independent test: all owned current versions equal 1.2.0; historical trees match published tag; source authority remains intact.

- [x] T004 [US1] Extend current advanced_cli assertion with answers in tests/release_test.py; demonstrate failure before correction (FR-003).
- [x] T005 [US1] Correct current capability metadata in scripts/release.py and pass the regression without changing historical fixtures (FR-003).
- [x] T006 [US1] Apply x/net 0.62.0 in go.mod/go.sum while preserving four required Wails/WebView sums; adopt #246 Next 16.4.3 site/package.json and package-lock.json diff (FR-002).
- [x] T007 [US1] Record concise highlights in specs/070-v1-2-release-preparation/highlights.md and dated dependency/preparation decision in CHANGELOG.md; run scripts/release.py prepare for 1.2.0 (FR-001, FR-003).
- [x] T008 [US1] Restore docs/v1.1.0 and schemas/v1.1.0 plus historical navigation entry in docs/versions.json from tag authority after current content advances; validate new docs/v1.2.0 and schemas/v1.2.0 (FR-001).

## Phase 4: User Story 2 (P1), executable evidence

Independent test: original current capture provenance and audited mandatory package/backend/installer evidence match source/version.

- [x] T009 [US2] Save and temporarily omit new-version walkthrough image blocks in docs/v1.2.0/desktop.md and windows-installation.md; run bootstrap checks, commit/push and dispatch .github/workflows/ci.yml on clean branch (FR-004, FR-005).
- [x] T010 [US2] Retain/audit bootstrap acceptance and artifact digests under .local/s070/ before another full run; report separate timing with target1800/hard3600/reserve60 (FR-006).
- [x] T011 [US2] Import only original receipt-enumerated captures into docs/screenshots/v1.2.0; restore matching walkthroughs, validate source/version/hash/dimensions and visually inspect images (FR-005).
- [x] T012 [US2] Complete root/Python/Go/frontend/site checks, full docs archive/transfer validation and existing security/workspace isolation coverage; retain logs under .local/s070/ (FR-001, FR-002, FR-004).

## Phase 5: User Story 3 (P2), owner handoff

Independent test: final PR and exact-head CI receipts, reviews and project status establish merge readiness without publication.

- [x] T013 [US3] Converge spec/plan/tasks against implementation and evidence in specs/070-v1-2-release-preparation/verification.md; correct any concrete gaps (FR-001 through FR-009).
- [ ] T014 [US3] Commit/push complete candidate, open official non-draft PR, attach to chat and Project7/S070/In Review; record review round1 in .local/s070/reviews.json without new issue or premature dependency closure (FR-007, FR-008).
- [ ] T015 [US3] Respond to every finding, implement necessary fixes and resolve all review threads; request at most one second review and record actual rounds in .local/s070/reviews.json (FR-007).
- [ ] T016 [US3] Retain/audit final PR and direct-head runs; digest-verify complete candidate and load via scripts/release.py; inspect all four package answer campaigns and MSI/setup receipts in .local/s070/ (FR-004, FR-006, FR-007, FR-009).
- [ ] T017 [US3] Publish retained compact source-bound handoff receipt in PR conversation, read back its digest, verify final head/checks/reviews/project and clean source, then notify owner for merge (FR-007, FR-008, FR-009).

## Phase 6: Cross-cutting closeout

- [x] T018 Preserve UTF-8 without BOM/LF, check mojibake and final diff, and record performed versus unverified evidence in specs/070-v1-2-release-preparation/verification.md before final-head qualification.

## Dependencies and execution

T001-T003 -> US1 -> capture bootstrap/import -> local checks/convergence -> official PR/reviews -> final CI/candidate retention -> external handoff. T018 occurs before the final source commit; any later change repeats affected checks and T016 on the resulting head. T017's exact source/review outcome lives externally so no self-referential commit is needed.

Research agents run read-only in parallel. Independent root/frontend, Go/Python checks can run concurrently if mutable outputs do not overlap; site builds/archive and terminal audits remain sequential. US1 is the first independently testable increment; all three stories are required before owner handoff. Review markers are never self-checked as implementation proof. No new tracking issue, tag, merge or publication.

## Discovered qualification correction

- [x] T019 [US1] Reproduce stale answer request versions, extend scripts/release.py preparation for owned answer schemas/docs, and add tests/release_test.py regression; regenerate current answer authority (FR-001, FR-003).
