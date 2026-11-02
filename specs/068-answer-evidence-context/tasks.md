# Tasks: S068 Answer evidence and context

Input: [spec](spec.md), [plan](plan.md), [research](research.md), [data model](data-model.md), [contracts](contracts/cli.md). Existing #220-#223, milestone 48. Test-first discipline required.

## Phase 1: Setup and gate

- [x] T001 Initialize specs/068-answer-evidence-context/ and verify branch/issue/selector identity; no new tracking issue.
- [x] T002 Complete research, contracts, checklists and plan; inspect existing bounded catalog/artifact patterns.
- [x] T003 Run read-only Spec Kit analysis; record coverage and resolve critical/high findings before implementation in specs/068-answer-evidence-context/verification.md.

## Phase 2: Foundation and general contracts

- [x] T004 [US4] Add failing request/evidence/claim/citation round-trip, forged-reference, unknown-time and large-identifier tests in internal/answers/answers_test.go and schemas tests (FR-001/FR-002/FR-010, SC-001/SC-005).
- [x] T005 [US4] Implement shared versioned types/validation/budgets in internal/answers/ and schemas/v1.1.0/; register current schemas/runtime requests without changing historical releases (FR-001/FR-002/FR-010).

## Phase 3: US1 bounded current evidence

Goal: inspect current evidence across recordings without whole-library hydration. Independent fixture: related, disconnected, stale and untimed sources with bounded continuations.

- [x] T006 [P] [US1] Add bounded projection, unmatched-page continuation, freshness, cancellation and isolation tests in internal/catalog/answer_evidence_test.go; reuse suite in postgres_test.go (FR-002/FR-003/FR-004/FR-010, SC-001/SC-002/SC-005).
- [x] T007 [P] [US1] Add bounded graph neighborhood and checkpoint/direction/workspace tests in internal/graph/adapter_test.go and focused fixtures for both adapters (FR-003/FR-004/FR-010).
- [x] T008 [US1] Implement catalog general cue projection/resolution/epoch and graph bounded node/edge reads in internal/catalog/answer_evidence.go, contract.go and internal/graph/ (FR-002/FR-003/FR-004).
- [x] T009 [US1] Implement one-budget retrieval orchestration, selection paths and graph-lag diagnostics in internal/answers/ and internal/app/answers.go; test multi-hop/current-source admission in internal/app/answers_test.go (FR-002/FR-003/FR-004, SC-001/SC-002).

## Phase 4: US2 independent support and ranking

Goal: repeated inputs cannot inflate corroboration. Independent fixture: duplicate, overlapping, quoted, independent and contradictory sources.

- [x] T010 [US2] Write failing ranking/lineage/contradiction tests, then implement explainable deterministic family grouping and ranking in internal/answers/ (FR-005/FR-006, SC-003).

## Phase 5: US3 bounded context and compatible cache

Goal: bounded hierarchical context with source-linked condensation and genuine compatible reuse.

- [x] T011 [US3] Write failing full-envelope budget/Unicode/hierarchy/corruption tests, then implement extractive context and semantic key validation in internal/answers/ (FR-007/FR-008, SC-002/SC-004).
- [x] T012 [US3] Add cache-hit/self-write, invalidation, corrupt artifact, CAS/lost-outcome, restart and backend tests, then implement artifact-backed cache pointers/retention in internal/app/answers.go and the scoped catalog setting behavior (FR-008/FR-009/FR-010, SC-004/SC-005).

## Phase 6: US4 usable inspection/export/validation

- [x] T013 [US4] Test then implement CLI/runtime answers prepare/show/export/validate, human summaries, help/schema inputs and stale/cancelled result handling in cmd/insonic/ and internal/app/; preserve shared desktop dispatch (FR-001/FR-009/FR-010, SC-001/SC-005).
- [x] T014 [US4] Add installed-CLI and supported backend fixture coverage, source mutation checks and public examples in scripts/ and affected test suites (FR-004/FR-009/FR-010, SC-005).

## Phase 7: Convergence and delivery

- [x] T015 Update authoritative docs/v1.1.0/ and CHANGELOG.md, run formatting/schema/repository/Go/CLI/backend/site checks, and complete Spec Kit convergence (all FR/SC).
- [x] T016 Commit, push and open the official PR closing only #220-#223; use existing milestone 48/Project 7 and retain source/test/CI evidence (all FR/SC).
- [ ] T017 Resolve every external finding within automatic round one plus at most one explicit second round; require green applicable exact-head CI, zero unresolved review threads, retained audited receipts and owner final merge handoff (all FR/SC).

## Dependencies and execution

T001-T003 precede implementation. T004 precedes T005. T006/T007 can run independently after contracts, then T008/T009. T010 builds on retrieved evidence; T011/T012 build on ranked source identity. T013/T014 complete usable integration after those foundations. T015-T017 close the slice. Tests precede corresponding code; shared-file edits are sequential. No partial issue closure: each issue closes only with its complete acceptance. No extra issue is created for slice bookkeeping.
