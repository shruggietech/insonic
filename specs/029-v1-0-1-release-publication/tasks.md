# Tasks: S029 v1.0.1 release and documentation publication

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/publication.md.
**Tests**: Meaningful regression-first discipline; all existing security/tenancy/source/publication guards remain mandatory.
**Organization**: Chronological pre-merge implementation, owner merge gate, then post-merge publication. PR readiness does not complete #97.

## Phase 1: Setup

- [x] T001 Confirm clean base/branch/selector, current constitution, hidden execution boundaries, #97 milestone/Project S029 In progress and absent conflicting release/tag; record in specs/029-v1-0-1-release-publication/verification.md.
- [x] T002 Complete installed Spec Kit specify/clarify/checklist/plan research and design artifacts under specs/029-v1-0-1-release-publication/.

## Phase 2: Foundational gate

- [x] T003 Run installed strict read-only analyze across specs/029-v1-0-1-release-publication/spec.md, plan.md and tasks.md; resolve critical findings outside analysis before implementation and record the report afterward.

## Phase 3: US1 accurate release cut (P1)

Independent test: truthful historical/current status across both targets, one complete dated changelog and synchronized current references.

- [x] T004 [US1] Add failing historical released/current candidate rendering and immutable-input regressions in site/tests/routes.test.mjs or a focused site/tests/release-status.test.mjs.
- [x] T005 [US1] Correct known-published 1.0.0 status in docs/versions.json and render-only released-entry index text for all entries in site/lib/docs.mjs and site/lib/release-status.mjs; retain current candidate snapshot semantics and frozen historical bytes.
- [x] T006 [US1] Consolidate root CHANGELOG.md into one 1.0.1 section dated 2026-10-15 with chronological decisions; empty Unreleased, refresh docs/v1.0.1/references/CHANGELOG.md and prepare specs/029-v1-0-1-release-publication/release-highlights.md (<=180 words ending in exact-tag master changelog).
- [x] T007 [US1] Update CONTRIBUTING.md current-contract links and docs/v1.0.1/references/CONTRIBUTING.md; verify existing 1.0.1 software/schema/package identities without rewriting capture inputs or historical records.

## Phase 4: US2 publication qualification (P1)

Independent test: existing mock-source/draft/asset/security suites plus structural qualification ordering prevent weaker publication.

- [x] T008 [US2] Add failing mandatory release-target site/diagram/UI qualification and fenced report-retention regressions in tests/release-versions.test.mjs.
- [x] T009 [US2] Add existing site tests, diagram rendering and bounded headless UI qualification before archival in .github/workflows/release.yml, with source/attempt-fenced documentation-ui evidence; preserve all privileged/job/cache/660-second boundaries.
- [x] T010 [US2] Record exact post-merge tag/source/trusted-main dispatch, draft interruption recovery, published-original-archive promotion and nine-asset verification in specs/029-v1-0-1-release-publication/contracts/publication.md and verification.md; preserve scripts/release.py guards.

## Phase 5: US3 documentation verification before merge (P2)

- [x] T011 [US3] Run snapshot and release-target public/offline builds, actual diagram and headless UI qualification, inspect representative screenshots and report identities/cleanup in specs/029-v1-0-1-release-publication/verification.md; preserve historical hashes and capture/brand integrity.

## Phase 6: Integration and owner handoff

- [x] T012 Run npm run check, npm test, site/Python release/Pages checks and Spec Kit convergence; inspect UTF-8/LF/mojibake and record analysis/convergence/results under specs/029-v1-0-1-release-publication/.
- [x] T013 Commit/push codex/029-v1-0-1-release-publication and open/attach the official PR with non-closing #97 reference, milestone and Project In review; record URL in specs/029-v1-0-1-release-publication/verification.md.
- [x] T014 Address every review comment and resolve all threads, counting automatic opening round and at most one manual second request; record in specs/029-v1-0-1-release-publication/verification.md and official PR.
- [ ] T015 Verify exact final-head elected CI green and compare full elapsed turnaround against 660 seconds, directly audit all review threads and ping owner for final review/squash merge; record post-commit results on official PR rather than infer them from this source snapshot.

## Phase 7: US2 after owner merge only (P1)

- [ ] T016 [US2] Verify owner-merged exact squash-main source, clean checkout, green source checks and immutable tag absence/identity; create/push v1.0.1 at that source and record identity in specs/029-v1-0-1-release-publication/verification.md or the delivery PR/issue checkpoint.
- [ ] T017 [US2] Dispatch trusted-main .github/workflows/release.yml for fresh exact-source qualification and verified draft publication with publish/promote_docs true, existing Pages and reviewed highlights; verify four archives, one MSI, both docs bundles, release manifest and checksums.

## Phase 8: US3 after publication (P2)

- [ ] T018 [US3] Verify original published documentation digest and exact Pages commit/live marker, manifest statuses, latest/fixed/archive/schema routes, all download links, company attribution and headless mobile/theme evidence per contracts/publication.md; retry only original published-asset promotion if needed.

## Phase 9: Delivery closeout

- [ ] T019 Update #97 acceptance and Project Done only after actual verified release/deployment; record published/deployed facts and limitations on the official PR/issue and ping owner.

## Dependencies and parallel opportunities

T001–T003 gate implementation. T004 precedes T005; T006/T007 use current release source and shared snapshots sequentially. T008 precedes T009 and can be independently validated from US1. T010 documents existing contracts without new product code. T011 integrates both stories. T012–T015 precede owner merge. T016–T019 are explicitly prohibited before owner merge and form the post-merge continuation of this same slice.

Research agents ran in parallel with disjoint read-only scopes. Root owns integration and implementation; no overlapping file mutations are planned. Focused site and workflow checks can run independently, while snapshot/release builds sharing output directories must be sequential.

## Implementation strategy and completion

All three stories are required for #97. Preparation is a reviewable intermediate delivery; do not close #97 or call publication complete at T015. Custom release-quality checklist markers are reviewer-owned requirements questions, not implementation completion or an extra human gate. Source checklists are a pre-CI snapshot; final post-commit and post-publication evidence lives on the official delivery PR/issue.
