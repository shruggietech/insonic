# Tasks: S009 Query assistance

Input: [spec](spec.md), [plan](plan.md), [research](research.md), [model](data-model.md), [contract](contracts/assistance.md). Autopilot requires meaningful tests before implementation.

## Phase 1: Setup

- [x] T001 Align S009/specs/009-query-assistance/codex/009-query-assistance/#13; execute specify, clarify, checklist, research and design.
- [x] T002 Complete blocking analyze across spec/plan/tasks and all constitution principles.

## Phase 2: Foundation

- [x] T003 Write per-setting CAS, replay and snapshot portability tests in internal/catalog/assistance_settings_test.go.
- [x] T004 Implement existing-table named setting read/CAS in internal/catalog/assistance_settings.go and contract.go.
- [x] T005 Write HTTP/schema/config validation, election, CI refusal, redirect, credential and timeout tests in internal/assistance/assistance_test.go.
- [x] T006 Implement strict assistance configuration/protocol and bounded HTTP in internal/assistance/assistance.go and http.go.

## Phase 3: US1 Inspect suggestions

Independent check: deterministic HTTP returns a complete proposal with zero generated-query executions.

- [x] T007 [US1] Write runtime suggestion/disabled/failure/invalid native output/workspace/fence tests in internal/app/assistance_test.go.
- [x] T008 [US1] Implement show/set/assist dispatch, selected schema/capabilities and pending native validation in internal/app/assistance.go and contracts/explore.go.
- [x] T009 [US1] Add shared CLI parsing/help and schema/example validation in cmd/insonic/explore_test.go, main.go and schemas/v0.0.0/.

## Phase 4: US2 Elected execution

Independent check: valid deterministic auto-run equals direct current results; invalid output has zero engine executions.

- [x] T010 [US2] Write auto-run/excerpt limits/exact clocks/cancellation/direct parity/profile/settings conflict tests in internal/app/assistance_test.go and internal/runtime/runtime_test.go.
- [x] T011 [US2] Extract shared context-aware query execution and bounded request contexts/disconnect supervision in internal/app/explore_ops.go, explore_fallback.go and internal/runtime/runtime.go/deadline.go.
- [x] T012 [US2] Implement validated native EXPLAIN and elected bounded current auto-run/context in internal/app/assistance.go; use real Ladybug/Arcade acceptance in existing backend suite.

## Phase 5: US3 Configuration and desktop

Independent check: configuration survives restart, complete proposal adopts without dropped fields and late requests cannot override edits.

- [x] T013 [US3] Write rendered full proposal/configuration/late-request/error/direct-query tests in desktop/frontend/tests/assistance.test.mjs and explore.test.mjs.
- [x] T014 [US3] Implement accessible optional assistance and complete editor adoption with request ownership in desktop/frontend/src/assistance.tsx and explore.tsx.
- [x] T015 [US3] Extend mounted deterministic qualification via runtime construction and desktop/qualification.go/frontend/src/qualification.ts for source and relocated packages.

## Phase 6: Delivery

- [x] T016 Publish self-contained contracts/help/capability status and changelog in docs/v0.0.0/graph.md, desktop.md, index.md, pipelines.md and CHANGELOG.md.
- [x] T017 Run npm check/test, Go acceptance/vet/affected race, frontend check/test/build, Python tests and site/offline/native qualification; record verification.md.
- [x] T018 Execute speckit-converge across 12 FRs, five SCs, nine story scenarios, nine decisions and all constitution principles; implement appended gaps.
- [x] T019 Commit/push verified feature branch, publish/attach official PR closing #13 and move Project status to In review.
- [x] T020 Resolve/reply every initial review; request only one additional code/security review round and address/resolve every finding.
- [x] T021 Verify final exact-head green hosted checks below ten minutes per job and hand off for owner final squash merge without merging/tagging/releasing.

## Dependencies and execution

Setup -> foundation -> US1 -> US2 -> US3 -> delivery. Test tasks precede implementation. Shared files stay sequential; research delegation is read-only. Independent test-file reads can batch, but no implementation delegation is assumed. All three stories are required; intermediate checks do not narrow the delivery scope.

## Phase 7: Convergence

- [x] T022 Invalidate pending assistance when a direct query starts; add a rendered competing-request regression in desktop/frontend/src/explore.tsx and tests/assistance.test.mjs per FR-008 and D08 (partial, HIGH).
- [x] T023 Count generated native query executions on real selected backend fixtures, requiring zero for suggestions/invalid output and exactly one for valid auto-run in internal/app/explore_backend_test.go per SC-002 and FR-012 (partial, HIGH).
- [x] T024 Verify configuration through actual runtime restart and exact current source clocks through elected excerpts in internal/app/assistance_test.go per SC-003 and FR-010 (partial, MEDIUM).

## Phase 8: Convergence

- [x] T025 Preserve generated 64-bit integer parameters through desktop JSON handoff as exact decimal strings, accept both representations in the shared typed/native contract, and qualify editor/native execution per FR-005/010 and SC-001 (partial, HIGH).

## Phase 9: Final review remediation

- [x] T026 Share and deduct one aggregate query budget across probing/context and validation/execution; verify valid phases and exhaustion without charging provider time in internal/app/assistance.go/test and docs/v0.0.0/graph.md per FR-008 and D06 (review P2).

## Phase 10: Hosted qualification remediation

- [x] T027 Reset Library media elements for every new verified playback ticket to prevent old-resource readiness from racing source seeks; add a red/green rendered regression and bounded numeric seek diagnostics for repeated macOS hosted qualification failure.

- [x] T028 Use the existing 25-second native journey readiness budget for asynchronous initial workspace mounting, requiring frame/form/library controls together; test delayed controls and missing-screen rejection after hosted Windows run 37826752729 exhausted the five-second mount window.
