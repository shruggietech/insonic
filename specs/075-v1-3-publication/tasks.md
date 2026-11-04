# Tasks: v1.3.0 publication and verification

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/publication.md.

## Phase 1: Setup

- [x] T001 Reconcile merged source, obsolete branch/selector and completed tracking; record in specs/075-v1-3-publication/verification.md. [FR-001, FR-010]
- [x] T002 Complete Spec Kit design and read-only analysis in specs/075-v1-3-publication/. [FR-008]

## Phase 2: Foundation

- [x] T003 Run root checks and existing release/Pages/public-verification/website/HTTP guard tests; retain .local/s075/ receipts. [FR-002, FR-010]
- [x] T004 Commit planning, switch to clean selected main, retain successful main CI and run scripts/release.py pages-preflight. [FR-001, FR-002]

## Phase 3: US1 - Official release (P1)

Independent test: every official download matches the retained accepted candidate by name, size and SHA-256.

- [x] T005 [US1] Prepare .github/workflows/release.yml with exact source and publication disabled; retain terminal lane times and original candidate in .local/s075/. [FR-002, FR-003]
- [x] T006 [US1] Validate all ten original files and both release-status documentation archives with scripts/release.py; require exact source and immutable identity. [FR-002, FR-005]
- [x] T007 [US1] Run scripts/qualify-speakers.py real CPU/NVIDIA against original Windows CLI archive and receipt; retain complete metrics, readiness/timing and cleanup in .local/s075/. [FR-004]
- [x] T008 [US1] Recheck source checks/tag conflicts and publish accepted bytes unchanged with scripts/release.py and S074 highlights. [FR-005]
- [x] T009 [US1] Independently download all ten files with scripts/release.py published and retain exact comparison in .local/s075/public-download-comparison.json. [FR-007]

## Phase 4: US2 - Live documentation (P1)

Independent test: exact completed Pages deployment passes live content/routes/downloads and all eight maintained visual cases.

- [x] T010 [US2] Promote original documentation using scripts/release.py pages-promote; retain exact attempt and bounded reconciliation if needed in .local/s075/. [FR-006]
- [x] T011 [US2] Run scripts/release.py verify-public and scripts/website.py verify-deployed against independently downloaded bytes and exact Pages commit. [FR-007]
- [x] T012 [US2] Run scripts/check-live-site-ui.mjs with hidden full Chromium and inspect eight screenshots; retain report/hashes in .local/s075/live-ui/. [FR-007, FR-010]

## Phase 5: US3 - Auditable handoff (P2)

Independent test: durable original-source evidence, review ledger and final terminal CI reconcile with the evidence PR.

- [x] T013 [US3] Return to S075; retain compact audited receipts and limitations in specs/075-v1-3-publication/evidence/ and verification.md; record actual publication in CHANGELOG.md. [FR-008]
- [x] T014 [US3] Converge artifacts, run applicable checks and scan UTF-8/LF/mojibake and outbound mention text before push. [FR-008, FR-009, FR-010]
- [ ] T015 [US3] Push/open official evidence PR, attach Project 7 metadata, resolve every finding within two review rounds and retain final CI acceptance outside workflow storage. [FR-009]

## Phase 6: Handoff

- [ ] T016 Independently read back release, documentation, durable evidence and final PR state; report owner merge boundary in specs/075-v1-3-publication/verification.md and final response. [FR-005, FR-006, FR-009, FR-010]

## Dependencies and execution

T001-T004 precede US1. US1 publishes before US2 promotes. US3 records actual US1/US2 outcomes before final PR. Operational mutations are sequential. Read-only asset comparisons may run concurrently after publication; independent route and asset reads may overlap within maintained verification; final CI and external review may run concurrently. These are execution opportunities, not independent write tasks.

The first increment is independently verified release downloads. Continue through live documentation and reviewed evidence to fulfill the owner request; do not stop after the first increment. All identity mismatches fail acceptance, uncertain states retain diagnostic receipts, and a full rerun requires prior retention.
