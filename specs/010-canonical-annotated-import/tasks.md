# Tasks: S010 Canonical annotated import

**Input**: [spec](spec.md), [plan](plan.md), [research](research.md), [data model](data-model.md), [contracts](contracts/admission.md).
**Tests**: Required risk-focused TDD and affected CI parity; no models in required checks.

## Phase 1: Setup

- [x] T001 Confirm S010 directory/branch/issues and execute specify/clarify in specs/010-canonical-annotated-import/spec.md.
- [x] T002 Research independent boundaries and generate design/checklists in specs/010-canonical-annotated-import/.
- [x] T003 Review requirements quality and resolve blocking analysis in specs/010-canonical-annotated-import/verification.md.

## Phase 2: Foundation

- [x] T004 Add acceptance failures for atomic admission, source/document revisions, replay and portable proofs in internal/catalog/admission_test.go.
- [x] T005 Refactor recording-save transaction and implement one composite admission receipt, current/snapshot proofs and sidecar cleanup in internal/catalog/admission.go, recording.go, library_state.go and catalog.go.
- [x] T006 Define bounded local speaker identifiers across internal/contracts/, internal/catalog/selection.go, prior_evidence.go and affected schemas without narrowing global UUID identity.

## Phase 3: US1 annotated transcript import

- [x] T007 [US1] Add preservation/native-label/historical/integer/invalid-input fixtures in internal/subtitles/admission_test.go.
- [x] T008 [US1] Package exact historical contracts and implement verified lossless translation, direct preservation and non-clobber derivation in internal/subtitles/admission.go and validation.go.
- [x] T009 [US1] Add standalone target/collision/replacement/retry failure tests in internal/library/admission_test.go and internal/app/.
- [x] T010 [US1] Implement common transcript admission and optional explicit diarization election through internal/library/admission.go and internal/app/domain.go.
- [x] T011 [US1] Convert/retire legacy managed sidecars through explicit current-document admission in internal/library/admission.go and internal/catalog/admission.go.

## Phase 4: US2 remote and embedded inputs

- [x] T012 [US2] Add independent HTTP credentials/bounds/MIME/redirect/cancellation and embedded precedence/offset fixtures in internal/library/admission_test.go.
- [x] T013 [US2] Extend independent transcript acquisition and byte validation in internal/library/acquisition.go, types.go and admission.go.
- [x] T014 [US2] Implement inspectable embedded track discovery/selection/extraction and unsupported OCR diagnostics in internal/library/embedded.go.
- [x] T015 [US2] Extend versioned manifests and consistent per-item/default precedence in internal/library/manifest.go and schemas/v0.0.0/.
- [x] T016 [US2] Implement shared CLI transcript grammar/flags/help and runtime operations in cmd/insonic/domain.go, main.go and internal/app/domain.go.

## Phase 5: US3 canonical audio admission

- [x] T017 [US3] Add stereo/surround/MP3/grouped offsets/duration/path scrubbing/recovery fixtures in internal/library/canonical_test.go.
- [x] T018 [US3] Implement owner-approved canonical conversion/probe/provenance and dedup in internal/library/canonical.go, extract.go and service.go.
- [x] T019 [US3] Scrub accepted source locators from durable input/metadata with receipt integrity and retry preservation in internal/catalog/admission.go, library_state.go and internal/library/extract.go.
- [x] T020 [US3] Reconcile selected canonical track clocks with processing/playback consumers in internal/processing/audio.go and internal/app/playback_preview.go.
- [x] T021 [US3] Qualify pinned MP3 V0 support on every platform and update required source packaging/notices in internal/qualification/media-tools.json and scripts/build-media-source.py.

## Phase 6: Integration and delivery

- [x] T022 Implement consistent desktop admission/selection/replacement controls and rendered tests in desktop/frontend/src/screens.tsx and desktop/frontend/tests/.
- [x] T023 Update affected schema examples, authoritative docs, glossary, constitution and changelog in schemas/, docs/, .specify/memory/constitution.md and CHANGELOG.md.
- [x] T024 Run repository/schema, Go/vet/affected race, frontend/site and native/backend checks; record actual evidence in specs/010-canonical-annotated-import/verification.md.
- [x] T025 Execute speckit-converge against every requirement and child acceptance, repair remaining gaps and mark actual completed tasks in specs/010-canonical-annotated-import/tasks.md.
- [x] T026 Commit/push official PR, address every review thread, request at most one second round, verify final-head CI and stop before owner merge in specs/010-canonical-annotated-import/verification.md.

## Dependencies and parallel opportunities

Foundation precedes current publication. US1 document preservation can be tested independently; US2 uses that common boundary; US3 canonical media is needed for integrated new admission. CLI/desktop/docs follow stable contracts. Read-only research ran in parallel; implementation touching shared transaction/service files runs sequentially. Independent tests and file inspections may be batched.

## Implementation strategy

Build failing document/transaction fixtures first, preserve compatibility for readable legacy rows, integrate new admission once current authority is proven, then run bounded native/backend acceptance. Partial work is never treated as child completion; full bulk legacy-audio migration and audio replacement remain #33/#30.
## Phase 7: Convergence

- [x] T027 Freeze standalone target identity and observed source/document revisions at runtime enqueue, preserve request replay identity, and test delayed stale replacement per FR-002/FR-014 and plan D06 (partial, HIGH).

## Phase 8: Convergence

- [x] T028 Validate supplied timed speaker intervals against known canonical source-clock audio bounds while preserving native cue conflicts and unknown duration, with acceptance tests for zero/nonzero clocks (missing, HIGH; FR-013 and issue #31 media-alignment acceptance).
