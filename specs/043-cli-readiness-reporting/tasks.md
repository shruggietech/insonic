# Tasks: CLI readiness and failure reporting

Input: spec.md, plan.md, research.md, data-model.md and contracts/reporting.md. Tests are required for the observed defects and shared security boundary.

## Phase 1: Setup

- [x] T001 Confirm S043 branch/directory/issues, specify and clarify scope in specs/043-cli-readiness-reporting/spec.md (FR-001-009).
- [x] T002 Complete design and requirements validation in specs/043-cli-readiness-reporting/plan.md and checklists/requirements.md (FR-001-009).
- [x] T003 Map the six issues to S043 milestone/Project metadata and keep #114/#147/#154 scope distinct (FR-008).

## Phase 2: Shared error foundation

- [x] T004 Add failing asset-observation reconstruction/redaction/schema regressions in internal/contracts/asset_error_test.go and schemas/asset_error_test.go (FR-002/007/009).
- [x] T005 Add fixed generated asset reasons and Error.asset role/name handling in internal/contracts/asset_error.go, contracts.go, operator.go and schemas/v1.0.1/runtime-response.schema.json; names are ASCII slash-relative, maximum 128 bytes, never absolute/traversal/control/device names (FR-002/007/009).

## Phase 3: US1 identify failed prerequisites and bytes

Independent test: corruption/missing/truncation matrices, no invalid publication, exact local asset reason/name and durable cause.

- [x] T006 [US1] Add failing filesystem/mock-S3 verify/materialize and expected-output-before-admission regressions in internal/artifact tests (FR-001, SC-001).
- [x] T007 [US1] Correct physical integrity categories and add stable-staging PublishVerified before BeginPublication in internal/artifact/{contract,filesystem,s3,service}.go; preserve real catalog/revision fences and pending recovery (FR-001).
- [x] T008 [US1] Test and correct inline/path trainer digest/size rejection before admission in internal/voicemodels/service.go and tests, retaining malformed/unknown output rejection and stale-definition conflicts (FR-001, SC-001).
- [x] T009 [US1] Test and retain generated local worker/model filename/role/reason/correction through internal/workers/setup.go, internal/models/service.go and package tests, preserving cancellation and existing verification/confinement (FR-002/009, SC-002).

## Phase 4: US2 truthful readiness and budgets

Independent test: offline transfer zero, exact declared local totals, no remote probe or adapter invocation, credential/trainer state matrices.

- [x] T010 [US2] Add failing offline budget, credential and training-only readiness regressions in internal/app/workers_test.go and readiness tests (FR-003-005, SC-003).
- [x] T011 [US2] Reuse declared asset inspection for required/verified/missing budgets and zero offline download_bytes in internal/workers and internal/app/workers.go; retain separately labeled nominal totals and verified environment reuse (FR-003).
- [x] T012 [US2] Check required local credential resolvability in internal/app/workers.go, clearing resolved bytes and preserving missing/locked states, frozen references/destinations and no-credential verified reuse; expose upstream not_checked (FR-004/009).
- [x] T013 [US2] Reuse pinned executable/support checks without invocation in internal/voicemodels/adapter.go and internal/app/worker_readiness.go; distinguish unknown/unavailable and compose elected pipeline tasks (FR-005).
- [x] T014 [US2] Align worker/model plan types and existing GUI facts in schemas/v1.0.1/worker-resources.schema.json and desktop/frontend/src/workers.tsx, with affected frontend tests (FR-007).

## Phase 5: US3 truthful wait outcomes

Independent test: controlled context boundaries preserve last state, caller timeout, cancellation and independently observed startup failures.

- [x] T015 [US3] Add failing deadline-precedence and unknown-state regressions in cmd/insonic work wait/operator tests (FR-006, SC-004).
- [x] T016 [US3] Normalize only caller-driven readiness expiry/cancellation in cmd/insonic/work_wait.go and use neutral generated advice in internal/contracts/operator.go, retaining one budget and existing Ensure semantics (FR-006/007).

## Phase 6: Packaged integration and documentation

- [x] T017 Extend existing scripts/cli_contract_qualification.py isolated regressions for offline assets/budgets, credential preview and short wait human/JSON/durable behavior; add only relevant reusable synthetic fixture code and tests (FR-008/009, SC-001-004).
- [x] T018 Extend existing deterministic trainer/documented workflows and packaged regressions for training-only readiness, wrong-hash/truncated/unknown-field/success output and real artifact verify/materialize failures in scripts/documented_workflows.py and current qualification tests (FR-008, SC-001/003/004).
- [x] T019 Update authoritative docs/v1.0.1 contracts, CHANGELOG.md and concise specs/043-cli-readiness-reporting/quickstart.md for changed fields/categories, preserving version and public/internal separation (FR-007).
- [x] T020 Run blocking Spec Kit analysis before implementation and convergence after it; append only demonstrated gaps to this tasks.md (FR-001-009, SC-001-005).
- [ ] T021 Run npm run check, npm test, affected Go/vet/Python/frontend/site checks and existing source/package qualification paths; remove owned temporary outputs and inspect UTF-8/LF/mojibake/diff (FR-007-009, SC-001-005).

## Phase 7: Authorized publication and bounded review

- [x] T022 Commit and automatically push codex/043-cli-readiness-reporting, publish/attach the ordinary PR with exact issue mapping and validation (SC-005).
- [ ] T023 Monitor exact-head complete CI and initial automatic Codex/security reviews; answer/resolve every finding, with at most one explicit second review round and no third round (SC-005).
- [ ] T024 Verify final head/checks/review threads/Project state, remove temporary execution files and hand off the unmerged PR for owner review/merge (SC-005).

## Dependencies and execution

T001-T003 -> T004-T005 -> US1/US2/US3 -> T017-T021 -> T022-T024. T020 analysis runs before T004; its convergence runs after implementation. Integrity, readiness and wait stories can be researched independently after shared contracts; edits to shared files and qualification integration stay sequential. No same-file parallel mutation is planned. MVP is US1; completion requires all three stories.

## Phase 8: Convergence

- [ ] T025 Preserve elected detached runtime ownership in scripts/qualify.py model-reference calls after isolated regressions outlive the owner's idle allowance, and keep the cancellable synthetic job alive across replay/client startup; rerun the existing native source CLI journey and retain final owner/image/history checks per FR-008 (partial).
