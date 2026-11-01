# Tasks: v1.1.0 release preparation and qualification

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [contracts/release-preparation.md](contracts/release-preparation.md).
**Tests**: Meaningful failing-first capability metadata regression plus maintained complete qualification.

## Phase 1: Setup

- [x] T001 Establish S065 branch/selector and specification, clarification, built-in quality, reviewer checklist and design artifacts in specs/065-v110-release/ (FR-001, FR-009).
- [x] T002 Create issue #237, milestone 50 and Project 7 metadata with Slice S065/Release 1.1.0, recorded in specs/065-v110-release/spec.md (FR-009).

## Phase 2: Foundation

- [x] T003 Run existing preparation and historical-authority regressions in tests/release_test.py, tests/release_history_test.py, tests/recovery_release_history_test.py and tests/backup_release_history_test.py before version migration; retain failures honestly under .local/s065/ (FR-003, FR-008).

## Phase 3: US1 - Coherent candidate

Independent test: current release authorities agree on 1.1.0, additive documented behavior remains usable, capability metadata is truthful and upstream identities do not change.

- [x] T004 [US1] Add and reproduce a current candidate collection regression in tests/release_test.py requiring advanced_cli to include backup, recordings coverage and recordings repair, preserving the historical fixture (FR-002; SC-001).
- [x] T005 [US1] Correct current candidate capability metadata in scripts/release.py and pass the focused collection tests without weakening manifest integrity checks (FR-002, FR-008; SC-001).
- [x] T006 [US1] Write concise specs/065-v110-release/highlights.md, record the release-metadata/preparation decision in CHANGELOG.md, and run maintained scripts/release.py prepare for version 1.1.0 dated 2026-10-31 (FR-001, FR-002; SC-001).
- [x] T007 [US1] Validate coordinated VERSION/manifests/current runtime/schema/worker envelopes and retained prior document support; add the current CLI/GUI matrix to docs/v1.1.0/desktop.md, distinguishing shared processing from advanced range controls (FR-001, FR-002, FR-008; SC-001).

## Phase 4: US2 - Historical and visual authority

Independent test: original historical docs/contracts retain tag identity; published/candidate status differs correctly; all current displayed images are original source-bound captures.

- [x] T008 [US2] After copy-forward, restore docs/v1.0.3/ and schemas/v1.0.3/ from exact published tag authority, remove only unpublished additions through that restoration, and mark 1.0.3 released in docs/versions.json while 1.1.0 remains candidate (FR-003, FR-004; SC-001, SC-002).
- [x] T009 [US2] Run historical/preparation/schema/site-release regressions and verify tag-tree equivalence, frozen fixtures and upstream resource pins using tests/release_test.py, tests/site-release.test.mjs and site/tests/release-status.test.mjs (FR-003, FR-004, FR-008; SC-001, SC-002).
- [x] T010 [US2] Save and temporarily omit only new-version screenshot walkthrough blocks in docs/v1.1.0/desktop.md and windows-installation.md; complete clean-bootstrap local checks under .local/s065/ (FR-005, FR-006; SC-002, SC-003).
- [x] T011 [US2] Commit/push the clean bootstrap and dispatch .github/workflows/ci.yml for its exact branch SHA without opening the official PR; retain its identity in specs/065-v110-release/verification.md (FR-005, FR-006, FR-009; SC-002).
- [x] T012 [US2] Retain/audit bootstrap terminal acceptance and digest-verify original capture archives; import only the 25 receipt-enumerated PNGs and four unchanged original receipts into docs/screenshots/v1.1.0/ (FR-005, FR-007; SC-002, SC-003).
- [x] T013 [US2] Restore new-version image embeds in docs/v1.1.0/desktop.md and windows-installation.md, validate original source/version/hash/dimensions and visually inspect all captures with maintained site screenshot checks (FR-005; SC-002).

## Phase 5: US3 - Qualified owner handoff

Independent test: complete final-source candidate/platform acceptance and individually resolved reviews support a clean unmerged PR with a reproducible continuation.

- [x] T014 [US3] Complete foreground root check/test, full Python and Go tests/vet, frontend check/test/build and site test/public/offline build with logs under .local/s065/; preserve all mandatory substantive and security/integrity assertions (FR-006, FR-008; SC-003).
- [x] T015 [US3] Finalize executable accepted-main release-workflow, separately elected real-engine and exact-retained-byte publication/public-verification commands in specs/065-v110-release/quickstart.md (FR-010, FR-011; SC-005).
- [x] T016 [US3] Converge actual implementation against every requirement, acceptance scenario, decision and constitution principle using specs/065-v110-release/tasks.md, adding any genuine gap before delivery (FR-001 through FR-011; SC-001 through SC-005).
- [x] T017 [US3] Commit/push complete source and genuine captures, open the official non-draft PR and update issue #237/Project tracking, recording the first review trigger in .local/s065/reviews.json (FR-009; SC-004).
- [x] T018 [US3] Monitor/fix final PR and direct-head CI, retain digest-verified acceptance archives and original run/job/artifact metadata, terminal-audit them sequentially, and retain/load-validate the complete final direct-head candidate archive using scripts/release.py against exact source (FR-006, FR-007, FR-008; SC-003).
- [x] T019 [US3] Resolve every actionable review and individual thread within the automatic first plus at most one explicit second round, retaining actual receipts and any absent security-review evidence in .local/s065/reviews.json (FR-009, FR-011; SC-004).

## Phase 6: Closeout

- [x] T020 Finalize specs/065-v110-release/verification.md and task status, commit/push records, re-establish T018 acceptance on the final head, verify clean source and issue/project/review state, and publish final exact-source receipts in the PR before owner handoff (FR-007, FR-009, FR-010, FR-011; SC-003, SC-004, SC-005).

## Dependencies and execution

Setup -> baseline preservation -> failing capability regression -> bounded fix -> preparation -> historical reconciliation -> clean capture bootstrap -> original images -> complete local checks -> convergence -> official PR/reviews/final CI -> closeout and final-head requalification. T018 is repeated after any later commit; the last external receipt avoids embedding a commit's own hash inside itself. Failure receipts are retained before any full rerun.

## Parallel examples

Read-only research agents cover independent release/history, capture and acceptance questions. After source readiness, independent root/frontend/site and Python/Go checks may run concurrently in foreground if they do not share mutable outputs; acceptance audits sharing terminal.json remain sequential. Package GUI and installer work stays on isolated CI desktops. Shared source changes remain serial.

## Implementation strategy

First establish coherent current candidate metadata, then historical/visual evidence, then full final acceptance. The bootstrap is an evidence producer, not a published release. Existing owner autopilot/push authority permits all routine stages without another permission request. Reviewer-owned custom checklist markers are not rewritten; built-in quality and mandatory analyze/converge gates remain required.

## Phase 7: Demonstrated release blocker

- [x] T021 Fix the 64 MiB static-export limit exposed by bootstrap CI with one shared 128 MiB bound across scripts/pages.py, scripts/site-transfer.py and scripts/website.py; reproduce the failure, preserve finite over-limit and historical-alias rejection, update current docs and pass affected tests (FR-006, FR-008; SC-003).
- [x] T022 Retain the rejected bootstrap terminal receipt, commit the bounded correction and repeat clean capture qualification before resuming T012 (FR-005, FR-007; SC-002, SC-003).

## Phase 8: Convergence

- [x] T023 Align contracts/release-preparation.md with plan D2: candidate metadata lists backup, recordings coverage and recordings repair; the documentation matrix separately describes advanced window/range configuration and shared automatic processing (contradicts; MEDIUM).
