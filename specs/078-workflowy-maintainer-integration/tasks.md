# S078 tasks

## Phase 1 - Specification and design

- [x] T001 Define and clarify the approved scope in `spec.md` and
  `checklists/`. (FR-001–FR-006)
- [x] T002 Record pinned source, privacy boundaries, acceptance and tracking
  in `plan.md`, `research.md`, `data-model.md` and `quickstart.md`.
  (FR-001–FR-006)
- [x] T003 Run the blocking specification analysis before changing the
  maintained files. (FR-001–FR-006, SC-001–SC-004)

## Phase 2 - Find the project notebook

**Independent test** The bundled files and opt-in match upstream, and the
connected lookup returns the established insonic branch.

- [x] T004 [US1] Vendor the three pinned files under
  `.agents/skills/workflowy/` and retain the upstream attribution in
  `NOTICE`. (FR-001, SC-001)
- [x] T005 [US1] Replace the previous section in `AGENTS.md` with the
  extracted opt-in and check it against the original outer text. (FR-002, SC-001)
- [x] T006 [US1] Verify the three upstream blobs, the two relative guide links,
  and the scoped Desktop MCP readback. (FR-001–FR-003, SC-001/SC-002)

## Phase 3 - Continue without the notebook

**Independent test** The fallback is written and the diff contains only
maintainer workflow material and slice records.

- [x] T007 [US2] Check the written fallback and inspect all intended paths for
  the approved privacy and product boundaries. (FR-003–FR-005, SC-002/SC-003)
- [x] T008 [US2] Add the maintainer change to `CHANGELOG.md` and run the root
  checks and both site exports. (FR-005/FR-006, SC-004)
- [x] T009 [US2] Converge against the specification, record results and limits
  in `verification.md`, and check encoding and diff integrity before the local
  commit and handoff. (FR-004–FR-006, SC-003/SC-004)

## Delivery after authorization

- [ ] T010 Publish the tracking issue, associate its milestone and Project, and
  link the slice from the PR. Check exact-head hosted CI before merge.
  (FR-006)

T001–T003 precede implementation. T004 and T005 precede T006. T007 and T008
precede convergence and commit. T010 is external delivery and follows the
local pre-push handoff.
