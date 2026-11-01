# Feature Specification: v1.1.0 release preparation and qualification

**Feature Branch**: `codex/065-v110-release`

**Created**: 2026-10-31

**Status**: Specified

**Input**: Owner-directed S065 using Spec Kit and autopilot, automatic push and official PR, at most two review rounds, final green-CI handoff for owner merge.

**Tracking**: Issue #237, milestone 50, Project 7. Builds on merged S061-S064 and the completed long-recording milestone.

## User Scenarios & Testing

### User Story 1 - Understand and use one coherent candidate (Priority: P1)

An operator receives an unpublished 1.1.0 candidate whose software identity, documentation, schemas and release highlights describe the same delivered behavior, including resumed long-recording processing, coverage and targeted repair.

**Why this priority**: Mixed version identities make installation and contract selection unreliable.

**Independent Test**: Inspect current version authorities, package manifests and shipped help, then execute the maintained installed recording journey.

**Acceptance Scenarios**:

1. Given accepted S061-S064 changes, when the new candidate is prepared, then every current owned release authority identifies 1.1.0 and the product remains explicitly unpublished.
2. Given existing supported requests and catalog data, when the candidate is qualified, then additive long-recording contracts remain usable without a new catalog migration and CLI features awaiting GUI controls are identified.
3. Given the accumulated changes, when release highlights are read, then they concisely explain user-visible additions, corrections and compatibility limits and end with the tagged changelog link.

### User Story 2 - Trust historical and visual evidence (Priority: P1)

A reader can distinguish published 1.0.3 from candidate 1.1.0 and inspect screenshots genuinely captured from the displayed version, while original released bytes remain immutable.

**Why this priority**: Copying current changes into old version routes or relabeling old images misrepresents release authority.

**Independent Test**: Compare historical documents/contracts with published tag authority and verify each current screenshot's original bytes, source/version identity and capture receipt.

**Acceptance Scenarios**:

1. Given current corrections on the former current-version pages, when preparing 1.1.0, then those corrections carry forward before historical pages/contracts are reconciled to published authority.
2. Given a published 1.0.3 release and an unpublished 1.1.0 candidate, when navigating documentation/downloads, then their status and available assets remain distinct and accurate.
3. Given required current-version images, when captures are imported, then they come from isolated version-bound qualification with original hashes and provenance; no old image is relabeled or fabricated.

### User Story 3 - Review a fully qualified delivery (Priority: P1)

The owner receives a ready official PR with complete platform evidence, resolved reviews and clear boundaries for later merge and publication.

**Why this priority**: Release readiness requires verified candidates and exact-source evidence, not only a version edit.

**Independent Test**: Inspect terminal receipts, candidate identities, review threads and final PR state without requiring an attended local desktop ritual.

**Acceptance Scenarios**:

1. Given the prepared source, when required qualification completes, then Windows/Linux CLI and desktop distributions, Windows installer paths, native workflows, supported backends and documentation pass the maintained checks.
2. Given a failed or superseded qualification run, when a full rerun is needed, then audited compact receipts are retained outside workflow expiry first, without changing the original conclusion.
3. Given the official PR, when at most two external review rounds complete, then every actionable finding is addressed and each thread resolved; final PR and direct branch-head CI are green before owner handoff.
4. Given a successful handoff, when the owner plans publication, then the continuation identifies fresh accepted-main qualification and publication of the exact retained bytes, without treating pre-merge artifacts as a release already published.

### Edge Cases

- Historical fixtures or upstream dependency/model versions contain the previous release number but are not current product authority.
- A copied screenshot has the correct dimensions but belongs to a different source/version, or its provenance artifact is missing/corrupt.
- A PR temporary merge revision differs from the branch head; receipts must identify both rather than conflate them.
- Preparation succeeds but installed qualification fails, is cancelled or exceeds the hard limit; acceptance remains false with retained diagnostics.
- The advisory target is exceeded within the hard limit; valid acceptance remains eligible with the measured overrun reported.
- A review bot returns only a positive reaction or no separate security receipt; report the observed evidence without inventing an approval.
- Hosted acquisition fails or a retained artifact expires; preserve original error and exact identity rather than substitute another candidate.

## Requirements

### Functional Requirements

- **FR-001**: Prepare one unpublished 1.1.0 candidate with coordinated software, documentation and master-schema versions and current owned entry points.
- **FR-002**: Include accepted S061-S064 behavior and concise compatibility/capability information without expanding into answer or speaker-discovery milestones or unrelated dependency upgrades.
- **FR-003**: Preserve original published assets, historical schema identities and narrow historical receipt rules. Carry current changes forward before restoring historical version authority where necessary.
- **FR-004**: Display verified published-versus-candidate status truthfully, retaining access to existing releases and historical documentation.
- **FR-005**: Provide genuine current-version visual evidence with original source-bound capture receipts and unchanged imported image hashes; keep local work headless and use isolated hosted desktops for interactive qualification.
- **FR-006**: Pass maintained Windows/Linux CLI/desktop, installer, native, backend, documentation and integrity checks on the exact final source/candidate identity, with separate direct branch-head evidence when PR CI uses a temporary merge commit.
- **FR-007**: Retain audited compact acceptance receipts outside workflow-run retention before full reruns and final handoff; keep preparation, qualification and combined time distinct, with an 1800-second advisory target, 3600-second hard limit and 60-second provisional reserve. Mandatory checks remain required independent of timing.
- **FR-008**: Keep source/credential/routing authority, catalog schema 11 and substantive integrity/security checks intact; repair only demonstrated release-blocking defects proportionally and record their scope.
- **FR-009**: Publish the authorized branch and official PR automatically, resolve every actionable comment within the automatic initial review plus at most one manual second round, and hand off an open, non-draft, green PR without merging, tagging or publishing a release.
- **FR-010**: Retain an executable post-merge procedure for fresh accepted-main candidate qualification, separately elected real-model acceptance, exact-retained-byte publication and public/documentation verification.
- **FR-011**: State acoustic/model/device evidence limits explicitly. Real-model CPU/NVIDIA qualification is not elected by this preparation request; deterministic fixtures do not establish recognition accuracy or hardware performance.

### Key Entities

- **Release candidate**: Version, source revision, platform package set, documentation/schema identity, checksums and unpublished state.
- **Historical release authority**: Published tag/source identity, original assets, versioned documentation/contracts and narrowly scoped retained receipts.
- **Capture evidence**: Original image bytes, source/version identity, isolated capture context and digest-bound receipt.
- **Acceptance record**: Run/attempt/source identity, prerequisite readiness, applicable checks, terminal result, separate timing and retained artifact hashes.
- **Review ledger**: At most two triggers, findings, responses, resolution state and any absent review receipt.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All current owned version surfaces agree on 1.1.0; all existing published assets and original historical contracts retain verified identity.
- **SC-002**: All current displayed screenshots have valid original source/version/hash evidence, and published/candidate status is accurate on public and offline documentation paths.
- **SC-003**: Every mandatory platform/package/backend/documentation check passes for the final candidate, with accepted retained terminal receipts and no hard-limit overrun.
- **SC-004**: The official PR has no unresolved actionable review findings after no more than two rounds and green final-source checks, while remaining unmerged for owner review.
- **SC-005**: The owner can follow the retained continuation to qualify accepted main and publish the exact accepted bytes without rebuilding after acceptance or mistaking deterministic evidence for acoustic/device qualification.

## Assumptions

- The accepted baseline is main at `a0f78d1898a4edf526ce2952c45ee440a9204ff5`; the public latest release remains 1.0.3.
- 1.1.0 is a minor release because S063/S064 add public capabilities; no breaking product contract is intentionally introduced.
- Current CI, package, capture and release tooling supplies the maintained qualification path; no new desktop acceptance gate or external route is introduced.
- Local calendar date is 2026-10-31; qualification receipts retain authoritative UTC timestamps.

## Clarifications

### Session 2026-10-31

- Q: Does this slice publish the software release? A: No. Push and official PR are authorized; owner merge and later tag/release/Pages publication remain subsequent actions.
- Q: Which version and compatibility scope apply? A: Additive 1.1.0 minor release, retaining existing catalog schema 11 and published historical identities.
- Q: Does preparation elect real-model CPU/NVIDIA acceptance? A: No; retain explicit limits and an executable separately elected post-merge path.

Autopilot resolved these choices from the owner instruction, accepted recommendation and constitution. All clarification taxonomy categories are clear; no outstanding or deferred requirement decision remains. The 14-item specification-quality checklist remains 14/14 passing.
