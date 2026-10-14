# Tasks: Windows MSI and installation guide

Input: [spec.md](spec.md), [plan.md](plan.md), research, data model and contracts. Tests are required by autopilot.

## Phase 1: Setup

- [x] T001 Confirm S027/#65/#64 branch, directory and authority in specs/027-windows-msi-installation/spec.md.
- [x] T002 Complete research/design and record retired native owner-host probe in specs/027-windows-msi-installation/plan.md.

## Phase 2: Foundation

- [x] T003 [P] Write MSI identity/client/ownership/table tests in tests/windows_msi_test.py.
- [x] T004 Implement deterministic native MSI model and exact verification in scripts/windows_msi.py.
- [x] T005 [P] Write launcher quoting and noninteractive contract tests in internal/launcher/launcher_test.go.
- [x] T006 [P] Write capture isolation/provenance/request-boundary tests in tests/capture-application-ui.test.mjs and cmd/insonic-capture-bridge/main_test.go.

## Phase 3: US1 Selected-client installation

Independent test: all three nonempty selections install and reach useful first-use; empty selection is rejected.

- [x] T007 [US1] Author shared/desktop/CLI features, actual license/directory/choices/native pages in scripts/windows_msi.py and scripts/package-msi.py.
- [x] T008 [P] [US1] Implement selected-client launcher and preflight/describe in cmd/insonic-launcher/ and internal/launcher/.
- [x] T009 [US1] Include launcher/session assets and exact inventory identity in scripts/package-desktop.py.
- [x] T010 [US1] Add per-user prerequisites, optional branded shortcuts, full-UI fresh-install launch suppression and Installed Apps metadata in scripts/windows_msi.py.

## Phase 4: US2 Maintenance and release identity

Independent test: lifecycle/rollback preserves fixture user data and unrelated integrations; tampered MSI/source identity is rejected.

- [x] T011 [US2] Add transactional feature/repair/remove, upgrade/downgrade and active-work behavior in scripts/windows_msi.py.
- [x] T012 [US2] Implement disposable-CI-only real install/maintenance/rollback qualification in scripts/qualify-windows-msi.py.
- [x] T013 [P] [US2] Write release missing/duplicate/wrong-format/source/qualification regressions in tests/release_test.py.
- [x] T014 [US2] Add exactly one typed MSI beside four archives in scripts/release.py and schemas/v1.0.1/release-candidate.schema.json.
- [x] T015 [US2] Add MSI publish/retrieval identity, checksums and current download links in scripts/release.py and site/lib/release-status.mjs.

## Phase 5: US3 Accurate guide

Independent test: complete text is usable without images and online/offline images/controls match receipts.

- [x] T016 [P] [US3] Publish verified conditional unsigned/prerequisite/MSI/ZIP guidance in docs/v1.0.1/windows-installation.md.
- [x] T017 [US3] Carry versioned screenshots into public/offline output with hash/alt/caption parity in site/scripts/ and site rendering.
- [x] T018 [US3] Integrate current guide/download navigation and exact proposed label set in docs/v1.0.1/desktop.md, docs/versions.json and specs/027-windows-msi-installation/verification.md.

## Phase 6: US4 Autonomous capture

Independent test: capture actual app/installer controls with source/renderer receipts and zero owner-host native UI.

- [x] T019 [US4] Implement bounded headless compiled-frontend/current-CLI bridge capture in scripts/capture-application-ui.mjs and cmd/insonic-capture-bridge/.
- [x] T020 [US4] Implement disposable-CI-only authored native page capture with exact owned-process cleanup in scripts/capture-installer-ui.py.
- [ ] T021 [US4] Generate and inspect current fixture PNGs with exact source/renderer/privacy receipts in docs/screenshots/v1.0.1/.

## Phase 7: Delivery

- [x] T022 Integrate parallel/cache-scoped MSI and capture gates and candidate artifacts in .github/workflows/ci.yml and native-qualification.yml.
- [x] T023 Record changed behavior and dated architecture/execution decisions in CHANGELOG.md.
- [ ] T024 Run configured integrity/domain/product/site checks and native CI evidence in specs/027-windows-msi-installation/verification.md.
- [ ] T025 Converge all FR/SC/journeys against implementation in specs/027-windows-msi-installation/convergence.md.
- [ ] T026 Commit/push official PR, address every finding in at most two rounds and verify final exact-head green CI in specs/027-windows-msi-installation/verification.md.

## Dependencies and parallel strategy

Foundation contracts precede integration. MSI agent owns windows_msi/package-msi/native lifecycle/capture and associated tests. Capture agent owns compiled-application capture/fixture/boundary tests. Release/docs agent owns release/schema/site/doc changes. Parent owns launcher, package-desktop integration, CI, changelog and integration. Separate files allow parallel implementation after analysis; never edit another owner's path concurrently. Native install/UI is CI-only, while local probes are noninteractive model/receipt/describe tests or verified headless Chromium.

## Implementation strategy

Establish typed identity/tests, deliver selected clients, transactional lifecycle, release format and captures, then integrate guide and exact-source candidate verification. All four stories are required; no partial MVP is declared complete. Label proposal approval remains the owner final-review shipping boundary.

## Convergence follow-up

- [x] T027 [US2] Require actual cancellation rollback and ordinary-account per-user context observations in scripts/qualify-windows-msi.py, verified in disposable CI through T024.
- [x] T028 [US1] Launch selected clients with an unelevated same-user token when the installer is elevated in internal/launcher/windows.go and elevation_windows.go, verified in disposable CI through T024.

- [x] T029 [US2] Resolve directory aliases before library containment and active-client comparison in internal/launcher/ with an alias-to-library regression.
- [x] T030 [US2] Record independent unsigned MSI signing and reject dirty or missing screenshot source inventories, including external review regressions.
- [x] T031 [US4] Bind headless Chromium to an explicit owned non-default browser profile for disposable Windows CI.

- [x] T032 [US2] Restore independently verified same-run desktop archives into an owned workspace path and separate MSI lifecycle checks into their own ten-minute job in .github/workflows/native-qualification.yml and scripts/package-msi.py.
- [x] T033 [US2] Replace MSI composite Upgrade key mutation with complete verified row replacement and an actual native database-only clone regression in scripts/qualify-windows-msi.py and tests/windows_msi_test.py.

- [x] T034 [US2] Run native Installer APIs in an exact owned hidden worker with checked UI/log initialization and bounded process/service diagnostics in scripts/qualify-windows-msi.py.
- [x] T035 [US2] Persist unchecked shortcut elections as zero and verify migration without automation overriding the saved choices in scripts/windows_msi.py and scripts/qualify-windows-msi.py.
- [x] T036 Hold only synthetic completion timers while exercising real durable active-capacity attempts in internal/app/capacity_test.go; keep the portable capture-source test independent of Windows subprocess constants.
- [x] T037 [US4] Bound capture phases and cleanup, create conventional isolated Windows profile directories, and retain failure diagnostics in scripts/capture-application-ui.mjs and native-qualification.yml.
- [x] T038 [US4] Reject untracked capture inputs and modified embedded Go builds, with a temporary repository regression for final external review.

- [x] T039 [US2] Author and independently verify required empty AppSearch locator tables and require intended negative-case messages/actions in scripts/windows_msi.py and scripts/qualify-windows-msi.py.
- [x] T040 [US2] Preserve explicit install-directory elections and check both incoming and registered installation roots during moving upgrades, with canonical-path and disposable active-process regressions.
- [x] T041 [US2] Refuse elected exact-name shortcut collisions outside MSI ownership, and verify preservation of unrelated sibling shortcuts and registry values during actual CI lifecycle.
- [x] T042 [US2] Give the CI worker an owned noninteractive desktop for its invoking account and startup diagnostics without changing shared desktop ACLs in scripts/qualify-windows-msi.py.
- [x] T043 [US4] Render verified property-backed authored pages through a restricted MSI session and retain both capture/lifecycle failures independently in scripts/capture-installer-ui.py and scripts/package-msi.py.

- [x] T044 [US4] Locate the owned native directory control by its verified normalized property value rather than a guessed window class.
- [x] T045 [US2] Diagnose the synthetic restricted-token startup and integrity failure; retire its unused construction after actual CI proved it unsuitable for ordinary-user qualification.

- [x] T046 [US2] Qualify a genuine owned ordinary CI account/profile with exact staged source/payload identities, same-actor elected children, isolated environment and bounded account/profile/stage cleanup; retain independent administrator lifecycle progress.

- [x] T047 [US2] Count retained local MSI features only when no action is scheduled, reject explicit removal of mandatory selections, and match GUI feature-add reinstall behavior.
- [x] T048 Join isolated package GUI checks before restoring development libraries, overlap them with the CLI journey, preserve the entire detached-runtime idle deadline and bound all read-only CLI checks to the existing four workers.

- [x] T049 Disable appearance Save until actual settings readback supplies a valid revision and make automated clicks wait for enabled controls within the existing deadline.
- [x] T050 Observe actual MSI action/installed feature states in a diagnostic-only clone and a temporary disposable CI lane with independently verified prior payload identity; exclude diagnostic results from publication eligibility.
