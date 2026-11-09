# Tasks: S077 v1.3.1 publication

## Phase 1: Setup

- [x] T001 Create specification, clarification and requirements checklists in specs/077-v131-publication/ (FR-009).
- [x] T002 Complete research and design in specs/077-v131-publication/plan.md, research.md, data-model.md, contracts/publication.md and quickstart.md (FR-001..FR-010).

## Phase 2: Foundation

- [x] T003 Complete blocking analysis and existing local publication/integrity guard checks; retain .local/s077 guard receipts (FR-008/FR-009, SC-004).
- [ ] T004 Wait for exact merged-main CI 42537775325 and retain its terminal audited metadata/receipts outside Actions in .local/s077 before release preparation (FR-001, SC-004).
- [ ] T005 Verify clean exact source, trusted current-main history, release/tag conflicts and existing Pages preflight through scripts/release.py; retain .local/s077 preflight receipts (FR-001/FR-004/FR-005).

## Phase 3: User Story 1 (P1)

**Goal**: Qualified exact-source public v1.3.1 downloads.

**Independent test**: Complete accepted ten-file candidate, installed seven-case answers, public saved-byte readback.

- [ ] T006 [US1] Dispatch existing .github/workflows/release.yml preparation with publish=false/promote_docs=false/pages and selected source; retain terminal metadata/timing and original artifacts in .local/s077 (FR-002/FR-008, SC-004).
- [ ] T007 [US1] Verify original API archive digest/size, confined extraction and exact-source ten-file candidate with both release-target docs archives through scripts/release.py; require four package catalog_startup_contracts/coverage results, and retain .local/s077/candidate-acceptance.json (FR-002/FR-003, SC-001/SC-002).
- [ ] T008 [US1] Copy-install original Windows CLI, verify full inventory/source/model blobs and run scripts/qualify-answers.py for all seven frozen cases; retain complete identities, null/not-applicable metrics, phase timing and owned cleanup under .local/s077 (FR-003/FR-008, SC-002/SC-004).
- [ ] T009 [US1] Recheck source checks and immutable tag, create/push v1.3.1 at selected source and publish unchanged accepted candidate using scripts/release.py and current highlights; retain publication receipt (FR-004, SC-001).

## Phase 4: User Story 2 (P2)

**Goal**: Matching released documentation and independent public proof.

**Independent test**: Exact completed Pages commit, complete downloaded/deployed inventories and eight sampled browser cases.

- [ ] T010 [US2] Promote original published documentation via scripts/release.py pages-promote with unique .local/s077 attempt; require completed exact Pages build or original-identity reconciliation (FR-005, SC-003).
- [ ] T011 [US2] Freshly download all public files through scripts/release.py published, compare every original name/size/digest and retain .local/s077 public-download proof (FR-006, SC-001).
- [ ] T012 [US2] Run verify-public, scripts/website.py verify-deployed against retained exact Pages tree, and scripts/check-live-site-ui.mjs in full headless Chromium; inspect all eight screenshots and retain complete .local/s077 content/render/cleanup receipts (FR-007, SC-003).

## Phase 5: Evidence and delivery

- [ ] T013 Retain compact digest/decompression-verified originals in specs/077-v131-publication/evidence/, write verification.md and successful publication history in CHANGELOG.md (FR-008/FR-009, SC-004).
- [ ] T014 Complete conformance/convergence, root/site/affected checks and UTF-8/LF/mojibake sanity without rebuilding published assets; retain specs/077-v131-publication/verification.md (FR-009, SC-004).
- [ ] T015 Commit/push one official evidence PR and refresh milestone/Project metadata, preserving selected product source and published assets (FR-010, SC-005).
- [ ] T016 Address all received review comments within at most two rounds; retain exact evidence-head green terminal CI and final owner handoff in PR conversation (FR-010, SC-005).

## Dependencies and parallel work

T001 -> T002 -> T003 -> T004/T005 -> T006 -> T007/T008 -> T009 -> T010 -> T011/T012 -> T013 -> T014 -> T015 -> T016. Selected source is exactly 1443198e0dec4738c129bb6dfe48baba9830d6e3. The ten-file candidate is immutable after acceptance. Research delegated under speckit-plan; product operations remain serial where state/authority depends on earlier outcomes. Independent local guard suites and retained-file comparisons may run concurrently. Actual-model qualification may overlap remaining hosted preparation only if the original final candidate is independently proven byte-identical before publication.

The full slice ships both stories; no partial-publication closeout. Final operational exact-head PR gates close externally to avoid self-referential acceptance commits.
