# Tasks: Native platform and shared-runtime foundation

**Input**: [Plan](plan.md), [specification](spec.md), research, data model and contracts.

## Phase 1: Setup

- [x] T001 Initialize exact module/toolchain pins in go.mod/go.sum and hidden maintainer launcher in scripts/run-hidden.ps1 (FR-008, FR-009).
- [x] T002 Embed/validate authoritative offline schemas in schemas/registry.go (FR-002, FR-005).

## Phase 2: User Story 1, shared workspace owner

- [x] T003 [US1] Write failing discovery/version/concurrency/alias tests in internal/workspace/workspace_test.go (FR-001, FR-002, SC-001).
- [x] T004 [US1] Implement platform paths, schema validation, atomic initialization and OS locks in internal/workspace/ (FR-001, FR-002).
- [x] T005 [US1] Write bounded protocol/identity/owner tests in internal/runtime/runtime_test.go (FR-003, FR-004, FR-007, SC-001).
- [x] T006 [US1] Implement user-only IPC, on-demand owner and idle lifetime in internal/runtime/ (FR-003, FR-004).
- [x] T007 [US1] Implement shared app dispatch and CLI in internal/app/ and cmd/insonic/ (FR-005, FR-007, SC-002).

## Phase 3: User Story 2, bounded attempts and supervision

- [x] T008 [US2] Write stale-generation/cancel/retry/disconnect tests in internal/app/app_test.go (FR-006, SC-001).
- [x] T009 [US2] Implement session-scoped attempt state and typed backend/secret-reference boundaries in internal/app/ and internal/contracts/ (FR-005, FR-006).
- [x] T010 [US2] Test and implement hidden no-shell bounded child launches in internal/process/ (FR-008, SC-004).

## Phase 4: User Story 3, native qualification

- [x] T011 [US3] Commit immutable asset/fixture pins and checksum-verified qualification tooling in scripts/qualify.py and internal/qualification/ (FR-009, FR-011, SC-003).
- [x] T012 [US3] Prove native Cueson, Ladybug/SQLite pairing and PostgreSQL/S3/ArcadeDB fixture operations in internal/qualification/ (FR-009, FR-011).
- [x] T013 [US3] Build common Wails bridge, local offline help and secret-service availability probes in cmd/insonic-desktop/ and desktop/ (FR-009, FR-010, SC-002, SC-003).
- [x] T014 [US3] Execute native matrix and retain receipts in .github/workflows/ci.yml; distinguish supported assets from executed architectures (FR-009, FR-012, SC-001, SC-003, SC-005).

## Phase 5: Convergence and publication

- [x] T015 Run CI parity, scope/security convergence and update CHANGELOG.md, authoritative docs and internal evidence (FR-012, SC-004, SC-005).
- [x] T016 Commit/push, publish and attach official PR; resolve all reviews; at most two external Codex rounds; green checks and owner merge handoff (FR-012).

## Dependencies and execution order

Setup precedes workspace/IPC. Workspace and shared app precede client integration. Native research precedes pinned probes; frontend/help assets precede Wails compilation. Tests precede their implementation. Native hosted execution follows explicitly authorized publication; no issue closure occurs before its evidence passes. T016 remains open until external review and green CI are complete.

## Phase 6: Convergence

- [x] T017 Register runtime request/response envelopes in the master schema and align discriminators, optional error identities and argument validation (FR-005, FR-007).
- [x] T018 Bound the native macOS secret-service subprocess itself and package exact upstream desktop typography assets (FR-008, FR-010).

## Phase 7: External review corrections

- [x] T019 Permit read-only metadata while retaining stable workspace ownership across control-directory edits; reclaim bounded terminal attempt history without evicting active work (FR-002, FR-003, FR-006).

- [x] T020 Correct final-review process-tree cancellation in runtime and qualification tooling, and prove noninteractive macOS missing-item lookup versus locked access (FR-008, FR-010).

- [x] T021 Validate Windows directory security by binary SID/ACL semantics rather than SDDL text aliases, retaining exact current-user grants (FR-001, FR-003).

- [x] T022 Preserve intentionally detached Windows owner lifetime across bounded qualification-client exits, without permitting ordinary supervised workers to break away (FR-004, FR-008).

- [x] T023 Include underscore-prefixed offline help assets in the executable and validate referenced local assets across all embedded HTML pages (FR-010, SC-003).

- [x] T024 Keep native desktop receipts valid JSON despite loader warnings and reject absent, ambiguous or failed qualification results (FR-012, SC-003).

- [x] T025 Apply the owner's removal of mandatory human PR approvals to branch protection, bootstrap readback and development policy while retaining automated checks and resolved review threads (FR-012).
