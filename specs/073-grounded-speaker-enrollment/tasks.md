# Tasks: Grounded speaker enrollment and unattended application

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/runtime.md.
**Tests**: Meaningful failure/security/isolation tests precede their implementation under autopilot.

## Phase 1: Setup

- [x] T001 Verify S073 identity, existing issues and publication boundaries; create planning artifacts in specs/073-grounded-speaker-enrollment/.
- [x] T002 Complete clarify/checklist/research/design using installed Spec Kit commands in .agents/skills/ and .specify/scripts/python/.

## Phase 2: Foundations

- [x] T003 Add failing independent-context proof and mapping replay/isolation tests in internal/catalog/speaker_application_test.go and internal/speakerdiscovery/contextual_test.go (FR-001, FR-004, FR-005, FR-007).
- [x] T004 Define server-derived contextual mapping provenance and atomic apply boundary in internal/catalog/speaker_application.go, internal/catalog/contract.go and internal/speakerdiscovery/contextual.go; retain existing stable mapping authority and no copied transcript (FR-001, FR-004).

## Phase 3: US1 - Grounded enrollment (Priority P1)

Independent test: context-only selection and enrollment preserve exact lineage, while circular, noisy, overlapping and stale inputs are excluded.

- [x] T005 [US1] Add failing contextual dataset/device/portable and mixed-speaker tests in internal/voicemodels/contextual_test.go and internal/catalog/speaker_application_test.go (FR-001, FR-002, FR-003).
- [x] T006 [US1] Extend authenticated independent selection/epoch/prior/reference gates in internal/catalog/{selection,selection_epoch,prior_evidence,speaker_models}.go without broadening legacyMappingDigest (FR-001).
- [x] T007 [US1] Retain exact evidence class and mapping provenance in current references and dataset/preparation manifests in internal/catalog/{evidence,selection,speaker_models}.go and internal/voicemodels/{preparation,portable}.go (FR-003).
- [x] T008 [US1] Diagnose route-aware mixed-speaker overlap alongside existing duration/channel/signal/duplicate exclusions in internal/voicemodels/preparation.go and internal/catalog/prior_evidence.go; retain CPU/NVIDIA execution policy (FR-002, FR-003).

## Phase 4: US2 - Correctable authority (Priority P1)

Independent test: correction/retraction and lost commit replay preserve newest mappings, stale dependent state and immutable outputs.

- [x] T009 [US2] Add failure/concurrency/retraction/snapshot-tampering tests in internal/catalog/speaker_application_test.go and internal/voicemodels/contextual_test.go (FR-005, FR-006, FR-007).
- [x] T010 [US2] Implement revision-fenced retraction, contradictory rerun and source/identity invalidation before output/work reconciliation in internal/catalog/{speaker_application,recording,exploration_outbox}.go (FR-005, FR-006).
- [x] T011 [US2] Preserve contextual mappings during matching and expose current profile usability while retaining historical outputs in internal/catalog/{speaker_models,speaker_models_state,speaker_output_reconciliation}.go (FR-003, FR-006).
- [x] T012 [US2] Verify portable mapping/lineage proof, cleanup/outbox recovery and SQLite/PostgreSQL parity through internal/catalog/*_test.go and internal/app/*_test.go (FR-007, FR-011).

## Phase 5: US3 - Unattended workflow (Priority P2)

Independent test: installed CLI complete flow, unknown/ambiguous outcomes, correction, cancellation/restart/repeat, no per-item gate or provider rerouting.

- [x] T013 [US3] Add failing durable orchestration/cancel/restart/four-parent capacity tests in internal/app/speaker_workflow_test.go and internal/catalog/work_test.go (FR-008, FR-010).
- [x] T014 [US3] Implement stable child IDs, immutable parent linkage, yield/readiness, fenced child effects and atomic cancel/retry propagation in internal/catalog/{work,speaker_workflow}.go and internal/app/work.go (FR-007, FR-010).
- [x] T015 [US3] Implement speakers.run/apply/retract shared operations and frozen stage/profile election in internal/app/speaker_workflow.go, reusing existing discovery and enrollment children (FR-004, FR-008, FR-010).
- [x] T016 [US3] Add strict saved speaker_workflow configuration and readiness/capability alignment in internal/pipeline/definition.go, internal/catalog/profiles.go and internal/app/{configured,worker_readiness}.go (FR-008, FR-010).
- [x] T017 [US3] Wire CLI, human/JSON summaries and shared desktop contract tests in cmd/insonic/{commands,explore,human_summary}.go and internal/contracts/ (FR-009, FR-012).
- [x] T018 [US3] Extend installed CLI journeys and help examples in scripts/{documented_workflows,cli_help_examples}.py with configured success/failure/correction/cancel/restart/repeat coverage (FR-008, FR-009, FR-011, FR-012).

## Phase 6: Convergence and handoff

- [x] T019 Update schemas/v1.2.0/, docs/v1.2.0/{speakers,voice-models,pipelines}.md, release metadata and CHANGELOG.md consistently; check encoding and source/docs scope (FR-012).
- [ ] T020 Run focused and required checks, site/frontend checks and qualification with maintained timing/receipt retention in specs/073-grounded-speaker-enrollment/verification.md (FR-011, FR-012, SC-001-SC-005).
- [ ] T021 Converge artifacts, commit/push, open official PR closing #230-#232, resolve all comments within at most two rounds and retain exact-head green CI evidence; owner final merge (FR-012).

## Dependencies and execution order

T001-T004 precede US1. US1 corpus changes and US2 authority depend on the shared mapping proof; US3 consumes both. Complete story-specific tests before integration. T019-T021 follow all stories. Parallel opportunities are read-only planning research and independent backend/CLI checks after the implementation stabilizes; shared catalog edits remain sequential.

## Implementation strategy

Deliver the mapping/provenance foundation first, then qualify grounded corpus behavior, correction/retraction, and finally durable orchestration. Each story has direct testable operations; the complete slice includes all three. No partial story is the requested handoff. Stage verification evidence and review state chronologically.

## Phase 7: Convergence

- [ ] T022 Run the installed configured speaker workflow, failure/correction/cancel/repeat and owned-runtime restart journeys against PostgreSQL/S3/ArcadeDB through scripts/qualify-answer-backends.py and scripts/cli_speaker_workflow_qualification.py; retain backend-bound CI receipts per SC-004 and FR-011 (partial).
