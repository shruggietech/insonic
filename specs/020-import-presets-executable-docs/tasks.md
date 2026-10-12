# Tasks: S020

**Input**: [plan.md](plan.md), [spec.md](spec.md), research, data model and contracts.

## Phase 1: Setup

- [x] T001 Establish S020 identity, owner authorization and issue tracking in specs/020-import-presets-executable-docs/spec.md.
- [x] T002 Complete research/design and constitution gates in specs/020-import-presets-executable-docs/plan.md.

## Phase 2: Foundation

- [x] T003 Add failing credential, scope, CAS/replay and snapshot tests in internal/catalog/import_presets_test.go (FR-001/002/012).
- [x] T004 Implement typed registry with at most 1000 retained identities, 1 MiB bound and name length 1..128 in internal/catalog/import_presets.go; validate generic settings and restore (FR-001/002/012).
- [x] T005 Add destination routing regression tests and immutable profile lookup in internal/artifact/routing_test.go and internal/catalog/profiles.go (FR-002/012).
- [x] T006 Implement immutable-profile publication/read routing in internal/artifact/routing.go and service.go (FR-002/012).

## Phase 3: US1 reusable choices

**Independent test**: Lifecycle and destination/date selection with synthetic direct/JSON/CSV/refresh inputs.

- [x] T007 [US1] Add failing preset validation/precedence tests in internal/app/import_presets_test.go and cmd/insonic/domain_test.go (FR-003/004/008).
- [x] T008 [US1] Add versioned current schemas and operation grammar in schemas/v1.0.1/runtime-request.schema.json, internal/contracts/configuration.go and cmd/insonic/commands.go (FR-001/003/012).
- [x] T009 [US1] Implement shared CRUD/resolution and explicit merge layers in internal/app/import_presets.go, internal/library/types.go, manifest.go and cmd/insonic/domain.go (FR-001/003/004/008).
- [x] T010 [US1] Add shared desktop preset management and import/refresh selectors in desktop/frontend/src/ (FR-001/008).

## Phase 4: US2 stable work

**Independent test**: Elected fixture pipeline queues once; edit/delete followed by retry retains original revision/destination/configuration.

- [x] T011 [US2] Add failing frozen-election, retry and readiness tests in internal/app/import_presets_test.go (FR-005/006/007).
- [x] T012 [US2] Freeze ordinal elections before enqueue in internal/app/import_presets.go and work.go; reject public injected frozen metadata (FR-003/005/006/012).
- [x] T013 [US2] Dispatch deterministic processing children after canonical admission with parent-generation fence in internal/catalog/work.go and library callbacks in internal/library/service.go (FR-005/006/007).
- [x] T014 [US2] Expose effective selections and queued_job_ids through work results and desktop feedback in internal/app/work.go and desktop/frontend/src/ (FR-006/008).

## Phase 5: US3 executable documentation

**Independent test**: Corrected introductory/advanced procedures execute against extracted CLI/desktop binaries; online/offline current pages agree.

- [x] T015 [P] [US3] Replace unsupported/incomplete current procedures and clarify mapping/date/capability semantics in docs/v1.0.1/ and current README links (FR-009/010).
- [x] T016 [US3] Qualify lifecycle, precedence, destinations, refresh, elected work and documentation procedures in scripts/cli_contract_qualification.py and package-desktop.py (FR-011).
- [x] T017 [US3] Add mounted desktop preset lifecycle/selection qualification in desktop/frontend/src/qualification.ts (FR-001/011).

## Phase 6: Convergence/publication

- [x] T018 Run security/scope, alternative-backend, runtime/schema/frontend/site and extracted-package checks; record evidence in specs/020-import-presets-executable-docs/verification.md (FR-001..012, SC-001..005).
- [x] T019 Update CHANGELOG.md with features and dated decisions, complete convergence and file integrity checks.
- [ ] T020 Commit/push official PR, attach it and set review tracking; resolve every finding in at most two rounds and record exact-head CI in specs/020-import-presets-executable-docs/verification.md.

## Dependencies and execution

Setup -> foundation -> US1 -> US2 -> US3 qualification -> convergence. Documentation repairs (T015) can run independently after design; catalog and artifact tests use independent files after their shared interface settles. Read-only research was parallel under speckit-plan. Production edits remain coordinated sequentially. US1 is the first usable increment; all three stories are required before publication. No release or merge is authorized.

## Phase 7: Convergence

- [x] T021 Preserve actionable incompatible-version diagnostics and test ambiguous UUID/name selections before queueing in internal/catalog/import_presets.go and internal/app/import_presets_test.go per FR-003 and US1/AC3 (partial, HIGH).
- [x] T022 Preserve supplied-transcript election for exact-title existing-record imports in internal/app/import_presets.go with regression coverage per FR-008 and US2/AC1 (partial, HIGH).
- [x] T023 Reuse identical per-request elections and avoid repeated immutable-profile registration in internal/app/import_presets.go per plan: frozen ordinal elections and Constitution V (partial, MEDIUM).
