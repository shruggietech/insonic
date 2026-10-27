# Tasks: Original release revalidation

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/revalidation.md.

## Phase 1: Setup

- [x] T001 Bind S056/#185, branch and feature selector; inspect authority and immutable original release identities (FR-001, FR-007).

## Phase 2: Foundation

- [x] T002 Add a compressed authentic original manifest fixture with public provenance in tests/fixtures/release-history/ (FR-001, SC-001).
- [x] T003 Add failing profile/source/receipt authority regressions in tests/release_history_test.py before implementation (FR-001, FR-002, SC-002).
- [x] T004 Implement immutable original profile and complete selected-source verification in scripts/release_history.py (FR-001, FR-002, FR-007).

## Phase 3: US1 - Original publication verification

**Independent check**: The original manifest and complete source elect the profile; all nine original assets validate with maintained code.

- [x] T005 [US1] Add failing candidate/profile propagation and legacy qualification acceptance cases in tests/release_history_test.py (FR-003, FR-005).
- [x] T006 [US1] Propagate resolved history authority through scripts/release.py and original MSI verification in scripts/windows_msi.py (FR-003 through FR-005).
- [x] T007 [US1] Verify all original published assets against complete selected source and retain compact identities/results outside workflow evidence (FR-007, SC-001).

## Phase 4: US2 - Reject altered history

**Independent check**: Altered source, receipts, first-use, signing, inventories, archive and MSI bytes fail despite recomputed local checksums.

- [x] T008 [US2] Add and run failing-first-use, source, manifest, inventory, signing, archive and MSI-table mutation regressions in tests/release_history_test.py, retaining existing tests/release_test.py boundaries (FR-001 through FR-005, SC-002).
- [x] T009 [US2] Complete original receipt/installer anchoring and before/after selected-source proof in scripts/release_history.py and scripts/release.py (FR-002 through FR-005).

## Phase 5: US3 - Preserve modern requirements

**Independent check**: Current or unknown receipts without modern terminal/setup/lifecycle evidence fail; existing v1.0.0 handling passes.

- [x] T010 [US3] Add unknown-profile evidence omission regressions in tests/release_history_test.py and run existing modern omission/lifecycle regressions in tests/release_test.py and tests/windows_msi_test.py (FR-006, SC-003).
- [x] T011 [US3] Verify modern default paths stay strict and existing historical v1.0.0 tests remain valid (FR-006, SC-003).

## Phase 6: Verification and Delivery

- [x] T012 Update docs/v1.0.2/releases.md and CHANGELOG.md with the authoritative historical verification contract and dated maintainer-tooling decision (FR-008).
- [x] T013 Run focused/full Python suites, npm run check, npm test and affected site tests/build; analyze/converge all requirements and retain compact verification facts in specs/056-historical-release-revalidation/verification.md (FR-001 through FR-008, SC-001 through SC-004).
- [ ] T014 Create S056 milestone/Project linkage, commit, automatically push and open the official PR linked to #185; attach it to this chat (SC-004).
- [ ] T015 Monitor exact-head CI, audit/retain terminal receipts before any full rerun, resolve all findings through at most two Codex rounds, and stop before merging for owner review (SC-004).

## Dependencies and Execution Strategy

Setup -> foundation -> US1 -> US2 -> US3 -> verification/delivery. Tests precede related code. Independent read-only source/fixture checks can run together; edits sharing release.py remain sequential. Deliver one coherent PR. No agent delegation is required. The immutable historical profile is the MVP, completed together with negative and modern compatibility boundaries before publication.
