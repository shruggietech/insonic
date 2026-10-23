# Tasks: Real CPU processing and speaker matching

**Input**: spec.md, plan.md, research.md, data-model.md, contracts/acceptance.md and quickstart.md. Tests precede implementation.

## Phase 1: Setup

- [x] T001 Establish S046 branch/selector, author quality checklist and issue #170/milestone 30 under specs/046-real-cpu-processing/ (FR-012).

## Phase 2: Foundations

- [x] T002 Acquire the hash-pinned official archive, freeze deterministic clip selection and retain licensed original/reference metadata in tests/fixtures/acoustic/; remove owned archive scratch (FR-003, FR-008).
- [x] T003 Add failing manifest/hash/role separation and derived-sample mapping regressions in tests/acoustic_reference_test.py (FR-003, FR-009).
- [x] T004 Implement strict corpus validation/derivation in scripts/acoustic_reference.py and bind its fixture inventory through tests/fixtures/cli-corpus.json (FR-003, FR-009).

## Phase 3: US1 - Prepare and process real audio

**Independent test**: Selected package, managed setup and model acquisition complete through ordinary CLI operations; real processing publishes current cue/audio authority.

- [x] T005 [US1] Add failing package-only managed election, legacy argument, no-CI and model/setup sequencing regressions in tests/journey_cpu_test.py (FR-001, FR-002, FR-010).
- [x] T006 [US1] Add reference-backed selection and sequential returned-plan managed/model preparation in scripts/qualify-journey.py and scripts/journey_cpu.py (FR-001, FR-002, FR-004).
- [x] T007 [US1] Add failing temporary-profile, first-failure, atomic receipt and descendant cleanup regressions in tests/journey_cpu_test.py, retaining the scripts/test-processing-worker.py suite (FR-008, FR-009).
- [x] T008 [US1] Reuse scripts/process_tree.py ownership, isolate temporary/profile paths and preserve first failures with atomic evidence in scripts/qualify-processing.py and scripts/qualify-journey.py (FR-008, FR-009, FR-011).

## Phase 4: US2 - Measure transcript and speaker outcomes

**Independent test**: Frozen references yield exact deterministic error counts, and real enrollment/matching returns compared decisions for every designated held-out case.

- [x] T009 [US2] Add failing normalization/edit-component, speaker-state accounting, mixed-window and receipt completeness regressions in tests/acoustic_reference_test.py and tests/journey_cpu_test.py (FR-005, FR-006, FR-009).
- [x] T010 [US2] Implement actual corpus processing, two independent reader enrollments, disjoint held-out/unseen/sequence matching and exact profile fetch in scripts/journey_cpu.py (FR-003, FR-004, FR-006).
- [x] T011 [US2] Implement per-case word/reader comparisons, frozen settings, complete stage/case identity validation and compact versioned receipts in scripts/acoustic_reference.py and scripts/journey_cpu.py (FR-005, FR-006, FR-009).

## Phase 5: US3 - Preserve query/playback/model results through restore

**Independent test**: Real generated results remain source-linked and exact before/after self-contained restore with the original library unavailable.

- [x] T012 [US3] Add failing timeline/playback source-position and restored authority/fetched-artifact comparison regressions in tests/journey_cpu_test.py (FR-007).
- [x] T013 [US3] Assert query/timeline/playback authority and release handles through current CLI operations in scripts/journey_cpu.py (FR-007, FR-008).
- [x] T014 [US3] Reuse source withdrawal/restore, compare fetched profile bytes and portable authority, and record destination worker prerequisites in scripts/journey_cpu.py (FR-007, FR-009).

## Phase 6: Validation and publication

- [x] T015 Document elected current-package real acceptance and statistical limits in docs/v1.0.1/development.md and CHANGELOG.md (FR-012).
- [x] T016 Run npm check/test, full Python maintainer suite and affected Go/site checks; record only compact results in specs/046-real-cpu-processing/quickstart.md (FR-010, FR-011).
- [ ] T017 Commit implementation and build/verify a clean current complete Windows CLI archive with scripts/package-desktop.py (FR-001).
- [ ] T018 Execute the elected real managed/package journey, repair demonstrated failures with regressions, retain compact canonical evidence in specs/046-real-cpu-processing/evidence/ or the official PR, and remove owned scratch (FR-001 through FR-009, FR-011).
- [x] T019 Converge code against specs/046-real-cpu-processing/spec.md and plan.md, resolving every buildable gap (FR-001 through FR-012).
- [x] T020 Commit final artifacts, automatically push codex/046-real-cpu-processing and open/attach the official PR linked to #170 (FR-012).
- [ ] T021 Monitor exact-final-head CI to terminal acceptance, report preparation/qualification/combined timing and resolve every finding within at most two Codex rounds in the official PR (FR-010, FR-012).
- [ ] T022 Reconcile #170/project/milestone evidence, retain #147/#154 limits, remove owned scratch and hand off the clean unmerged PR (FR-008, FR-009, FR-012).

## Dependencies and execution

Setup → Foundations → US1 → US2 → US3 → Validation/publication. Pure reference/scoring tests and orchestration tests can be developed in separate files after the fixture contract is frozen; shared journey mutations stay sequential. Test groups are inference-free. US1 preparation can be validated before measurements; US2 metrics are independently testable against frozen values; US3 comparison helpers are independently testable without engines. The delivered slice includes all three stories, not only the US1 MVP.

No additional agent implementation is delegated. Phase 0 research agents were explicitly required by the installed planning skill and remained read-only. Source/package/receipt revision differences must be stated truthfully; metadata-only closeout commits do not retroactively change a tested archive identity.

## Phase 7: Demonstrated integration repair (2026-10-23)

- [x] T023 Regress and repair recoverable package-smoke wait handling in tests/package_wait_test.py and scripts/package-desktop.py, retaining exact work identity, original 180-second operation bound, 30-second wait requests and unchanged CI deadlines (FR-008, FR-009, FR-011).
- [x] T024 Regress and repair cold-bootstrap qualification timing in tests/cli_invocation_timing_test.py and scripts/cli_contract_qualification.py, recording owned launch/process wait/total independently while retaining the two-second classification limit and full hosted CI accounting (FR-009, FR-011).
- [x] T025 Regress and enforce clean exact harness identity before execution and source stability at completion; attribute final runtime-release failures to cleanup, select only disposable session credentials and include temporary removal in qualification timing in scripts/qualify-journey.py, scripts/journey_cpu.py and tests/journey_cpu_test.py (FR-001, FR-008, FR-009).
- [x] T026 Regress and repair the observed Community-1 zero-source-overlap padding rejection in scripts/processing-worker.py and scripts/test-processing-worker.py; retain numeric diagnostics/boundary policy and compare exact restored diagnostics/provenance in scripts/journey_cpu.py, scripts/qualify-journey.py and tests/journey_cpu_test.py (FR-004, FR-007, FR-009, FR-011).

## Phase 8: External review repair (2026-10-23)

- [x] T027 Pin real election to the committed frozen manifest independently of caller-supplied hashes; reject self-consistent substituted references before package execution in scripts/acoustic_reference.py, scripts/journey_cpu.py and tests/journey_cpu_test.py (FR-003, FR-006, FR-009).
- [x] T028 Replace assertion-based archive admission with explicit validation and a single verified snapshot used by both parsing passes in tests/fixtures/acoustic/admit.py; regress optimized Python and source-path replacement in tests/acoustic_admission_test.py (FR-003, FR-009).
- [x] T029 Own and verify Linux session members across nested worker process groups in scripts/process_tree.py, with bounded termination and an actual Linux nested-group/unrelated-session regression in tests/process_tree_test.py (FR-008, FR-011).
