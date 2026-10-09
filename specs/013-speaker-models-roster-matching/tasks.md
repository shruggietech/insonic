# Tasks: Speaker models and roster-constrained matching

**Input**: [spec.md](spec.md), [plan.md](plan.md), research, data model and contracts.

Tests are required by FR-015 and autopilot; write meaningful failing cases before implementation. Task completion records implementation, not model accuracy.

## Phase 1: Setup

- [x] T001 Resolve active templates and persist S013 feature identity in specs/013-speaker-models-roster-matching/spec.md.
- [x] T002 Complete clarify/research/design and requirement review in specs/013-speaker-models-roster-matching/plan.md and contracts/speaker-work.md.

## Phase 2: Foundation

- [x] T003 Define portable corpus/model/profile/matching authority types and API boundaries in internal/catalog/speaker_models.go and internal/voicemodels/types.go (FR-001 through FR-014).
- [x] T004 Test and implement schema migration, manual mapping origin, real work linkage and immutable output metadata in internal/catalog/migration.go, recording_records.go and migration_test.go (FR-005, FR-007, FR-011, FR-014).

## Phase 3: US1 - Elected current-corpus training

Goal: publish usable validated speaker output from exact current independent evidence.

Independent test: complete multi-page fixture corpus training and reject source/mapping edits before acceptance.

- [x] T005 [US1] Write corpus completeness, tenancy, invalidation and stale publication tests in internal/catalog/speaker_models_test.go and internal/voicemodels/service_test.go (FR-001, FR-002, FR-006, SC-001).
- [x] T006 [US1] Implement reference-only datasets, summaries, filters and independently attributed complete corpus selection in internal/catalog/speaker_models.go and internal/voicemodels/dataset.go (FR-001, FR-002).
- [x] T007 [US1] Implement canonical source-clock preparation and configurable visible decode/duplicate/quality exclusions in internal/voicemodels/preparation.go (FR-003).
- [x] T008 [US1] Test then implement pinned local and selected hosted actual training adapters, declared output compatibility, bounded I/O and no fallback in internal/voicemodels/adapters.go and adapters_test.go (FR-004, FR-015).
- [x] T009 [US1] Implement durable train/resume/checkpoint/cancel/retry and immutable fenced model acceptance in internal/voicemodels/training.go and internal/catalog/speaker_models.go (FR-005, FR-006, SC-002).

## Phase 4: US2 - Exact discovery and retrieval

Goal: discover by speaker, revise current profile and retrieve immutable output.

Independent test: exact fetch verifies durable bytes; hosted-only and missing objects are diagnosed.

- [x] T010 [US2] Test then implement speaker-filtered lineage discovery and revisioned profile association in internal/catalog/speaker_models.go and speaker_models_test.go (FR-007).
- [x] T011 [US2] Test then implement exact output retrieval, destination confinement, hashes, hosted-only diagnostics and immutable compatibility metadata in internal/voicemodels/retrieval.go and retrieval_test.go (FR-007).

## Phase 5: US3 - Roster-constrained matching

Goal: identify existing voices only from eligible frozen roster profiles without overriding corrections.

Independent test: all score decision cases and roster/profile/source/manual races pass without recognition/diarization execution.

- [x] T012 [US3] Write empty/missing/unknown/ambiguous/multiple-voice/manual/stale-candidate tests in internal/voicemodels/matching_test.go and internal/catalog/speaker_models_test.go (FR-008 through FR-012, SC-003).
- [x] T013 [US3] Implement frozen roster/profile/evidence snapshots, bounded current matching provenance and atomic manual-preserving mapping acceptance in internal/catalog/speaker_models.go (FR-010, FR-011).
- [x] T014 [US3] Implement real offline embedding profile enrollment and compatible matching consumer in internal/voicemodels/embedding.go and scripts/processing-worker.py with independent CI guard (FR-004, FR-008, FR-009, FR-015).
- [x] T015 [US3] Implement matching-only preparation/scoring/unknown/ambiguity decisions and stale-result fencing in internal/voicemodels/matching.go (FR-008 through FR-012).

## Phase 6: US4 - Shared surfaces and portability

Goal: equivalent CLI/desktop runtime and portable backends.

Independent test: strict runtime/CLI/UI payloads and snapshot/backend tests preserve exact authority.

- [x] T016 [US4] Test then implement shared admission/dispatch/renewable-worker routing in internal/app/speaker_models.go and work.go (FR-013).
- [x] T017 [P] [US4] Implement strict CLI dataset/train/speaker-model/profile/match commands and help in cmd/insonic/speaker_models.go and recordings.go with command tests (FR-013).
- [x] T018 [P] [US4] Implement dataset/training/model and Library matching controls with stale-response guards in desktop/frontend/src and request fixtures (FR-013).
- [x] T019 [US4] Extend runtime/master snapshot/model contracts and portable graph/state validation in schemas/v0.0.0, tests and internal/explore/corpus.go (FR-014).

## Phase 7: Verification and publication

- [x] T020 Update authoritative docs/v0.0.0 voice-models/models/speakers/pipelines/media/schema/contracts/desktop/help/glossary/index and CHANGELOG.md to exact shipped contracts (FR-016).
- [x] T021 Run targeted tests, npm check/test, affected product/desktop/site builds and inspect encoding through specs/013-speaker-models-roster-matching/quickstart.md (FR-015, SC-001 through SC-004).
- [x] T022 Run read-only convergence and complete appended gaps in specs/013-speaker-models-roster-matching/tasks.md.
- [x] T023 Publish official PR automatically, resolve all findings within two rounds and verify green final-head CI; record receipts in specs/013-speaker-models-roster-matching/verification.md (SC-004).

## Dependencies and execution order

Setup -> Foundation -> US1. US2 and US3 use frozen catalog APIs and independently supplied output fixtures. US4 surfaces can be authored in separate files after boundary types are published; integration waits for service APIs. Verification/publication follows every story.

## Parallel examples and team strategy

- US1: catalog corpus/acceptance tests and service adapter tests use distinct files after T003; catalog and service owners coordinate API signatures.
- US2: catalog lineage/profile tests and retrieval tests can proceed separately against explicit fixtures.
- US3: catalog acceptance and embedding/scoring tests run separately, then integrate.
- US4: CLI and desktop edits are independent; root handles schemas/docs/backend projection, surface owner handles shared app integration.

Catalog owner: internal/catalog only. Adapter/service owner: internal/voicemodels, scripts/processing-worker.py and required processing export helper. Surface owner: internal/app, cmd/insonic, desktop. Root: schemas/tests/docs/specs, graph integration and all GitHub publication/review. Shared files are never edited concurrently.

## Implementation strategy

Validate elected training first, then exact retrieval, then constrained matching. This is an incremental verification order, not a reduced delivery scope. All stories complete before publication; owner final review/merge remains the stopping boundary.

## Phase 8: Convergence

- [x] T024 Retire obsolete owned preparation/model manifests and clear current lineage digests/publication references on evidence correction while retaining exact weights/checkpoints; scrub affected frozen matching inputs and validate fresh portable receipt proofs in internal/catalog/speaker_models.go and speaker_models_state.go per FR-003, FR-011, FR-014 (contradicts).
- [x] T025 Add exact speaker retrieval corruption/missing-object, hosted-only and malicious role/path fixtures with no partial destination acceptance in internal/voicemodels/retrieval.go and service_test.go per FR-007, FR-015, US2/AC2 (partial).
- [x] T026 Add a source-edit-during-training service fixture proving no accepted version and retirement of unaccepted owned outputs in internal/voicemodels/service_test.go per FR-006, FR-015, SC-002 (partial).
- [x] T027 Bind desktop speaker training/profile/matching mutations and result pages to the newest request/election, preventing earlier response replacement or cross-job outcome mixing, with late-response regressions in desktop/frontend/src/speaker-models.tsx and tests per FR-013 (partial).
- [x] T028 Require an explicit positive exact saved training profile revision at runtime admission, rejecting missing/zero revisions before enqueue in internal/voicemodels/configuration.go with regression coverage per FR-004, FR-010, FR-013 (contradicts).
- [x] T029 Preserve observed usable duration in unknown matching diagnostics when the minimum-evidence rule avoids embedding, with a quantitative regression in internal/voicemodels/match_service.go per FR-009 (partial).
- [x] T030 Preserve accepted public-intent replay after obsolete matching payload scrubbing using only request/input digests and portable proof validation in internal/catalog/speaker_work_reconciliation.go per FR-010, FR-013, FR-014 (partial).
- [x] T031 Reject empty current output publications before indexing and validate portable metadata state against catalog version state in internal/catalog/speaker_models.go and speaker_models_state.go per FR-006, FR-014 (partial).
