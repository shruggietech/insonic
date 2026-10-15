# Feature Specification: Landing-page download clarity

**Feature Branch**: `codex/031-landing-download-clarity`

**Created**: 2026-10-15

**Status**: Specified

**Input**: S031, issue #101. Make landing-page package choices and all link purposes clear, separate crowded controls, and verify the whole page autonomously.

## User Scenarios & Testing

### User Story 1 - Choose the right package (Priority: P1)

A Windows or Linux x86-64 visitor understands installation versus portable use and chooses Desktop + CLI or CLI-only without guessing what a file contains.

**Why this priority**: Downloads are the page's primary action and the current controls run together.

**Independent Test**: Read each package label by itself, compare it with the release manifest, and activate each distinct link without activating a neighbor.

**Acceptance Scenarios**:

1. **Given** a released version, **When** a Windows visitor reaches downloads, **Then** the MSI is described as a normal installation with Desktop, CLI or both selectable, both selected by default, and the two portable ZIP choices remain visible.
2. **Given** a Linux visitor, **When** they choose a package, **Then** Desktop + CLI and CLI-only tar.gz archives have standalone platform/format labels and adjacent prerequisites.
3. **Given** candidate or specification status, **When** the page renders, **Then** it does not advertise published package availability or provide premature file links.

### User Story 2 - Navigate predictable links (Priority: P1)

A visitor can distinguish the hero jump, file downloads, installation guidance, documentation and repository navigation, including when reading a link list.

**Why this priority**: Matching labels to destinations prevents false promises and redundant choices.

**Independent Test**: Inventory every visible landing and shared-chrome link, resolve its actual href, and compare its purpose with destination content.

**Acceptance Scenarios**:

1. **Given** the hero action, **When** activated, **Then** it moves to the download choices and remains the page's single primary visual action.
2. **Given** a platform guidance link, **When** followed, **Then** its label names the platform and promises only the instructions actually present at that destination.
3. **Given** the remaining landing/header/footer links, **When** audited, **Then** documentation uses stable latest routes, archive navigation remains distinct, and no accidental duplicate packaging link remains.

### User Story 3 - Use all controls after reflow (Priority: P1)

A visitor using a narrow screen, enlarged text, either theme, a keyboard, a mouse or touch can read and activate every landing control.

**Why this priority**: A layout correction must hold beyond one desktop screenshot.

**Independent Test**: Inspect full-page headless captures and measured controls in light/dark at 320, 390, 768 and 1440 CSS pixels, plus 200% text and a 400%-zoom-equivalent reflow viewport; exercise keyboard, pointer and touch paths.

**Acceptance Scenarios**:

1. **Given** any required viewport/theme, **When** the page lays out, **Then** there is no unintended horizontal overflow, clipped label, overlapping action or missing choice.
2. **Given** keyboard navigation, **When** focus traverses the page, **Then** order follows the document and each interactive control shows focus and activates its intended destination.
3. **Given** mouse or touch input, **When** each package/guidance choice is activated, **Then** the intended native link handles activation and has at least the governed minimum target size or valid spacing.

### Edge Cases

- Long standalone package labels wrap inside the card at 320 CSS pixels and enlarged text.
- All five release assets retain exact version, operating system, amd64 architecture, client variant and file extension.
- A missing, mismatched or unpublished release cannot silently become an apparently downloadable package.
- The public export and documentation-only offline export retain their different purposes.
- Browser verification must use owned, isolated headless processes and profiles, with no personal Chrome interaction or installer execution.
- Public deployment follows merge/publication authorization; pre-merge evidence cannot be described as a live deployed fix.

## Requirements

### Functional Requirements

- **FR-001**: Preserve five released packages: Windows MSI, Windows Desktop + CLI ZIP, Windows CLI-only ZIP, Linux Desktop + CLI tar.gz, and Linux CLI-only tar.gz. Verify their identities against the official release manifest and asset inventory.
- **FR-002**: Explain MSI component selection/defaults and maintenance separately from portable archives. Keep both platforms, both client variants and prerequisites visible.
- **FR-003**: Provide one primary hero action. Give card downloads subordinate emphasis, with visibly distinct alternative package and guidance treatment.
- **FR-004**: Give card controls explicit alignment, wrapping and horizontal/vertical spacing; labels and target rectangles must remain distinct after reflow.
- **FR-005**: Make package/guidance labels predictable without their card context; retain native links for navigation and files.
- **FR-006**: Inventory every visible landing/header/footer link with label, actual destination, intent, navigation type and current version. Correct misleading destination promises and remove accidental duplication without losing information access.
- **FR-007**: Preserve stable latest-documentation routing, version archives, release-status truth, five asset destinations, pinned brand tokens and governed kit bytes.
- **FR-008**: Verify light/dark at 320, 390, 768 and 1440 CSS pixels; 200% text; 400%-zoom-equivalent reflow; focus order and visible focus; keyboard, mouse and touch activation; target size/spacing; and complete label visibility.
- **FR-009**: Add focused rendered-link and action-grouping regressions plus visual/reflow checks against the generated public export. Keep browser activity isolated and headless. Inspect the live counterpart after an authorized deployment and distinguish that later proof from pre-merge verification.
- **FR-010**: Keep new checks targeted so the complete CI remains within 660 seconds. Report measured evidence and untested assistive technology; do not claim certification.

### Key Entities

- **Package choice**: release version, platform, architecture, client variant, format, exact asset destination and install/portable purpose.
- **Landing link**: visible label, resolved href, intent, navigation category and release/version context.
- **Verification case**: export, theme, viewport, text scale, input mode, measurements, screenshot and result.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every visible landing/chrome link is inventoried; all five downloads match the official release inventory and retain distinct activation targets.
- **SC-002**: One hero action has primary emphasis; each card exposes a principal package, alternatives and truthful guidance without any touching or overlapping controls.
- **SC-003**: All eight base viewport/theme combinations plus enlargement/reflow cases pass full-page inspection and control measurements with zero unintended overflow, clipped labels or lost choices.
- **SC-004**: Keyboard, pointer and touch checks reach the expected destinations, with visible focus and adequate target geometry.
- **SC-005**: Repository/site checks pass, CI is green on the final PR revision within 660 seconds, and all review findings are resolved in at most two rounds.

## Assumptions

- The published v1.0.1 package set and release-status mechanism remain authoritative; this slice does not publish or rebuild product binaries.
- Scope includes landing/shared chrome and narrowly necessary guidance labels/content, not another site-wide documentation redesign.
- Owner authorization covers branch push and an official PR. Final merge remains with the owner; live post-deployment verification follows deployment authorization.

## Clarifications

### Session 2026-10-15

- Q: Add Linux update/removal procedures or use a narrower accurate link? A: Use a platform-specific prerequisites/installation link to the existing Linux section; avoid unrelated lifecycle-document expansion.
- Q: Keep the bottom duplicate packaging link? A: Remove the accidental duplicate while retaining platform guidance and capability documentation access.
- Q: Publish updated site or product before merge? A: Stop at the requested final-review/merge handoff. Record the live check as a post-deployment follow-up, not pre-merge proof.
