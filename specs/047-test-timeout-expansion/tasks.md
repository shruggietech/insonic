# Tasks: Comprehensive test timeout expansion

**Input**: S047 [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md).

**Tracking**: #172, milestone31, branch `codex/047-test-timeout-expansion`.

## Phase 1: Setup

- [x] T001 Specify, clarify and check S047 requirements in `specs/047-test-timeout-expansion/spec.md` and `checklists/requirements.md`.
- [x] T002 Complete research/design in `plan.md`, `research.md`, `data-model.md` and `quickstart.md`.
- [x] T003 Apply the explicit timing governance amendment in `.specify/memory/constitution.md` outside the read-only analysis step.

## Phase 2: Foundation

- [x] T004 Define canonical positive-integer stage policy in `internal/qualificationtiming/policy.json` and updated aggregate `scripts/ci-timing-policy.json` (FR-002/003/004/005/009).
- [x] T005 Add validated Go/Python/JS/frontend policy consumers in `internal/qualificationtiming/`, `scripts/qualification_timing.py`, `scripts/qualification-timing.mjs` and frontend configuration (FR-009/011).
- [x] T006 Establish checked timing owners/exceptions in `specs/047-test-timeout-expansion/deadline-inventory.json` (FR-001/011).

## Phase 3: US1 - Slow healthy execution

**Independent test**: Controlled delayed completion beyond the former bound succeeds without sleeping minutes.

- [x] T007 [P] [US1] Write former-bound success/new-bound expiry tests in `tests/ci-artifact.test.mjs` and `tests/package_desktop_test.py` before integration (FR-007).
- [x] T008 [P] [US1] Expand native/frontend fixture limits and enclosing matrices in `cmd/insonic-desktop/`, `desktop/qualification.go`, `desktop/frontend/src/qualification.ts` (FR-003/004/006).
- [x] T009 [P] [US1] Expand Python source/package/installer/CLI/processing/capture/preparation helpers in `scripts/` including embedded native/PowerShell waits (FR-001/003/004/005).
- [x] T010 [P] [US1] Expand Node/browser/site verification and artifact joins in `scripts/*.mjs`, `site/lib/screenshots.mjs`, `site/scripts/prepare.mjs` (FR-001/005).
- [x] T011 [P] [US1] Expand all preparation, qualification, release/site/diagnostic job watchdogs in `.github/workflows/` (FR-002/005).
- [x] T012 [US1] Verify real cumulative matrix and explicit shared-deadline relationships in policy tests and inventory (FR-001/003/004/005).

## Phase 4: US2 - Failure and evidence truth

**Independent test**: Expiry/cancellation/forced cleanup remain distinct; help progress uses monotonic elapsed observations.

- [x] T013 [P] [US2] Add native/frontend monotonic fixed-stage, allowlist and ordinary-session exclusion regressions in `desktop/qualification_test.go` and frontend tests before observer implementation (FR-007/008/011).
- [x] T014 [P] [US2] Preserve earlier cancellation, original absolute deadlines and late-response refusal in `scripts/wait-ci-artifact.mjs` and affected Python polling helpers (FR-005/006/007).
- [x] T015 [P] [US2] Implement bounded help progress in `desktop/qualification.go`, `desktop/frontend/src/main.tsx`, `help_qualification.go` and retain elapsed child outcomes in package qualification (FR-008).
- [x] T016 [US2] Verify bounded cleanup, nonzero timeout and genuine natural-release assertions in `tests/process_tree_test.py`, package/native tests (FR-006/010).

## Phase 5: US3 - Complete consistent coverage

**Independent test**: Inventory/consistency checks expose stale owners and distinguish product/failure-fixture exceptions.

- [x] T017 [P] [US3] Add cross-layer policy/workflow/inventory checks in `tests/qualification-timing.test.mjs` and affected policy tests (FR-001/009/011).
- [x] T018 [US3] Synchronize `AGENTS.md`, `docs/v1.0.1/development.md` and dated `CHANGELOG.md` policy effect; remove temporary constitution sync comment before commit (FR-009).
- [x] T019 [US3] Update source/cache recipe identity for consumed policy inputs in `scripts/media_source_build.py` and verification tests (FR-009/010).
- [x] T020 [US3] Audit all remaining active timing literals and justify retained bounds in `deadline-inventory.json` (FR-001/011).

## Phase 6: Verification and publication

- [x] T021 Run root `npm run check`, `npm test` and affected Python/Go/frontend/site suites; retain compact results in `verification.md` (FR-007/010).
- [ ] T022 Converge against all S047 requirements in `specs/047-test-timeout-expansion/` and repair findings (FR-001 through FR-013).
- [x] T023 Commit, automatically push `codex/047-test-timeout-expansion` and open official PR closing#172; attach PR and update GitHub Project (FR-013).
- [ ] T024 Monitor exact-head CI and automatic review, repair failures, answer/resolve every actionable thread in the PR (FR-010/013).
- [ ] T025 Dispatch isolated cold CI and full same-run warm rerun on stable implementation source; independently validate each terminal attempt/artifact (FR-012).
- [x] T026 Trigger at most a second Codex round if needed and resolve all findings; do not request a third (FR-013).
- [ ] T027 Complete final exact-head/check/review/clean-state verification and compact external evidence; stop before merge (FR-012/013).

## Dependencies and parallel execution

T001-T003 precede analysis. T004 establishes names before independent consumers. T005-T006 precede integration. File-owned native, Python and workflow/Node tasks can run in parallel after the analysis gate. Tests precede corresponding implementation. T012/T016/T020 integrate all branches of the work; publication follows required checks. Cold and warm campaigns run sequentially on one stable source, without intervening pushes that cancel the ref.

## Implementation strategy

Deliver expanded healthy execution with intact failures first, then complete consistency/docs, then hosted evidence/reviews. A partial policy or passing unrelated tests is not completion. Main post-merge acceptance remains outside the unmerged handoff.

## Phase 7: Convergence

- [x] T028 Expand and inventory remaining internal/process controlled child readiness and negative clock origins per FR-001/FR-007 (partial).
- [x] T029 Expand cmd/insonic and internal/runtime test observation/context allowances while retaining explicit cancellation fixtures per FR-001/FR-011 (partial).
- [x] T030 Diagnose models/backendsetup fixture failures and expand healthy test contexts/lease fixture headroom without changing production contracts per FR-001/FR-010 (partial).
- [x] T031 Audit remaining internal Go test timers, include all internal owners in deadline-inventory.json, and rerun full Go verification per FR-001/SC-001 (partial).

## Phase 8: Convergence

- [ ] T032 Diagnose and repair Linux successful-exit help receipt rejection, retain bounded stream identity/progress, and rerun affected hosted acceptance per FR-008/FR-010 (partial).

- [ ] T033 Verify the isolated runtime replacement fixture on hosted Windows and retain cold-run failure attribution without claiming a product permission repair per FR-006/FR-010/FR-011 (partial).
