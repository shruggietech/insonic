# Feature Specification: S053 v1.0.2 release preparation

**Feature Branch**: `codex/053-capture-source-correction`

**Created**: 2026-10-27

**Status**: Owner-merged preparation; post-merge capture-source correction in progress

**Input**: Prepare v1.0.2 using Spec Kit/autopilot, push/open the official PR, resolve CI failures and at most two Codex review rounds, then stop before owner merge. Official publication is a later phase.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Identify a coherent new candidate (Priority: P1)

A maintainer can distinguish the upcoming v1.0.2 candidate from released versions and review the accumulated corrections under one consistent version.

**Why this priority**: The fixes need a new distributable release with accurate version and availability information.

**Independent Test**: Inspect the prepared software, contracts, documentation, package identities and release highlights, requiring v1.0.2 throughout their current-version surfaces.

**Acceptance Scenarios**:

1. **Given** the reviewed development baseline, **When** v1.0.2 is prepared, **Then** current software, documentation, owned schemas, package identities and changelog agree on v1.0.2 and identify it as unpublished.
2. **Given** the published earlier versions, **When** preparation completes, **Then** their immutable tags/assets and original contract identities remain intact and historical navigation identifies their released status.

### User Story 2 - Read truthful release documentation (Priority: P1)

Operators can read documentation and visual instructions whose version, package choices and capability statements match their selected release.

**Why this priority**: Copying obsolete visual evidence or contradictory platform/download claims misrepresents what an operator receives.

**Independent Test**: Render public/offline candidate documentation with validated image authority and verify historical routes, current package links and platform descriptions.

**Acceptance Scenarios**:

1. **Given** version-bound screenshots and release pages, **When** v1.0.2 documentation renders, **Then** it uses verified compatible visual authority without relabeling historical capture receipts.
2. **Given** supported Windows/Linux package choices, **When** current release instructions are read, **Then** archive/setup/MSI availability, shared CLI/GUI capabilities and stated limitations match maintained distribution contracts.

### User Story 3 - Review a qualified release preparation (Priority: P2)

The owner receives a concrete release-preparation PR with completed automated checks and bounded review evidence, ready for the final review and merge.

**Why this priority**: Source preparation, CI acceptance and actual release publication are separate outcomes.

**Independent Test**: Verify the final PR source identity, required checks, review receipts and post-merge execution guide, with publication still pending.

**Acceptance Scenarios**:

1. **Given** the prepared changes, **When** delivery completes, **Then** all elected final-source checks pass and at most two Codex rounds have completed with every actionable finding addressed.
2. **Given** the owner merge boundary, **When** this turn ends, **Then** the PR remains unmerged, no official tag/release/deployment exists from this slice, and exact-main candidate/CPU-restore/publication steps are clearly recorded for the next phase.

### Edge Cases

- Version preparation must reject overwriting an existing version or claiming an unpublished candidate is released.
- Preserve original published history separately from post-release edits used to seed the new candidate.
- Historical screenshot bytes/receipts cannot acquire a new version by string replacement.
- Preparation or capture failure must not produce partial successful activation or false acceptance.
- External dependency versions, upstream schemas and independently versioned protocols retain their identities.
- No visible or focus-stealing windows may launch on the owner's desktop.
- An unavailable check, failed cleanup or functional failure cannot be hidden by timing headroom or cached proof.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Prepare a consistent v1.0.2 software/documentation/owned-contract/package candidate using maintained preparation mechanisms.
- **FR-002**: Preserve published version/tag/asset/schema authority and accurate historical released navigation while retaining current corrections in v1.0.2.
- **FR-003**: Provide valid version-bound visual documentation for current candidate pages and retain immutable historical capture evidence.
- **FR-004**: Correct release-facing platform, download, capability and availability descriptions where demonstrated inconsistencies affect v1.0.2.
- **FR-005**: Curate the accumulated changes into a dated v1.0.2 changelog and concise highlights ending with the tagged master changelog link.
- **FR-006**: Qualify the exact prepared source through maintained checks, preserving integrity, signing disposition, hidden execution and separate preparation/qualification/combined accounting.
- **FR-007**: Record an executable post-merge handoff for a fresh exact-main candidate, targeted managed CPU/restore acceptance, immutable publication and matching documentation promotion, without claiming those future actions happened.
- **FR-008**: Publish the official PR automatically, resolve failures and every actionable finding within at most two Codex rounds, retain compact audited evidence and remove owned scratch before stopping unmerged.

### Key Entities

- Release candidate: version, exact source identity, synchronized public contracts, package choices and unpublished state.
- Historical release authority: original version/source/tag, schemas, documentation and capture receipts with immutable identities.
- Acceptance record: exact source/check/review outcomes, timing, cleanup and explicit unverified publication state.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All maintained current-version checks identify v1.0.2; historical contract checks and complete public/offline rendering pass without capture-version substitution.
- **SC-002**: Every elected final-source delivery check succeeds within the maintained hard limit after verified readiness; all compact acceptance/review receipts are retained outside workflow-artifact expiry.
- **SC-003**: The owner receives one official unmerged PR with zero unresolved actionable findings, no more than two Codex review rounds and a complete post-merge release guide.

## Assumptions

- This request authorizes preparation and PR publication, with the owner explicitly retaining the merge boundary. Exact-main CPU/restore execution and official tag/release/Pages publication follow owner merge.
- v1.0.2 is unused and the latest published product is v1.0.1 from October15. Broad #147 remains open for continuing quality and installed-release observations.
- Existing supported platforms, configured providers, signing policy and publication destination remain unchanged; no new feature scope or attended pre-release ritual is introduced.
- Qualification retains the1800s target/3600s hard limit,60s provisional reserve, original product deadlines and separate phase accounting.

## Post-merge continuation (2026-10-27)

The owner merged PR #186, then authorized housekeeping, exact-main CPU/restore acceptance and v1.0.2 release/documentation publication. Main960fbf7 failed screenshot source validation because later PR corrections removed the complete capture snapshot from squashed ancestry. T019 restores the existing FR-003/SC-001 proof contract in a fresh checkout. Preserve the original two completed PR review rounds and every capture byte; new source must receive its own checks and reach reviewed main before release acceptance/publication.
