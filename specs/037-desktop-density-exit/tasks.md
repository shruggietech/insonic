# Tasks: S037 Desktop density and standard Exit

## Phase 1: Setup

- [x] T001 Verify #112/#113, codex/037-desktop-density-exit and specs/037-desktop-density-exit/spec.md agree; create the quality checklist.
- [x] T002 Complete plan.md, research.md, data-model.md, contracts/interaction.md and quickstart.md with the required read-only planning agents.

## Phase 2: Foundation

- [x] T003 Complete the read-only spec/plan/tasks analysis gate and record its result in verification.md before product edits (FR-001..FR-009, SC-001..SC-005).

## Phase 3: US1 Compact desktop density (P1)

Independent test: actual six core workflows preserve square glyphs, clickable label targets, intrinsic actions and reachable commands across the declared window/renderer/text matrix.

- [x] T004 [US1] Add meaningful failing selection-label and rendered-geometry regressions in desktop/frontend/tests/density.test.mjs and scripts/desktop-density-capture.mjs before CSS/component edits (FR-001..FR-005, SC-001..SC-003).
- [x] T005 [US1] Apply documented token-based consumer composition in desktop/frontend/src/components.tsx and app.css: square glyphs, label targets at least44 by44, intrinsic action widths and contained overflow (FR-001..FR-005).
- [x] T006 [US1] Add the source-shared observer in desktop/frontend/src/density.ts; integrate the full renderer matrix and clean-source representative image receipt in scripts/capture-application-ui.mjs and scripts/qualify-ui-capture.py (FR-004/FR-008, SC-001..SC-003).
- [x] T007 [US1] Require real compact geometry observations during core native Windows/Linux workflow visits in desktop/frontend/src/qualification.ts and cmd/insonic-desktop/main.go (FR-005/FR-008, SC-003).

## Phase 4: US2 Native Exit (P2)

Independent test: actual owned native menu activation quits without/with a workspace; playback ownership closes while a held GUI-independent acquisition retains the same runtime session/work and finishes after release.

- [x] T008 [US2] Add failing menu-label/composition and synchronized early/repeated-quit tests in desktop/menu_test.go before desktop/menu.go implementation (FR-006/FR-007).
- [x] T009 [US2] Integrate the native menu and guarded lifecycle in cmd/insonic-desktop/main.go, preserving ordinary close/playback/runtime ownership (FR-006/FR-007, SC-004).
- [x] T010 [US2] Implement bounded explicit native test hooks in cmd/insonic-desktop/exit_qualification*.go and the private runtime/acquisition owner in cmd/insonic-desktop/exit_qualification.go with ownership/error regressions in exit_qualification_test.go (FR-006..FR-008, SC-004).
- [x] T011 [US2] Require native source/package Exit proofs through scripts/qualify.py and scripts/package-desktop.py without dropping existing journey/recovery gates (FR-008, SC-004/SC-005).

## Phase 5: Delivery

- [ ] T012 Document the actual density/menu/shutdown behavior and approved consumer adaptation in docs/v1.0.1/desktop.md, development.md and CHANGELOG.md; publish representative clean-source images and original receipt in docs/screenshots/v1.0.1 (FR-005/FR-009).
- [ ] T013 Run required root/frontend/Python/Go/site checks, rendered/native proofs and convergence; record exact evidence and limits in specs/037-desktop-density-exit/verification.md (FR-008/FR-009, SC-001..SC-005).
- [ ] T014 Commit/push, open the two-issue PR, maintain milestone/Project tracking, resolve at most two code/security review rounds, verify exact-head CI/all threads and stop unmerged for owner final review (SC-005).

## Dependencies and execution

T001 -> T002 -> T003 -> T004 -> T005 -> T006/T007; independently T008 -> T009 -> T010 -> T011. Both stories -> T012 -> T013 -> T014. US1 is the MVP but the owner-approved slice delivers both stories. Source edits are sequential under the primary agent. Independent read-only research and read-only test suites may run concurrently; do not run competing installers or source mutations. Parallel examples: menu unit checks can run alongside renderer measurements after both implementations stabilize; source inventory reads can run alongside GitHub metadata reads. No additional agent implementation delegation is required.
