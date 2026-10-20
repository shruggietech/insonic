# Tasks: Verified CI readiness and elected acceptance

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md).

## Phase 1: Setup

- [x] T001 Verify #161, exact main and feature mapping in specs/041-ci-readiness-acceptance/spec.md.
- [x] T002 Record owner timing decision and constitution 3.1.0 in .specify/memory/constitution.md.

## Phase 2: Foundation

- [x] T003 Complete design/research and Spec Kit consistency gate for specs/041-ci-readiness-acceptance/plan.md.

## Phase 3: US1 readiness (P1)

**Independent test**: Actual workflow graph delays consumers past a 1,342-second cold producer; failed/cancelled inputs never start them.

- [x] T004 [US1] Retain helper 180-second characterization and failing-before actual graph simulation in tests/ci-readiness.test.mjs (FR-001, FR-009, SC-001).
- [x] T005 [US1] Add terminal source ready join and gate all qualification consumers in .github/workflows/media-source.yml and .github/workflows/ci.yml (FR-001, FR-002).
- [x] T006 [US1] Record resolved toolchain package observation through scripts/source-toolchain.py and source artifacts in .github/workflows/media-source.yml (FR-008).
- [x] T007 [US1] Update obsolete dependency assertions in tests/ci-scope.test.mjs and run existing transfer trust tests (FR-003, FR-004).

## Phase 4: US2 elected acceptance (P1)

**Independent test**: Every elected non-success, unknown scope and over-budget interval rejects acceptance; intentional non-election is recorded.

- [x] T008 [US2] Add result/timing/identity/API failure regressions in tests/ci-acceptance.test.mjs (FR-005, FR-007, SC-002, SC-003).
- [x] T009 [US2] Implement always-running acceptance and terminal audit in scripts/ci-acceptance.mjs and .github/workflows/ci.yml (FR-005, FR-007).
- [x] T010 [US2] Declare required aggregate in .github/bootstrap.json without mutating hosted protection before publication (FR-006).

## Phase 5: Verification and handoff

- [x] T011 Update docs/v1.0.1/development.md and CHANGELOG.md with current contracts and dated decision (FR-007).
- [x] T012 Run root/site/affected checks, encoding sanity and final diff, record evidence in specs/041-ci-readiness-acceptance/verification.md.
- [x] T013 Commit verified local work and report remaining hosted cold/warm, protection application and post-merge evidence in specs/041-ci-readiness-acceptance/verification.md (SC-004).

## Dependencies and execution

Setup -> foundation -> US1 -> US2 -> verification. Different test files may be authored independently; shared workflow edits remain sequential. US1 is the first independently testable increment; US2 adds the merge acceptance contract. Hosted closure follows authorized publication and is explicitly outside local completion evidence.

## Phase 6: Convergence

- [ ] T014 After authorized publication, retain exact cold/warm hosted preparation, qualification and terminal timing receipts per SC-004 (partial).
- [ ] T015 After the aggregate is published, apply/read back the existing bootstrap required check and reconcile #161 milestone/project tracking per plan: hosted enforcement (partial).
- [ ] T016 Verify exact owner-authorized squash main package acceptance and terminal timing before closing #161 per plan: post-merge evidence and SC-004 (partial).

## Phase 7: Measured hosted timing repair

- [x] T017 Record first hosted failure and analyze dependency-only preparation against the approved timing contract in plan.md, research.md and verification.md.
- [x] T018 Add failing-before preparation, identity, archive and consumer-graph regressions; implement scripts/go-dependency-transfer.py and the Windows preparation job.
- [x] T019 Wire verified package consumers and readiness/aggregate; preserve product builds and all acceptance; run root, site and affected checks.
- [ ] T020 Push the verified repair and monitor exact-head CI to terminal; retain final timing and package evidence, stop before merge.

- [x] T021 Analyze the second measured budget failure; add setup compiler dependencies, shared-deadline input gate and joined independent CI setup work; preserve every product and native check.
