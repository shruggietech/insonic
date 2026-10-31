# Tasks: Resumable long-recording processing

**Input**: [Plan](plan.md), [specification](spec.md), [data model](data-model.md), [contracts](contracts/windows.md).

## Phase 1: Setup

- [x] T001 Confirm S063 branch, feature selector, issue scope, constitution and installed command workflow in specs/063-resumable-long-recording/spec.md.
- [x] T002 Research existing streaming, checkpoint, word evidence and recovery boundaries in specs/063-resumable-long-recording/research.md.
- [x] T003 Complete requirements review and blocking cross-artifact analysis for specs/063-resumable-long-recording/.

## Phase 2: Shared foundations

- [x] T004 Freeze public window/plan/result and catalog checkpoint interfaces in internal/processing/windows.go and internal/catalog/processing_window_work.go; bind all identity-bearing inputs (FR-001, FR-003).
- [x] T005 [P] Add hidden supervised bounded stream-output tests and implementation in internal/process/stream.go and internal/process/stream_test.go (FR-004, FR-005, FR-013).
- [x] T006 [P] Add typed plan/checkpoint domain migration, schema and admission/proof validation in internal/catalog/, schemas/v1.0.3/catalog-snapshot.schema.json (FR-008, FR-009, FR-010).

## Phase 3: US1 accurate bounded processing

- [x] T007 [P] [US1] Add deterministic plan tests for zero/short/exact/final, stereo/grouped/uneven tracks, ranges, offsets, identity changes and overflow in internal/processing/windows_test.go (FR-001, FR-002, SC-001).
- [x] T008 [US1] Implement normalized core 1..300 seconds, context 0..half core, concurrency cap 1..4, exact native sample plans and inward mapping in internal/processing/windows.go (FR-001, FR-002).
- [x] T009 [US1] Implement direct bounded verified source streaming and linear resample windows, owned scratch cleanup and cancellation in internal/processing/windows_audio.go (FR-004, FR-005, SC-002).
- [x] T010 [P] [US1] Add native PCM comparisons and tenfold-duration resource-bound tests in internal/processing/windows_native_test.go (FR-004, FR-005, SC-002).
- [x] T011 [P] [US1] Add mapped-relative segment/word contracts, validation and local/hosted production in internal/processing/results.go, internal/pipeline/http.go, scripts/processing-worker.py and associated tests (FR-006).
- [x] T012 [US1] Add repeated phrase, pause, multilingual, track overlap, ambiguity and changed-output tests before deterministic reconciliation in internal/processing/stitching_test.go and stitching.go (FR-006, FR-007, SC-003).
- [x] T013 [US1] Connect shared app window orchestration and final cue provenance/current acceptance in internal/app/processing_window_work.go with integration tests (FR-007, FR-014).
- [x] T014 [US1] Normalize automatic long-source selection and explicit short-source windows through runtime, saved pipeline, CLI help and JSON schemas in internal/app/election.go, internal/pipeline/definition.go, cmd/insonic/command_guidance.json, schemas/v1.0.3/ (FR-002, FR-014).

## Phase 4: US2 durable resume

- [x] T015 [P] [US2] Add stale owner, source/document changes, forged snapshot/proof, cross-plan substitution and partial publication tests in internal/catalog/processing_window_work_test.go (FR-003, FR-008, FR-009, SC-004).
- [x] T016 [US2] Implement fenced plan/checkpoint commits, digest/size/kind verification, stable logical IDs and generation-qualified publications in internal/catalog/processing_window_work.go (FR-008, FR-009).
- [x] T017 [US2] Implement app publication/rehash/resume and compact progress with fault-boundary tests in internal/app/processing_window_work.go and processing_window_work_test.go (FR-008, FR-009, SC-004).
- [x] T018 [US2] Implement final acceptance/replacement retirement and artifact reference/cleanup rules in internal/catalog/processing_windows_state.go, exploration_outbox.go and library_state.go (FR-005, FR-010).
- [x] T019 [US2] Add backup selection, snapshot integrity and portable restore coverage in internal/catalog/backup.go and internal/catalog/processing_window_work_test.go, preserving ready checkpoint bytes across catalog/storage backends (FR-010, FR-013).
- [x] T020 [US2] Prove cancellation retains only verified windows and explicit retry skips them in internal/app/processing_window_work_test.go (FR-009, FR-010, SC-004).

## Phase 5: US3 finite resource recovery

- [x] T021 [P] [US3] Add CUDA OOM/batch/CPU/deadline/route preservation tests in internal/processing/device_policy_test.go and scripts/test-processing-worker.py (FR-011, FR-012, SC-005).
- [x] T022 [US3] Implement at most one compatible pyannote batch reduction before allowed CPU fallback in internal/processing/device_policy.go, scripts/processing-worker.py (FR-011, FR-012).
- [x] T023 [US3] Extend actual-device attempt receipts/validators and runtime schemas consistently in internal/contracts/device_attempt.go and schemas/v1.0.3/runtime-response.schema.json; preserve historical forms (FR-012, FR-014).
- [x] T024 [US3] Prove checkpoint reuse through compatible device recovery and finite active execution in internal/app/processing_window_work_test.go (FR-003, FR-011, SC-005).

## Phase 6: Convergence and delivery

- [x] T025 Update authoritative docs/v1.0.3/ and CHANGELOG.md with defaults, sample units, resume/cleanup, uncertainty, provider policy and dated architecture decisions (FR-014).
- [x] T026 Run npm run check, npm test, affected Go test/vet, Python worker tests, frontend/site checks; record scoped evidence in specs/063-resumable-long-recording/verification.md (SC-006).
- [x] T027 Run elected native media and applicable alternate-backend/backup fixtures; separately retain elected actual CPU/NVIDIA engine evidence or state exact unverified limits in specs/063-resumable-long-recording/verification.md (SC-002, SC-006).
- [x] T028 Converge requirements, implementation and task evidence; check encoding, formatting and git diff; commit on codex/063-resumable-long-recording (all requirements).
- [x] T029 Push and open official PR, attach it, map #212-#216 to In review/S063, and retain CI receipts under .local/s063/ (FR-014).
- [ ] T030 Address every review comment, use at most two rounds, verify final-head CI and unresolved threads, then hand off for owner merge without merging (SC-006).

## Dependencies and parallel execution

T001-T003 gate implementation. T004 freezes shared interfaces; T005/T006 may proceed independently with assigned files. Extraction T007-T010, worker/stitching T011-T012/T021-T023, and catalog T015-T016/T018-T019 use the Spec Kit parallel team strategy, with root integrating app/configuration/docs. Shared-file changes are coordinated. App T013/T017 uses those exported interfaces. All acceptance tasks gate publication; partial issue implementation is not completion.

US1 proves bounded accurate processing; US2 adds interruption/retry; US3 proves resource behavior across both. Tests precede associated behavior. Focused regressions precede broad checks; all three stories are required for this authorized slice.

## Phase 7: Convergence

- [x] T031 Preserve source-clock end bounds for positive/negative offsets in window assembly and catalog acceptance per FR-002, FR-005, US1/AC1 (partial; HIGH).
- [x] T032 Fence implicit supplied-current transcript reuse against changed source tracks and retain cue/voice source scope per FR-003, FR-007 (partial; HIGH).
- [x] T033 Reconcile available window publications orphaned before checkpoint commit while preserving live producer authority and ready references per FR-008, FR-009, US2/AC1 (partial; HIGH).
