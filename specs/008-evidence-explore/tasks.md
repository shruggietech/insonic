# Tasks: S008 Evidence and Explore

Input: [spec](spec.md), [plan](plan.md), [research](research.md), [data model](data-model.md), [contracts](contracts/explore.md). Tests are required by autopilot.

## Phase 1: Setup

- [x] T001 Verify S008, specs/008-evidence-explore, codex/008-evidence-explore and issues #11/#12 alignment; complete specify/clarify/checklist/plan in specs/008-evidence-explore/.
- [x] T002 Freeze schema5 DDL/digest and add migration/restore regressions in internal/catalog/freeze_v5_test.go and historical_v5.go.
- [x] T003 Add shared exploration types and runtime/schema contract checks in internal/contracts/explore.go and schemas/v0.0.0/runtime-request.schema.json.

## Phase 2: Foundation

- [x] T004 Add current extraction, saved query versions and separate layouts with typed validation/proofs in internal/catalog/exploration.go, migration.go and snapshot.go.
- [x] T005 Append atomic mutation invalidations and bounded publisher renewal/status in internal/catalog/exploration_outbox.go, recording.go, library.go and identity.go.
- [x] T006 Add reference graph entities/edges and native read validation with isolation/input-limit tests in internal/graph/types.go and native_test.go.
- [x] T007 Add graph lifecycle/shared dispatch in internal/app/explore.go and app.go.

## Phase 3: User Story 1 - Current evidence

Goal: Real current evidence extraction/publication/search on both graph engines.
Independent test: Supplied cue fixture -> accepted assertions -> real graph publish/query -> original source reference.

- [x] T008 [P] [US1] Write chunk/attribution/negation/invalid-output/timeout protocol tests in internal/evidence/evidence_test.go.
- [x] T009 [P] [US1] Write atomic graph replay/fence/rollback/uncertain-commit/parity tests in internal/graph/adapter_test.go.
- [x] T010 [US1] Implement current cue chunking, literal statement/default and elected HTTP extraction in internal/evidence/evidence.go and http.go.
- [x] T011 [US1] Implement real tagged Ladybug and explicit untagged unavailable adapter in internal/graph/ladybug.go and ladybug_unavailable.go.
- [x] T012 [US1] Implement real Arcade transactional/session adapter in internal/graph/arcade.go.
- [x] T013 [US1] Implement ordered reference-only publication/refresh/rebuild/receipt reconciliation in internal/app/graph.go.
- [x] T014 [US1] Integrate durable fenced extraction acceptance/recovery/current replacement in internal/app/evidence.go, work.go and internal/catalog/exploration.go.
- [x] T015 [US1] Implement common query filtering/order/pagination/current hydration and native/explain operations in internal/app/query.go.
- [x] T016 [US1] Add CLI parsing/search/query/graph/extraction commands and acceptance tests in cmd/insonic/explore.go and explore_test.go.
- [x] T017 [US1] Qualify real current-reference backend parity and correction through prepared graph/integration checks in internal/graph/adapter_test.go and internal/app/explore_test.go.

## Phase 4: User Story 2 - Timelines

Goal: Discover every media date and current original-clock evidence with source playback.
Independent test: Multiple library pages including uncertain/unknown dates; navigate/seek/replace.

- [x] T018 [US2] Write calendar pagination/precision/correction/current timeline tests in internal/app/timeline_test.go.
- [x] T019 [US2] Implement whole-library date filtering/uncertainty/original timeline operations in internal/app/timeline.go.
- [x] T020 [US2] Add Explore route/calendar/recording controls and reuse source-bound playback in desktop/frontend/src/explore.tsx and App.tsx.
- [x] T021 [US2] Add rendered keyboard/pagination/stale-request/playback/date uncertainty tests in desktop/frontend/tests/explore.test.mjs.
- [x] T022 [US2] Extend mounted native qualification with current Explore calendar/query/playback journeys in desktop/frontend/src/qualification.ts and desktop/qualification.go.

## Phase 5: User Story 3 - Saved definitions and graph views

Goal: Persist versioned queries/layouts, inspect current graph/table and incompatibility.
Independent test: Save/revise/restart/conflict/backend switch and graph/table expansion.

- [x] T023 [US3] Write immutable definition/CAS/restart/incompatibility/layout tests in internal/catalog/exploration_test.go and internal/app/explore_test.go.
- [x] T024 [US3] Implement saved definitions/validation/run/layout operations in internal/app/saved_queries.go.
- [x] T025 [US3] Implement bounded force-directed relationships/filter/provenance/omissions/expansion/accessibility in desktop/frontend/src/graph-view.tsx.
- [x] T026 [US3] Implement saved query/version/layout controls and rendered graph/table/reduced-motion tests in desktop/frontend/src/explore.tsx and desktop/frontend/tests/explore.test.mjs.

## Phase 6: Integration and publication

- [x] T027 Reconcile #21 pinned-result schema/prose/examples and publish exact CLI/contracts/implemented status in schemas/v0.0.0/graph-query.schema.json, docs/v0.0.0/graph.md, desktop.md, pipelines.md and CHANGELOG.md.
- [x] T028 Integrate real prepared graph tests/package support and bounded adapter qualification in scripts/qualify.py, scripts/package-desktop.py and .github/workflows/ci.yml.
- [x] T029 Run npm check/test, Go acceptance/vet/race, frontend checks and site/offline build; record evidence in specs/008-evidence-explore/verification.md.
- [x] T030 Run speckit-converge against all FRs/SCs/stories/design decisions; append and implement gap tasks in specs/008-evidence-explore/tasks.md.
- [x] T031 Push verified branch, publish/attach official PR closing #11/#12 and update Project tracking; record URL in specs/008-evidence-explore/verification.md.
- [x] T032 Address/reply/resolve every initial bot finding and reaction; request at most one additional review round; record rounds in specs/008-evidence-explore/verification.md.
- [ ] T033 Address/reply/resolve second-round findings and verify final exact-head green CI without triggering a third review in specs/008-evidence-explore/verification.md.
- [ ] T034 Hand off completed reviewed PR for owner final review/merge; preserve release/signing boundary in specs/008-evidence-explore/verification.md.

## Dependencies and execution order

Setup -> foundation -> US1 -> US2/US3 -> integration/convergence/publication. Tests precede implementation in each phase; shared files remain sequential. T008/T009 may run concurrently on disjoint files after foundation, but no implementation delegation is assumed. US2 date operations independently test catalog data; US3 storage independently tests saved definitions.

## Implementation strategy

Deliver usable current search first, then integrate timeline/playback, then saved definitions/graph views. All three stories are required for this authorized slice; intermediate checkpoints are validation rather than reduced delivery scope.
## Phase 7: Convergence
- [x] T035 Close FR-005 recovery evidence with real-engine uncertain-commit reconciliation and atomic failed/successful/replayed rebuild tests in internal/graph/failure_test.go and adapter_test.go.
- [x] T036 Close FR-011 persistence evidence with compatibility metadata, immutable replay, separate layout restore/CAS and cross-catalog acceptance in internal/catalog/exploration_test.go and postgres_test.go.
- [x] T037 Close FR-015/FR-016 delivery evidence with mounted native Explore/package qualification and current self-contained public docs/schema contracts.

- [x] T038 Close FR-002 elected-extractor dispatch with deterministic HTTP-to-durable-acceptance coverage in internal/app/explore_http_test.go.
- [x] T039 Close FR-005/FR-010 exact recovery/source-span gaps with idempotent rebuild requests and disjoint cited-clock filtering in internal/catalog/exploration_outbox.go and internal/explore/traversal_test.go.

## Phase 8: Review corrections

- [x] T040 Correct all four initial review findings and verify lifecycle, independent snapshot budget/paged reads, native output parity and complete saved-query edits in internal/app/explore_lifecycle_test.go, internal/catalog/explore_snapshot_limit_test.go, internal/graph/query_limits_test.go and desktop/frontend/tests/explore.test.mjs.
- [x] T041 Verify the second-round comment-aware function boundary across every supported native dialect, fix misleading SQL test dialects and prove rejected calls never reach the engine in internal/graph/native_comments_test.go and native_test.go.
- [x] T042 Correct hosted YAML parsing failure and add prepublication YAML syntax/duplicate-key validation and regression coverage in .github/workflows/ci.yml, scripts/check.mjs and tests/check.test.mjs.
- [x] T043 Update historical schema-3 fixture teardown for schema-6 foreign-key dependencies and pass full PostgreSQL catalog integration acceptance in internal/catalog/evidence_migration_test.go.
- [ ] T044 Bound native-webview playback startup separately and scroll real media into view without weakening decoded time advancement in desktop/frontend/src/qualification.ts and tests/frontend.test.mjs; requalify in hosted CI.

- [x] T045 Address Linux mirror download timeout through official HTTPS sources, bounded network retries and targeted dependency installation in .github/workflows/ci.yml; retain all required checks and ten-minute job limits.

- [x] T046 Serialize shared adapter fixtures, bound administrative setup clients and test hosted-only macOS qualification visibility in internal/app/explore_arcade_test.go, internal/graph/arcade_test.go, desktop/qualification.go and qualification_test.go; hosted acceptance remains T033/T044.
