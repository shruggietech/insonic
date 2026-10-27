# Tasks: Windows packaged runtime startup diagnosis and repair

**Input**: spec.md, plan.md, research.md, data-model.md, quickstart.md in this directory.

## Phase 1: Setup

- [x] T001 Establish S055/#191 identity in spec.md and .specify/feature.json; read authority and retained main failure.
- [x] T002 Complete specify, clarify and design in plan.md and research.md with no unsupported historical cause claim.

## Phase 2: Foundation

- [x] T003 Run the blocking Spec Kit analysis over spec.md, plan.md and tasks.md; verify checklist and ignored temporary output boundaries.

## Phase 3: US1 - Attributable failure evidence (P1)

**Independent test**: Inject a known startup failure, unwind owned fixtures and recover only bounded matching generated evidence.

- [x] T004 [US1] Add failing privacy, PID/correlation, malformed/oversized data and pre-cleanup regressions in tests/terminal_failure_evidence_test.py and tests/cli_terminal_qualification_test.py (FR-001, FR-002, FR-006).
- [x] T005 [US1] Implement bounded failure capture in scripts/terminal_failure_evidence.py, allowing only generated identities/enums/numbers and at most 16 matching native observations per client (FR-001, FR-002).
- [x] T006 [US1] Elect existing CI-only native diagnostics and capture before terminal/runtime unwind in scripts/cli_terminal_qualification.py; retain original failure and cleanup ownership (FR-003, FR-005, FR-006).

## Phase 4: US2 - Cause-driven owner repair (P1)

**Independent test**: Real IPC first-probe failure with held authority must join that owner and never call the forbidden launcher.

- [x] T007 [US2] Add and run failing owner-probe, held-lock cancellation/deadline and released-owner regressions in internal/runtime/startup_test.go; record the before result (FR-003, FR-004).
- [x] T008 [US2] Repair pre-launch authority arbitration in internal/runtime/runtime.go within the same ten-second context; preserve genuine startup errors, cold conflict races and protected-input load (FR-003, FR-004).
- [x] T009 [US2] Run focused Go and terminal Python regressions, retaining all eight scenarios and deadline/cleanup checks in tests/cli_terminal_qualification_test.py and tests/terminal_transport_test.py (FR-005, FR-007).

## Phase 5: US3 - Automated delivery (P2)

**Independent test**: Final head has terminal green applicable checks and resolved findings; PR remains open.

- [x] T010 [US3] Update authoritative docs/v1.0.2/cli.md and docs/v1.0.2/development.md plus CHANGELOG.md; run npm run check, npm test, go test ./... and go vet ./... (FR-008).
- [ ] T011 [US3] Commit/push codex/055-packaged-runtime-startup, open/attach PR for #191, and update milestone/Project tracking (FR-008).
- [ ] T012 [US3] Monitor exact-head CI; retain compact audited failed-attempt evidence before full reruns and correct every relevant failure (FR-006, FR-007, FR-008).
- [ ] T013 [US3] Resolve all actionable reviews within at most two Codex rounds and verify final SHA/checks/threads; record separate preparation/qualification/combined accounting in verification.md (FR-008).

## Phase 6: Polish and handoff

- [ ] T014 Converge the acceptance matrix in verification.md, scan encoding and remove only owned .local/s055-work staging after durable evidence retention; hand off without merging.

## Dependencies and execution

Setup -> analysis -> US1/US2 focused tests and implementation -> integrated verification -> US3 -> handoff. US1 evidence tests and US2 owner tests can run independently in separate files; each implementation follows its own failing regression. No parallel edits to shared files. MVP diagnostics alone are not slice completion; the demonstrated race repair and final package acceptance remain required.
