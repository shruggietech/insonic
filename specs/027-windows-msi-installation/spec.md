# Feature Specification: Windows MSI and installation guide

**Feature Branch**: `codex/027-windows-msi-installation`

**Created**: 2026-10-13

**Status**: Specified

**Input**: S027 completes GitHub #65 and #64, with an autonomous headless application UI screenshot facility. Push and official PR publication are explicitly authorized; at most two review rounds, then owner final review/merge.

## User Scenarios & Testing

### User Story 1 - Install the selected clients (Priority: P1)

A Windows x86-64 user installs the desktop application, CLI or both through an initially unsigned installer, chooses the program directory and optional launch integrations, and reaches useful first-run setup.

**Why this priority**: Existing ZIP copying does not provide installation registration, feature selection or maintenance.

**Independent Test**: Install each selection into an isolated disposable location and use its selected client and first-run entry.

**Acceptance Scenarios**:

1. **Given** a fresh supported Windows account, **When** the user accepts the actual Apache 2.0 license and keeps defaults, **Then** both clients and their exact shared dependencies are installed and registered.
2. **Given** a Desktop-only selection, **When** installation completes, **Then** shared CLI/runtime bytes remain for the desktop while exposed CLI launch integrations are absent; CLI-only omits the GUI.
3. **Given** a spaced or non-ASCII program directory, **When** selected optional shortcuts and final Launch are used, **Then** the selected client starts under the invoking user with correct quoting and no transient useless CLI window.
4. **Given** no selected client or a missing prerequisite, **When** installation is attempted, **Then** the installer explains the correction and does not leave a unusable installation.

### User Story 2 - Maintain without losing user data (Priority: P1)

A user changes client selection, repairs, upgrades or removes the application while preserving workspaces, media, models, credentials and settings.

**Why this priority**: Installation owns program bytes and integrations, not libraries or durable user work.

**Independent Test**: Exercise maintenance and forced replacement failure with sentinel user data and compare ownership before/after.

**Acceptance Scenarios**:

1. **Given** an accepted installation, **When** feature selection changes or repair runs, **Then** shared dependencies and chosen integrations agree with the selected clients without duplicate registration.
2. **Given** a newer package, **When** replacement succeeds or fails after removal begins, **Then** selections persist on success and the prior usable installation returns on failure.
3. **Given** active clients or runtime work, **When** maintenance would replace their files, **Then** the operation reports how to close them and never silently terminates durable work.
4. **Given** user data and an unrelated ZIP installation, **When** this MSI is removed, **Then** only this installer's owned bytes and integrations are removed.

### User Story 3 - Follow accurate installation guidance (Priority: P2)

A nontechnical user follows numbered instructions and screenshots, understands unsigned download/application warnings, installer choices and optional setup, and can identify policy or dependency failures.

**Why this priority**: Different Windows checks need different actions and may offer no continuation.

**Independent Test**: Render matching online/offline help and follow each documented branch using current authoritative controls and captured authored screens.

**Acceptance Scenarios**:

1. **Given** the matching official download, **When** the user reads the guide without images, **Then** the version/architecture/checksum and complete normal install/open/remove path remain usable.
2. **Given** browser reputation, SmartScreen, UAC, Smart App Control, S mode or organization restrictions, **When** the matching branch is read, **Then** actual available controls and missing-continuation support paths are distinguished without promising universal bypass.
3. **Given** captured installer/application screens, **When** the guide renders online or offline, **Then** numbered captions, meaningful alternative text and adjacent steps match the built controls and provenance.

### User Story 4 - Capture UI without owner assistance (Priority: P2)

A maintainer generates repeatable application and authored installer screenshots without showing windows on the owner's desktop, requiring clicks or reading the owner's profile.

**Why this priority**: The owner explicitly requires autonomous headless screenshots, even when new capture capability is necessary.

**Independent Test**: Run the capture entry point unattended against isolated fixture state and inspect resulting PNGs and machine-readable provenance.

**Acceptance Scenarios**:

1. **Given** an isolated fixture and the current source build, **When** the capture command runs, **Then** real rendered application controls and installer pages are captured without foreground/focus-stealing windows or owner interaction.
2. **Given** secrets, private paths or incidental host windows, **When** capture runs, **Then** those are excluded by fixture isolation and bounded capture surfaces rather than post-hoc concealment.
3. **Given** a platform/browser/runtime/rendering mismatch or missing UI landmark, **When** capture is attempted, **Then** it fails with diagnostics and does not publish a fabricated or stale screenshot.

### Edge Cases

- No clients, absent GUI-only prerequisites, unattended install, cancellation and maintenance must not launch clients.
- Invalid/root/installation-library overlapping paths, disk exhaustion, concurrent installer transactions, active work and permission failures retain prior accepted state.
- Equal-version reinstall, upgrade, downgrade, feature changes, shared runtime ownership and unrelated integration values have explicit behavior.
- Unsigned policy blocks may offer no continuation. Secure-desktop or reputation prompts may not occur in automation; observed and source-documented evidence remain distinct.
- Browser-rendered application captures and native installer/OS captures identify their actual renderer and platform; neither masquerades as the other.
- Capture outputs must bind to exact source/version, fixture identity, viewport, locale and renderer. No owner directories, credentials or network media are capture inputs.

## Requirements

### Functional Requirements

- **FR-001**: Deliver a Windows x86-64 unsigned MSI with actual Apache 2.0 license text, initially unchecked acceptance, editable install directory and Desktop/CLI selections, both selected by default. At least one client is required.
- **FR-002**: Desktop-only keeps mandatory shared runtime bytes with truthful dependency wording and optional exposed CLI integrations; CLI-only omits GUI bytes and GUI prerequisites.
- **FR-003**: Optional desktop and Start menu integrations use governed icons and exact recorded launcher wording. The default checked final Launch starts the selected experience as the invoking user only after successful interactive fresh installation.
- **FR-004**: The CLI launcher opens a useful persistent help/setup session with robust spaced/non-ASCII quoting and a standard Windows fallback. Launch is suppressed on unattended installation, failure, cancellation, uninstall and maintenance.
- **FR-005**: Installation context and elevation behavior are explicit, with sensible defaults and validated capacity, destination and prerequisite conditions. No user data or secrets enter installer-owned program files or custom actions.
- **FR-006**: Feature-specific prerequisites have actionable official online/offline remediation; installation has declared network/restart behavior and never removes shared Microsoft runtimes.
- **FR-007**: Installed Apps metadata, feature changes, repair, uninstall, stable upgrade identity, release-version mapping and downgrade policy are coherent. Upgrade rollback restores the prior installation and accepted feature choices.
- **FR-008**: Installer ownership covers only its own program files/integrations. User data and unrelated ZIP installations survive maintenance/removal; active application/runtime work is not silently killed.
- **FR-009**: MSI receives explicit format identity, payload inventory, exact source/help/license/version receipts, checksums and release collection/retrieval/download handling while retaining qualified archive choices and strict completeness checks.
- **FR-010**: Publish numbered installation text before screenshots, with exact package selection, checksum meaning, unsigned status, normal MSI and ZIP procedures, first-run entry and removal.
- **FR-011**: Document browser reputation, SmartScreen, UAC, license acceptance, prerequisites, Smart App Control, S mode, app-source recommendations and organization policy as distinct branches. No global security-control disabling is the ordinary recipe.
- **FR-012**: Capture the application UI and authored installer screens autonomously on bounded offscreen/headless surfaces, without owner help or foreground windows. Captures use the current implementation and isolated nonsecret fixtures.
- **FR-013**: Screenshot receipts record version, exact source revision, Windows build when native, renderer/browser version, locale, viewport, signing disposition, fixture identity and image hashes. Only genuinely observed OS prompts are represented as captures; unobserved conditions use verified textual guidance.
- **FR-014**: Screenshots have numbered captions, useful alternative text and adjacent instructions, remain usable offline, and contain no private paths, secrets or incidental windows.
- **FR-015**: Exact human-visible Start menu folder, GUI/CLI shortcut, desktop shortcut and Launch-control text must be presented as a concrete reviewable set for owner approval before product shipping. Pending proposal status is retained in internal records; approval is never inferred.
- **FR-016**: Meaningful automated Windows qualification covers client selections, spaced/non-ASCII locations, prerequisites, offline behavior, scopes/users, change/repair/remove, upgrade/downgrade/cancel/rollback and active-work ownership. Capture/rendering checks and release checks retain the under-ten-minute CI budget through scoped parallelism/caching.

### Key Entities

- **Installation identity**: Stable product family, numeric release mapping, program location, selected clients/integrations and exact payload authority.
- **Maintenance transaction**: Current accepted installation, requested change, active-client/prerequisite observations and rollback disposition.
- **Launcher wording proposal**: Exact visible label set, version and explicit owner approval status.
- **Capture receipt**: Exact implementation/fixture/renderer/platform identity, bounded captured surface and image hashes.
- **Installation guide**: Version-matched normal steps, conditional warning/dependency branches and accessible captured evidence.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All three nonempty client selections install and reach their selected first-use experience; empty selection is rejected without partial ownership.
- **SC-002**: Every tested maintenance/rollback/removal scenario preserves all sentinel user data and unrelated installation integrations.
- **SC-003**: Required authored installer pages and application first-run entry are captured unattended with zero owner interaction and zero foreground console windows.
- **SC-004**: Every shipped guide image has exact-source/hash provenance, accessible text and a matching numbered instruction in both online/offline output.
- **SC-005**: Release validation rejects missing, duplicate, stale or wrong-format MSI artifacts while all existing archive variants remain qualified.
- **SC-006**: Final exact-head CI is green within ten minutes and every external review finding is answered/resolved within at most two requested rounds.

## Assumptions

- S026's shared optional setup is the first-run integration; installation does not configure user accounts, models or workspaces.
- Windows x86-64 MSI is additional to the existing four Windows/Linux archives. macOS and code signing remain outside scope.
- Per-user installation with optional exposed CLI integrations is the preferred bounded default, subject to implementation evidence; all-users support is evaluated explicitly rather than implied.
- No optional prerequisite bootstrapper or PATH integration is required unless a concrete implementation benefit justifies it; direct MSI and explicit remediation remain mandatory.
- Actual Windows OS warning availability depends on policy and reputation; their absence never authorizes fabricated screenshots or a universal continuation claim.
- Push and official PR are authorized. Merge, product tagging/release and deployment remain owner actions. Launcher label signoff is presented with the concrete final review artifacts before shipping.

## Clarifications

### Session 2026-10-13

- Q: May screenshots require owner actions? A: No. A repeatable headless/offscreen capture feature is explicitly within S027.
- Q: Does Desktop-only remove the sibling CLI runtime? A: No. Shared runtime bytes remain mandatory; exposed CLI launch integrations are independently selected.
- Q: Must all-users or PATH integration be added? A: Evaluate scope explicitly; prefer per-user defaults and omit unnecessary global integrations while preserving every required client selection.
- Q: Are launcher labels already approved? A: No. Produce concrete candidate controls/captures, record proposed wording and request signoff at final owner review before shipping.
- Q: Can missing OS warnings be synthesized as screenshots? A: No. Capture observed screens and document policy-dependent prompts through verified primary guidance, with capture provenance and limits explicit.
