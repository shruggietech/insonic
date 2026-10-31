# Tasks: Backup/restore capacity-failure recovery

**Input**: [spec.md](spec.md), [plan.md](plan.md), research and contracts. S061 / #200.

## Phase 1: Setup

- [x] T001 Bind branch, selector, #200, milestone and Project 7 to `specs/061-backup-capacity-recovery/spec.md`.
- [x] T002 Complete clarify, recovery requirements review and research in `specs/061-backup-capacity-recovery/`.

## Phase 2: Foundation

- [x] T003 Add failing wrapped capacity, classification preservation and cleanup-redaction regressions in `internal/backup/capacity_test.go` (FR-001, FR-006).
- [x] T004 Centralize OS capacity recognition in `internal/contracts/capacity_*.go`; reuse from `internal/artifact/contract.go`; classify catalog capacity in `internal/catalog/store.go`; add generated cleanup reason to `internal/contracts/operator.go` and `schemas/v1.0.3/runtime-response.schema.json` (FR-001, FR-006).
- [x] T005 Implement private per-call file operations and checked finalization/cleanup in `internal/backup/io.go` (FR-002, FR-003).

## Phase 3: US1 - Incomplete backup (P1)

**Independent test**: Each creation fault refuses completion, preserves source/existing output, retains recoverable pins and completes identical retry after the fault clears.

- [x] T006 [US1] Add open/write/short-write/flush/close/publication/durability/cleanup failure and retry tests in `internal/backup/capacity_test.go` (FR-002, FR-003, FR-005; SC-001 through SC-003).
- [x] T007 [US1] Apply checked I/O and observed stage cleanup, correct publication classification and repeated retry durability in `internal/backup/service.go` (FR-002, FR-003, FR-005).

## Phase 4: US2 - Restore recovery (P1)

**Independent test**: Publication/progress/activation failure preserves bundle and original configuration, retains fencing and exact progress identity, and allows identical retry.

- [x] T008 [US2] Add confined catalog capacity rollback/retry in `internal/catalog/capacity_test.go` and restore publication/progress/activation/cleanup regressions in `internal/backup/capacity_test.go`, retaining occupied-target/tamper/backend tests (FR-004 through FR-006; SC-001 through SC-003).
- [x] T009 [US2] Apply checked progress, reference scratch and activation output plus observed cleanup in `internal/backup/progress.go` and `internal/backup/service.go` (FR-002 through FR-005).

## Phase 5: US3 - Bounded acceptance evidence (P2)

**Independent test**: Strict validator rejects missing/false/mislabeled cases and accepts actual source/package obstruction, exact retry and restored-byte evidence.

- [x] T010 [P] [US3] Write negative receipt tests in `tests/cli_backup_qualification_test.py` (FR-007, FR-008; SC-004).
- [x] T011 [US3] Add actual owned backup-parent/publication/progress obstruction and source-independent restore qualification in `scripts/cli_backup_qualification.py` (FR-003 through FR-008).
- [x] T012 [US3] Require `backup_contracts` from source/package receipt paths in `scripts/cli_contract_qualification.py`, `scripts/release.py` and affected validator tests (FR-007, FR-008; SC-004).

## Phase 6: Documentation, verification and handoff

- [x] T013 [P] Update authoritative `docs/v1.0.3/storage.md` and unreleased `CHANGELOG.md` with resulting behavior and dated tooling decision.
- [x] T014 Run root, affected Go/Python/site and encoding checks from `specs/061-backup-capacity-recovery/quickstart.md`; resolve all failures and complete Spec Kit convergence.
- [x] T015 Commit/push, open and attach official PR, monitor exact-head CI, resolve every review finding within two Codex rounds, retain/read back evidence and leave PR open for owner merge; record concise verification in `specs/061-backup-capacity-recovery/quickstart.md`.

## Dependencies and execution

T001-T005 precede stories. US1 and US2 share I/O helpers and service edits, so execute sequentially. US3 follows runtime regressions; its validator test file can be authored independently. Documentation can proceed alongside evidence integration. T014-T015 require all stories. Implement US1 as the first testable increment, then US2, then US3; every increment must preserve existing security and backend checks. No concurrent service edits are planned.

## Phase 7: Convergence

- [x] T016 Preserve capacity causes during filesystem/catalog initialization and observe initial SQLite file close in `internal/artifact/filesystem.go` and `internal/catalog/store.go`, using the existing tested classifiers; rerun affected initialization, capacity and recovery checks per FR-001 (partial, HIGH).
