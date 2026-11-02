# Feature Specification: v1.2.0 release preparation and dependency qualification

**Feature Branch**: `codex/070-v1-2-release-preparation`

**Created**: 2026-11-02

**Status**: Specified

**Input**: Owner-authorized S070 under Spec Kit autopilot, automatic push and official PR, at most two review rounds, owner-controlled final merge. Incorporate dependency PRs #245 and #246 without creating a tracking issue.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Review one consistent release candidate (Priority: P1)

A maintainer can inspect an unpublished v1.2.0 candidate containing accepted evidence-grounded answer work and the two pending dependency updates, with matching product, documentation and contract versions.

**Why this priority**: A candidate must identify the same behavior and version everywhere before qualification is meaningful.

**Independent Test**: Validate the current candidate's version bindings and compare historical documentation and contracts to their published authority.

**Acceptance Scenarios**:

1. **Given** accepted main and the two dependency proposals, **When** preparation completes, **Then** all owned current version identities equal 1.2.0 and the dependency updates retain upstream integrity evidence.
2. **Given** published v1.1.0 documentation and contracts, **When** current changes advance to v1.2.0, **Then** original published history remains intact and current additions remain available in the candidate.
3. **Given** the unpublished candidate, **When** its help or website is read, **Then** it describes current supported behavior, CLI/GUI differences and release status accurately.

### User Story 2 - Assess executable package evidence (Priority: P1)

A maintainer receives auditable qualification of the exact candidate across supported hosts, package variants, installers and backend combinations.

**Why this priority**: Source-only success cannot establish installed product behavior.

**Independent Test**: Inspect retained, source-bound terminal qualification receipts and current-version screenshots from isolated hosted desktops.

**Acceptance Scenarios**:

1. **Given** verified prerequisites, **When** qualification runs, **Then** mandatory functional, integrity, provenance, security and elected checks complete with separately reported preparation, qualification and combined timing.
2. **Given** packaged answer workflows, **When** deterministic qualification runs, **Then** all seven maintained cases pass across supported variants without claiming unelected real-model accuracy.
3. **Given** versioned screenshot examples, **When** the candidate is prepared, **Then** displayed images and original receipts describe genuine current-version captures.

### User Story 3 - Finish review without premature publication (Priority: P2)

The owner receives a green official PR ready for final review and squash merge, with every review finding addressed and candidate artifacts retained independently of expiring runs.

**Why this priority**: Preparation must support the owner's final decision without publishing an unaccepted version.

**Independent Test**: Inspect final PR head, checks, review threads, project state and retained evidence.

**Acceptance Scenarios**:

1. **Given** automatic review on PR creation, **When** findings arrive, **Then** each receives an appropriate response and any required fix, using no more than one additional requested review round.
2. **Given** final green checks and resolved reviews, **When** the owner is notified, **Then** the PR remains unmerged, no release/tag is published, and later publication requires the accepted post-merge source and exact retained bytes.

### Edge Cases

- Dependency updates that discard required source checksums must preserve checksum authority rather than weakening verification.
- Historical content changed since release must move forward before the historical tree is restored from its tag.
- Old screenshots cannot be relabeled as new captures; bootstrap qualification may omit only new-version walkthrough embeds until genuine captures exist.
- A changed final commit needs current source-bound evidence; a temporary PR merge commit is reported separately from branch-head evidence.
- A failed check or missing receipt remains a failure or unverified claim, even if other checks pass. Retain audited receipts before a full rerun.
- Qualification target overruns are advisory; hard-limit or functional failures block acceptance.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The candidate MUST align software, versioned documentation and master contract version at 1.2.0, include accepted current behavior, and preserve published historical authority.
- **FR-002**: The candidate MUST incorporate the two dependency proposals while retaining required upstream source-verification checksums and checks.
- **FR-003**: Current documentation and concise release highlights MUST explain supported behavior and CLI/GUI differences, distinguish unpublished status, and end highlights with the versioned master changelog link.
- **FR-004**: Qualification MUST cover Windows and Linux x86-64 CLI and desktop-with-CLI packages, Windows MSI/setup, default and supported alternative backends, and the seven maintained packaged answer cases.
- **FR-005**: Current screenshots MUST have genuine current-version source-bound capture receipts from isolated hosted desktops; local acceptance MUST remain headless.
- **FR-006**: Qualification MUST target 1,800 seconds after verified preparation, enforce 3,600 seconds with a 60-second provisional finalization reserve, and retain separate preparation, qualification and combined terminal timing. No mandatory checks may be weakened.
- **FR-007**: The official PR MUST have final-head verification, all review comments addressed, no unresolved actionable review threads, and at most two review rounds. Compact audited receipts MUST survive outside workflow runs.
- **FR-008**: Preparation MUST NOT publish a release/tag, promote production documentation, merge the PR or create a tracking issue. Existing dependency PRs remain open until this candidate is merged.
- **FR-009**: Evidence MUST distinguish deterministic packaged behavior, unelected real-model evaluation, pre-merge source qualification and later publication qualification.

### Key Entities *(include if feature involves data)*

- **Candidate**: Release version, exact source revision, package inventory, documentation and contract versions, unpublished state.
- **Historical release**: Published tag and immutable version-specific documents, schemas and captures.
- **Qualification receipt**: Source/version/platform identity, original artifact digests, readiness boundary, terminal check results and timing.
- **Review handoff**: Final PR head, review-round count, findings/resolutions, CI receipts, project status and owner merge boundary.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All owned current version bindings agree on 1.2.0 and historical published documents/contracts pass preservation checks.
- **SC-002**: Four supported package variants and both Windows installer forms pass mandatory qualification; seven packaged answer cases pass in each required variant.
- **SC-003**: Every displayed current-version screenshot has an original verified capture receipt; no historical capture is presented as current.
- **SC-004**: Final branch-head and PR checks are green within the hard limit, with separately retained audited timing and artifact provenance.
- **SC-005**: The official PR has zero unresolved actionable findings within at most two review rounds; the owner receives a concrete merge handoff with no publication or merge performed.

## Assumptions

- S069 accepted main is the source baseline; existing feature backlog #227-#233 is outside this release-preparation slice.
- Dependency #245 and #246 are pull requests, not new issues. Their intended updates are incorporated together without prematurely closing them.
- Maintained release and qualification contracts remain authoritative. Real-model evaluation is not elected by this preparation request.
- Exact post-squash source must be accepted separately for later release publication; pre-merge receipts do not establish published bytes.

## Clarifications

### Session 2026-11-02

- Autopilot resolves routine choices from the accepted recommendation and constitution: prepare/qualify only, preserve historical authority, create no issue, and stop at owner merge. No critical ambiguity requires an owner question.
