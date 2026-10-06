# Tasks: Portable catalogs and durable jobs

**Input**: [spec.md](spec.md), [plan.md](plan.md), research/data-model/contracts/quickstart.

Tests follow the owner-authorized autopilot TDD discipline. Every requirement maps below. No custom unchecked reviewer gate is introduced.

## Phase 1: Setup

- [x] T001 Freeze qualified pins and resolve research/design choices in specs/002-portable-catalogs/research.md (FR-010, FR-012).

## Phase 2: Foundation

- [x] T002 Add shared catalog acceptance tests in internal/catalog/catalog_test.go before adapter implementation (FR-001, FR-002, FR-003, FR-008, FR-009).
- [x] T003 Implement selected SQLite/PostgreSQL configuration, portable migrations and typed records in internal/catalog/store.go, records.go and migration.go; UUID scope, signed int64 and exact timestamp agreement are mandatory (FR-001, FR-002, FR-008, FR-010).

## Phase 3: User Story 1 - Durable state

Independent test: reopen, immutable metadata admission, concurrent expected revision, scoped FK rejection and cross-backend transfer.

- [x] T004 [US1] Implement atomic mutation/receipt/revision acceptance and typed reads in internal/catalog/mutations.go (FR-002, FR-003, FR-008).
- [x] T005 [US1] Implement consistent digest snapshots and atomic empty restore with imported claims expired in internal/catalog/snapshot.go (FR-009).
- [x] T006 [US1] Run same server acceptance and cross-backend round trips in internal/catalog/postgres_test.go (FR-001, FR-009, FR-010, SC-001).

## Phase 4: User Story 2 - Durable work

Independent test: stop/reopen, preserve attempts, recover after expiry and reject stale publication/cancel races.

- [x] T007 [US2] Add lease/recovery/cancellation/idempotency tests in internal/catalog/jobs_test.go and internal/app/app_test.go before integration (FR-004, FR-005, FR-006).
- [x] T008 [US2] Implement fenced job/workspace ownership and durable histories in internal/catalog/jobs.go (FR-004, FR-005, FR-006).
- [x] T009 [US2] Integrate selected catalog startup/errors, recovery, renewal and shutdown into internal/app/app.go and internal/runtime/runtime.go (FR-004, FR-005, FR-006, FR-010).
- [x] T010 [US2] Expose history/catalog status/transfer/migrate through cmd/insonic/main.go and aligned schemas/v0.0.0/runtime-request.schema.json (FR-011).

## Phase 5: User Story 3 - Ordered handoff

Independent test: next-only event claim, expiry/takeover and stale acknowledgement rejection.

- [x] T011 [US3] Add ordered outbox contract tests in internal/catalog/outbox_test.go before implementation (FR-007).
- [x] T012 [US3] Implement transactional event allocation and fenced claims/checkpoints in internal/catalog/outbox.go (FR-003, FR-007).

## Phase 6: User Story 4 - Relevant qualification

Independent test: scope table classifies docs/core/native/schema/unknown paths conservatively and pin drift fails.

- [x] T013 [US4] Add scope/pin tests then implement scripts/ci-scope.mjs, scripts/qualify.py and .github/workflows/ci.yml with stable checks (FR-012).

## Phase 7: Completion

- [x] T014 Update authoritative docs and CHANGELOG.md with demonstrated behavior, remaining scope and dated decisions (FR-013).
- [x] T015 Run local CI parity, server/platform hosted qualification, encoding checks and Spec Kit convergence for specs/002-portable-catalogs (SC-001 through SC-005).
- [x] T016 Automatically publish an official PR, attach it, satisfy every review finding within two rounds and hand off for owner merge (SC-005).

## Dependencies and execution

T001 -> T002/T003 -> US1 -> US2 -> US3 -> US4 -> completion. Tests precede implementation; schema/mutation ownership is shared so implementation runs sequentially. Independent acceptance tests can run in parallel after infrastructure exists (for example record round trips and scope classification). Deliver US1 first as a reopenable catalog, then durable work, ordered handoff and affected CI. No partial increment is reported as slice completion.

## Phase 8: Convergence

- [x] T017 Enforce and preserve configured backend-version constraints in internal/catalog/store.go, profiles.go and records.go per FR-010 and plan: selected profile authority (partial).
- [x] T018 Exercise bounded history, cancellation/publication races, invalid credentials/TLS without fallback and snapshot event-revision consistency in internal/catalog/*_test.go per SC-003, FR-009, FR-010 and FR-011 (partial).

- [x] T019 Preserve active retry attachment when replaying an earlier cancellation and reconcile accepted starts/retries at capacity in internal/app and internal/catalog regression tests (FR-003, FR-004, SC-003).

- [x] T020 Address first-round review findings with shared-backend credential-key alias rejection and durable outbox acknowledgement reconciliation after expiry/takeover (FR-003, FR-007, FR-010).

- [x] T021 Reject backend scalar coercion during snapshot restore using explicit portable cell types, UUID checks and shared-backend rollback regressions (FR-001, FR-009).

- [x] T022 Cancel in-flight catalog operations before acquiring the application shutdown mutex, with a blocked-call regression in internal/app/app_test.go (FR-004, FR-006).

- [x] T023 Remove asymmetric snapshot input limits, read validated file streams through a private temporary spool, and prove an actual catalog export above 64 MiB restores (FR-009, FR-011).
- [x] T024 Require matching deterministic acknowledgement receipts and outcomes for restored checkpoints, rejecting missing/mismatched receipts and checkpoint rewinds on both backends (FR-007, FR-009).
