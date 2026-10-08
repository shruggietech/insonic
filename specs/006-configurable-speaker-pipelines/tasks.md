# Tasks: S006 Configurable pipelines, speakers and terminology

Input: [spec](spec.md), [plan](plan.md), research, data model, runtime contract and quickstart. Tests precede implementation under the autopilot protocol.

## Phase 1: Setup and foundation

- [x] T001 Confirm requested S006/#8/#9 directory/branch/issue alignment in `spec.md`.
- [x] T002 Complete specify/clarify/checklist/plan design artifacts in this feature directory.
- [x] T003 Review requirements checklist and run cross-artifact analysis before code changes in `analysis.md`.
- [x] T004 Freeze historical schema4 and specify schema5 snapshot/migration boundaries in `internal/catalog/`.

## Phase 2: User story 1, elected pipelines

- [x] T005 [P] [US1] Add failing pipeline CAS/proof/migration tests in `internal/catalog/pipelines_test.go` (FR001/014).
- [x] T006 [P] [US1] Add failing definition/election/capability/HTTP fixtures in `internal/pipeline/` (FR002..005).
- [x] T007 [US1] Implement current pipeline persistence/replay/snapshot proofs in `internal/catalog/` (FR001/014).
- [x] T008 [US1] Implement Local/Connected/Custom/full-stage overrides in `internal/pipeline/` (FR002/003).
- [x] T009 [US1] Implement bounded cancellable hosted worker client and synthetic-secret redirect/error tests in `internal/pipeline/` (FR003/004).
- [x] T010 [US1] Add election/independent-rerun/concurrent-default tests in `internal/app/pipelines_test.go` (FR005/006).
- [x] T011 [US1] Wire immutable elected pipeline and hosted/local stages into `internal/app/processing.go` (FR002..006).
- [x] T012 [P] [US1] Implement shared request schemas/CLI pipeline commands in `schemas/`, `cmd/insonic/` (FR013).

## Phase 3: User story 2, speaker continuity

- [x] T013 [US2] Add failing identity/alias/revision/selection/correction tests in `internal/catalog/` (FR007/008/012).
- [x] T014 [US2] Implement current speaker identities/aliases and CAS link validation in `internal/catalog/` (FR007).
- [x] T015 [US2] Implement bounded current mapping/document reference selection and exact timing resolution in `internal/catalog/`, `internal/speakers/` (FR008/012).
- [x] T016 [US2] Verify correction/invalidation, stale references, overlap, untimed and cross-recording deduplication in `internal/catalog/` (FR008/012).
- [x] T017 [US2] Wire speaker shared handlers in `internal/app/` (FR007/008/013).
- [x] T018 [P] [US2] Expose shared CLI/bridge speaker operations and schemas in `cmd/insonic/`, `schemas/` (FR013).

## Phase 4: User story 3, recognition context

- [x] T019 [US3] Add failing terminology/link/scope/budget/determinism tests in `internal/catalog/`, `internal/speakers/` (FR009/010).
- [x] T020 [US3] Implement current CAS terms and deterministic compiler in `internal/catalog/`, `internal/speakers/` (FR009/010).
- [x] T021 [US3] Add deterministic worker hint-forwarding tests before implementing hotwords/context digest in `scripts/processing-worker.py`, `internal/processing/` (FR010).
- [x] T022 [US3] Freeze compiled context at enqueue and expose compile/term handlers in `internal/app/` (FR005/009/010).
- [x] T023 [P] [US3] Add term/context request/CLI schema examples in `schemas/`, `cmd/insonic/` (FR009/010/013).

## Phase 5: User story 4, quality diagnostics

- [x] T024 [P] [US4] Add failing configurable quality/mandatory-validation tests in `internal/processing/`, `internal/app/` (FR011).
- [x] T025 [US4] Implement pure quality configuration/thresholds and source-independent diagnostics in `internal/processing/` (FR011).
- [x] T026 [US4] Implement current speaker-correlation diagnostics and shared inspection in `internal/speakers/`, `internal/app/` (FR011/012).

## Phase 6: End-to-end verification and delivery

- [x] T027 Reconcile authoritative pipeline/speaker/worker/schema/docs and changelog in `docs/`, `schemas/`, `CHANGELOG.md` (FR016).
- [x] T028 Prove full catalog/snapshot contract and hosted PostgreSQL/S3 parity in `internal/catalog/`, `internal/qualification/` (FR014).
- [x] T029 Prove committed audio/video deterministic CLI/bridge fixtures in `scripts/qualify.py`, `internal/app/` (FR013/015).
- [x] T030 Run affected Go/race/vet, deterministic Python, npm check/test and site/offline checks; scan UTF8/LF/mojibake and record evidence in `verification.md` (FR014..016).
- [x] T031 Commit/push official PR linked to #8/#9, attach it and update tracking. Record at most two review rounds in `verification.md` (FR016).
- [x] T032 Respond to every finding, resolve all threads, verify final-head CI green and hand off for owner merge (FR016).

## Dependencies and parallel examples

T001..003 precede implementation. Catalog owner serializes T004/005/007/013..016/019/020/028. Pipeline owner works T006/008/009/021/024/025 in parallel after analysis, coordinating context types. CLI/schema/docs owner handles T012/018/023/027/029 once interfaces are agreed. Parent owns T010/011/017/022/026 and app integration. Shared catalog/schema files have one writer each, no concurrent edits. Every story is validated before final checks/publication; all32 tasks are required, not an MVP-only delivery.
