# Tasks: NVIDIA-first local processing and CLI reliability

**Input**: S062 spec, plan, research, data model and contracts. Tests are required. All seven issues are in scope.

## Phase 1: Setup

- [x] T001 Establish S062 identity, owner authorization and seven-issue mapping in specs/062-local-processing-reliability/spec.md.
- [x] T002 Complete design and compatibility decisions in specs/062-local-processing-reliability/plan.md and research.md.

## Phase 2: Foundation

- [x] T003 Review requirements checklists and run read-only Spec Kit analysis across specs/062-local-processing-reliability/{spec,plan,tasks}.md; resolve blocking findings before code.
- [x] T004 Freeze generated failure vocabulary and execution fields in specs/062-local-processing-reliability/contracts/processing.md; preserve empty legacy CPU and one shared worker deadline (FR-001 to FR-004, FR-006).

## Phase 3: US1 - Reliable accelerated processing

**Independent test**: New omitted local requests use auto, old frozen omitted requests stay CPU, explicit CPU stays CPU, CUDA failures recover once with exact source/model identity and truthful observed device.

- [x] T005 [P] [US1] Add native loader/readiness regression fixtures in tests/worker_probe_test.py and scripts/test-processing-worker.py before fixing internal/workers/probe.py and shared embedded loader (FR-004).
- [x] T006 [P] [US1] Add admission, bounded fallback and provenance regressions in internal/processing/*test.go and internal/app/*test.go (FR-001 to FR-004).
- [x] T007 [US1] Implement canonical verified native-library helper and common probe/worker bootstrap in internal/workers/; reconcile scripts/qualify-processing.py and packaging (FR-004).
- [x] T008 [US1] Implement new-local auto normalization and legacy CPU execution, fresh-child CUDA/CPU recovery under one deadline, generated fallback causes and strict provenance in internal/processing/, internal/app/election.go and worker_readiness.go (FR-001 to FR-004).
- [x] T009 [US1] Propagate device policy and embedding provenance through internal/voicemodels/, internal/processing/embedding.go and internal/app/speaker_models.go; preserve vectors/profile compatibility (FR-001, FR-003).
- [x] T010 [US1] Reconcile current request/pipeline/speaker manifest schemas in schemas/v1.0.3/ and desktop/frontend/src/ device defaults and controls (FR-009).

## Phase 4: US2 - Honest transcription and failures

**Independent test**: Exact silence yields no fabricated cues; quiet nonzero speech survives; disagreement is observable; invalid language is rejected before durable admission; failures preserve accepted bytes.

- [x] T011 [P] [US2] Add zero/nonzero-signal, language and redacted worker-failure tests in scripts/test-processing-worker.py before scripts/processing-worker.py changes (FR-005, FR-006).
- [x] T012 [P] [US2] Add local-admission, stage-disagreement and typed-error tests in internal/app/, internal/processing/ and internal/contracts/ (FR-005, FR-006).
- [x] T013 [US2] Implement exact-silence recognition, typed engine failures and shared local language validation across scripts/processing-worker.py, internal/processing/ and internal/app/ (FR-005, FR-006).
- [x] T014 [US2] Register generated worker reasons and corrections in internal/contracts/ and schemas/v1.0.3/runtime-response.schema.json (FR-006).

## Phase 5: US3 - Concise usable CLI guidance

**Independent test**: Golden successful/empty/missing/unknown/lifecycle cases have one useful outcome and valid actions while complete JSON and warnings remain.

- [x] T015 [P] [US3] Add golden/operator and lifecycle reason tests in cmd/insonic/, internal/catalog/artifacts_test.go and internal/artifact/ (FR-007, FR-008).
- [x] T016 [US3] Implement concise work rendering and state/resource-aware actions in cmd/insonic/human*.go; preserve generic public fields, warnings and detailed modes (FR-007).
- [x] T017 [US3] Distinguish retained references, active leases, grace and live retirement owners in internal/catalog/artifacts.go and internal/contracts/operator.go without changing deferred retirement/fencing (FR-008).
- [x] T018 [US3] Update docs/v1.0.3/ processing/device/adapter guidance and executable model setup examples in scripts/documented_workflows.py, scripts/cli_operator_qualification.py and their tests (FR-009).

## Phase 6: Integration and delivery

- [x] T019 Update CHANGELOG.md and verify UTF-8/LF, schema contracts, frontend/site and npm run check/npm test plus Go/Python regressions; retain results in specs/062-local-processing-reliability/verification.md (FR-009 to FR-011).
- [ ] T020 Qualify Windows/Linux source/package journeys and supported backend integrity with existing scripts/cli_contract_qualification.py and CI; retain independent receipts and separate timing (FR-010, FR-011).
- [x] T021 Run separately elected real CPU/NVIDIA processing/enrollment/matching acceptance through scripts/qualify-processing.py or maintained owned fixtures and record actual device/model evidence and limits in specs/062-local-processing-reliability/verification.md (SC-001, SC-002, SC-005).
- [ ] T022 Commit/push one S062 branch, publish official PR closing #202-#208, resolve every review/CI finding, request at most one additional Codex review and retain exact-head merge handoff in specs/062-local-processing-reliability/verification.md.

## Dependencies and parallel execution

T001-T004 precede implementation. T005/T007/T011/T013 Python work and T006/T008/T009/T012 Go work proceed in coordinated separate files after the contract is frozen. T015-T017 are an independent operator/lifecycle group. T010/T014 schema and T018 docs integration follow corresponding contracts. T019-T022 follow all stories. Shared files have a single owner; tests precede associated fixes.

Spec Kit parallel team execution: worker agent owns Python worker, workers native helper/probe, their tests and direct qualification/bootstrap; runtime agent owns processing/app/pipeline/voicemodels Go behavior and tests; operator agent owns human output/catalog lifecycle/contracts messages and tests; lead owns schemas/frontend/docs/qualification integration and final delivery. Coordinate shared error enums before edits.

## Implementation strategy

Validate each story independently, then deliver the complete seven-issue milestone. No partial issue closure or unverified GPU claim. Preserve all required integrity, provenance, security and backend checks. Publication is authorized; merge/release remain reserved.
