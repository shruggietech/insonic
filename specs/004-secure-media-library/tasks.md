# Tasks: S004 Secure media library

**Input**: [spec.md](spec.md), [plan.md](plan.md), research/data-model/contracts.
**Tests**: Required by autopilot, before implementation; security and workspace isolation are blocking.

## Phase 1: Setup

- [x] T001 Verify S004/branch/directory/issues alignment and read authority in specs/004-secure-media-library/spec.md.
- [x] T002 Execute specify/clarify/checklist/plan and research decisions in specs/004-secure-media-library/.
- [x] T003 Establish hidden noninteractive tooling and fixture prerequisites in build/run-hidden.ps1.

## Phase 2: Foundational

- [x] T004 Freeze legacy catalog migration definitions and test actual v1/v2 upgrades in internal/catalog/migration.go and migration tests.
- [x] T005 Add typed current library/model and durable fenced workflow/cleanup records with portable snapshots in internal/catalog/.
- [x] T006 Add separate bounded stdout/stderr preserving failure outcomes in internal/process/process.go.
- [x] T007 Define exact runtime/library/model/credential payload schemas and routing validation in internal/contracts/ and schemas/v0.0.0/.

## Phase 3: US1 persistent credentials

- [x] T008 [P] [US1] Write native/vault lifecycle, restart, replacement, isolation and disclosure tests in internal/secrets/*_test.go.
- [x] T009 [P] [US1] Implement workspace-scoped native Windows/macOS/Linux noninteractive stores in internal/secrets/.
- [x] T010 [US1] Implement fixed Argon2id/AES-GCM vault, session choice, atomic exclusive updates and status-only manager in internal/secrets/.
- [x] T011 [US1] Integrate catalog-independent bootstrap and shared provider into cmd/insonic/, internal/app/ and desktop/bridge.go.

## Phase 4: US2 downloaded models

- [x] T012 [P] [US2] Write exact multi-file manifest/digest/retry/cancel/restart tests in internal/models/*_test.go.
- [x] T013 [US2] Implement exact registry/acquisition/publication/verification and materialization in internal/models/.
- [x] T014 [US2] Add shared model CLI/runtime/bridge operations in cmd/insonic/, internal/app/ and desktop/.

## Phase 5: US3 original media admission

- [x] T015 [P] [US3] Write stable-original/reference/mutation/manifest/extractor/recovery tests in internal/library/*_test.go.
- [x] T016 [US3] Implement stable copy/reference/remote acquisition and CSV/JSON manifests in internal/library/.
- [x] T017 [US3] Implement pinned-tool raw metadata extraction/provenance, nullable measured duration and exact stream mapping in internal/library/.
- [x] T018 [US3] Implement fenced admission/list/show/metadata/relocation operations and supplied subtitle assets in internal/library/.
- [x] T019 [US3] Integrate actual work executor/recovery/cancel/retry and media CLI/bridge routes in internal/app/, internal/runtime/ and cmd/insonic/.

## Phase 6: US4 dates/current metadata

- [x] T020 [P] [US4] Write historical timezone/offset/fold/gap/date-only/conflict tests in internal/library/date_test.go.
- [x] T021 [US4] Implement recorded local/IANA/offset rules, versioned precedence and per-item date resolution/correction in internal/library/.
- [x] T022 [US4] Implement atomic current-metadata replacement plus durable physical retirement and original retention in internal/library/ and internal/catalog/.
- [x] T023 [US4] Verify refresh/restart/cleanup and SQLite/PostgreSQL plus filesystem/S3 parity in internal/catalog/ and internal/library/ integration tests.

## Phase 7: Cross-cutting verification/publication

- [x] T024 Qualify real native secrets/tools/media on three OSes and affected backend fixtures in scripts/qualify.py and .github/workflows/ci.yml.
- [x] T025 Reconcile authoritative docs/schema examples and current metadata contracts in docs/v0.0.0/, schemas/v0.0.0/ and CHANGELOG.md.
- [ ] T026 Run analyze, implement/converge and all affected repository/site/native checks; record exact evidence in specs/004-secure-media-library/verification.md.
- [x] T027 Commit and automatically push codex/004-secure-media-library, publish official PR closing #5/#6 and attach it to this chat.
- [ ] T028 Resolve every initial review/comment, request at most one second review round, verify final head green and stop before owner merge.

## Dependencies and parallel execution

Setup precedes analyze and implementation. Foundational catalog contracts block service integration. US1 supplies selected authenticated providers; US2 and US3 can use injected fixture providers until US1 integration. US4 extends US3. Native credential files, catalog files, library files and model/integration files have separate owners, so their independent tests/design can run in parallel under the team strategy in the installed tasks template. Shared files are edited sequentially. All story acceptance must complete; independent increments do not reduce the authorized two-issue outcome.

## Implementation strategy

Complete core contracts, then independently test story services, integrate one shared executor and CLI/bridge, run convergence, fix every gap, verify and publish. Passing partial checks does not satisfy closure. No third external review round.

## Phase 8: Convergence

- [x] T029 [US4] Preserve approximate recording-date bounds through current selection and refresh in internal/library/date.go and acceptance tests (FR-008, US4/AC1).
- [x] T030 [US1] Verify live backend selection, unlocked status, credential replacement for subsequent S3/PostgreSQL connections and rejected selection rollback in internal/app/, internal/catalog/store.go and desktop/ (FR-001/002, US1/AC1/4).
- [x] T031 [US2] Verify restored exact manifest digest and reconciled-download scratch cleanup in internal/models/ and internal/catalog/ (FR-003/010, US2/AC1/2).
- [x] T032 [US3] Qualify complete extractor support-file configuration beyond 16 KiB through CLI/runtime loading in internal/app/work.go and cmd/insonic/main.go (FR-006, US3/AC2).
- [x] T033 [US3] Preserve large batch acceptance with bounded result journals, paged list/work-result responses and leased oversized-metadata reports in internal/catalog/work.go, internal/app/work.go and runtime/schema/client tests (FR-004/005/010, US3/AC3/4).

## Phase 9: Convergence

- [x] T034 Reject historical current library/model/work records when later receipts remain in imported snapshots, including cancelled-work replay regressions in internal/catalog/ (FR-010/012, SC-003, partial, CRITICAL).
- [x] T035 Validate selected backend and required secret input before cold-start credential unlock/load in cmd/insonic/, matching runtime/desktop behavior (FR-001/002, US1/AC2/4, partial, HIGH).
- [x] T036 Compare recording dates by actual precision/instant/bounds and preserve explicit owner interpretation changes on duplicate import in internal/library/ (FR-008, US4/AC1/2, partial, HIGH).
- [x] T037 Expose approximate UTC recording-date bounds through CLI import/refresh/correction flags with exact integer validation in cmd/insonic/ (FR-004/008, CLI-first constitution, partial, HIGH).

## Phase 10: Convergence

- [x] T038 Reconcile externally removed native credentials on explicit deletion while retaining locked/unavailable failures in internal/secrets/ (FR-001, US1/AC2, partial, HIGH).
- [x] T039 Reject malformed apparent remote URLs before durable admission and require strict numeric timezone offsets in internal/library/ (FR-002/008, US4/AC2, partial, HIGH).
- [x] T040 Resume unselected report retirement after transient storage deletion failure on work retry in internal/library/ (FR-010/011, SC-004, partial, HIGH).

## Phase 11: External review

- [x] T041 Retire unreferenced original/subtitle candidates after failed copy admission using the current-reference fence, with failure/retry regressions (FR-007/010/011, review 4210975392).
- [x] T042 Use stable logical downloaded-model version identity and reject changed manifests before acquisition, preserving unchanged replay (FR-003, US2/AC2, review 4210975402).
- [x] T043 Align bounded normalized manifest input, durable payload and IPC ingress budgets while retaining smaller paged responses, with real-runtime large-batch regression (FR-004/005/010, review 4210975415).
- [x] T044 Share long-operation deadlines between desktop and runtime calls with deadline/cancellation coverage (FR-004, desktop parity, review 4210975459).
- [x] T045 Keep failed batch admissions retryable with preserved per-item results and no duplicate successful entries in internal/app/ (FR-010, review 4210975392).
