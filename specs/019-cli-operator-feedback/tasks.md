# Tasks: S019 CLI operator feedback

**Input**: [spec.md](spec.md), [plan.md](plan.md), research/data-model/contracts/quickstart.

## Phase 1: Setup

- [x] T001 Specify and clarify scope #56/#57/#58/#61 in specs/019-cli-operator-feedback/spec.md.
- [x] T002 Complete requirements review and design artifacts in specs/019-cli-operator-feedback/.

## Phase 2: Foundation

- [x] T003 Write generated reason/redaction regressions in internal/contracts/operator_test.go and schemas/operator_test.go (FR-002, FR-009).
- [x] T004 Implement bounded generated reason context in internal/contracts/operator.go, current schemas/v1.0.1/ and durable internal/app/work.go (FR-002, FR-009).

## Phase 3: US1 readable results (P1)

- [x] T005 [US1] Write human/credential/corruption/source-map regressions in cmd/insonic/human_test.go and affected internal tests (FR-001-FR-004).
- [x] T006 [US1] Fix actual credential states, duplicate/corrupt/reuse reasons in internal/secrets/, internal/backup/, internal/app/ (FR-002, FR-003).
- [x] T007 [US1] Implement 128-line/64-entry bounded human renderer and exact-document actions in cmd/insonic/human.go and main.go (FR-001, FR-004, FR-009).

## Phase 4: US2 startup and diagnostics (P1)

- [x] T008 [US2] Write early exit/input/report/redaction/detached lifetime tests in internal/process/startup_test.go, internal/runtime/ and internal/diagnostics/ (FR-005, FR-006).
- [x] T009 [US2] Implement 4096-byte report and shared ten-second startup deadline in internal/process/startup.go, internal/runtime/, cmd/insonic/main.go and desktop/bridge.go (FR-005).
- [x] T010 [US2] Implement global diagnostic controls and private three-file 1 MiB logs in cmd/insonic/commands.go and internal/diagnostics/ (FR-006, FR-009).

## Phase 5: US3 work feedback (P2)

- [x] T011 [US3] Write state/progress/redirected/partial warning tests in cmd/insonic/work_wait_test.go and human_test.go (FR-007-FR-009).
- [x] T012 [US3] Add terminal-only changed observations to same-process cmd/insonic/work_wait.go (FR-007, FR-009).
- [x] T013 [US3] Render final/last counts, item date warnings and exact continuation/correction commands in cmd/insonic/human.go and affected docs (FR-008).

## Phase 6: US4 model context (P2)

- [x] T014 [US4] Write inspection/election/readiness/role/lease/dataset regressions in internal/models/, internal/app/, schemas/ and cmd/insonic/human_test.go (FR-010-FR-012).
- [x] T015 [US4] Preserve incompatible inspections and declared model role/readiness responses in internal/models/ and internal/app/work.go (FR-010, FR-011).
- [x] T016 [US4] Expose preparation readiness and evidence remedies in internal/app/speaker_models.go and human.go (FR-012).

## Phase 7: Polish and delivery

- [x] T017 Update current docs/v1.0.1/, CHANGELOG.md and internal verification evidence (FR-013).
- [x] T018 Run required local checks and affected extracted-package qualification from specs/019-cli-operator-feedback/quickstart.md (FR-014).
- [x] T019 Execute consistency/convergence review and resolve findings in specs/019-cli-operator-feedback/verification.md (FR-014).
- [x] T020 Commit/push official PR, resolve all reported findings within two rounds, verify final-head CI and hand off for owner merge (FR-014).

## Dependencies and implementation strategy

Setup and foundation precede stories. US1 supplies rendering consumed by US3/US4; US2 is independently testable and integrates error rendering. Tests precede each production change. Deliver all four stories together, with no intermediate owner gates.

Independent checks: US1 readable/machine equivalent outputs and actual causes; US2 early failure/hidden detached lifetime/redaction; US3 all work states/partial dates and clean redirected waits; US4 exact base/alias targets, strict election, roles/leases and dataset admission.

Parallel opportunities: independent read-only research and test suites; file edits to shared CLI/contracts/runtime remain sequential.

## Phase 8: Convergence

- [x] T021 Validate elected diagnostic destinations before workspace initialization in cmd/insonic/main.go, with a mutation-free failure regression per FR-006 and US2/AC3 (partial).
- [x] T022 Detect early child termination even if a descendant retains the startup report pipe in internal/process/startup.go, with a bounded regression per FR-005 and SC-002 (partial).

- [x] T023 Classify proven model publication/manifest mismatch as damaged readiness in internal/models/service.go and internal/app/model_readiness.go, with metadata-mismatch regressions per FR-011 (partial).
- [x] T024 Retain bounded warning omission/paging guidance in quiet human output and offer cancellation only for active or unobserved work in cmd/insonic/human.go, with operator result regressions per FR-008 (partial).

## Phase 9: Review and packaged startup convergence

- [x] T025 Stop and reap unready children on deadline, cancellation and rejected reports in internal/process/startup.go, preserving owner lifetime only after validated readiness; add lifecycle regressions per FR-005 and US2/AC4 (partial, final-round P1 review).
- [x] T026 Retry startup when a competing owner finishes shutdown within the shared deadline in internal/runtime/runtime.go; add a closing-owner regression and repeat installed qualification per FR-005 and SC-002 (partial, recurring package readiness timeout).
