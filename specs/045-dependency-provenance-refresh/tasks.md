# Tasks: Dependency refresh and provenance qualification

**Input**: [spec.md](spec.md), [plan.md](plan.md), [research.md](research.md), [data-model.md](data-model.md), [quickstart.md](quickstart.md).

## Phase 1: Setup

- [x] T001 Create S045 specification, clarification and requirements records in specs/045-dependency-provenance-refresh/spec.md and checklists/.
- [x] T002 Link #168, milestone 29 and Project 7 in specs/045-dependency-provenance-refresh/plan.md.

## Phase 2: Foundation

- [x] T003 Complete the blocking cross-artifact/constitution analysis for specs/045-dependency-provenance-refresh/spec.md, plan.md and tasks.md; preserve exact slice identity.

## Phase 3: User Story 1 - Adopt intended updates (P1)

**Independent test**: The selected direct versions account for all three PRs, capture authority matches the exact root pin, and the resolved graph contains no unintended downgrade.

- [x] T004 [US1] Add pin-coherence and mismatched current-capture receipt regressions in tests/capture-application-ui.test.mjs.
- [x] T005 [US1] Update playwright-core in package.json and package-lock.json; demonstrate the capture authority regression before fixing it.
- [x] T006 [US1] Derive capture Playwright authority in scripts/capture-application-ui.mjs and pass the focused regressions.
- [x] T007 [P] [US1] Refresh only elected site versions and required closure in site/package.json and site/package-lock.json.
- [x] T008 [US1] Recompute the elected runtime upgrades and required transitive resolution in go.mod and go.sum using Go 1.27.1.

## Phase 4: User Story 2 - Preserve provenance (P1)

**Independent test**: Both maintained modules accept exact upstream identities and reject missing or mismatching sums; local source trees/manifests remain unchanged.

- [x] T009 [P] [US2] Add both-module missing/changed/restored checksum regressions in tests/go_module_notices_test.py and tests/webview-source.test.mjs.
- [x] T010 [US2] Restore verified pristine upstream archive/module sums in go.sum after tidy and prove scripts/go_module_notices.py and scripts/webview-source.mjs still reject pruned authority.
- [x] T011 [US2] Document the actual repeatable refresh/restore/verify procedure in docs/v1.0.1/development.md and record maintenance changes in CHANGELOG.md.

## Phase 5: User Story 3 - Qualify and publish (P2)

**Independent test**: Complete exact-head CI, terminal timing and received review threads are accepted; final PR stays unmerged.

- [x] T012 [US3] Run root npm check/test, Python unittest and affected pinned Go tests/vet under plan.md verification requirements.
- [x] T013 [P] [US3] Run site check/test/build/public-offline export for site/package.json updates.
- [x] T014 [US3] Commit S045 changes, automatically push codex/045-dependency-provenance-refresh and publish/attach its official PR linked to #168.
- [x] T015 [US3] Monitor and repair complete exact-head .github/workflows/ci.yml execution; verify all native/package/backend/CLI receipts and terminal phase timing.
- [x] T016 [US3] Resolve every received PR review finding, complete at most two Codex rounds, and requalify the final head.
- [x] T017 [US3] Reconcile #168 and supersede #157-#159 only after the replacement PR represents all intended changes and passes qualification.

## Phase 6: Polish and handoff

- [x] T018 Converge specs/045-dependency-provenance-refresh/ against implementation, check UTF-8/LF/mojibake and remove owned temporary workflow files.
- [x] T019 Verify final Git/CI/review/tracking state and hand off the unmerged PR with evidence and remaining limits in specs/045-dependency-provenance-refresh/quickstart.md.

## Dependencies and execution

Setup -> blocking analysis -> US1 -> US2 -> local verification/publication -> full CI/reviews -> superseded PR reconciliation -> convergence/handoff. US1 is the first independently inspectable increment; it is not a publishable candidate until US2 integrity and US3 qualification pass. Root/site lock refreshes affect separate contexts. Both checksum regression files can be developed independently; all Go graph/sum mutations stay sequential. Site checks and root/Go checks may run in parallel with separate outputs. No implementation task edits reviewer checklist markers.

## Requirement coverage

FR-001: T004-T008,T017; FR-002: T008,T010,T012-T013; FR-003: T009-T010; FR-004: T009-T010; FR-005: T012-T015; FR-006: T015,T019; FR-007: T012-T013,T018; FR-008: T001-T002,T011,T017; FR-009: T014-T016,T019; FR-010: T017,T019. SC-001: T005-T010,T017; SC-002: T009-T010; SC-003: T012-T015; SC-004: T016-T019.

## Phase 7: Convergence

- [x] T020 Enforce exact archive and `/go.mod` upstream checksum entries for both maintained modules in scripts/webview-source.mjs and scripts/go_module_notices.py; cover missing, changed and conflicting entries in tests/webview-source.test.mjs and tests/go_module_notices_test.py per FR-003 and SC-002 (partial).
