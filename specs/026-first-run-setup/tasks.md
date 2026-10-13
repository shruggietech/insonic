# Tasks: First-run setup

Input: [plan.md](plan.md), [spec.md](spec.md), research, data model and runtime contracts. Tests are required by autopilot.

## Phase 1: Setup

- [x] T001 Confirm S026/#66 scope, branch and authority in specs/026-first-run-setup/spec.md.
- [x] T002 Execute clarification, research, checklist and design in specs/026-first-run-setup/plan.md.

## Phase 2: Foundation

- [x] T003 Write strict validation/restart/replay/conflict tests in internal/firstsetup/setup_test.go.
- [x] T004 Implement bounded private bootstrap and typed state in internal/firstsetup/setup.go.
- [x] T005 [P] Write catalog CAS and workspace-isolation/security tests in internal/app/setup_test.go.
- [x] T006 Implement shared state/observation/readiness orchestration in internal/app/setup.go.

## Phase 3: US1 Local defaults and first import

Independent test: fresh local defaults reach ordinary import with no credential or model download.

- [x] T007 [P] [US1] Write command/location/bootstrap tests in cmd/insonic/setup_test.go.
- [x] T008 [US1] Add explicit noninteractive setup/location/draft/provision and settings setup routes in cmd/insonic/setup.go and commands.go.
- [x] T009 [P] [US1] Add shared pre-workspace bridge and native wrapper tests in desktop/setup.go and setup_test.go.
- [x] T010 [P] [US1] Add accessible five-stage entry and normal import navigation in desktop/frontend/src/setup.tsx and App.tsx.

## Phase 4: US2 Resume and reconcile

Independent test: unknown-save replay and stale competing writes preserve accepted state after restart.

- [x] T011 [US2] Add stable request identity and lost-response readback in desktop/frontend/src/client.ts and setup.tsx.
- [x] T012 [US2] Seed/reuse normalized nonsecret backend drafts without persisting protected values in desktop/frontend/src/backend-setup.tsx.
- [x] T013 [US2] Add restart/conflict/secret/navigation/focus UI journeys in desktop/frontend/tests/setup.test.mjs.
- [x] T014 [US2] Validate portable destination proofs and deferred ongoing work in internal/app/setup_test.go.

## Phase 5: US3 Optional capability orchestration

Independent test: elected optional controls and readiness preserve ready import despite failed dependencies.

- [x] T015 [US3] Reuse Workers, Assistance and Jobs controls in desktop/frontend/src/setup.tsx.
- [x] T016 [US3] Expose setup request validation/schema and exact revision reconciliation in internal/contracts/desktop.go and schemas/v1.0.1/runtime-request.schema.json.
- [x] T017 [US3] Add nonsecret friendly credential labels and registered/configured/usable distinctions in desktop/frontend/src/setup.tsx.

## Phase 6: Polish and delivery

- [x] T018 [P] Document actual shared setup and remove stale claims in docs/v1.0.1/setup.md, desktop.md, cli.md and index.md.
- [x] T019 Update current contracts, changelog and version navigation in schemas/v1.0.1 and CHANGELOG.md.
- [x] T020 Run meaningful domain/CLI/UI checks and complete npm/product/site parity in specs/026-first-run-setup/verification.md.
- [x] T021 Converge implementation against all FR/SC requirements in specs/026-first-run-setup/convergence.md.
- [ ] T022 Commit/push/publish official PR, address at most two review rounds and verify green exact-head CI in specs/026-first-run-setup/verification.md.

## Dependencies and parallel examples

Foundation precedes story integration. T003 precedes T004; T005 precedes T006. After typed contracts, CLI/native tests T007/T009 and UI scaffolding T010 can run on separate files. US2/US3 follow the working defaults path and join at shared UI state. Documentation T018 can run separately after the contract stabilizes. Never concurrently edit the same file.

Spec Kit's parallel team strategy: one agent owns catalog/runtime contracts; another owns desktop frontend; the parent owns bootstrap/CLI/native wrapper and integration. Source-owned paths are assigned explicitly, with tests before code.

## Implementation strategy

Deliver defaults first, then restart/concurrency behavior, then optional existing capabilities. Validate each increment; completion requires all stories and the publication/review boundary.
