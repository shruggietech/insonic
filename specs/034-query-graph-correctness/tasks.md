# Tasks: S034 Query and graph correctness

## Phase 1: Setup

- [x] T001 Verify six issue IDs, branch and feature directory; author spec/checklists/plan in specs/034-query-graph-correctness.
- [x] T002 Research runtime, schema and consumers; document decisions and contracts in specs/034-query-graph-correctness/research.md and contracts/query.md.

## Phase 2: Foundation

- [x] T003 Add shared pinned timezone resolver below contracts/library, preserving existing provenance in internal/timezone and internal/library/date.go (FR-003, FR-007).

## Phase 3: US1 Complete results (P1)

Independent test: complete graph identity unions and more than 500 search matches.

- [x] T004 [US1] Add failing star/dense graph pagination and cursor regressions in internal/explore/query_test.go (FR-001, SC-001).
- [x] T005 [US1] Implement deterministic two-stream byte-bounded continuation in internal/explore/query.go and update current schema contract (FR-001, FR-007).
- [x] T006 [US1] Add search --after, human summary/help regressions and >500-match continuation coverage in cmd/insonic (FR-002, SC-001).
- [x] T007 [US1] Align desktop accumulation/status and shared backend parity tests in desktop/frontend/src/explore.tsx and internal/app/explore_backend_test.go (FR-007).

## Phase 4: US2 Advertised query contracts (P2)

Independent test: accepted timezone forms and live schema-derived save example.

- [x] T008 [US2] Add query/calendar timezone and DST regressions; use shared resolver in internal/contracts/explore.go and internal/explore/calendar.go (FR-003, SC-002).
- [x] T009 [US2] Require save-only nonblank title and validate emitted example through live dispatch in schemas/v1.0.1/runtime-request.schema.json and internal/app tests (FR-004, SC-003).

## Phase 5: US3 Unambiguous native reads (P2)

Independent test: exact parameter decoding and native statement/effect regression matrix.

- [x] T010 [US3] Add failing duplicate parameter/nested-member/exact-int tests then validate JSON before decoding in cmd/insonic/explore_aliases.go (FR-005, SC-003).
- [x] T011 [US3] Add terminator/quote/comment/function/effect regressions then implement actionable bounded native reasons in internal/graph/native.go and internal/contracts/operator.go with response-schema alignment (FR-006, SC-004).
- [x] T012 [US3] Explain supported native grammar/functions through cmd/insonic contextual help and docs/v1.0.1/graph.md (FR-006).

## Phase 6: Verification and delivery

- [x] T013 Update authoritative query docs and CHANGELOG.md; check encoding and public/internal separation (FR-007).
- [x] T014 Execute affected Go/CLI/desktop/backend and npm/site checks; record results and convergence in specs/034-query-graph-correctness/verification.md (SC-001..SC-005).
- [x] T015 Prepare verified commits, linked PR description and GitHub milestone/Project assignments for authorized publication; use the live PR as the authoritative exact-head CI and bounded review record, stopping before merge (SC-005).

## Dependencies and execution strategy

T001 -> T002 -> analysis gate -> regression tests -> implementations -> T013 -> T014 -> T015. T003 precedes T008. US1 is the first independently useful increment; all six issues remain required for delivery. Test files for independent stories may be checked concurrently, but shared-file mutations remain sequential. Research agents performed read-only parallel inspections. No implementation subagent is required.

## Phase 7: Convergence

- [x] T016 Refresh graph layout effects for edge-only continuation in desktop/frontend/src/graph-view.tsx per FR-001/FR-007 (partial).
- [x] T017 Capture the actual local-zone provenance for the Local alias in internal/library/date.go and add its regression per FR-003/FR-007 (partial).

- [x] T018 Accept bounded opaque search cursors in the top-level invocation parser, retaining UUID validation for catalog cursors in cmd/insonic/commands.go; extend full-parser regression per FR-002 (partial).

- [x] T019 Normalize the accepted trailing terminator before native engine dispatch in internal/graph/native.go and internal/graph/types.go; retain literal/comment distinctions and all guards, with cross-dialect regression coverage per FR-006/FR-007 (partial).

- [x] T020 Distinguish parenthesized read clauses from function calls while rejecting qualified calls outside the declared pure subset in internal/graph/native.go and native_terminator_test.go; document grammar per FR-006 (partial).
