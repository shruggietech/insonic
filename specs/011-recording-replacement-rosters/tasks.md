# Tasks: S011 Recording replacement and declared speaker rosters

**Input**: [spec](spec.md), [plan](plan.md), [research](research.md), [data model](data-model.md), [contracts](contracts/recording-updates.md).
**Tests**: Risk-focused TDD, workspace/current proofs, failure/replay and affected CI parity; no acoustic models.

## Phase 1: Setup

- [x] T001 Confirm S011 branch/directory/issues and execute specify/clarify in specs/011-recording-replacement-rosters/spec.md.
- [x] T002 Research independent boundaries and generate design artifacts in specs/011-recording-replacement-rosters/.
- [x] T003 Review requirements checklists and blocking analyze coverage in specs/011-recording-replacement-rosters/verification.md.

## Phase 2: Foundation

- [x] T004 Add failing elected-source/clear/keep/replay/current-evidence/leased-cleanup tests in internal/catalog/admission_test.go and evidence_test.go.
- [x] T005 Add failing audio-only roster, workspace/alias/CAS/current-proof and portable tests in internal/catalog/roster_test.go and postgres_test.go.
- [x] T006 Define roster header/member domains, normal additive schema migration and snapshot validation in internal/catalog/roster_records.go, records.go, migration.go and schemas/v0.0.0/catalog-snapshot.schema.json (membership maximum 1,000; selector maximum 512 UTF-8 bytes; persisted header/member revision >0, zero only the absent expectation; member revision equals header revision).
- [x] T007 Implement shared selector resolution, roster read/mutation/save transaction and current proofs in internal/catalog/roster.go, identity.go, records.go, library_state.go and contract.go, retaining independent expected_revision and undeclared/empty semantics.
- [x] T008 Extend elected CommitAdmission/source update, truthful untranscribed current state, source-bound evidence/mapping reconciliation and structural artifact retirement in internal/catalog/admission.go, library_state.go, recording.go and evidence.go.

## Phase 3: US1 Audio replacement

- [x] T009 [US1] Add failing targeted-media skip, transcript clear/keep/replace, stale/cancelled/response-lost and shared-artifact tests in internal/library/admission_test.go and internal/app/import_retry_test.go.
- [x] T010 [US1] Define targeted-media discriminator, explicit replacement policies and validation/merging in internal/library/types.go; preserve standalone transcript aliases by request kind.
- [x] T011 [US1] Extend target/current/roster freezing and durable accepted replay in internal/library/admission.go and internal/app/work.go without retaining local locators.
- [x] T012 [US1] Implement staged canonical audio replacement, applicability/known-bound checks, source-map validation and one current acceptance in internal/library/admission.go and extract.go.
- [x] T013 [US1] Reconcile playback/processing/extraction/current corpus fences and exercise identical kept document with changed audio in internal/app/playback_preview_test.go, processing tests and internal/catalog/evidence_test.go.

## Phase 4: US2 Declared roster editing

- [x] T014 [US2] Add failing shared runtime/CLI roster grammar, repeated selectors, idempotency and inactive lifecycle tests in internal/app/roster_test.go and cmd/insonic/domain_test.go.
- [x] T015 [US2] Expose roster operations through shared runtime validation/dispatch and plural CLI grammar in internal/app/roster.go, internal/contracts and cmd/insonic/recordings.go, domain.go and main.go.
- [x] T016 [US2] Add declared-speaker context projection and graph invalidation without acoustic/training evidence in internal/explore/corpus.go and corpus_test.go; include both graph adapter fixtures.
- [x] T017 [US2] Verify portable roster proofs, null-document current state and alternative catalog/backend parity in internal/catalog/roster_test.go, snapshot tests and postgres_test.go.

## Phase 5: US3 Consistent import and interfaces

- [x] T018 [US3] Add failing atomic initial roster, replacement retain/clear, alias freezing and per-item/default precedence tests in internal/library/admission_test.go and manifest_test.go.
- [x] T019 [US3] Integrate initial roster admission and explicit audio-replacement roster policy inside the same current catalog transaction in internal/library/admission.go and internal/catalog/admission.go.
- [x] T020 [US3] Add repeatable import speaker selectors and JSON/CSV/versioned request contracts in cmd/insonic/domain.go, internal/library/manifest.go and schemas/v0.0.0/import-manifest.schema.json and runtime-request.schema.json.
- [x] T021 [US3] Implement accessible shared desktop replacement and roster controls plus conflict/empty/undeclared presentation in desktop/frontend/src/screens.tsx and client.ts.
- [x] T022 [US3] Add rendered desktop matrix, new audio-only roster and replacement authority regressions in desktop/frontend/tests/frontend.test.mjs and qualification.ts.

## Phase 6: Integration and delivery

- [x] T023 Update authoritative contracts/help/examples/docs/glossary and changelog in schemas/, docs/v0.0.0/, cmd/insonic/ and CHANGELOG.md; include constitution 2.2.1 and remove hypothetical pre-v1 bulk audio migration claims.
- [x] T024 Extend bounded native CLI/desktop/package journeys for replacement and rosters in internal/qualification/, scripts/qualify.py and desktop/frontend/src/qualification.ts without inference.
- [x] T025 Run required root check/test, Go/vet/affected race, frontend/site and native/backend checks, recording actual evidence in specs/011-recording-replacement-rosters/verification.md (local qualification passed; alternative-backend and cross-platform results are verified by T027 exact-head CI).
- [x] T026 Execute speckit-converge against every requirement, story acceptance and plan decision; append and implement remaining build findings in specs/011-recording-replacement-rosters/tasks.md.
- [ ] T027 Commit/push official PR, close #30/#33 acceptance, identify #35 remaining matching scope and address every finding within two rounds, verify exact final-head CI, then stop before owner merge in specs/011-recording-replacement-rosters/verification.md.

## Dependencies and parallel opportunities

Foundation/current proofs precede publication. Roster and replacement tests can be researched independently; shared admission/migration/records implementation is sequential. US1 and US2 build on foundation; US3 integrates both. CLI and graph reads/test planning may proceed independently, but implementation touching common contracts stays ordered. Independent checks may be batched except native/package isolation.

## Implementation strategy

Build failure fixtures first, implement one elected current transaction and independent revisioned roster, then expose shared interfaces and qualify real bounded media plus portable adapters. All three stories are required for issue completion; an MVP is not slice completion. No roster acoustic matching, real models, bulk user audio migration, merge or release is performed.

## Phase 7: Convergence

- [x] T028 Require explicit transcript policy for retained subtitle publications and stage their compatible current document during keep, before atomic acceptance, in internal/library/replacement.go and admission.go per FR-004, FR-007 and US1/AC3 (partial, high).
- [x] T029 Invalidate current source evidence on source revision changes and validate retained selected stream/channel mapping against candidate layout in internal/catalog/evidence.go and internal/library/replacement.go per FR-004, FR-008 and plan D03 (partial, high).
- [x] T030 Exercise manifest empty-list precedence and frozen alias elections, and compare desktop retained roster with the pre-replacement revision in internal/library/manifest_test.go and desktop/frontend/src/qualification.ts per FR-006, FR-017 and SC-001 (partial, medium).
- [x] T031 Complete authoritative contract, architecture and voice-model documentation plus CLI replacement help in docs/v0.0.0/ and cmd/insonic/main.go per FR-019 and T023 (partial, low).
