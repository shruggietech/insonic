# Tasks: S052 Terminal progress and interruption acceptance

**Input**: spec.md, plan.md, research.md and data-model.md in this directory. Tests are required. Branch/feature/selector/issue/milestone/Project agree on S052.

## Phase 1: Setup

- [x] T001 Specify/clarify/plan S052, validate checklists and link #182 under #147 with milestone36 and Project7 Slice S052 in specs/052-terminal-interruption/.

## Phase 2: Foundation

- [x] T002 Add test-first actual-terminal, interrupt, overflow and forced-cleanup regressions in tests/terminal_transport_test.py and receipt mutations in tests/cli_terminal_qualification_test.py (FR-002, FR-006, FR-007, SC-003).
- [x] T003 Implement hidden bounded Windows/Linux terminal ownership in scripts/terminal_transport.py (FR-001, FR-002, FR-006, FR-007).

## Phase 3: US1 Real progress (P1)

**Independent test**: held two-role acquisition emits correlated running/count progress before release, then succeeds; JSON/quiet/redirected controls contain no progress.

- [x] T004 [US1] Build owned runtime/server barrier and success/output-mode controls in scripts/cli_terminal_qualification.py (FR-001, FR-004, US1/AC1-2).

## Phase 4: US2 Client interruption (P1)

**Independent test**: actual Ctrl+C naturally exits while independently observed work persists and completes; cancellation, timeout and failure remain distinct.

- [x] T005 [US2] Implement terminal interrupt/continued-work, explicit cancel, timeout and checksum-failure scenarios in scripts/cli_terminal_qualification.py (FR-002, FR-003, FR-004, US2/AC1-3, SC-002).

## Phase 5: US3 Evidence (P2)

**Independent test**: strict receipt mutations refuse missing/false/mismatched proof and source/package qualification elects the real campaign.

- [x] T006 [US3] Validate exact-executable receipt/lifecycle evidence and integrate scripts/cli_contract_qualification.py, scripts/cli_corpus.py and tests/fixtures/cli-corpus.json with focused regression updates (FR-005, FR-007, FR-008, US3/AC1-2, SC-001/003).
- [x] T007 [US3] Require nested terminal acceptance in scripts/release.py and update tests/release_test.py fixtures (FR-005, FR-008, SC-003).

## Phase 6: Verification and Delivery

- [x] T008 Run actual current-host source campaign, full relevant Python, Go checks if product changes, root check/test and affected site checks; update docs/v1.0.1/development.md, CHANGELOG.md and verification.md (FR-001-008, SC-001-003).
- [x] T009 Converge implementation against all FR/SC/acceptance/plan/constitution obligations and resolve gaps in specs/052-terminal-interruption/ (FR-008).
- [ ] T010 Commit/push/open official PR, attach artifact and update #182/Project7 In review; monitor exact final-source CI and at most two Codex rounds, fix/reply/resolve every finding (FR-008, SC-003).
- [ ] T011 Retain compact audited receipts/review evidence, remove owned scratch after readback, and stop unmerged for owner review (FR-007, FR-008).

## Dependencies and Strategy

T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008 -> T009 -> T010 -> T011. Actual terminal transport is the shared prerequisite; each story then has its own independently checkable scenario. Runtime/server ownership and cleanup are always required, including negative paths. Parallel opportunities are read-only research, separate transport/receipt tests after design, and independent final root/site checks; shared-file edits remain sequential. Build the observable progress MVP first, then interruption/durable comparisons, then strict delivery integration. No deployment, tag or merge.
