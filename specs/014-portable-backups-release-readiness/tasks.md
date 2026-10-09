# Tasks: Portable backups and release readiness

**Input**: S014 spec, plan, research and contracts. **Date**: 2026-10-09

## Phase 1: Setup

- [x] T001 Establish literal S014 branch/feature and Spec Kit specify/clarify/checklist artifacts in specs/014-portable-backups-release-readiness/.
- [x] T002 Complete research and design for full artifact portability, corresponding sources, release promotion and taxonomy in plan.md, research.md, data-model.md and contracts/.

## Phase 2: Foundation

- [x] T003 Add transfer write fencing and receipt-preserving portable catalog authority in internal/catalog/backup.go and focused tests (FR-001, FR-004, FR-005, FR-007).
- [x] T004 [P] Define strict portable bundle manifest/completion and streamed artifact contracts in internal/backup/ (FR-001, FR-002, FR-003, FR-008).
- [x] T005 [P] Establish exact-source/version release validation and corresponding-source inputs in scripts/ and internal/qualification/media-tools.json (FR-010, FR-011, FR-015).

## Phase 3: US1 Complete recovery

**Independent test**: Source-store loss, current audio/transcript/model lineage survives restore.

- [x] T006 [US1] Add current/durable-authority filtering and provenance retention tests in internal/catalog/backup_test.go (FR-002, FR-003).
- [x] T007 [US1] Implement streamed self-contained copy, final manifest publication and verify in internal/backup/service.go (FR-001, FR-007, FR-008).
- [x] T008 [US1] Implement empty-target receipt-preserving relocation restore and graph checkpoint reset in internal/catalog/backup.go and internal/backup/service.go (FR-004, FR-005).
- [x] T009 [US1] Add offline CLI backup create/verify/restore/release and machine envelopes in cmd/insonic/backup.go; document advanced maintenance availability (FR-009).
- [x] T010 [US1] Add corruption/missing-byte/traversal/no-overwrite/interruption and exact metadata/lineage tests in internal/backup/ and cmd/insonic/ (FR-005, FR-007, FR-008).
- [ ] T011 [US1] Qualify alternative-backend restore and accepted-evidence graph rebuild using internal/backup integration fixtures and .github/workflows/ci.yml (FR-004, FR-016).

## Phase 4: US2 Reference lifetime

**Independent test**: Backup pins block cleanup until explicit release.

- [x] T012 [US2] Implement reference-only immutable dependencies, atomic durable retention pins and release in internal/catalog/backup.go and internal/backup/ (FR-006).
- [x] T013 [US2] Test cleanup fencing, stale transfer generation, failed preparation and source dependency loss in internal/catalog/ and internal/backup/ (FR-006, FR-007).

## Phase 5: US3 Native and release delivery

**Independent test**: Six native variants relocate successfully, complete sources and exact release identity validate.

- [ ] T014 [P] [US3] Build all-platform source-complete FFmpeg companions preserving required input capabilities in scripts/build-media-source.py and scripts/media-tools.py (FR-010).
- [x] T015 [US3] Add source/build inventories, CLI-only/desktop variants and configured native signing before hashes in scripts/package-desktop.py and package tests (FR-010, FR-012).
- [x] T016 [P] [US3] Implement owned version preparation, immutable historical docs/schema and comprehensive version checks in scripts/ and schemas/registry.go (FR-011, FR-015).
- [x] T017 [US3] Implement six-variant candidate manifests/checksums, integrity/source/signing rejection tests in scripts/ and tests/ (FR-011, FR-012).
- [x] T018 [US3] Implement draft-first complete publication/readback and configured exact-version docs promotion in .github/workflows/release.yml and scripts/ with failure tests (FR-013).
- [ ] T019 [US3] Qualify real media and relocated packages on all three operating systems with measured cache/build inputs in .github/workflows/ci.yml (FR-010, FR-016).

## Phase 6: US4 Tracking and final authority

**Independent test**: New canonical outcomes with historical edits/states preserved.

- [x] T020 [P] [US4] Align bootstrap canonical labels and new-outcome fields preserving history in .github/bootstrap.json and scripts/github-bootstrap.mjs (FR-014).
- [x] T021 [US4] Align issue forms and regressions for edited/closed/project-value preservation in .github/ISSUE_TEMPLATE/ and tests/ (FR-014).
- [x] T022 [US4] Audit and align public storage/release/import/model contracts, schema/help/glossary and CHANGELOG.md (FR-015).
- [x] T023 [US4] Reassess open dependency PRs #17/#23 and record tested disposition in research.md (FR-011, FR-016).

## Phase 7: Integration and handoff

- [x] T024 Run npm run check, npm test, full Go/vet, affected race/Python/frontend/site checks and quickstart validation; record evidence in verification.md (FR-016).
- [ ] T025 Run Spec Kit converge against all FR/acceptance/decisions, complete every identified gap and record evidence in convergence.md.
- [ ] T026 Commit/push, publish official PR with issue closures only for completed acceptance, satisfy every CI/review finding within two rounds and verify exact final head; stop for owner merge.

## Dependencies and parallel strategy

T001/T002 and analyze precede implementation. Backup foundation T003/T004 precedes US1/US2. Source/version foundations precede package/publication promotion. Independent backup, package and release/tracking owners may work in parallel after analyze, coordinated on shared files. Root integrates CLI/contracts/CI/docs. Local story checks precede initial branch/PR publication; live PR CI establishes supported-backend and all-platform qualification before final convergence and owner handoff. No public product tag/release/deployment occurs in this slice.

## Phase 8: Convergence

- [ ] T027 Complete actual alternative-backend backup/multipart/graph fixtures and both relocated native package variants on all three operating systems, then record exact revision evidence per FR-004, FR-010, SC-002 and SC-004 (partial).
- [ ] T028 Measure complete required-CI turnaround and cold/cached source stage durations, optimize any over-budget critical path without reducing checks or supported input capabilities, and record evidence per FR-016, SC-007 and Constitution V (partial).
