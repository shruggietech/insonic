# Tasks: Local import storage-failure recovery

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [receipt contract](contracts/storage-receipt.md).
**Tests**: Required by this acceptance slice; reproduce faults before repairing them.

## Phase 1: Setup

- [x] T001 Specify/clarify bounded S058 scope and built-in quality in specs/058-import-storage-failures/spec.md and checklists/requirements.md (FR-009, SC-004).
- [x] T002 Complete research/design and tracking linkage, with confined .local/s058-work fixtures and verified native/media prerequisites (FR-007, FR-009, FR-010).

## Phase 2: Foundational

- [x] T003 Add failing-first injected capacity/short-write/flush/rewind/cleanup regressions in internal/artifact/storage_failure_test.go (FR-001, FR-003, FR-004).
- [x] T004 Repair demonstrated staging finalization/capacity classification in internal/artifact/stable.go, service.go, contract.go and capacity_*.go without product fault switches (FR-001, FR-003, FR-004, FR-006).

## Phase 3: US1 - Refuse incomplete authority

**Independent test**: Faults at source staging, canonical/report transfer and final persistence leave no new accepted recording, preserve source/sentinel bytes and retain uncertain recovery authority.

- [x] T005 [US1] Extend native publication partial-write/persistence coverage in internal/artifact/storage_failure_test.go using existing Store/persist boundaries (FR-001, FR-002, FR-004).
- [x] T006 [US1] Add canonical/report fault and sentinel/attempt integrity tests in internal/library/storage_failure_test.go (FR-001, FR-002, FR-003, FR-006; SC-001).

## Phase 4: US2 - Recover one verified admission

**Independent test**: Clearing the fault permits existing native reconciliation and one distinct ordinary public import, with idempotent replay and correct bytes/provenance.

- [x] T007 [US2] Prove cleared-fault and uncertain-publication recovery, metadata/date equality and duplicate refusal in internal/library/storage_failure_test.go (FR-005, FR-006; SC-002).
- [x] T008 [US2] Implement actual confined filesystem obstruction/removal and ordinary re-import in scripts/cli_storage_qualification.py, preserving failed-item/partial-batch semantics and marker-bound cleanup (FR-001 through FR-009; SC-001, SC-002).

## Phase 5: US3 - Trust bounded platform evidence

**Independent test**: Receipt omission/mutation, mislabeled physical exhaustion, stale identities and incomplete cleanup are rejected; actual source/package campaigns pass.

- [x] T009 [US3] Add failing receipt mutations and cleanup/blocker regressions in tests/cli_storage_qualification_test.py, then implement strict validation in scripts/cli_storage_qualification.py (FR-008, FR-009; SC-003).
- [x] T010 [US3] Integrate storage_contracts and corpus ownership in scripts/cli_contract_qualification.py and tests/fixtures/cli-corpus.json, updating affected receipt consumers (FR-007, FR-008; SC-003).
- [x] T011 [US3] Enforce modern storage receipt requirements with exact original-history compatibility in scripts/release.py and affected tests/release_test.py and tests/recovery_release_history_test.py (FR-010; SC-003).

## Phase 6: Verification and delivery

- [x] T012 Update current storage/development/release documentation, CHANGELOG.md and affected test-clock inventory, preserving public/internal planning boundaries (FR-003, FR-007, FR-008, FR-010).
- [x] T013 Complete focused/all affected native/Python, npm check/test and affected site verification; run local actual campaign and Spec Kit convergence; record concise results in specs/058-import-storage-failures/verification.md (SC-001 through SC-004).
- [ ] T014 Commit/push, open official PR, monitor/fix exact-head CI, resolve up to two Codex rounds, retain independently audited receipts, clean owned staging, and stop before owner merge (SC-004).

## Dependencies and execution

T001 -> T002 -> T003 -> T004. T005/T006 use distinct files but run serially here. T007 follows native failure acceptance. T009 defines receipt failures before T008/T010 integration. T011 follows the stable receipt contract. T012 -> T013 -> T014 completes delivery.

## Parallel examples

Native artifact and library test reviews can be inspected independently after T004. Python receipt mutation design can be prepared while native suites execute. Shared files and source edits remain sequential; no additional implementation agents are assigned.

## Implementation strategy

Complete native refusal first, then native recovery, then actual source/package evidence. Final completion requires every story, source-bound CI and bounded reviews; no release or merge is part of this slice.
