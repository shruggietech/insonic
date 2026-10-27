# Tasks: Bounded documentation publication

**Input**: spec.md, plan.md, research.md, data-model.md and contracts/publication.md.

## Phase 1: Setup

- [x] T001 Bind S054, #189, feature selector and branch; inspect authority and maintained implementation.

## Phase 2: Foundation

- [x] T002 Extend isolated hosting fixtures in tests/pages_test.py to retain staged objects and request sizes.

## Phase 3: US1 - Complete bounded publication

**Goal/independent test**: A 1,110-file/54 MB export stages completely with bounded requests and original identities.

- [x] T003 [US1] Add failing representative-export, large-text, Unicode/binary and unchanged/deleted-content tests in tests/pages_test.py (FR-001 through FR-003, SC-001/SC-003).
- [x] T004 [US1] Implement bounded incremental trees and verified standalone blobs in scripts/pages.py (FR-001/FR-002).
- [x] T005 [US1] Verify complete staged inventory before publication and compare a staged mixed-content tree to real local Git in tests/pages_test.py (FR-003).

## Phase 4: US2 - Failure, concurrency and deadlines

**Goal/independent test**: Failed staging/inventory never advances the branch; original authority and shared time remain effective.

- [x] T006 [US2] Add failure, malformed inventory/blob, competing ref, uncertain response and decreasing-timeout tests in tests/pages_test.py (FR-003 through FR-006, SC-002/SC-003).
- [x] T007 [US2] Preserve publication fences and thread the remaining enclosing allowance through staging, updates and build waiting in scripts/pages.py and scripts/release.py (FR-004 through FR-006).

## Phase 5: US3 - Useful credential-free errors

**Goal/independent test**: JSON and asset-upload errors identify the operation and useful bounded server reasons without injected secrets.

- [x] T008 [US3] Add failing structured/malformed/oversized/sensitive response and timeout tests in tests/release_test.py (FR-007, SC-004).
- [x] T009 [US3] Implement bounded HTTP diagnostics in scripts/release.py for JSON calls and asset uploads (FR-007).

## Phase 6: Verification and Delivery

- [x] T010 Update current maintainer publication contract and dated unreleased decision in docs/v1.0.2/development.md and CHANGELOG.md (FR-008).
- [x] T011 Run focused/full maintained Python suites, npm run check, npm test and affected site tests/build, then analyze/converge against all requirements (FR-001 through FR-008, SC-001 through SC-004).
- [ ] T012 Commit and automatically push/open official PR linked to #189 and S054 milestone/Project; attach the PR to this chat.
- [ ] T013 Monitor exact-head CI, retain compact audited evidence before any full rerun, resolve every comment through at most two Codex rounds, then stop before merge.

## Dependencies and Execution Strategy

Setup -> foundation -> US1 -> US2 -> US3 -> verification/delivery. Tests precede corresponding implementation. A representative-export test and Git oracle can run independently from diagnostic fixtures; source edits in scripts/pages.py and scripts/release.py remain sequential. Deliver the whole slice through one PR. Root and site verification may run concurrently once source is stable. No agent delegation is required.
