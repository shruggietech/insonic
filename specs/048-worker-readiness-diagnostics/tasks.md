# Tasks: S048 Managed worker readiness diagnostics

**Input**: [spec.md](spec.md), [plan.md](plan.md), [contracts/probe.md](contracts/probe.md).
**Tracking**: #174, parent #147, milestone 32. Tests precede corresponding fixes.

## Phase 1: Setup

- [x] T001 Specify/clarify scope, research alternatives and complete design in specs/048-worker-readiness-diagnostics/.
- [x] T002 Verify clean main baseline, feature selector, existing ignore rules and S048 tracking in .specify/feature.json and .gitignore.

## Phase 2: Foundational contract

- [x] T003 Reproduce missing/ambiguous probe facts with current APIs in internal/workers/probe_diagnostics_test.go and tests/worker_probe_test.py (FR-001, FR-002, FR-007, FR-008).
- [x] T004 Add optional validated fixed-label probe facts in internal/workers/types.go and probe_diagnostics.go, with at most 16 observations, 0..3600000 ms durations, 0..60000 ms budget, 0..30000 files and 0..9007199254740991 bytes (FR-002, FR-006, FR-007).
- [x] T005 Register safe worker reason strings and schema bounds in internal/contracts/operator.go, schemas/v1.0.1/runtime-response.schema.json and worker-resources.schema.json (FR-001, FR-006, FR-007).

## Phase 3: US1 Understand inspection failure

**Independent acceptance**: Distinct generated reasons/stages survive hostile input, failed setup and both client displays.

- [x] T006 [US1] Add private atomic bounded checkpoints and fixed failure mapping in internal/workers/probe.py and probe.go (FR-001, FR-002, FR-007).
- [x] T007 [US1] Preserve failed inspection in setup checkpoints and fixed safe errors in internal/workers/setup.go and internal/app/workers.go, without publishing or retaining failed attempts (FR-001, FR-005, FR-007).
- [x] T008 [US1] Prove import/version/inventory/output and forged-progress rejection plus legacy/changed-byte acceptance boundaries in internal/workers/probe_diagnostics_test.go and tests/worker_probe_test.py (FR-005, FR-006, FR-007, FR-008).
- [x] T009 [US1] Surface bounded probe summaries in cmd/insonic/human_summary.go and desktop/frontend/src/workers.tsx, with focused human/frontend regressions (FR-001, FR-006, FR-007).

## Phase 4: US2 Truthful bounded readiness

**Independent acceptance**: Earlier expiry/cancellation never becomes integrity mismatch or fresh preparation; already observed roles remain distinguishable.

- [x] T010 [US2] Reproduce canceled hash/receipt/reuse and exhausted role budgets in internal/workers/probe_diagnostics_test.go and internal/app/worker_diagnostics_test.go (FR-003, FR-004, FR-005, FR-008).
- [x] T011 [US2] Preserve deadline/cancellation through internal/workers/types.go, probe.go, setup.go and internal/app/workers.go (FR-003, FR-005).
- [x] T012 [US2] Preserve partial readiness and mark unattempted role prerequisites unknown with fixed actions in internal/app/worker_readiness.go (FR-003, FR-004).
- [x] T013 [US2] Add actual ephemeral IPC disconnect/cancellation regressions in internal/runtime/worker_disconnect_test.go; fix scoped cancellation in runtime.go and desktop/bridge.go without canceling durable setup (FR-003, FR-006, FR-008).
- [x] T014 [US2] Verify healthy readiness, genuine integrity failure and cancellation fencing across workers/app/runtime/desktop suites (FR-003, FR-004, FR-005, FR-006).

## Phase 5: US3 Source-bound managed CPU acceptance

**Independent acceptance**: Fresh snapshot and repeated same-mode measurements are labeled accurately, and actual CPU processing uses the exact clean candidate.

- [x] T015 [US3] Extend maintained scripts/journey_cpu.py measurement/receipt projection and relevant harness tests for bounded probe facts (FR-002, FR-009).
- [ ] T016 [US3] Run an isolated source-bound complete-package managed CPU journey, retain fresh snapshot/repeated inspect observations and actual processing outcomes on the PR, then verify owned cleanup (FR-009, FR-010).

## Phase 6: Qualification and delivery

- [x] T017 Document fixed probe reasons, clock origins, partial readiness and deadline/cancellation behavior in docs/v1.0.1/processing-setup.md and CHANGELOG.md (FR-001, FR-003, FR-006).
- [x] T018 Run root check/test, focused/full affected Go/Python/frontend/site checks and encoding checks; record compact outcomes in specs/048-worker-readiness-diagnostics/verification.md (SC-001, SC-002, SC-003).
- [ ] T019 Converge current implementation against every FR/SC/scenario/design/constitution obligation; append only concrete remaining work to this tasks.md.
- [x] T020 Commit clean source, automatically push and publish official PR for #174/#147; keep #147 open and retain exact-head CI evidence on GitHub (SC-004/FR-010).
- [ ] T021 Resolve every individual comment from at most two Codex rounds and verify terminal applicable checks after repairs; retain earlier failures separately (SC-004).
- [ ] T022 Complete source/evidence/tracking/cleanup audit and hand off unmerged for owner review in specs/048-worker-readiness-diagnostics/verification.md (SC-004/FR-010).

## Dependencies and execution

Setup -> contract -> US1 -> US2 -> US3 -> final delivery. US1/US2 have independent injected failure fixtures; US3 uses the completed shared contract. Python fixture tests and Go validation tests may execute concurrently after setup; schema/human/frontend changes touch separate files but stay on the same authoritative contract. No implementation agent delegation is required. Deliver all three stories before handoff; a partial MVP is not completion.

## Phase 7: Convergence

- [ ] T023 CRITICAL: Make bounded probe history compatible with the existing durable-result privacy contract, preserve the raw-observations rejection, and prove healthy setup checkpoint/publication through the real catalog in internal/app/worker_diagnostics_test.go; rerun real CPU acceptance and applicable CI per FR-006/SC-003 (contradicts).

## Phase 8: Convergence

- [ ] T024 Give the real Arcade functional fixture the maintained request allowance without changing its production client default; prove delayed completion and earlier caller cancellation with virtual time in internal/graph/arcade_test.go, then rerun all elected CI checks per SC-004 (partial).

## Phase 9: Convergence

- [ ] T025 Wait for the existing correlated owned-runtime startup report before issuing qualification commands; retain bounded ready/failure facts, honor the original readiness/overall deadline and prove gating, invalid-report refusal and expiry in tests/journey_cpu_test.py before rerunning source-bound CPU acceptance per FR-003/FR-009 (partial).

## Phase 10: Convergence

- [ ] T026 Reject an optional native success diagnostic whose validated child records failure, while retaining valid native receipt-verification failure after child success; reproduce through the real receipt validator in internal/workers/probe_diagnostics_test.go, repair probe_diagnostics.go and rerun acceptance per FR-005/FR-007 (partial).
