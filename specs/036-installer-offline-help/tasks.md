# Tasks: S036 Installer progress and offline documentation

## Phase 1: Setup

- [x] T001 Verify exact issues/branch and author specs/036-installer-offline-help/spec.md and requirements checklist.
- [x] T002 Complete research, plan, state model, contracts and quickstart in specs/036-installer-offline-help/ using read-only MSI/help research agents.

## Phase 2: Foundation

- [x] T003 Add failing installed-help identity, path and opener/launch-user regressions in internal/launcher and desktop before product changes (FR-005/FR-007).
- [x] T004 Implement shared help resolution/association dispatch in internal/launcher and explicit documentation mode in cmd/insonic-launcher, preserving linked-token refusal and actionable warnings (FR-005/FR-007).

## Phase 3: US1 Native installer progress/results (P1)

Independent test: real disposable install/maintenance/cancel/failure journeys report engine progress and actual matching outcomes.

- [x] T005 [US1] Add failing portable native model/state regressions in tests/windows_msi_test.py for progress subscriptions, operation text, terminal buttons and focus/choices (FR-001..FR-003).
- [x] T006 [US1] Repair modeless progress, action text, cancellation confirmation and operation-specific ready/terminal pages in scripts/windows_msi.py (FR-001..FR-003).
- [ ] T007 [US1] Add bounded real full-UI transaction automation with passive native callbacks, owned HWND controls/progress/keyboard observations and result receipts to scripts/qualify-windows-msi.py or its helper; retain silent/ownership/rollback gates (FR-008/FR-009, SC-001/SC-004).

## Phase 4: US2 Packaged offline help (P1)

Independent test: actual native help activation renders installed content/resources/navigation offline in Windows/Linux packages.

- [x] T008 [US2] Add failing click/error/compact-control regressions in desktop/frontend/tests and bridge tests (FR-004/FR-005).
- [x] T009 [US2] Bind native help in desktop/bridge.go, frontend/src/client.ts and App.tsx; apply one compact consumer control in app.css without governed kit edits (FR-004/FR-005).
- [ ] T010 [US2] Extend native desktop/package qualification with actual controlled-help rendering before workspace selection and local-only resource/navigation checks on Windows/Linux, including spaces/Unicode and missing-help errors. Prove the installer OS association separately in the disposable ordinary actor (FR-005/FR-008, SC-002).

## Phase 5: US3 Independent completion documentation (P2)

Independent test: all client selections support either/both/neither post-action, excluded modes launch nothing, help dispatch remains same-user/non-elevated.

- [x] T011 [US3] Add failing completion election/default/mode/error regressions in tests/windows_msi_test.py and launcher tests (FR-006/FR-007).
- [x] T012 [US3] Add independent unchecked documentation election and nonfatal successful-fresh-interactive action to scripts/windows_msi.py; reuse explicit launcher mode and preserve checked client launch (FR-006/FR-007).
- [ ] T013 [US3] Qualify actual elected documentation opening and independent combinations through installer completion in scripts/qualify-windows-msi.py and update capture-installer-ui.py for corrected result/progress screenshots (FR-008, SC-003/SC-004).

## Phase 6: Delivery

- [ ] T014 Update docs/v1.0.1/desktop.md, windows-installation.md, screenshot evidence and CHANGELOG.md with actual defaults/modes/verification and dated build-script decisions (FR-008/FR-009).
- [ ] T015 Run regression-first and required full checks; converge all nine requirements/five criteria and record specs/036-installer-offline-help/verification.md (FR-001..FR-009, SC-001..SC-005).
- [ ] T016 Commit, push, open the three-issue PR, update milestone/Project tracking, resolve at most two code/security rounds, verify exact-head CI/threads and stop unmerged (SC-005).

## Dependencies and execution

T001 -> T002 -> read-only analyze gate -> T003/T004 -> T005/T006/T007 and T008/T009/T010 -> T011/T012/T013 -> T014/T015 -> T016. US1 and US2 can be qualified independently, while US3 integrates their shared boundaries. Primary agent owns sequential shared-file edits; independent read-only research and hidden verification may run in parallel. Do not run competing MSI transactions concurrently. Deliver all three issues, no release or merge.

## Phase 7: Convergence

- [ ] T017 [HIGH, partial, FR-001/FR-002/FR-003/FR-006/FR-007/FR-008, SC-001/SC-003/SC-004] Complete the actual full-UI driver in scripts/qualify_installer_ui.py using effective native UI-level evidence and a supported unrestricted transaction entry point or proven isolated desktop context. Observe all authored pages, engine progress, real outcomes/rollback and all independent completion elections without supplying license acceptance or substituting callback cancellation/results. Initial hosted receipts showed effective NONE before entering any authored UI; later native proof records full UI and actual authored pages.
- [ ] T018 [HIGH, partial, FR-008, SC-001/SC-005] Retain, verify and publish actual transaction progress/operation/cancel/failure captures and final proof in docs/screenshots/v1.0.1 and verification.md after T017 passes; keep the seven restricted-session captures truthfully distinguished. Verify exact-head CI and the bounded review/thread closure before owner handoff.
