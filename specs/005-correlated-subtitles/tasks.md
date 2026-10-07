# Tasks: S005 Current correlated subtitles

**Input**: specs/005-correlated-subtitles/spec.md, plan.md, research.md, data-model.md and contracts/runtime.md.
**Scope**: #21/#7; codex/005-correlated-subtitles; all four P1 stories. Tests are explicitly required by the specification.
**Rules**: Run tests before their corresponding implementation. Mark completion only with evidence. No inference/model loading/downloads in CI.

## Phase 1: Setup

- [X] T001 Confirm literal slice/branch/issues and complete requirement review plus read-only analysis in specs/005-correlated-subtitles/analysis.md.
- [X] T002 [P] Pin exact Cueson 1.2.0 packages/schema and retire old version references in internal/qualification/dependencies.json, schemas/ and scripts/qualify.py.
- [X] T003 [P] Extend exact FFmpeg executable qualification and hidden bounded launch configuration in internal/qualification/media-tools.json and scripts/media-tools.py.

## Phase 2: Foundation

- [X] T004 Freeze historical schema3 DDL/digest and add migration regression coverage in internal/catalog/historical.go and internal/catalog/migration_test.go before changing reflected records.
- [X] T005 Specify and test current Recording and external mapping validation, sole document authority, explicit ready/no-speech/no-timed-subtitles states and exact integer source mapping in internal/catalog/recording_test.go.
- [X] T006 Implement current Recording/mapping domains, master schema and schema4 migration in internal/catalog/records.go, internal/catalog/migration.go and schemas/v0.0.0/master.schema.json.
- [X] T007 Test and implement latest recording/mapping receipt proofs, workspace isolation and snapshot tamper rejection in internal/catalog/library_state.go and internal/catalog/recording_test.go.

## Phase 3: US1 Assemble current annotated subtitles (P1)

**Goal**: Supplied subtitles plus deterministic turns produce the sole validated current document.
**Independent check**: Real fixtures, exact Cueson, overlap/untimed/integer cases and strict export.

- [X] T008 [P] [US1] Write driver/assembly/export regressions for unknown/zero duration, overlap, cue intersections, sub-ms collapse and bounded upstream losses in internal/subtitles/subtitles_test.go.
- [X] T009 [US1] Implement exact verified Cueson driver, local validation/inspection and loss-aware strict export in internal/subtitles/driver.go.
- [X] T010 [US1] Implement recording-local UUID assembly and deliberate millisecond projection with diagnostics in internal/subtitles/assemble.go.
- [X] T011 [US1] Test atomic accepted recording replacement, immutable source inputs and stale candidate rejection in internal/catalog/recording_test.go.
- [X] T012 [US1] Implement fenced recording commit and current retrieval with receipt-only digests/references in internal/catalog/recording.go.
- [X] T013 [US1] Add shared supplied-subtitle processing, show and export routes with CLI/desktop parity in internal/app/, internal/contracts/ and cmd/insonic/.

## Phase 4: US2 Process audio/video locally (P1)

**Goal**: Configured local extraction/recognition/diarization works without automatic provider fallback.
**Independent check**: Deterministic adapter checks, followed by explicit non-CI maintainer real-engine qualification.

- [X] T014 [P] [US2] Write deterministic extraction/session/adapter regressions for cancellation, confined managed models, malformed outputs and cleanup in internal/processing/processing_test.go and scripts/test-processing-worker.py.
- [X] T015 [US2] Implement mapped PCM audio session with explicit channel/stream, rational source clock, bounded decoding and leased private scratch in internal/processing/audio.go.
- [X] T016 [US2] Implement lazy offline exact local recognition/diarization workers and capability failures in scripts/processing-worker.py and internal/processing/adapters.go.
- [X] T017 [US2] Implement automatic quality diagnostics and truthful no-speech/unusable timing results in internal/processing/quality.go and internal/subtitles/assemble.go.
- [X] T018 [US2] Integrate real adapters, managed model/settings selection and phase-only checkpoints with durable work in internal/app/processing.go and internal/app/work.go.
- [X] T019 [US2] Expose shared CLI/runtime/desktop processing contracts and bounded paged results in cmd/insonic/, internal/contracts/ and desktop/.

## Phase 5: US3 Replace results and recover work (P1)

**Goal**: Reruns and mapping corrections leave one current authority and retire obsolete derived bytes.
**Independent check**: Diarization-only/text-preserving rerun, replacement races, restart cleanup and downstream invalidation.

- [X] T020 [P] [US3] Write independent rerun/mapping/CAS/cancellation/cleanup and original-preservation regressions in internal/app/processing_test.go and internal/catalog/recording_test.go.
- [X] T021 [US3] Implement independently elected transcription/diarization reruns, current cue reuse and external known-speaker correction in internal/app/processing.go and internal/catalog/recording.go.
- [X] T022 [US3] Migrate timed legacy segment attribution to reference-only evidence, remove stale preparation/membership alternatives and document invalidation in internal/catalog/records.go, internal/catalog/migration.go and internal/catalog/recording.go.
- [X] T023 [US3] Queue/recover physical retirement of superseded mapped audio and evidence clips with current reference fences in internal/catalog/library_state.go, internal/artifact/ and internal/library/service.go.
- [X] T024 [US3] Update downstream query/playback/export/training reference resolution and schema validation in internal/catalog/, internal/app/ and schemas/.
- [X] T025 [US3] Qualify migrations/current replacement/snapshot restoration on SQLite/PostgreSQL and filesystem/S3 in internal/qualification/ and scripts/qualify.py.

## Phase 6: US4 Reproducible inexpensive qualification (P1)

**Goal**: Ordinary Git checkout supplies real media and required checks remain inference-free.
**Independent check**: Both fixture paths pass deterministic native qualification; explicit maintainer command reports actual engine evidence separately.

- [X] T026 [P] [US4] Commit lossless speech and short Sintel dialogue media, exact hashes/size/license/source revision/crop/tool recipe and checked references in tests/fixtures/media/.
- [X] T027 [US4] Write fixture provenance/digest/reference checks and deterministic overlap/silence/untimed/integer coverage in scripts/test-media-fixtures.py and internal/subtitles/subtitles_test.go.
- [X] T028 [US4] Integrate real FFmpeg/Cueson fixture qualification on three native OS jobs using supplied/deterministic results in scripts/qualify.py and .github/workflows/ci.yml.
- [X] T029 [US4] Implement explicit opt-in non-CI real-engine qualification, guard tests and bounded resource/quality receipts in scripts/qualify-processing.py and scripts/test-processing-worker.py.
- [X] T030 [US4] Run actual locally elected engines on committed fixtures outside CI and record measured behavior and limitations in specs/005-correlated-subtitles/verification.md.

## Phase 7: Convergence, validation and publication

- [X] T031 Reconcile public authoritative contracts, legacy planning and changelog with current sole-document behavior in docs/, specs/ and CHANGELOG.md.
- [X] T032 Run npm run check, npm test, affected Go/race/vet/site/schema/native/backend checks and quickstart scenarios; inspect UTF-8/LF/mojibake in specs/005-correlated-subtitles/verification.md.
- [X] T033 Run Spec Kit convergence, fix every critical finding and record final evidence in specs/005-correlated-subtitles/analysis.md and verification.md.
- [X] T034 Commit/push codex/005-correlated-subtitles and publish official PR closing #21/#7; attach PR to this chat and record URL in specs/005-correlated-subtitles/verification.md.
- [X] T035 Respond to and resolve every external review, request at most one combined second round, finish review rounds before waiting on final CI and record exact rounds in specs/005-correlated-subtitles/verification.md.
- [X] T036 Confirm affected CI green and all reviews satisfied, then hand the PR to the owner for final review/squash merge; do not merge or release.

## Dependencies and execution order

Setup precedes foundation. T004 freezes historical schema before T006 changes reflected types. T005 precedes T006; T007 precedes accepted snapshot qualification. Story integration requires foundation, while isolated story tests and research can proceed against agreed interfaces. US1 is the first independently validated increment; US2 adapter implementation can proceed in parallel after foundation without editing US1/catalog files. US3 integrates US1/US2 and verifies replacement; US4 fixture preparation can proceed independently, while native qualification needs their interfaces. Publication follows all stories and convergence.

Parallel examples: the subtitle owner handles T008-T010; the processing owner handles T014-T017; the fixture owner handles T026-T027. Parent serializes shared catalog/runtime/schema/docs integration and common qualification files. T002/T003 may proceed while catalog foundations are built because they affect separate files. Collaborators coordinate shared scripts before editing.

## Implementation strategy

Complete setup and immutable historical foundation first. Validate US1 before runtime engine integration, then independently exercise adapters, reruns and references. Complete all four stories in this session; the supplied-subtitle increment is a checkpoint, not the delivery boundary. Real inference qualification remains explicit maintainer evidence, never a CI dependency. Resolve external reviews before final CI wait and stop at owner merge handoff.

## Requirement coverage

FR-001 T002/T009/T031; FR-002 T009/T012/T013; FR-003 T005/T006/T010/T022/T024; FR-004 T008/T010/T015; FR-005 T003/T015/T016/T018/T019; FR-006 T017/T030; FR-007 T020/T021; FR-008 T011/T012/T022/T023; FR-009 T007/T011/T018/T020/T025; FR-010 T008/T009/T013; FR-011 T026/T027; FR-012 T028/T029/T032; FR-013 T004/T006/T007/T022/T024/T025/T031; FR-014 T033-T036.


## Phase 8: Convergence

- [X] T037 Preserve maximum per-cue speaker multiplicity when reconstructing reusable timed activity, and retain untimed evidence only for unchanged cue content/timing, with overlap/repetition regressions in internal/app/processing.go and internal/app/recording_bounds_test.go per FR-003/FR-007 (partial, HIGH).
- [X] T038 Validate elected processing configuration identities and resource bounds consistently through runtime validation and the packaged master schema, without loading tools/models, in internal/processing/, internal/app/processing.go and cmd/insonic/recordings.go per FR-005/FR-012 (partial, HIGH).
