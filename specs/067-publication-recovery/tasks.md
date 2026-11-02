# Tasks: S067 Publication recovery and timing guidance

## Phase 1: Setup

- [x] T001 Create specification, clarify scope and link #240/#241 in `specs/067-publication-recovery/spec.md` and milestone 52/Project 7.
- [x] T002 Complete research, plan, data model, CLI contract and quickstart in `specs/067-publication-recovery/`.

## Phase 2: Foundation

- [x] T003 Run blocking cross-artifact analysis against `spec.md`, `plan.md` and `tasks.md`; record gate in `verification.md`.

## Phase 3: US1 Recover uncertain publication

Independent proof: delayed/lost response fixtures recover exact original identity; conflicts, corrupt receipts/trees and storage failures refuse without repeating mutation.

- [x] T004 [US1] Add failing stale/lost/bounded/conflict/receipt persistence cases in `tests/pages_test.py` (FR-001/002/003/006).
- [x] T005 [US1] Implement atomic receipt persistence, phase callbacks and three-observation exact-head reconciliation in `scripts/pages.py` (FR-001/002/006).
- [x] T006 [US1] Add original-parent/tree read-only recovery and no-op handling in `scripts/pages.py`, covered by `tests/pages_test.py` (FR-003/004).
- [x] T007 [US1] Integrate durable promotion and separate GET-only recovery CLI in `scripts/release.py`; test archive/source/public-verification authority and unchanged original receipt in `tests/release_test.py` (FR-001/004/006).
- [x] T008 [US1] Retain failure receipts using always-upload and attempt-qualified names in `.github/workflows/release.yml`; cover retention in `tests/site-release.test.mjs` (FR-001).

## Phase 4: US2 Truthful timing guidance

Independent proof: audit each current publication timing claim against its executed caller; repository and documentation checks pass.

- [x] T009 [P] [US2] Correct action acquisition, content, Pages aggregate, website Git/wrapper, browser and cleanup guidance in `docs/v1.1.0/releases.md` (FR-005).
- [x] T010 [US2] Document recovery usage and immutable release boundaries in `docs/v1.1.0/releases.md`, add behavior and dated workflow decision to `CHANGELOG.md` (FR-004/005/006).

## Phase 5: Qualification and delivery

- [x] T011 Run affected Python suites, npm check/test and site test/check/build; record outcomes and encoding sanity in `specs/067-publication-recovery/verification.md` (SC-001/002/003/004).
- [x] T012 Converge artifacts, commit/push/open official PR linking both issues and milestone 52; record identity in `specs/067-publication-recovery/verification.md` (SC-005).
- [ ] T013 Resolve all actionable review comments within two rounds, retain CI receipts outside Actions, and hand off green final head for owner merge in `specs/067-publication-recovery/verification.md` (SC-005).

## Dependencies and strategy

T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008. T009 can be audited independently after T003; T010 integrates recovery guidance. T011 follows T008/T010, then T012/T013. US1 is the first independently testable increment; US2 follows without changing runtime budgets. Research was parallelized per Spec Kit; code edits remain sequential to avoid shared-file conflicts.
