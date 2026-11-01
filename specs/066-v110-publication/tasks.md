# Tasks: v1.1.0 publication and independent verification

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [publication contract](contracts/publication.md).
**Tests**: Maintained repository and publication/security/integrity regressions plus complete fresh candidate qualification and independent public evidence. No new product behavior or prose-mirroring test is introduced.

## Phase 1: Setup

- [X] T001 Establish branch/selector, Spec Kit artifacts and issue #239/milestone 51/Project 7 identity in specs/066-v110-publication/ (FR-009).
- [X] T002 Complete clarification, built-in quality, reviewer checklist, researched plan, tasks and blocking analysis in specs/066-v110-publication/; leave custom checklist markers reviewer-owned (FR-001 through FR-010).

## Phase 2: Foundation

- [X] T003 Run npm run check and npm test in the foreground, retaining timestamped logs under .local/s066/ (FR-008, FR-009; SC-003).
- [X] T004 Run existing tests/release_test.py, tests/pages_test.py and tests/public_verification_test.py regressions without weakening negative security/integrity assertions, retaining .local/s066/ logs (FR-002, FR-004, FR-006, FR-007, FR-008; SC-003).
- [X] T005 Prepare confined retention/validation helpers under .local/s066/ and retain/audit exact-main CI 41270520010 separately through scripts/ci-acceptance.mjs, preserving source/workflow identities and complete original metadata (FR-001, FR-003, FR-008; SC-003).
- [X] T006 Commit planning, switch cleanly to main and run scripts/release.py validate/trusted/pages-preflight for accepted source 82a1bfd7e8f4e389d451789ec7840d874127f39c; record command/output identities under .local/s066/ (FR-001, FR-006, FR-008; SC-003).

## Phase 3: US1 - Official accepted release

Independent test: every official asset and the immutable tag identify the accepted source and match retained qualification bytes.

- [X] T007 [US1] Dispatch .github/workflows/release.yml for accepted source with publish=false, promote_docs=false and pages transport; monitor all elected lanes to terminal success and retain failures before any full rerun under .local/s066/ (FR-002, FR-003; SC-001, SC-003).
- [X] T008 [US1] Retain the unique unexpired release-candidate-ATTEMPT archive, original run/jobs/artifact metadata and complete ten-asset inventory under .local/s066/; require exact ZIP size/SHA256, confined regular paths, complete loader checks and release-target docs using scripts/release.py; record supported observed lane and combined timing (FR-001, FR-002, FR-003, FR-008; SC-001, SC-003).
- [X] T009 [US1] Recheck clean trusted main/source/check/tag authority, create immutable v1.1.0 at accepted source, publish only original retained bytes through scripts/release.py publish with specs/065-v110-release/highlights.md and retain exact remote release readback (FR-004, FR-005; SC-001).
- [X] T010 [US1] Independently retrieve all ten official assets using scripts/release.py published into a fresh .local/s066/ directory and compare every original/downloaded name, size and SHA256, retaining an explicit byte-comparison receipt (FR-003, FR-007; SC-001).

## Phase 4: US2 - Matching public documentation

Independent test: exact Pages commit and live bytes match original accepted documentation, with bounded fresh landing render evidence.

- [X] T011 [US2] Promote original published documentation via scripts/release.py pages-promote; retain parent/promoted/build identities and require exact commit completion under .local/s066/ (FR-006; SC-002).
- [X] T012 [US2] Run explicit scripts/release.py verify-public with the returned Pages commit and retain the complete live-resource/asset-metadata receipt under .local/s066/ (FR-003, FR-007; SC-002).
- [X] T013 [US2] Materialize exact Pages files as confined data and run scripts/website.py verify-deployed, retaining complete source/product/download identities under .local/s066/ (FR-006, FR-007, FR-008; SC-002).
- [X] T014 [US2] Run scripts/check-live-site-ui.mjs with the complete website receipt and explicit full GUI-subsystem Chromium in headless mode; require eight cases and cleanup to pass, inspect all captures and retain original render evidence under .local/s066/ (FR-007, FR-008, FR-010; SC-002).
- [X] T015 [US2] Record the stale qualification-timeout prose in docs/v1.1.0/releases.md as a separate scoped GitHub follow-up and reference it in specs/066-v110-publication/verification.md without altering accepted release bytes (FR-005, FR-010; SC-003).

## Phase 5: US3 - Auditable owner handoff

Independent test: source records, retained/public evidence, final CI and every review support one complete open PR for owner merge.

- [X] T016 [US3] Return to codex/066-v110-publication; record actual version/source/release/Pages IDs, phase timing, original artifact hashes, independent readback/render evidence and explicit limits in specs/066-v110-publication/verification.md, compact receipts and CHANGELOG.md (FR-003, FR-005, FR-010; SC-003).
- [ ] T017 [US3] Converge actual results against all requirements, acceptance scenarios, plan decisions and constitution principles using specs/066-v110-publication/tasks.md; append genuine gaps before delivery (FR-001 through FR-010; SC-001 through SC-004).
- [ ] T018 [US3] Complete affected foreground local checks, commit/push evidence, open and attach the official non-draft PR, link issue #239 and update Project tracking; record the automatic first review in .local/s066/reviews.json (FR-009; SC-004).
- [ ] T019 [US3] Resolve every actionable comment and individual thread using at most two triggers, preserving actual review receipts and absent security evidence in .local/s066/reviews.json; no third trigger is permitted (FR-009, FR-010; SC-004).
- [ ] T020 [US3] Require final applicable PR CI on the actual final evidence head, retain/audit its terminal receipt and original metadata under .local/s066/, and requalify after any subsequent source change (FR-003, FR-009; SC-003, SC-004).

## Phase 6: Closeout

- [ ] T021 Finalize task/verification state, read back retained compact receipts, verify final source/tracking/reviews/publication identities and publish the exact-source receipt in the PR before owner final-review/merge handoff; keep any final self-referential evidence external to source (FR-003, FR-009, FR-010; SC-001 through SC-004).

## Dependencies and execution

Setup -> local guards -> clean trusted main -> fresh release preparation -> retained candidate acceptance -> immutable publication/public retrieval -> exact documentation promotion -> independent byte/render verification -> evidence and convergence -> PR/reviews/final CI -> owner handoff. T020 is repeated after any later evidence commit; final external receipts avoid embedding a commit's own hash inside itself. An accepted publication candidate is never rebuilt for publication.

## Parallel opportunities

Planning research is read-only across publication, workflow retention and public verification. Independent root and Python regression commands may run concurrently in the foreground if outputs do not overlap. Release platform jobs parallelize through maintained workflows. Dependent tag/release/Pages mutations and shared-terminal audits remain sequential. Review and CI observation may overlap after PR publication.

## Implementation strategy

US1 delivers verified official bytes, US2 verifies their matching public documentation, and US3 completes durable evidence and bounded review. Complete all three under the owner's explicit authorization. No new local runtime feature or unattended real-engine campaign is inferred.
