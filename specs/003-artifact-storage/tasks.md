# S003 tasks

## Phase 1: Setup
- [x] T001 Confirm S003/issue #4 mapping and preserve dependency pins in specs/003-artifact-storage/plan.md.
- [x] T002 Add streaming contract and typed publication journal in internal/artifact/contract.go and internal/catalog/artifacts.go.

## Phase 2: Authority foundation
- [x] T003 Write upgrade/restore and fencing tests first in internal/catalog/artifacts_test.go (FR-003/007/009).
- [x] T004 Implement schema-v2 migration preserving v1 digest in internal/catalog/migration.go.
- [x] T005 Implement durable claims, receipts, retention/lease/retirement barriers and structural reference guard in internal/catalog/artifacts.go.
- [x] T006 Extend and validate lifecycle snapshots with expired imported authority in internal/catalog/snapshot.go and schemas/v0.0.0/catalog-snapshot.schema.json.

## Phase 3: US1 verified publication
- [x] T007 [US1] Write shared empty/binary/large/replay/interruption/security tests in internal/artifact/store_test.go (FR-001/002/003/004/007).
- [x] T008 [US1] Implement confined staged/flush/no-overwrite filesystem store in internal/artifact/filesystem.go.
- [x] T009 [US1] Implement exact-key/version streaming S3 and resumable multipart journal in internal/artifact/s3.go.
- [x] T010 [US1] Implement selected-profile service with renewable publication claims/readback/admission/reconcile/abort in internal/artifact/service.go.
- [x] T011 [US1] Add pinned fixture and deterministic signing/range/cancellation tests in internal/artifact/s3_test.go and .github/workflows/ci.yml.

## Phase 4: US2 read/materialize
- [x] T012 [US2] Write bounded cache, corrupt-object, lease/cancellation tests in internal/artifact/service_test.go.
- [x] T013 [US2] Implement verified leased materializations, renew/release and ranged reads in internal/artifact/service.go.
- [x] T014 [US2] Connect shared runtime/CLI request schemas and capability diagnostics in internal/app/app.go, internal/contracts/contracts.go, internal/runtime/runtime.go, cmd/insonic/main.go and schemas/v0.0.0/runtime-request.schema.json.

## Phase 5: US3 retirement
- [x] T015 [US3] Write retain/materialize versus retire races and structural reference tests in internal/catalog/artifacts_test.go and internal/artifact/service_test.go.
- [x] T016 [US3] Implement grace, claim, exact deletion/reconciliation and lifecycle history in internal/artifact/service.go.

## Phase 6: Verification and publication
- [x] T017 Update truthful storage/development/index contracts and CHANGELOG.md.
- [x] T018 Execute repository/site/product/fixture/race checks and converge evidence in specs/003-artifact-storage/verification.md.
- [ ] T019 Commit, push official PR and resolve at most two external review rounds; record final green head in specs/003-artifact-storage/verification.md.

## Dependencies and strategy

T001->T002->T003->T004/T005->T006. US1 tests->adapters->service->fixtures, US2 depends on verified US1, US3 depends on authority and leases. Tests under distinct adapters can run independently, but mutations of shared schema/runtime files remain sequential. US1 is the first useful increment; all stories are required for slice completion.

## Phase 7: Convergence

- [x] T020 Close HIGH partial FR-003 recovery claim gap: maintain publication authority through recovery verification and staged-source hashing; test slow reconciliation and active abort in internal/artifact/service.go and failure_test.go.
- [x] T021 Close HIGH partial FR-009 lifecycle restore gap: bind the current journal to the latest publication receipt and validate admission intent; test old-journal replay against newer receipts in internal/catalog/artifacts.go and artifacts_test.go.
- [x] T022 Close MEDIUM partial FR-008 bounded response gap: expose bounded artifact receipts with journal counts and strengthen multipart metadata validation in internal/app/artifacts.go and internal/catalog/artifacts.go.
- [x] T023 Close MEDIUM partial FR-010 CLI qualification gap: exercise publish/replay/verify/materialize/lease/retention/cache commands through the built CLI in scripts/qualify.py; update CLI help and document bounded inspection.

## Phase 8: Convergence

- [x] T024 Close HIGH partial FR-003 unknown-completion abort gap: refuse to abandon a pending journal when its final object already exists; keep it reconcilable and test the lost-response abort path in both adapters.

## Phase 9: External review

- [x] T025 Resolve first-round workspace/profile key isolation, final-link durability, indeterminate S3 completion/abort and typed location retirement findings with regressions on both storage/catalog backends.
