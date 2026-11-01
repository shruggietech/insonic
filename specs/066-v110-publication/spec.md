# Feature Specification: v1.1.0 publication and independent verification

**Feature Branch**: `codex/066-v110-publication`

**Created**: 2026-11-01

**Status**: Specified

**Input**: Owner-authorized S066 using Spec Kit/autopilot: publish and verify v1.1.0 and its docs, push/open the official PR, resolve every actionable review within two rounds, and stop for owner final review and merge after green CI.

**Tracking**: Issue #239, milestone 51, Project 7. Product source is the reviewed S065 squash commit `82a1bfd7e8f4e389d451789ec7840d874127f39c`; its exact-main CI passed. This slice adds publication and delivery evidence without changing that accepted product source.

## Clarifications

### Session 2026-11-01

- Q: Which source is shipped? A: The reviewed S065 squash source; S066 records are separate.
- Q: Is publication already authorized? A: Yes, including the immutable tag, official release and existing Pages destination.
- Q: Are real-model CPU/NVIDIA checks elected? A: No; maintained deterministic/native/package qualification and independent public verification are required.

All clarification coverage categories are clear. These answers come from the owner request, accepted S065 continuation and existing project contracts; no user questions were needed.

## User Scenarios & Testing

### User Story 1 - Obtain the accepted official release (Priority: P1)

An operator downloads v1.1.0 for supported Windows/Linux hosts and receives the same complete packages that passed release qualification, with truthful platform, signing and engine-evidence claims.

**Why this priority**: Prepared code and temporary CI artifacts are not official downloadable software.

**Independent Test**: Compare the official tag, release manifest, all ten downloaded assets and checksums with the retained accepted candidate and selected source.

**Acceptance Scenarios**:

1. Given the reviewed merged source and green exact-main CI, when a fresh publication candidate is qualified, then all four CLI/desktop packages, the MSI/setup pair, both documentation archives, manifest and checksums identify 1.1.0 and that exact clean source.
2. Given accepted retained bytes, when publication completes, then the immutable tag and official release point to the selected source and every official asset matches those bytes without a rebuild.
3. Given an interrupted upload or an existing remote release, when publication is retried, then only identical verified state may be reconciled; conflicting tags/assets stop publication and remain unchanged.

### User Story 2 - Read the matching published documentation (Priority: P1)

A reader reaches the existing public documentation site and sees released v1.1.0 guidance, working exact-version downloads, the matching contracts and retained historical documentation.

**Why this priority**: Product and documentation publication must describe one authoritative release.

**Independent Test**: Compare the completed deployment identity and every required live documentation byte with the accepted original publication archive, and retrieve all official release assets independently.

**Acceptance Scenarios**:

1. Given the verified published release, when its documentation is promoted, then the existing domain, HTTPS settings and historical routes remain intact and the completed deployment names the promoted commit.
2. Given a completed deployment, when independent public verification runs, then all official asset bytes, download identities, documentation resources and schema routes match the accepted release authorities.
3. Given stale bytes, missing resources or concurrent deployment, when verification fails, then the failure is retained and investigated without claiming success, overwriting release assets or silently rebuilding accepted content.

### User Story 3 - Audit and merge the completed work (Priority: P2)

The owner receives an evidence PR that distinguishes candidate qualification, actual publication, public verification and unexecuted dimensions, with all applicable checks green and every actionable review addressed.

**Why this priority**: Durable evidence supports later diagnosis and an informed final merge.

**Independent Test**: Read the retained receipts and public PR, resolve each cited identity, and confirm the final source/check/review/tracking state.

**Acceptance Scenarios**:

1. Given a terminal campaign, when its outcome is recorded, then original artifacts and compact audited receipts survive workflow expiry and preparation, qualification and combined timings are separate.
2. Given the evidence PR, when at most two review rounds finish, then every actionable comment is individually addressed, no actionable thread remains unresolved, and absent bot evidence is stated accurately.
3. Given successful publication/public verification and final green applicable CI, when S066 is handed off, then the PR remains open and unmerged for the owner's review while official release and docs are already published.

### Edge Cases

- A failed candidate campaign is retained before a full rerun; a later passing run does not erase it.
- A moved source, dirty checkout, expired or ambiguous artifact, changed archive digest, incomplete asset inventory or conflicting tag cannot satisfy acceptance.
- An interrupted draft upload may resume only through verified identical state; a failed documentation verification does not imply rollback.
- A changed Pages parent or unrelated deployment is a visible conflict, not a reason for force-pushing.
- Real-model CPU/NVIDIA accuracy, hardware performance, publisher signing and fresh visual/accessibility behavior are claimed only when supported by an elected corresponding receipt.

## Requirements

### Functional Requirements

- **FR-001**: The campaign MUST bind v1.1.0 software, documentation and schema identity to the reviewed S065 squash source and completed green exact-main CI; S066 planning/evidence changes MUST remain distinct from shipped source.
- **FR-002**: A fresh publication-target candidate MUST contain the complete ten-asset release inventory and pass all maintained elected platform, native, installer, functional, integrity and provenance checks without weakening gates or substituting pre-merge artifacts.
- **FR-003**: Original candidate bytes, run/job/artifact metadata, checksums, successful and failed terminal outcomes and compact audited receipts MUST be retained outside workflow-run expiry; observed preparation, qualification and combined timing MUST remain distinct, with the thirty-minute target and sixty-minute limit applied according to the actual workflow's maintained timing authority.
- **FR-004**: Publication MUST create or reconcile the immutable v1.1.0 tag and official release at the accepted source, publish only the verified complete retained candidate, reject conflicting remote state and never rebuild accepted bytes to publish them.
- **FR-005**: Release highlights MUST stay concise, describe delivered compatibility and material limitations accurately, and end with a link to the master changelog at v1.1.0; historical versions and upstream identities MUST remain unchanged.
- **FR-006**: Documentation promotion MUST use the published release's original publication-target archive, preserve the existing destination/HTTPS/history contracts, and require successful completion of the exact promoted Pages commit.
- **FR-007**: Independent verification MUST retrieve and validate all official release asset bytes and verify all required live documentation resources, source/version identities, download targets and Pages identity against the original accepted release; failures MUST be retained and resolved before completion.
- **FR-008**: All activity MUST preserve existing credential isolation, bounded archive/path/network validation and hidden noninteractive Windows execution; no unrelated product feature, provider routing, host configuration or historical release mutation is included.
- **FR-009**: Spec Kit artifacts, branch, issue, milestone and Project metadata MUST consistently identify S066; the official evidence PR MUST be pushed automatically, receive no more than two requested review rounds, resolve every actionable comment, and reach final green applicable CI before owner handoff.
- **FR-010**: Reports MUST distinguish source/package/native qualification, official publication, public byte verification, browser behavior and real-engine evidence; missing separate security-review receipts MUST remain explicit, and owner PR merge MUST not be performed by the agent.

### Key Entities

- **Accepted source**: reviewed immutable product revision, synchronized version and exact-main CI identity.
- **Publication candidate**: complete original assets, manifests, qualification/signing dispositions and hashes bound to accepted source.
- **Release identity**: immutable tag, official release ID, exact public asset inventory and concise highlights.
- **Documentation deployment**: original archive, configured destination, parent/promoted commit and terminal build identity.
- **Evidence ledger**: original metadata/archives, phase timing, independent readback results, review-round count and final handoff state.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All ten official release assets match the retained accepted candidate in identity, size and digest, and v1.1.0 resolves to the accepted product source.
- **SC-002**: Every required live documentation resource and download target passes maintained byte/identity verification for the exact completed deployment, with historical paths retained.
- **SC-003**: All required qualification gates and final PR checks pass, all failed campaigns remain visible, and complete durable receipts identify separate observed phase timings and evidence limitations.
- **SC-004**: No actionable review remains unresolved after at most two rounds; the owner receives an open unmerged PR with matching S066 tracking and verified publication evidence.

## Assumptions

- The owner explicitly authorizes tag/release publication, existing Pages promotion, automatic push and official PR creation in this request. No additional routine permission gate is needed.
- Existing GitHub credentials and the configured insonic.io Pages destination remain the elected publication route.
- Maintained deterministic/native/package qualification is required. Real-model CPU/NVIDIA acceptance is not elected; no attended workstation test gate is introduced.
- The prepared S065 release highlights and version alignment are reused. Product changes are limited to a demonstrated publication blocker, which would require a newly reviewed/qualified source before acceptance; no blocker is currently known.
- The evidence PR can merge after publication because its records are separate from the immutable product source.
