# Tasks: Speaker quality and release qualification

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/evaluation.md.
**Tests**: Failing-first meaningful corpus, measurement, election and receipt regressions; maintained security/integrity suites remain mandatory.

## Phase 1: Setup

- [x] T001 Establish S074 identity, specification, clarification, reviewer checklist and research/design in specs/074-speaker-release-qualification/; associate existing #233/#211 with S074/In progress.
- [x] T002 Complete read-only analyze gate and validate ignored local work/evidence paths in .gitignore and specs/074-speaker-release-qualification/verification.md (FR-007, FR-011).

## Phase 2: Foundation

- [x] T003 Record current executable/tool/model/corpus prerequisites and evidence identity plan in specs/074-speaker-release-qualification/verification.md; keep preparation distinct from inference (FR-001, FR-004, FR-007, FR-010).

## Phase 3: User Story 1 - Measured speaker quality

**Goal**: Account for every contextual/acoustic outcome without overstating quality.
**Independent test**: Deterministic scorer assertions plus real installed contextual/CPU/CUDA receipts.

- [x] T004 [US1] Write failing corpus/scoring tests in tests/speaker_reference_test.py for malformed/duplicate truth, unknown identities, false-positive precision, coverage, terminal errors and zero denominators (FR-001, FR-003, SC-001).
- [x] T005 [US1] Freeze eight cases in tests/fixtures/speakers/manifest.json and implement strict bounded validation/metrics in scripts/speaker_reference.py; every metric has numerator/denominator and null when denominator is zero (FR-001, FR-002, FR-003).
- [x] T006 [US1] Write failing tests in tests/speaker_quality_test.py for election, exact model identity, missing cases, source separation, explicit actual CPU/CUDA and fallback rejection (FR-004, FR-005, FR-007).
- [x] T007 [US1] Implement installed contextual fixture/provider execution and complete terminal case accounting in scripts/speaker_quality.py and scripts/qualify-speakers.py (FR-002, FR-003, SC-001).
- [x] T008 [US1] Implement real product enrollment/matching on frozen licensed evidence with explicit device, source/model identity and actual provenance in scripts/speaker_quality.py, preserving scripts/journey_cpu.py historical authority (FR-001, FR-004, FR-005, SC-002).
- [ ] T009 [US1] Execute model-free installed evaluation and explicit real contextual/CPU/CUDA qualification; retain audited receipts and limitations under specs/074-speaker-release-qualification/evidence/ (FR-003, FR-004, FR-010, SC-001, SC-002).

## Phase 4: User Story 2 - Complete configured workflow

**Goal**: Demonstrate configured autonomy and correction/recovery without self-certifying evidence.
**Independent test**: Installed speaker workflow and default/alternative backend receipts.

- [ ] T010 [US2] Run targeted current-authority, mixed-speaker, independent/circular and correction/retraction tests in internal/catalog, internal/app and internal/voicemodels; add failing-first fixes only for observed gaps (FR-005, FR-006, FR-007, SC-003).
- [ ] T011 [US2] Verify scripts/cli_speaker_workflow_qualification.py and scripts/qualify-answer-backends.py terminal installed workflow/restart results on default and PostgreSQL/S3/ArcadeDB CI, retaining their model-quality limits (FR-006, SC-003).
- [ ] T012 [US2] Demonstrate integrated configured contextual discovery with independently mapped acoustic enrollment sources and verify correction invalidates current profile use; retain evidence in specs/074-speaker-release-qualification/evidence/ (FR-005, FR-006, FR-007).

## Phase 5: User Story 3 - Release candidate

**Goal**: Prepare one compatibility-reviewed candidate with genuine current documentation.
**Independent test**: Aligned versions, historical tree equality, capture proofs and exact-source candidate validation.

- [x] T013 [US3] Reconcile current speakers/voice-models/pipelines/development documentation, catalog upgrade limits and CHANGELOG.md; write specs/074-speaker-release-qualification/highlights.md ending in changelog link (FR-008, FR-009).
- [x] T014 [US3] Run scripts/release.py prepare for 1.3.0, preserve discovery protocol version 1, restore tagged docs/v1.2.0 and schemas/v1.2.0 and released navigation in docs/versions.json; verify owned version migration (FR-008, SC-004).
- [ ] T015 [US3] Push a clean hosted-capture bootstrap with temporary new-version image omissions; retain audited terminal CI evidence before reruns in specs/074-speaker-release-qualification/evidence/ (FR-009, FR-010, FR-011).
- [ ] T016 [US3] Import 25 original screenshots/four unchanged capture receipts into docs/screenshots/v1.3.0, restore current walkthrough embeds, inspect captures and verify site provenance (FR-009, SC-004).
- [ ] T017 [US3] Validate final exact-source ten-file release candidate with scripts/release.py; retain original assets and compact acceptance proof (FR-008, FR-010, SC-004).

## Phase 6: Cross-cutting verification and handoff

- [ ] T018 Run npm run check/npm test, full Go tests/vet/module verification, Python discovery, frontend check/test/build and site checks/tests/build; retain terminal results in specs/074-speaker-release-qualification/verification.md (FR-010).
- [ ] T019 Run Spec Kit convergence against specs/074-speaker-release-qualification/spec.md, plan.md and tasks.md; implement any appended gaps and repeat until no buildable gaps remain.
- [ ] T020 Commit/push final implementation and open official PR with #233/#211, S074 and Project 7; resolve all external findings with at most two review rounds (FR-011, FR-012).
- [ ] T021 Audit green terminal PR and exact-branch CI, candidate bytes and review threads; reconcile issue/epic criteria in the PR acceptance packet and hand off for owner review/merge (FR-010, FR-011, FR-012, SC-005).

## Dependencies and execution order

Setup/analyze precede foundation and implementation. US1 tests precede implementation and real runs. US2 reuses merged workflows and can be verified independently; T012 depends on real enrollment readiness. T014 version preparation may follow T008 before the final T009 campaign so real receipts bind the release version. US3 follows harness/product changes, then captures, complete checks and candidate validation. T017 and T021 consume final terminal CI, so their authoritative completion is attached to the PR after source commits.

Read-only research ran in parallel as required by the plan skill. Independent local checks may run concurrently within resource limits; acoustic CPU/CUDA and contextual inference run sequentially to avoid device contention. File mutations remain ordered. No new tracking issue is created.

## Implementation strategy

Deliver deterministic corpus/scoring first, then installed real measurements, then release preparation/capture and final acceptance. Report observed errors rather than silently replacing failed cases. Preserve failed receipts before reruns. Existing external acceptance comments close CI-dependent tasks without self-referential future-source claims.
