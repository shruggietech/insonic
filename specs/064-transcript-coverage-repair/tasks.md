# Tasks: Transcript coverage and targeted repair

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/ and quickstart.md. Tests are required by the slice and autopilot. Task order is chronological; all work remains S064/#217-#219.

## Phase 1: Setup

- [x] T001 Establish S064 branch, feature selector, issue/project linkage and specification in specs/064-transcript-coverage-repair/spec.md.
- [x] T002 Complete research, design contracts and requirement-quality review in specs/064-transcript-coverage-repair/plan.md and checklists/requirements.md.

## Phase 2: Foundation

- [x] T003 Verify ignore/prerequisite setup and run blocking artifact analysis for specs/064-transcript-coverage-repair/tasks.md.

## Phase 3: US1 coverage (P1)

**Goal**: Read-only coverage with truthful per-track denominators. Independent validation uses known-truth gaps, repeats, unequal tracks, silence/unknowns and snapshot non-mutation.

- [x] T004 [US1] Write known-truth positive/negative, overflow and pagination tests in internal/processing/coverage_test.go before implementation (FR-001..005, SC-001..002).
- [x] T005 [US1] Implement text-free coverage evidence and interval calculator in internal/processing/coverage.go: at most 32,768 observations, 1,000,000 combined intervals, 65,536 cues and 1,000 detail regions per page; explicit stage status and bounded comparison work (FR-001..005, FR-013).
- [x] T006 [US1] Retain verified per-core acoustic observations before assembly in internal/app/processing_window_work.go and expose snapshot adaptation in internal/app/recording_coverage.go; legacy or mismatched evidence remains unknown (FR-001..005).
- [x] T007 [US1] Add coverage dispatch/schema/CLI and named human summaries in internal/app/processing.go, cmd/insonic/recordings.go, cmd/insonic/commands.go and schemas/v1.0.3/runtime-request.schema.json (FR-005).
- [x] T008 [US1] Verify read-only snapshot identity, ambiguous lanes, supplied clocks and evidence persistence in internal/app/recording_coverage_test.go (SC-001..002).

## Phase 4: US2 repair (P1)

**Goal**: Bounded durable repair preserves current authority unless a valid candidate measurably improves. Independent validation covers target selection, exact unaffected semantics, faults and idempotency.

- [x] T009 [P] [US2] Write native-format merge and out-of-scope preservation cases in internal/subtitles/repair_test.go for SubRip/WebVTT/ASS/SSA (FR-008, SC-003).
- [x] T010 [P] [US2] Write mapping/source/document/owner fence and unchanged-decision cleanup tests in internal/catalog/recording_repair_test.go (FR-010..011, SC-004).
- [x] T011 [US2] Extract build-only window candidate construction in internal/app/processing_window_work.go; filter selected windows while preserving full validated plan and ordinary processing behavior (FR-007).
- [x] T012 [US2] Implement pure finite policy/target normalization in internal/processing/repair.go: max_attempts 1..3 default 1 per selected window across restarts, at most 32 normalized regions and bounded duration/window/context selection (FR-006..009, FR-013).
- [x] T013 [US2] Implement native-compatible raw cue merge and dependency reconciliation in internal/subtitles/repair.go, preserving unaffected semantic fields and allowing only structural ordinal/root-summary changes (FR-008).
- [x] T014 [US2] Implement transactional speaker-mapping fence, durable attempt consumption and accepted/unchanged decision receipts in internal/catalog/recording_repair.go; retire only appropriate temporary artifacts and reuse ordinary dependent invalidation (FR-010..011, FR-013).
- [x] T015 [US2] Implement frozen repair election, fixed-basis comparison and candidate acceptance in internal/app/recording_repair.go; fail or return unchanged for stale, ambiguous, exhausted or non-improving candidates (FR-006..013).
- [x] T016 [US2] Integrate repair work kind with internal/app/work.go, model_dependencies.go, portable_work.go and internal/catalog/backup_input.go, processing_window_work.go, processing_windows_cleanup.go, preserving model, backup and cleanup authority (FR-007, FR-010..012).
- [x] T017 [US2] Add optional saved coverage/repair policy and full-policy overrides in internal/pipeline/definition.go and internal/app/election.go without implicitly starting repair during ordinary processing (FR-012).
- [x] T018 [US2] Add repair CLI, request/output schemas, examples and completion guidance in cmd/insonic/ and schemas/v1.0.3/, with strict validation tests in schemas/ and internal/contracts/ (FR-005..006, FR-012..013).
- [x] T019 [US2] Test target-only calls, disjoint/whole-cue scope, cancelled resume, stale mapping and source, corrupt checkpoints, unknown commit replay and rejected-candidate cleanup in internal/app/recording_repair_test.go and internal/catalog/recording_repair_test.go (SC-003..005).

## Phase 5: US3 qualification (P2)

**Goal**: Demonstrate the complete milestone workflow and provide a compatibility-based release recommendation. Independent validation executes installed CLI examples and audits platform/backend receipts.

- [x] T020 [US3] Add deterministic admission/window interruption/resume/coverage/repair/retrieval/cleanup journey in scripts/cli_recording_qualification.py and scripts/documented_workflows.py with exact public JSON examples and invocation ledger assertions (FR-014, SC-006).
- [x] T021 [US3] Integrate meaningful alternative-backend acceptance and cleanup cases into internal/catalog/ integration fixtures and existing installed package qualification without weakening CI inference refusal (FR-014, FR-016).
- [x] T022 [US3] Document coverage denominators, repair elections, native preservation, GUI lag and executable examples in docs/v1.0.3/cli.md, pipelines.md, subtitles.md and development.md (FR-014..015).
- [x] T023 [US3] Prepare compatibility/semantic-version recommendation, concise changelog-linked highlights and acoustic/hardware evidence limits in specs/064-transcript-coverage-repair/release-recommendation.md (FR-015, SC-007).

## Phase 6: Convergence, verification and publication

- [x] T024 Run Spec Kit converge against specs/064-transcript-coverage-repair/ and complete every required remaining task; preserve traceability and scope.
- [x] T025 Run required npm check/test, Go tests/vet, Python and affected native/frontend/site/package/backend checks; retain preparation/qualification/combined evidence under .local/s064/ and summarize in specs/064-transcript-coverage-repair/verification.md (FR-016, SC-006).
- [x] T026 Update CHANGELOG.md and generated reference projections, verify UTF-8/LF/no mojibake and synchronized contract versions, then commit and push the authorized branch.
- [x] T027 Publish official PR closing #217-#219, mark project items In review/S064, and retain the bounded review ledger under .local/s064/.
- [x] T028 Address every external finding, request at most one second review round, verify exact final-head green CI and resolved threads, and hand off for owner final review/merge in specs/064-transcript-coverage-repair/verification.md.

## Dependencies and execution

T001-T003 gate all implementation. US1 precedes US2 comparison; native merge and catalog fault research/tests can proceed independently (T009/T010). US2 integration precedes the complete US3 journey. Documentation/release packet drafting can proceed while qualification runs. No two mutations of the same file run concurrently. US1 is the first independently demonstrable increment, but all three stories are required for this slice. Tests precede the corresponding implementation. Publication follows convergence and verification; final handoff follows reviews and exact-head CI.

## Traceability

FR-001..005: T004-T008. FR-006..009: T009,T011-T015,T018-T019. FR-010..013: T010,T014-T019. FR-014: T020-T022. FR-015: T022-T023,T026. FR-016: T021,T025-T028. SC-001..002: T004-T008. SC-003..005: T009-T019. SC-006: T020-T025. SC-007: T023,T027-T028.

## Phase 7: Convergence

- [x] T029 Recheck unresolved cue lanes against the final whole-cue expanded scope in internal/processing/repair.go and add a regression in repair_test.go per FR-008, US2/AC2 (partial, HIGH).
- [x] T030 Bound whole-cue component selection against the fixed initial scopes, rather than an expanding component list, and exercise a dense multi-component fixture in internal/processing/repair_test.go per FR-013 and plan: bounded resources (partial, MEDIUM).
