# Feature Specification: Installer progress and offline documentation

**Feature Branch**: codex/036-installer-offline-help
**Created**: 2026-10-17
**Status**: Clarified
**Input**: S036 autopilot for #109, #110 and #111; publish PR, up to two review rounds, stop before owner merge.

## User Scenarios & Testing

### User Story 1 - Understand installation work and outcomes (Priority: P1)

The operator sees real installation/maintenance progress, can cancel where supported, and receives a result matching the actual operation.

**Why this priority**: #109 hides lengthy work and uses misleading completion controls.
**Independent Test**: Drive disposable native transactions and compare progress, terminal controls, installed state and native results.

**Acceptance Scenarios**:

1. Given fresh install, upgrade, change, repair or removal, when work executes, then real engine progress and concise current phase remain visible without invented timing.
2. Given cancellable work, when cancelled, then rollback is communicated and native results distinguish cancellation, failure and success.
3. Given a terminal result, when shown, then operation-specific text and close/finish replace misleading Next/Cancel controls.
4. Given configuration navigation or prerequisite correction, when moving backward/forward, then selections, license/defaults, focus and keyboard behavior remain usable.

### User Story 2 - Open usable offline help (Priority: P1)

The operator activates one compact help control and reads installed-version documentation offline, including before selecting a library.

**Why this priority**: #110 currently opens an unresolved private-origin popup.
**Independent Test**: Activate help in packaged Windows/Linux applications under spaces/Unicode paths and inspect version, local assets and navigation offline.

**Acceptance Scenarios**:

1. Given the packaged application, when activated by keyboard or pointer, then supported opening reaches existing installed-version help without an unhandled private-origin popup.
2. Given missing or unopenable help, when activated, then a useful correction is displayed.
3. Given the header at supported sizes, when rendered, then help is one compact, named, focusable control without combined link/button styling.

### User Story 3 - Choose documentation independently (Priority: P2)

After successful interactive fresh installation, the operator can open documentation independently of launching the selected client.

**Why this priority**: #111 reuses repaired completion states and help opening.
**Independent Test**: Exercise either, both or neither choice for Desktop-only, CLI-only and combined installations.

**Acceptance Scenarios**:

1. Given successful interactive fresh installation, when finished, then client launch retains its checked default and Open documentation is independently selectable, initially unchecked.
2. Given elected documentation, when Finish is activated, then actual installed HTML opens through the invoking user's configured Windows association without elevation.
3. Given unattended installation, cancellation, failure, removal, upgrade, change or repair, when finished, then fresh-install launch/documentation choices are neither offered nor executed.
4. Given committed installation and failed help opening, when reported, then installation remains successful with an actionable correction.

### Edge Cases

- Missing prerequisites, occupied destinations, shortcut collisions and initially unchecked license acceptance.
- Cancel before execution versus during work; rollback and controlled post-copy failure.
- All independent completion choices and installed client selections.
- Relocated paths with spaces, Unicode and shell-sensitive characters.
- Missing help, unsupported handler, local links/assets and installed-version identity.
- Existing libraries, credentials, durable runtime work and unrelated files remain outside installer ownership.

## Requirements

### Functional Requirements

- **FR-001**: Display real engine-driven progress and phases for install, upgrade, change, repair, removal and rollback (#109).
- **FR-002**: Preserve supported cancellation and distinguish actual success/cancel/failure with correct terminal controls (#109).
- **FR-003**: Preserve navigation, selections, unchecked license, approved launch defaults, focus/keyboard access and supported scaling (#109).
- **FR-004**: Expose one compact offline-help affordance that works before workspace selection (#110).
- **FR-005**: Open installed-version help offline on Windows/Linux with local assets/navigation and useful failures; never expose private application origins externally (#110).
- **FR-006**: Successful interactive fresh installation offers independent unchecked documentation and existing checked client-launch choices, excluding other outcomes/modes (#111).
- **FR-007**: Honor invoking-user non-elevated Windows associations for every client selection and relocated paths; opening failures must not fail committed installation (#111).
- **FR-008**: Qualify actual packaged/native activation and full-UI transactions; retain silent lifecycle, inventory, ownership and rollback checks, and refresh truthful guidance/screenshots (#109..#111).
- **FR-009**: Preserve the 660-second CI budget, hidden local console launches, UTF-8/LF integrity, exact issue scope and owner merge boundary.

### Key Entities

- Installer operation/result: install, upgrade, change, repair, remove; success, cancel, failure; engine phase/progress.
- Completion election: independent client/documentation choices, successful interactive fresh installation only.
- Installed help identity: actual package directory, version, entry and local resource tree.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Native transaction journeys observe real progress and correct terminal controls/results throughout the elected matrix.
- **SC-002**: Packaged Windows/Linux help activation renders matching-version content/resources/navigation while external networking is unavailable.
- **SC-003**: All client selections support either/both/neither completion action; excluded modes execute neither.
- **SC-004**: Existing lifecycle, rollback, active-work, data-ownership and launch-user checks pass.
- **SC-005**: Required local checks and exact-head hosted CI pass within the existing budget; at most two review rounds precede an unmerged owner handoff.

## Assumptions and Clarifications

- Exact scope is #109/#110/#111; desktop-wide density #112 and File/Exit #113 remain separate.
- Documentation defaults unchecked. Upgrade/change/repair/removal do not offer fresh-install post-actions.
- Prefer a shared supported native help opener; an internal GUI help view is allowed if needed.
- Interactive transactions/capture run only on acknowledged disposable hosted runners; no attended owner gate.
- No release, deployment, merge or memory update is authorized.
