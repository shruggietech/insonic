# Tasks: S051 Restored receipt recording authority

Input: [spec.md](spec.md), [plan.md](plan.md), [data-model.md](data-model.md). Test-first execution; #180 under #147.

## Phase 1: Setup

- [x] T001 Align S051 branch/selector, quality checklist, milestone 35 and Project 7 through specs/051-restored-receipt-authority/spec.md and plan.md (FR-008).

## Phase 2: Foundation

- [x] T002 Recover and verify immutable retained receipt into owned temporary storage; capture its raw digest and original identities for specs/051-restored-receipt-authority/verification.md (FR-006).

## Phase 3: US1 - Explicit authority (P1, MVP)

Independent test: valid restored receipt passes; every missing/malformed/duplicate authority and invalid revision ordering is refused.

- [x] T003 [US1] Add failing source identifier/revision, enrollment duplicate and destination lineage mutations in tests/journey_restored_test.py; enforce "canonical UUID string, unique across all six source cases", "exact integer type and is >=1" and "source revision <= prior_revision < revision" (FR-001..004, SC-001).
- [x] T004 [US1] Guard complete source authority before set membership in scripts/journey_cpu.py and remove its fabricated source revision fallback; preserve maintained ValueError refusal (FR-001..004).

## Phase 4: US2 - Evidence continuity (P2)

Independent test: unchanged legacy v1 and hash-verified actual v2 pass; dated S050 reconciliation preserves the event sequence and remaining historical limitations.

- [x] T005 [US2] Keep explicit legacy source-only and newer-prior positive controls in tests/journey_restored_test.py; validate unchanged actual receipt with corrected scripts/journey_cpu.py and record current validator hashes separately in specs/051-restored-receipt-authority/verification.md (FR-005,006, SC-002).
- [x] T006 [US2] Append chronological evidence/reconciliation to specs/050-restored-cpu-processing/tasks.md and verification.md, identifying completed acceptance and owner merge before unfinished final convergence/handoff (FR-007).

## Phase 5: Verification and delivery

- [x] T007 Run root and affected Python/site checks, update CHANGELOG.md and specs/051-restored-receipt-authority/verification.md, and complete Spec Kit convergence against both slices' bounded implementation requirements (FR-007,008, SC-003).
- [x] T008 Commit/push/open official PR for #180, monitor elected final-head CI and resolve all review threads within two Codex rounds; retain compact receipts outside runs before full reruns using specs/051-restored-receipt-authority/verification.md (FR-008, SC-003).
- [x] T009 Verify final source/check/review/tracking state, retain evidence, remove owned scratch and provide the unmerged owner handoff recorded in specs/051-restored-receipt-authority/tasks.md (FR-006,008, SC-003).

## Dependencies and execution

T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008 -> T009. US1 is the smallest useful increment; both stories and delivery remain required. No parallel edits to the shared validator/tests. After implementation, independent root/site checks may overlap full Python checks; research is read-only. The retained real campaign is revalidated, never relabeled as current-source inference.
