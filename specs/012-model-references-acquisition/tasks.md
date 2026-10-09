# Tasks: S012 Model references and acquire-on-demand

Input: [plan.md](plan.md), [spec.md](spec.md), research/data-model/contracts/quickstart.
Tests: Meaningful test-first integrity, tenancy, recovery and parity tests are required by autopilot and FR-014.

## Phase 1: Setup

- [x] T001 Verify S012/#34/branch/directory, explicit publication authorization and clean baseline in specs/012-model-references-acquisition/verification.md.
- [x] T002 Run installed specify/clarify/checklist/plan/tasks workflows and dispatch research in specs/012-model-references-acquisition/.

## Phase 2: Foundation

- [x] T003 Preserve schema-7 migration/snapshot proofs before alias/source additions in internal/catalog/historical_v7.go and migration.go.
- [x] T004 Define shared target/reference/resolution/compatibility and dependency payload contracts in internal/models/references.go, internal/catalog/model_references.go and internal/app/model_dependencies.go.

## Phase 3: US1 exact selection

Goal: Revisioned references independently resolve exact identities. Test: multiple versions, ambiguity, tenancy and remove/recreate stale edits.

- [x] T005 [US1] Add failing alias/source concurrency, replay, workspace and tombstone tests in internal/catalog/model_references_test.go.
- [x] T006 [US1] Implement bounded lowercase aliases, typed targets and CAS source/alias domains in internal/catalog/model_references.go, records.go, migration.go and snapshot.go.
- [x] T007 [US1] Preserve base/trained/hosted identity and lineage in exact resolution tests and implementation in internal/models/references.go and references_test.go.
- [x] T008 [P] [US1] Expose alias/reference runtime and CLI operations in internal/app/model_references.go and cmd/insonic/model_references.go once shared foundation contracts are ready.

## Phase 4: US2 acquire-on-demand

Goal: Freeze selections and acquire shared bundles before processing. Test: more than four consumers, cancel/retry/restart/retarget and import parity without engines.

- [x] T009 [P] [US2] Add dependency/concurrency/frozen retry/import regression tests in internal/app/model_dependencies_test.go and internal/library/manifest_test.go.
- [x] T010 [US2] Implement stable shared explicit/automatic acquisition identity and identical verified recovery in internal/models/service.go and internal/app/work.go.
- [x] T011 [US2] Freeze processing/pipeline models and gate scheduler claims on dependencies in internal/app/election.go, processing.go, model_dependencies.go and work.go.
- [x] T012 [US2] Preserve request replay, frozen dependency repair, independent cancellation and phase/status in internal/app/model_dependencies.go and internal/catalog/work.go where required.
- [x] T013 [US2] Freeze import aliases/digests, propagate expected digest and preserve partial retry/proof scrubbing in internal/app/work.go, internal/library/types.go, manifest.go and internal/catalog/library_state.go.
- [x] T014 [US2] Test corrupt/missing available bytes, resumability and shared acquisition fence in internal/models/service_integration_test.go.

## Phase 5: US3 discovery and compatibility

Goal: Declared metadata discovers pinned compatible bundles. Test: bounded mutable selectors, wrong task/runtime/roles, source routing and credential exclusion.

- [x] T015 [P] [US3] Add compatibility/source metadata negative and mutable selector tests in internal/models/compatibility_test.go and sources_test.go.
- [x] T016 [US3] Implement optional manifest compatibility and exact legacy role inference shared with loaders in internal/models/manifest.go, compatibility.go and internal/processing/adapters.go.
- [x] T017 [US3] Implement configured catalog discovery/resolution with bounded transport and pinned complete manifests in internal/models/sources.go.
- [x] T018 [US3] Wire discovery/source CRUD/inspection into internal/app/model_references.go and cmd/insonic/model_references.go.

## Phase 6: US4 surfaces and portability

Goal: Shared consistent references and portable records. Test: CLI/rendered desktop/native journeys and populated snapshot/backend parity.

- [x] T019 [P] [US4] Update model/source/alias runtime, manifest/pipeline/import and snapshot JSON contracts in schemas/v0.0.0/ and tests/schemas.test.mjs.
- [x] T020 [P] [US4] Add shared model choices, alias/discovery/settings and acquisition phases with rendered tests in desktop/frontend/src/ and desktop/frontend/tests/frontend.test.mjs.
- [ ] T021 [US4] Qualify alias/source/model portable proof relationships and graph/backend parity in internal/catalog/ and internal/graph/ tests.
- [x] T022 [P] [US4] Add executable CLI/native/desktop synthetic acquisition journeys in cmd/insonic/ tests, scripts/qualify.py and desktop/qualification.go.

## Phase 7: Polish and delivery

- [x] T023 Audit and update authoritative models/pipelines/voice-models/schema/desktop/contracts/glossary docs and CHANGELOG.md together.
- [x] T024 Run blocking speckit-analyze and resolve coverage/constitution findings in specs/012-model-references-acquisition/verification.md before implementation.
- [x] T025 Run full affected root/Go/vet/race/frontend/site/Python/native checks and record truthful evidence in specs/012-model-references-acquisition/verification.md.
- [x] T026 Run speckit-converge, resolve findings and check UTF-8/LF/mojibake and clean diff.
- [ ] T027 Commit, automatically push, publish official PR closing #34 and attach it to the chat; no merge/release.
- [ ] T028 Wait for CI and every review/reaction, answer and resolve all findings, request at most one second round and verify exact final head.
- [ ] T029 Complete delivery records and ping owner for final review/squash merge with remaining matching/training/release scope explicit.

## Dependencies and parallel team strategy

Setup and analysis gate precede implementation; T003/T004 establish foundation. US1 precedes integrated US2, US3 compatibility feeds US2 election, US4 consumes finalized shared contracts. Tests precede each implementation seam. Independent [P] tasks may use the installed tasks-template parallel team strategy: catalog/model developer, application/dependency developer and surface/schema developer coordinate interface contracts before edits. Shared files and mutation dependencies remain sequential; root integrates and verifies. No independent agent may push, request reviews or merge.

US1 MVP is exact reference resolution, but S012 completion requires all four stories. Parallel examples: T009 application fixture tests alongside T015 models fixture tests; T019 schemas alongside T020 rendered UX after contracts are stable. T023 docs may proceed after final shared behavior is known.

## Phase 8: Convergence

- [x] T030 Investigate and repair catalog single-record read contention revealed by large pagination race qualification without weakening JSON/workspace integrity, then rerun affected qualification per FR-007/FR-014 (partial).

## Phase 9: Convergence

- [x] T031 Include bounded capability and default-adapter compatibility summaries in actual model inventory, and use them in shared desktop choices with runtime/rendered regressions per FR-005/FR-010 and US4/AC1 (partial).
