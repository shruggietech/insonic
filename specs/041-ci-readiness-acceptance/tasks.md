# Tasks: Verified CI readiness and elected acceptance

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md).

## Phase 1: Setup

- [x] T001 Verify #161, exact main and feature mapping in specs/041-ci-readiness-acceptance/spec.md.
- [x] T002 Record owner timing decision and constitution 3.1.0 in .specify/memory/constitution.md.

## Phase 2: Foundation

- [x] T003 Complete design/research and Spec Kit consistency gate for specs/041-ci-readiness-acceptance/plan.md.

## Phase 3: US1 readiness (P1)

**Independent test**: Actual workflow graph delays consumers past a 1,342-second cold producer; failed/cancelled inputs never start them.

- [x] T004 [US1] Retain helper 180-second characterization and failing-before actual graph simulation in tests/ci-readiness.test.mjs (FR-001, FR-009, SC-001).
- [x] T005 [US1] Add terminal source ready join and gate all qualification consumers in .github/workflows/media-source.yml and .github/workflows/ci.yml (FR-001, FR-002).
- [x] T006 [US1] Record resolved toolchain package observation through scripts/source-toolchain.py and source artifacts in .github/workflows/media-source.yml (FR-008).
- [x] T007 [US1] Update obsolete dependency assertions in tests/ci-scope.test.mjs and run existing transfer trust tests (FR-003, FR-004).

## Phase 4: US2 elected acceptance (P1)

**Independent test**: Every elected non-success, unknown scope and over-budget interval rejects acceptance; intentional non-election is recorded.

- [x] T008 [US2] Add result/timing/identity/API failure regressions in tests/ci-acceptance.test.mjs (FR-005, FR-007, SC-002, SC-003).
- [x] T009 [US2] Implement always-running acceptance and terminal audit in scripts/ci-acceptance.mjs and .github/workflows/ci.yml (FR-005, FR-007).
- [x] T010 [US2] Declare required aggregate in .github/bootstrap.json without mutating hosted protection before publication (FR-006).

## Phase 5: Verification and handoff

- [x] T011 Update docs/v1.0.1/development.md and CHANGELOG.md with current contracts and dated decision (FR-007).
- [x] T012 Run root/site/affected checks, encoding sanity and final diff, record evidence in specs/041-ci-readiness-acceptance/verification.md.
- [x] T013 Commit verified local work and report remaining hosted cold/warm, protection application and post-merge evidence in specs/041-ci-readiness-acceptance/verification.md (SC-004).

## Dependencies and execution

Setup -> foundation -> US1 -> US2 -> verification. Different test files may be authored independently; shared workflow edits remain sequential. US1 is the first independently testable increment; US2 adds the merge acceptance contract. Hosted closure follows authorized publication and is explicitly outside local completion evidence.

## Phase 6: Convergence

- [x] T014 After authorized publication, retain exact cold/warm hosted preparation, qualification and terminal timing receipts per SC-004 (partial).
- [x] T015 After the aggregate is published, apply/read back the existing bootstrap required check and reconcile #161 milestone/project tracking per plan: hosted enforcement (partial).
- [x] T016 Verify exact owner-authorized squash main package acceptance and terminal timing before closing #161 per plan: post-merge evidence and SC-004 (partial).

## Phase 7: Measured hosted timing repair

- [x] T017 Record first hosted failure and analyze dependency-only preparation against the approved timing contract in plan.md, research.md and verification.md.
- [x] T018 Add failing-before preparation, identity, archive and consumer-graph regressions; implement scripts/go-dependency-transfer.py and the Windows preparation job.
- [x] T019 Wire verified package consumers and readiness/aggregate; preserve product builds and all acceptance; run root, site and affected checks.
- [x] T020 Push the verified repair and monitor exact-head CI to terminal; retain final timing and package evidence, stop before merge.

- [x] T021 Analyze the second measured budget failure; add setup compiler dependencies, shared-deadline input gate and joined independent CI setup work; preserve every product and native check.

## Phase 8: Convergence

- [x] T022 Add an opt-in isolated compiled-media cache campaign to .github/workflows/ci.yml and .github/workflows/media-source.yml, retain cold/warm path outputs, and verify ordinary cache keys and all acceptance checks remain unchanged per SC-004 and FR-002/003/009 (missing).

## Phase 9: Convergence

- [x] T023 CRITICAL Include first-attempt initial queue time in preparation and combined durations in scripts/ci-acceptance.mjs, preserve the readiness-based qualification clock and per-rerun start, add regression coverage in tests/ci-acceptance.test.mjs, and update timing documentation per Constitution V, FR-007 and SC-003 (contradicts).

## Phase 10: Bounded Windows qualification convergence

- [x] T024 Amend the S041 calibration scope, preserve historical T014/T015 and pending T016, predeclare two/ten-second cohorts and pass Spec Kit consistency analysis before experiments (FR-004/010-013, SC-005/006).
- [x] T025 Add passive archive-child desktop/input and created-environment identity observations, successful/failed monotonic UI and stage timings, retained successful child evidence, and meaningful native/parser regressions (FR-010/011).
- [x] T026 Reuse the existing unattended diagnostic workflow for one same-executable sequential two/ten-second twelve-case comparison with equivalent separate profiles, unchanged outer bounds and retained first failures (FR-012, SC-005).
- [x] T027 Adopt the justified documented fixture allowance or correct the demonstrated bounded defect, separating any necessary production behavior repair into the next available dependent slice in PR #162 (FR-004/013).
- [x] T028 Complete required root, affected Windows/product/site checks, source-authority hashes, encoding/diff self-review and Spec Kit convergence; publish the final frozen candidate without another hosted Codex review (FR-003/013, SC-006).
- [x] T029 Monitor final ordinary-path CI through terminal acceptance, retain exact identities/all twelve archive cases/natural release and phase/queue accounting, update PR and tracking evidence while keeping #161/#154 open and T016 pending (FR-005/007/013, SC-004/006).

## Phase 11: Convergence

- [x] T030 CRITICAL Read first-attempt initial creation from the identity-verified workflow record, retain attempt-record creation separately, reject mismatched authority/invalid ordering, and verify terminal accounting against retained hosted metadata in scripts/ci-acceptance.mjs and tests/ci-acceptance.test.mjs per Constitution V, FR-007 and SC-003 (contradicts).
- [x] T031 Use a non-expiry fixture lease for portable-history assertions in internal/catalog/backup_processing_test.go after the hosted one-second claim expired; preserve production fencing and dedicated expiry tests, and run affected/full tracked-package checks per FR-003 and SC-006 (partial).

## Phase 12: Owner-authorized timing-policy correction (2026-10-21)

T029 evidence: final PR head e578f3c0339d086bbb851a1bdae4522820b35773, tested merge d9c08b0a80bfe83b07350e5d1f90127a817a9033, [run 39558146331](https://github.com/shruggietech/insonic/actions/runs/39558146331), attempt 1; 251s preparation, 566s qualification, 817s combined, zero initial queue. [Existing PR evidence](https://github.com/shruggietech/insonic/pull/162#issuecomment-6310309533) completes T029. The subsequent main run 39576651956 failed under the old 660-second policy and remains failed.

- [x] T032 Amend constitution, active S041 requirements and current development contracts; complete Spec Kit consistency analysis (FR-007/014-016, SC-003/007).
- [x] T033 Add focused fake-clock, terminal/provisional boundary, fixed-deadline, Python remaining-interval and scoped watchdog regressions (FR-005/014-016, SC-002/003/007).
- [x] T034 Implement the shared policy, advisory receipts, fixed 900-second input deadline, Python wrapper/calibration metadata and applicable workflow watchdogs without changing native/transfer/preparation/release limits (FR-003/004/007/014-016).
- [x] T035 Run required root, affected Python and site checks; inspect final diff, encoding and policy consistency; prepare the frozen small policy PR for publication (SC-007).
- [x] T036 Publish and verify complete PR acceptance, squash merge under owner authorization, clean up the branch and retain exact-main terminal acceptance externally; complete T016 only from that evidence (FR-005/007, SC-004/007).
- [ ] T037 Run the existing isolated cold campaign once and its planned full warm comparison on the final S041 revision if still needed for #161 closure; retain revision-bound results separately from later unrelated product commits (FR-001-003/009, SC-004).

T032 -> T033 -> T034 -> T035 -> T036. T037 follows the final S041 revision and remains separate from acceptance of the policy correction and subsequent product work.
