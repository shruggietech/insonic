# Tasks: Reliable website publishing from main

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/automation.md.

## Setup and gates

- [x] T001 Record S032 scope, research, design and issues #106/#107.
- [x] T002 Analyze requirement coverage/checklist criteria in specs/032-main-website-publication/analysis.md before source edits.

## User Story 1: qualified publication (P1)

Independent test: invalid data causes zero writes; transport preserves concurrent work/schema bytes.

- [x] T003 [US1] Write failing source/release/archive/schema/transport/replay tests in tests/website_test.py.
- [x] T004 [US1] Implement scripts/website.py with existing archive/process/destination contracts and bounded owned Git transport.
- [x] T005 [US1] Add trusted-main .github/workflows/website.yml and release.yml destination serialization.

## User Story 2: live proof (P1)

Independent test: stale bytes, broken downloads and measured layout failures reject verification with evidence retained.

- [x] T006 [US2] Write failing live/download/browser tests in tests/website_test.py and tests/live-site-ui.test.mjs.
- [x] T007 [US2] Implement full live verification and scripts/check-live-site-ui.mjs with shared assertions.
- [x] T008 [US2] Add optional read-only hosted live smoke to ci.yml.

## User Story 3: CI headroom (P2)

Independent test: drift fails, selection conservative, final full CI below 660 seconds.

- [x] T009 [US3] Write failing transfer/scope/workflow tests in tests/site_transfer_test.py and tests/ci-scope.test.mjs.
- [x] T010 [US3] Implement scripts/site-transfer.py; split producer/qualifier, overlap scope/foundation, reuse exact offline help in native-qualification.yml without fallback.

## Delivery

- [x] T011 Update docs/v1.0.1/releases.md, development.md, CHANGELOG.md; check encoding and planning separation.
- [ ] T012 Run root/affected/hosted checks and Spec Kit converge; record exact evidence in verification.md.
- [ ] T013 Commit/push official PR, resolve at most two actual review rounds, inspect every thread and final exact-head CI; request owner merge review.

## Dependencies and execution

T001–T002 block source edits. Tests precede each behavior group. US2 uses US1 identities; US3 transfer is independent after analysis. Delivery follows implementation. Separate test files permit independent checks; no implementation delegation required. Deliver the complete slice together.

## Phase 6: Convergence

- [x] T014 Complete bounded owned-browser termination in scripts/check-live-site-ui.mjs per FR-008 and SC-005 (partial). Convergence inspected 12 FRs, five success criteria, nine acceptance scenarios, seven architecture decisions and all applicable constitution constraints. One partial/high finding; no missing, contradicting or unrequested work. Hosted evidence remains T012, delivery remains T013.
