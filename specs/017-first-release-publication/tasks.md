# S017 source preparation tasks

## Phase 1: Specification and foundations

- [x] T001 Specify and clarify the owner-approved publication scope in `spec.md`; establish issue #46, milestone and Project tracking (FR-007).
- [x] T002 Research and define the existing release/deployment identities and sequencing in `research.md`, `data-model.md`, `plan.md` and `quickstart.md` (FR-003, FR-005, FR-006, FR-008).

## Phase 2: US1, first official release

- [x] T003 [US1] Finalize the actual date and consolidate pending changes in `CHANGELOG.md` and `docs/v1.0.0/references/CHANGELOG.md`, preserving synchronized versions (FR-001).
- [x] T004 [US1] Author original bounded release notes in `highlights.md` ending with the tagged changelog link (FR-004, FR-006, SC-002).
- [x] T005 [US1] Run repository checks and existing release/Pages tests; validate the guarded exact-source/tag/six-package publication procedure in `quickstart.md` (FR-003, FR-007, SC-001).

## Phase 3: US2, matching documentation

- [x] T006 [US2] Rescan current documentation terminology against `docs/v1.0.0/glossary.md` and correct demonstrated omissions (FR-002).
- [x] T007 [US2] Build/check ordinary and released public/offline exports and validate original-archive Pages promotion/recovery in `quickstart.md` (FR-005, SC-003).

## Phase 4: Convergence and launch handoff

- [x] T008 Assess prepared source against `spec.md`, `plan.md` and `tasks.md`; preserve concise planning and external temporary-file cleanup instructions in `quickstart.md` (FR-007, FR-008, SC-004).

Execute phases in order; T006 can be assessed alongside T003, but site exports sharing an output directory run sequentially. Existing interfaces and tests are reused; no implementation-mirroring tests are added for prose edits. Source preparation does not establish actual publication: execute the PR/qualification/tag/publication/deployment procedure and close issue #46 only after both live outcomes are verified. The issue holds launch results rather than another repository evidence archive.

## Phase 5: Convergence

- [x] T009 Paginate the exact-source check inventory in `scripts/release.py` and exercise later-page required/red/pending checks plus incomplete/changing/duplicate inventories in `tests/release_test.py` per FR-003 (partial). Preserve verified current-suite exclusion and every external check gate.
