# Tasks: S018 CLI contract reliability

Input: [plan](plan.md), [spec](spec.md), [research](research.md), [contracts](contracts/cli.md). Tests precede matching fixes.

## Phase 1: Setup

- [x] T001 Verify S018 directory/branch/nine-issue scope and read governance/CI/skills (FR-011/012).
- [x] T002 Complete specify/clarify/checklist/plan/tasks and blocking analysis in specs/018-cli-contract-reliability (all FR/SC).

## Phase 2: Foundations

- [x] T003 Prepare synchronized 1.0.1 using scripts/release.py and add historical-document compatibility tests/helpers in internal/contracts, schemas and affected persisted readers (FR-011).

## Phase 3: US1 Command invocation

- [x] T004 [US1] Add failing command/help/token/flag inventory tests in cmd/insonic/commands_test.go (FR-001/002).
- [x] T005 [US1] Implement cmd/insonic/commands.go pure lexical/structural admission and full command/help metadata; integrate main.go and existing parsers (FR-001/002).

## Phase 4: US2 Existing operations

- [x] T006 [US2] Add failing settings/model-list/protected-delete route tests in cmd/insonic and internal/runtime (FR-003/004/005).
- [x] T007 [US2] Consolidate shared operation admission, omit absent cursor and read/clear protected vault-delete input once in main.go/speaker_models.go (FR-003/004/005).

## Phase 5: US3 Data fidelity

- [x] T008 [US3] Add actual decoder/nested signed numeric regression in internal/runtime plus CLI consumer tests (FR-006).
- [x] T009 [US3] Preserve json.Number through runtime/client consumers and output (FR-006).
- [x] T010 [US3] Add PrepareImport/import/set-origin timezone-provenance/replay/DST tests in internal/library (FR-007).
- [x] T011 [US3] Implement explicit/implicit timezone provenance through options/merge/date/recovery and matching current schema (FR-007).

## Phase 6: US4 Durable responses

- [x] T012 [US4] Add typed durable-reload/receipt/retry/mixed-cause tests in internal/catalog and internal/app (FR-009).
- [x] T013 [US4] Propagate bounded classified causes through checkpoints, model acquisition and result paging without changing Work proof shape (FR-009).
- [x] T014 [US4] Add envelope/stream/exit/correlation/output-failure tests in cmd/insonic and current response schema tests (FR-008/010).
- [x] T015 [US4] Normalize direct/live/maintenance/help/version/wait output and preserve context/numeric fidelity in main.go/work_wait.go (FR-008/010).

## Phase 7: Qualification and delivery

- [x] T016 Extend scripts/qualify.py/package consumer coverage for all nine issue matrices and historical populated workspace/backup/restore (FR-012, SC-001/002/003/004).
- [x] T017 Update docs/current, schemas/current and CHANGELOG.md with exact command/stream/compatibility/date/failure behavior and inspect rendered/offline output (FR-011).
- [ ] T018 Run focused and full configured npm/Go/Python/site/frontend/native/backend checks, encoding and diff integrity (FR-012, SC-002).
- [x] T019 Run speckit-converge, fix any gaps and record exact verification in this slice (all FR/SC).
- [ ] T020 Commit/push/open official PR closing nine issues; track project status, satisfy every review thread with at most two rounds and green exact-head CI; stop for owner merge (SC-005).

## Dependencies and execution order

Setup -> foundations -> US1 -> US2 -> US3 -> US4 -> qualification/delivery. Tests run before the corresponding implementation. Work sharing main.go is sequential; independent read-only research/checks can run concurrently. US2/US3/US4 each retain independent acceptance tests. MVP validation never substitutes for completion of the full nine-issue slice.

## Parallel examples

Within US1, independent source/flag inventory reads may run together. Within US2, settings and cursor fixture reads may run together. Within US3, numeric and date research may run together. Within US4, schema and durable-persistence research may run together. Mutations remain ordered; validation is awaited.

## Implementation strategy

Complete all four stories and package qualification before publication. Preserve old document bytes and journal proofs. Never widen schema UUID/secret validation or introduce new mandatory reviews. Publication is authorized; merge/tag/product release are not part of this handoff.

## Phase 8: Review convergence

- [x] T021 Normalize historical import request copies before current IPC validation, preserve source bytes, add a CLI-to-shared-schema regression, and qualify the final source (FR-011, initial Codex review).
- [x] T022 Retain immutable historical model manifest hashes during current request admission with explicit packaged historical schema references and CLI/shared-schema regression (FR-011, adjacent review boundary).
- [x] T023 Use the shared schema catalog in frontend request validation and pass type checking/all 73 frontend tests after CI exposed missing historical dependencies (FR-012).
