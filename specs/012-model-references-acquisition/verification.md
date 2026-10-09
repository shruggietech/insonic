# S012 Verification ledger

## Chronological workflow (2026-10-09)

Baseline: merged S011 `11ce241507b98afc66dd451495b77b80d37c8a73`, clean main and origin/main. Literal S012 maps to #34, specs/012-model-references-acquisition and codex/012-model-references-acquisition. #29 advances; #35/#15/#14 remain distinct work. Five open issues were rechecked; no new scope changes. Owner explicitly authorizes push/official PR and at most two review rounds, with final merge handoff only.

Installed hyphenated Spec Kit skills are used. No extensions.yml exists, so no hooks apply. Specify resolved the installed template stack and persisted .specify/feature.json; clarify helper resolved correct feature, five routine decisions were answered under autopilot with all coverage categories clear. Checklist helper requires plan existence; setup_plan materialized its template before checklist generation, then plan produced research/design/contracts/quickstart. Three read-only research agents examined actual catalog/models, scheduler/import and surfaces/provider boundaries.

Custom checklist: ten generated reviewer-owned items remain unchecked. Requirements review separately confirms all ten are explicitly covered by spec/design. The owner's full autonomous kickoff and autopilot decision policy authorize proceeding; no checkbox is changed to manufacture implementation evidence.

## Pre-implementation analyze gate

Coverage (14 functional requirements, 6 buildable success criteria, 29 tasks):

| Requirements | Tasks |
| --- | --- |
| FR-001/002, SC-001 | T004-T008 |
| FR-003/004, SC-002 | T007, T011-T013, T015-T018 |
| FR-005, SC-004 | T015-T017 |
| FR-006/007, SC-003 | T009-T014 |
| FR-008/009 | T010, T014-T017 |
| FR-010, SC-005 | T008, T013, T018-T022 |
| FR-011/012 | T003, T006/T007, T019/T021 |
| FR-013 | T019/T020, T023 |
| FR-014, SC-006 | T005/T009/T014/T015, T021/T022, T025-T029 |

Analysis found no constitution conflicts, uncovered requirements, ambiguity or duplicate authority. Task T024 is a gate despite its ledger listing in final phase; explicit dependencies require it before all implementation. No unmapped task. Coverage 100%; critical/high findings 0. Proceed after prerequisite helper confirms all artifacts.

## Implementation and qualification

Implemented schema-8 aliases/sources with CAS, tombstone and replay proofs; frozen schema-7 opening/snapshot migration; exact base/trained/hosted resolution; complete source catalogs and non-loading compatibility checks; frozen processing/import elections and shared acquisition dependencies. Explicit acquisition retains independently cancellable parent work. Retry resets terminal frozen dependencies atomically with the consumer after its replay check, preserving the original bundle and avoiding a cross-runtime scheduler race. Corrupt/missing bytes recover through new immutable publications; stale claims fail before file effects.

CLI and desktop share reference inspection, source discovery and alias/source operations. Selectors cover import, processing and pipelines. Jobs expose frozen identities and acquisition phases. Graph projections preserve exact targets without source endpoints or credential IDs. Authoritative models/import/pipelines/voice-models/schema/contracts/desktop/glossary/index and changelog were audited together. Hosted and trained aliases are inspected; local model selections require compatible base bundles, while existing explicit hosted pipelines retain their established execution route. Matching/training consumers remain separate.

Test-first evidence: alias/source authority, mutation replay, workspace isolation, tombstone fencing, forged snapshot rejection, compatibility/transport/role negatives, dependency starvation and cancellation/retry fixtures, CLI parsing and graph projection were added at their implementation seams. Integration exposed a stale-claim recovery defect, then a missing native receipt field; both were fixed and rerun. Review before publication also fixed multi-capability discovery and atomic parent/dependency retry replay.

Local qualification:

- Root text/schema checks pass (21 schemas, 19 contracts, 26 examples, version 0.0.0); 62 root tests pass.
- Full Go suite and vet pass. Final affected race suites pass: catalog 61.441s, application 100.900s (including large pagination), models 2.915s, exploration 1.126s and pipelines 1.141s; processing race passed separately. Final held-writer/foreign-workspace/duplicate-JSON and runtime inventory regressions also pass.
- 59 rendered frontend tests, typecheck and frontend build pass.
- Documentation website and offline export pass (27 checked documentation pages, resolved relative links/anchors).
- 20 Python tests pass. Real pinned media/Cueson/SQLite/Ladybug native checks pass without acoustic execution.
- Cross-process CLI reference/source/alias/acquisition/verification journey passes using four tiny synthetic files. Shared acquisition identity is checked across independent requests.
- Source desktop bridge and native WebView pass the model-reference journey (`ui_model_references: passed`).
- Relocated Windows package passes inventory, real media import, replacement/rosters, Cueson assembly/export, bridge and native WebView, including model references; development companion paths were isolated and restored. Local package receipt is a dirty-source qualification at the baseline revision, not a promoted release or final-commit artifact.
- Actual PostgreSQL reference CAS/snapshot and S3 missing-byte recovery tests are added; CI includes internal/models in adapter fixtures. Execution of these services and Linux/macOS packages remains pending final-head CI.

No required check downloaded real weights, loaded an acoustic model or ran inference/training. Those behaviors are unverified and outside the required-check contract. No official binary release is created.

## Spec Kit convergence

Prerequisite helper confirms this feature's spec/plan/tasks and supporting artifacts. Current source assessment covers 14 functional requirements, 6 success criteria (with CI/review delivery pending), 14 acceptance scenarios, ten plan decisions and five core constitution principles plus repository/workflow constraints. First convergence found one partial HIGH gap: broad affected race qualification failed the existing 10,000-item work-results pagination read with unavailable catalog authority. Single-record reads retain SQLite's immediate transaction through large JSON validation/decoding. T030 is appended as Phase 8 to investigate and fix proportional read contention while preserving integrity. Missing 0, partial 1, contradicts 0, unrequested 0; critical 0, high 1, medium/low 0. Publication/review tasks remain ordinary delivery work. No extension hooks exist. Final convergence remains pending T030.

Second convergence found T031 (Phase 9): the actual bounded model inventory omitted capabilities/compatibility expected by the desktop selector. That partial HIGH FR-005/FR-010 gap is implemented with capabilities and per-operation default-adapter compatibility; runtime and rendered regressions pass. T030 now scans a coherent single row without an immediate read transaction and validates/decodes afterward; transactional writes and multi-query snapshots retain their authority. Held-writer reads expose only committed state, foreign workspace reads fail and duplicate JSON fails. The unchanged large pagination integrity test and final broad affected race suites pass.

Final read-only convergence checks the same 34 requirement/criterion/acceptance items, ten decisions and governing constraints. No remaining application gap: missing/partial/contradicts/unrequested 0, all severity counts 0. Implementation satisfies spec/plan/tasks; no further convergence section is appended. Final source desktop/WebView qualification passes after inventory/read fixes. PostgreSQL/S3/ArcadeDB, supported platform packages and review satisfaction remain the explicit delivery gate in T021/T028; no release/acoustic claim follows from convergence.

## Publication and review ledger

No PR yet. Round 1 begins on official PR publication; at most one second explicit request is permitted. No merge/release authorization.
