# Tasks: Managed local processing and task readiness

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/runtime.md.

## Phase 1: Setup

- [x] T001 Establish S025/#69 branch, feature selector and Project tracking in specs/025-managed-local-processing/spec.md.
- [x] T002 Complete researched plan, contracts and blocking consistency analysis in specs/025-managed-local-processing/plan.md and analysis.md.

## Phase 2: Foundational

- [x] T003 Add exact runtime/uv assets and full platform hash locks in internal/workers/assets/ per FR-001/006/017.
- [x] T004 Add schema/request definitions and worker-profile/receipt entities in schemas/v1.0.1/ and internal/workers/types.go per FR-001/003/015/019.

## Phase 3: User Story 1, usable environments

- [x] T005 [US1] Add failing isolation, archive/identity, incompatible probe, cancellation and offline fixtures in internal/workers/workers_test.go per FR-003/004/005/015/016/018.
- [x] T006 [US1] Implement bounded weight-free capability inspection and automatic dependency identity in internal/workers/probe.go per FR-002/003/010/016.
- [x] T007 [US1] Implement confined verified runtime acquisition/extraction and hash-locked isolated preparation in internal/workers/setup.go per FR-004/005/009/017.
- [x] T008 [US1] Add failing durable setup/replay/cancel/revision conflict tests in internal/app/workers_test.go per FR-005/014/018.
- [x] T009 [US1] Implement shared plan/setup/reuse and guarded settings publication in internal/app/workers.go and work.go per FR-001/004/005/014.
- [x] T010 [US1] Freeze and revalidate environment receipts before engine invocation in internal/processing/adapters.go and internal/voicemodels/adapter.go per FR-003/011/014.

## Phase 4: User Story 2, exact model acquisition

- [x] T011 [US2] Add failing builtin-role, local reuse, redirect credential, destination and byte-progress tests in internal/models/defaults_test.go and service_test.go per FR-007/008/009/015/018.
- [x] T012 [US2] Consolidate qualified model pins into internal/models/defaults.json and expose immutable choices in defaults.go per FR-007/017.
- [x] T013 [US2] Extend existing acquisition for verified elected local resources, bounded byte progress, frozen storage and scoped HF/CDN transport in internal/models/service.go per FR-008/009/015.
- [x] T014 [US2] Expose default preview/acquisition through shared runtime and current model jobs in internal/app/workers.go and model_dependencies.go per FR-001/007/008/014.

## Phase 5: User Story 3, task and device readiness

- [x] T015 [US3] Add failing five-task/device/model/storage readiness fixtures in internal/app/workers_test.go per FR-002/009/010/011/012/013/018.
- [x] T016 [US3] Compose import/subtitle/transcription/diarization/training readiness and corrections in internal/app/workers.go per FR-002/009/010/011/012/013.
- [x] T017 [US3] Reuse readiness in doctor and pipeline inspection in internal/app/app.go and configured.go per FR-012/016.
- [x] T018 [US3] Add shared CLI inventory/parser/help and human corrective output in cmd/insonic/ per FR-001/019.
- [x] T019 [US3] Add frontend contract/device/stale-operation fixtures in desktop/frontend/src/workers.test.ts per FR-001/011/014/018.
- [x] T020 [US3] Implement Settings worker/default-model/readiness controls and pipeline facts in desktop/frontend/src/workers.tsx, settings.tsx and screens.tsx per FR-001/007/012/019.

## Phase 6: Closure and verification

- [x] T021 Declare and verify shipped worker metadata/lock bytes in scripts/package-desktop.py and internal/packageenv/packageenv.go per FR-006/019.
- [x] T022 Consolidate maintainer model pins and document executable processing setup in scripts/qualify-processing.py, docs/v1.0.1/ and CHANGELOG.md per FR-017/018/019.
- [x] T023 Run root/frontend/Python/Go/schema/site/public/offline checks and record evidence in specs/025-managed-local-processing/verification.md per FR-018/019 and SC-001/006.
- [x] T024 Run owned opt-in actual CPU engine/profile qualification, distinguish unexecuted combinations and record evidence in specs/025-managed-local-processing/verification.md per FR-018/019.
- [x] T025 Converge against all scenarios/requirements and append any gaps to specs/025-managed-local-processing/tasks.md before publication.
- [ ] T026 Commit/push, publish official PR, resolve all comments in at most two rounds and verify final-head green CI/review threads in specs/025-managed-local-processing/verification.md.

## Dependencies and implementation strategy

Setup -> shared identities -> US1. US2 retains existing acquisition and can be designed independently; US3 composes both. Client integration follows stable shared contracts. Tests precede corresponding implementations. No independent source edits run concurrently in this shared checkout. Read-only research/verification may run in parallel; the Spec Kit plan research agents cover engines, model acquisition and client surfaces. Each story's independent validation is in spec.md; full delivery includes all stories, not only environment preparation.

## Phase 7: Convergence

- [x] T027 Add bounded Linux glibc prerequisite inspection before managed acquisition in internal/workers/platform_*.go and shared setup previews per FR-009/017 and plan: target platform (partial).
- [x] T028 Sanitize host-specific setup/local-model work in portable capture and bind destination environment receipts alongside platform tools, with restore regressions in internal/catalog/backup_input.go and internal/app/portable_work.go per FR-014/018 and Constitution IV (partial).
- [x] T029 Complete the current CLI/desktop capability matrix and document portable setup prerequisites in docs/v1.0.1/desktop.md and processing-setup.md per FR-019 (partial).

## Implementation evidence notes

T010 is satisfied by the shared processing adapter receipt check, which also covers the existing app speaker-profile embedding/training bridge; no duplicate check was added to the custom adapter. T019 lives in desktop/frontend/tests/workers.test.mjs and workers-ui.test.mjs under the existing frontend build harness. T027 uses internal/workers/platform.go with a runtime platform selector rather than duplicated platform files. T023/T024 evidence is recorded in verification.md; T026 remains open until final-head CI and external reviews complete.

## Phase 8: Initial external review

- [x] T030 Combine peak capacity on physical filesystem volumes and conservatively handle unknown relationships, with runtime/volume regressions in internal/app/workers_test.go and internal/workers/workers_test.go per FR-009 and review r4262506536.
- [x] T031 Reject a new gated builtin download without its protected credential reference before enqueue, preserving previews/verified local reuse and prior references, per FR-015/017 and review r4262506548.

## Phase 9: Final external review

- [x] T032 Bind runtime/uv extraction to privately owned byte snapshots hashed during copy, with source-replacement/tampering regressions in internal/workers/workers_test.go per FR-003/005/015 and review r4262667308.
- [x] T033 Apply the actual bounded receipt/probe space budget to reuse plans and execution, with shared-plan regression coverage per FR-004/009 and review r4262667323.
