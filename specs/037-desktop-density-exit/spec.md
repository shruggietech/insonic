# Feature Specification: S037 Desktop density and standard Exit

**Feature Branch**: `codex/037-desktop-density-exit`

**Created**: 2026-10-17

**Status**: Specified

**Input**: Owner-approved autopilot for #112 and #113, including push, PR creation and at most two review rounds; stop before owner final review and merge.

## User Scenarios & Testing

### User Story 1 - Work at desktop density (Priority: P1)

A mouse/keyboard user can see and operate Setup, Library/import/processing, Jobs, Pipelines, Models and Settings without oversized or stretched controls wasting the window. Shared navigation, ordinary actions, fields, selection controls and tables use a consistent compact presentation.

**Why this priority**: #112 is the remaining high-priority desktop usability defect.

**Independent Test**: Render the actual core workflows at default and minimum window sizes, including scaled/enlarged text; measure shared control geometry and exercise keyboard focus and clickable labels.

**Acceptance Scenarios**:

1. Given the default 1200 by 800 desktop window, when a user opens each core workflow, then ordinary action buttons use content-appropriate widths and all shared controls use the documented compact presentation.
2. Given checkbox or radio options in narrow and wide containers, when their labels wrap or the container grows, then visible glyphs remain square and proportional, labels remain clickable, and controls retain visible focus and usable targets or spacing.
3. Given the minimum 720 by 600 window, display scaling of 100%, 125%, 150% or 200%, or text enlarged to 200%, when a user navigates the core workflows, then labels and commands remain reachable without clipping or unintended page-wide horizontal overflow; scrollable data tables may retain contained horizontal scrolling.
4. Given supported Windows and Linux desktops, when the actual application opens, then its compact controls retain readable hierarchy, keyboard access and the release-matched help behavior.

### User Story 2 - Exit through the standard application lifecycle (Priority: P2)

A user can choose native File > Exit before or after opening a workspace and can access the command through normal menu keyboard navigation. Exiting releases desktop playback resources while existing durable runtime work continues according to the close contract.

**Why this priority**: #113 is a small related desktop-shell omission that shares native qualification with US1.

**Independent Test**: Exercise native menu composition and activation with no workspace and an open workspace, including active playback and durable work; compare its lifecycle with normal window close.

**Acceptance Scenarios**:

1. Given an unopened workspace, when the user opens the native application menu, then File > Exit is available and the application exits through its ordinary close lifecycle.
2. Given a Windows desktop, when the user uses the File and Exit mnemonics or normal menu navigation, then Exit works; Alt+F4 still closes the window and ordinary text-entry shortcuts remain unaffected.
3. Given playback resources and running durable work in an open workspace, when the user selects Exit, then desktop playback handles close, the desktop terminates normally, and the runtime/job is neither killed nor silently cancelled.
4. Given a supported Linux desktop host, when a user selects its native File > Exit command, then the same orderly shutdown behavior occurs.

### Edge Cases

- Long labels, validation/error messages, disabled and busy controls, empty tables, wrapping actions, minimum window size and enlarged text.
- Checkbox/radio checked, unchecked, disabled and keyboard-focus states; current selection and error cues remain understandable without colour alone.
- Exit before startup finishes, without a workspace, with playback active, or while durable work runs; repeated requests follow ordinary lifecycle semantics.
- A delivered component adapter may lack a compact control API even when the component recipe declares density. Resolve that gap with an allowed consumer adaptation or a formally synchronized upstream kit; never edit governed bytes or silently reduce scope.

## Requirements

### Functional Requirements

- **FR-001**: Apply a documented compact desktop presentation to shared navigation, action buttons, fields, checkbox/radio controls and table controls across Setup, Library/import/processing, Jobs, Pipelines, Models and Settings.
- **FR-002**: Selection glyphs MUST remain proportional squares independent of parent width, with clickable associated labels, visible keyboard focus, understandable states and usable pointer targets or spacing.
- **FR-003**: Ordinary actions MUST use content-appropriate widths; larger/full-width actions require an intentional task-specific layout. Primary, secondary and navigation actions remain distinguishable.
- **FR-004**: Core commands and labels MUST remain reachable at default/minimum windows, declared display scales and 200% text size, without clipped controls or accidental page-wide overflow. Tables may scroll within their own region.
- **FR-005**: Preserve governed brand tokens, keyboard accessibility and supported Windows/Linux behavior. Component gaps MUST be resolved consistently with the kit contract without hand-editing governed files.
- **FR-006**: Expose native File > Exit before and after workspace selection on Windows and supported Linux hosts, with standard Windows mnemonics/menu navigation and preserved Alt+F4/text entry.
- **FR-007**: Exit MUST use the ordinary application shutdown lifecycle, release desktop playback resources and preserve shared-runtime ownership and durable jobs; no force-kill or implicit job cancellation.
- **FR-008**: Add actual rendered geometry regressions, representative screenshots and focused menu/lifecycle checks; include bounded native activation coverage. Distinguish portable rendering proofs, native platform proofs and unexecuted field behavior.
- **FR-009**: Update user help with the actual compact presentation, menu wording and shortcuts. Keep planning/evidence internal and public documentation self-contained.

### Key Entities

- **Desktop density**: one compact presentation using the existing brand vocabulary; no new user preference or third density is required.
- **Exit request**: a native application-menu action routed to the same lifecycle as window close.
- **Qualification evidence**: source-bound geometry, keyboard/lifecycle observations and representative images with explicit platform and scenario identity.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All six core workflow groups use compact shared controls; selected checkbox/radio glyph width equals height within one rendered pixel and remains stable across narrow/wide containers.
- **SC-002**: Representative ordinary actions remain content-sized at both window sizes; every declared layout/scaling/text scenario retains reachable labels and commands and contains any table overflow.
- **SC-003**: Keyboard focus and associated-label activation pass for representative controls in each core workflow group on the supported rendering paths.
- **SC-004**: Native Exit is available without and with a workspace; observed Windows menu keyboard and Linux native activation use ordinary shutdown and preserve the running durable job while releasing desktop playback ownership.
- **SC-005**: Required automated checks pass, screenshots match their recorded source and scenario, CI remains within 660 seconds, review findings from at most two rounds are resolved, and the published PR stops unmerged for owner review.

## Assumptions

- Scope is #112 and #113 only; CLI backlog, installer recovery redesign, new product features and release publication remain outside this slice.
- Existing brand vocabulary and supported host contracts govern sizing. A component-capability fix must preserve token authority and shipped kit identity.
- No new confirmation dialog or mandatory owner-attended pre-release gate is introduced. Existing close and durable-work contracts determine Exit behavior.
- The owner has authorized publication and bounded review handling; the skill's generic pre-push pause is already satisfied by that authorization.

## Clarifications

### Session 2026-10-17

- Density is a shared desktop default, not a new preference or a restyling of the documentation site.
- Native File > Exit uses the existing close path; durable runtime work keeps its existing ownership and cancellation rules.
- The verification matrix separates renderer scaling/text-size checks from actual native platform activation and reports precisely which ran.
