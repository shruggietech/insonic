# Tasks: S022 backend configuration and local query assistance

**Input**: [plan.md](plan.md), [spec.md](spec.md), research, data-model and contracts.

## Phase 1: Setup

- [x] T001 Establish S022 identity, issue mapping and resolved Spec Kit artifacts in specs/022-backend-service-setup/.

## Phase 2: Foundation

- [x] T002 Define complete versioned backend plans, stable generated identity and field corrections in internal/backendsetup/plan.go and schemas/v1.0.1/backend-plan.schema.json (FR-001/002/003/005).

## Phase 3: US1 configure roles

Independent test: fresh defaults and elected alternate backends initialize with exact persisted profiles and protected credentials before the initial catalog connection.

- [x] T003 [US1] Add plan/bootstrap/failure/retry tests first in internal/backendsetup/setup_test.go (FR-001/002/003/005).
- [x] T004 [US1] Implement explicit provisioning and atomic publication in internal/backendsetup/provision.go and internal/workspace/workspace.go; identical generated plan retries reconcile, populated transfers remain explicit migration (FR-003/005).
- [x] T005 [US1] Expose equivalent CLI and desktop bootstrap operations/forms in cmd/insonic/backend_setup.go, desktop/backend_setup.go and desktop/frontend/src/backend-setup.tsx (FR-001/002/003).

## Phase 4: US2 preflight

Independent test: six-backend plans produce bounded separate readiness dimensions without database/object/file-tree mutations.

- [x] T006 [US2] Add read-only, transport, denied/cancelled/capped probe tests in internal/backendsetup/probe_test.go and internal/catalog/transport_test.go (FR-004/006/013/015).
- [x] T007 [US2] Extract shared validated PostgreSQL connection construction and enforce local versus explicit insecure remote transport in internal/catalog/store.go (FR-006).
- [x] T008 [US2] Implement dedicated six-backend non-mutating probes in internal/backendsetup/probe.go and adapter helpers in internal/artifact/s3.go; finite 30-second aggregate deadline and 1 MiB response cap, write capabilities untested (FR-004/006).
- [x] T009 [US2] Display readiness/corrections/current selections in desktop/frontend/src/backend-setup.tsx and settings.tsx, preserving selected config on failure and cancelling underlying probes (FR-002/004/005/016).

## Phase 5: US3 discovery

Independent test: discovered hints/candidates/models have accurate separate states and do not alter route/model election or load inference.

- [x] T010 [US3] Add metadata/discovery redaction, redirects, caps, cancellation, stale/multiple candidate tests first in internal/assistance/ollama_test.go (FR-007/008/009/013).
- [x] T011 [US3] Implement metadata-only inventory/probes in internal/assistance/discovery.go and ollama.go; 16 candidate, 128 model and bounded detail limits, source/time and distinct evidence (FR-007/008/009).
- [x] T012 [US3] Expose shared discovery and actual cancellation through internal/app/assistance.go, internal/contracts/explore.go, internal/runtime/runtime.go, cmd/insonic/explore.go and desktop bridge/client (FR-002/008/016).

## Phase 6: US4 native assistance

Independent test: deterministic native protocol proposals pass unchanged validated read-only machinery, without fabricated local keys or silent hosted routing.

- [x] T013 [US4] Add native proposal/cloud/unknown route/provider-error/synthetic tests in internal/assistance/ollama_test.go and internal/app/assistance_test.go before implementation (FR-009/010/011/012/013).
- [x] T014 [US4] Implement versioned native translation and synthetic test receipt in internal/assistance/ollama.go and internal/app/assistance.go; preserve existing budgets and fences (FR-009/010/011/012).
- [x] T015 [US4] Add native adapter/discovery/model election/test controls and rendered parity tests in desktop/frontend/src/assistance.tsx and desktop/frontend/tests/assistance.test.mjs (FR-002/009/010/011/012/013).

## Phase 7: Cross-cutting verification and publication

- [x] T016 Update current schemas/examples/help and docs/v1.0.1/storage.md, graph.md, desktop.md, cli.md, contracts.md, secrets.md plus CHANGELOG.md; retain frozen 1.0.0 bytes (FR-014).
- [ ] T017 Run npm check/test, affected Go/vet, frontend/site and backend integrations, record exact evidence and converge all acceptance in specs/022-backend-service-setup/verification.md (FR-015/016).
- [ ] T018 Push official PR, handle every review in at most two rounds, verify final-head green CI and hand off for owner merge (FR-016).

## Dependencies and implementation strategy

T001 -> T002 -> US1/US2 -> US3 -> US4 -> T016/T017/T018. Tests precede their implementation. Backend and native-protocol test construction can be batched, but shared files are edited sequentially. Each story has independent fixtures; completing US1 alone is an internal checkpoint, not completion of the authorized slice. All four stories and both issues must be satisfied.

Parallel examples: independent backend unit fixtures and Ollama protocol fixtures can run together after their implementations; product/site verification can run together using hidden redirected child launches. No independent task edits a shared file concurrently.

## Phase 8: Convergence

- [x] T019 Use the existing opaque-origin UUID helper for cancellable client request identity in desktop/frontend/src/client.ts per FR-002/016 (partial, HIGH).
- [x] T020 Reuse shared backend definitions in schemas/v1.0.1/backend-plan.schema.json and update the explicit registered contract count in tests/schemas.test.mjs per FR-014/015 (partial, MEDIUM).

## Phase 9: Review remediation

- [x] T021 Initialize the selected graph/schema/owner/version before config publication and share runtime construction in internal/graph/initialize.go per FR-002/005 and review discussion_r4256618070.
- [x] T022 Persist and enforce a private pending plan/credential-backend identity before backend mutation in internal/backendsetup/intent.go per FR-005 and review discussion_r4256618082.
- [x] T023 Preserve successful version/reachability evidence when later model metadata fails in internal/assistance/discovery.go per FR-009/013 and review discussion_r4256618087.
- [x] T024 Correct cancellation ref initialization in desktop/frontend/src/assistance.tsx and run the configured frontend type check per FR-015/016 and initial CI failure.

## Phase 10: Second review remediation

- [x] T025 Observe S3 HeadBucket access before successful workspace publication in internal/backendsetup/provision.go, with inaccessible/missing/denied/cancelled regression fixtures per FR-003/005/013/015 and review discussion_r4256827921.
- [x] T026 Verify retained intent and persisted credential mode under workspace ownership for completed retries, with all three mode/conflict/receipt regression cases in internal/backendsetup/second_review_test.go per FR-005/013/015 and review discussion_r4256827933.
- [x] T027 Preserve typed selected-model metadata failures and per-model redacted corrections in internal/assistance/ollama.go and desktop/frontend/src/assistance.tsx, with protocol and rendered fixtures per FR-009/013/015 and review discussion_r4256827939.
- [x] T028 Make the S3 cancellation fixture wait for the real HEAD request before cancellation, avoiding a pre-network initialization timer race per FR-015/016 and CI run 38376803237.
