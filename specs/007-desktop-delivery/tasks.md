# Tasks: S007 Desktop delivery

Input: [spec](spec.md), [plan](plan.md), [research](research.md), [data model](data-model.md), [contracts](contracts/desktop.md). Tests are required by authorized autopilot and the acceptance gates.

## Phase 1: Setup

- [x] T001 Align S007/#10/branch/directory and create specification in specs/007-desktop-delivery/spec.md.
- [x] T002 Dispatch research, clarify boundaries and write design in specs/007-desktop-delivery/plan.md and research.md.
- [x] T003 Analyze requirements/task coverage and resolve gate findings in specs/007-desktop-delivery/plan.md.

## Phase 2: Foundation

- [x] T004 [P] Add shared envelope/settings/current-cue/playback contract regressions in internal/app/desktop_ops_test.go and schemas/desktop_test.go.
- [x] T005 [P] Establish pinned static React/TypeScript build and stale-response regressions in desktop/frontend/.
- [x] T006 [P] Add installed companion discovery/hash/path tests in internal/packageenv/.

## Phase 3: User Story 1 (Library)

Goal: independent workspace/library desktop journey; compare with CLI fixtures.

- [x] T007 [US1] Add synchronized workspace selection/native chooser and stale-workspace tests in desktop/workspace.go and desktop/workspace_test.go.
- [x] T008 [US1] Implement workspace/Library/import/detail/metadata/date/relocation forms in desktop/frontend/.
- [x] T009 [US1] Exercise actual desktop library/date/relocation paths with committed audio/video in desktop/frontend/ tests and native qualification.

## Phase 4: User Story 2 (Configuration/work)

Goal: dedicated controls use current shared authority and durable work.

- [x] T010 [P] [US2] Implement settings CAS/read/write and shared CLI configuration in internal/app/desktop_ops.go, internal/contracts/desktop.go and cmd/insonic/.
- [x] T011 [P] [US2] Implement Jobs/detail/recovery and pipeline controls in desktop/frontend/.
- [x] T012 [US2] Implement speakers/aliases/current mappings/evidence and Terms/context controls in desktop/frontend/.
- [x] T013 [US2] Implement Models/credentials/tool settings and readable backend profiles in desktop/frontend/.
- [x] T014 [US2] Qualify mutation conflicts, route/no-fallback, current refresh and closing-window work persistence in desktop/ and frontend tests.

## Phase 5: User Story 3 (Playback)

Goal: actual audio/video playback and original-time seeking; reject stale/unscoped access.

- [x] T015 [P] [US3] Implement bounded current cue projections with exact-clock strings and revision/digest fences in internal/app/desktop_ops.go.
- [x] T016 [US3] Implement runtime playback registry/leases/lifecycle/current-authority checks in internal/app/desktop_ops.go and app.go.
- [x] T017 [US3] Implement native opaque tickets/ranged GET/HEAD handler and scope/current-ref regressions in desktop/playback.go and playback_test.go.
- [x] T018 [US3] Integrate native audio/video, cue/span seek and replacement invalidation in desktop/frontend/.
- [x] T019 [US3] Wire dialogs/playback/workspace lifecycle and screen qualification in cmd/insonic-desktop/main.go.

## Phase 6: User Story 4 (Native delivery)

Goal: relocated extracted packages work on each executed OS/architecture.

- [x] T020 [P] [US4] Implement installation-relative companion defaults and explicit-config precedence in internal/packageenv/ and app configuration loaders.
- [x] T021 [P] [US4] Implement three-platform layouts/install helpers/inventory/notices/corresponding-source retention in scripts/package-desktop.py and packaging tests.
- [x] T022 [US4] Add extracted-package CLI/media/Cueson/native-webview checks and CI artifacts in scripts/package-desktop.py and .github/workflows/ci.yml.
- [x] T023 [US4] Qualify keyboard/focus/theme/reduced-motion/offline-help controls in desktop/frontend/ and desktop/.

## Phase 7: Integration, convergence and publication

- [x] T024 Reconcile capability matrix, authoritative docs and dated packaging decisions in docs/v0.0.0/desktop.md, development.md, technology.md and CHANGELOG.md.
- [x] T025 Run repository/schema/Node/frontend/Go/race/vet/Python/site/native/package checks and record evidence in specs/007-desktop-delivery/verification.md.
- [x] T026 Run speckit-converge against all FRs/stories; append and complete any gap tasks in specs/007-desktop-delivery/tasks.md.
- [x] T027 Commit, automatically push and publish official PR closing #10; update Project status and attach PR.
- [x] T028 Address every received review, request at most one second round, confirm exact-head CI and hand off before owner merge.

## Dependencies and parallel execution

Setup/analysis precedes all code. Independent foundation tests/build/package discovery may run in parallel. Runtime-owned contract changes precede bridge integration; UI follows the agreed wire. Package construction follows static/native builds; extracted qualification follows package inventory. Files shared between tasks are edited sequentially by their designated owner. Research agents continue independent file-owned implementation under the plan. Each story has independent fixture acceptance; all four are required for final completion.

## Implementation strategy

Deliver workspace/library first, then current configuration/work flows, scoped evidence playback and relocated packages. Integrate after each story, preserve public status truth and finish all acceptance gates before publication. Do not claim graph/training/release features from unrelated checks.

## Phase 8: Convergence

- [x] T029 Revalidate mounted buffered playback through a scoped native ticket check, stop obsolete media and clear cue/mapping selections after external replacement; add rendered and native regressions per FR-009/FR-010 (partial, HIGH).
- [x] T030 Replace repeated package-smoke CLI polling processes with one bounded shared-runtime wait caller, and qualify timeout/terminal-state behavior per plan: hidden Windows tooling and US4/AC1 (contradicts, HIGH).

## Phase 9: Review closure

- [x] T031 Replace checkout-dependent Unix loader paths with installation-relative loading and isolate checkout libraries during extracted acceptance per FR-012/US4/AC1 (review P1, contradicts).
- [x] T032 Keep effective installed defaults separate from editable workspace overrides and provide shared revision-checked reset per FR-008/plan: installation-relative defaults (review P2, contradicts).
- [x] T033 Qualify Linux webview startup with explicit readiness/progress and include the macOS fixture encoder per SC-002/FR-014 (hosted acceptance, partial).
- [x] T034 Preserve schema-valid optional FFmpeg settings through save/reload per FR-008 (self-review, partial).
- [x] T035 Load the authoritative persisted credential backend in Settings without changing it on reopen per FR-008 (second-round review P2, contradicts).
- [x] T036 Replace unsupported native custom-scheme media streaming with process-scoped loopback ranges, validate authority and shutdown, and qualify all platform players per FR-009/SC-002 (hosted acceptance, partial).

## Phase 10: Receipt provenance correction

- [x] T037 Pin the exact-source FFmpeg reported version independently of enclosing checkout Git state, invalidate its build cache and qualify reproducible provenance per FR-012/SC-002 (receipt audit, contradicts).
