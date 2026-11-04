# Feature Specification: v1.3.0 publication and verification

**Feature Branch**: `codex/075-v1-3-publication`

**Created**: 2026-11-03

**Status**: Publication and verification complete; final PR delivery gates pending

**Input**: PR #252 is merged and its remote branch deleted. Perform standard housekeeping and publish the next version release. Existing publication, documentation destination and model elections persist; no new tracking issue.

## User Scenarios & Testing

### User Story 1 - Obtain the official release (Priority: P1)

Users can download the accepted v1.3.0 Windows/Linux products and matching documentation from the official release.

**Why this priority**: The reviewed speaker milestone is complete but its prepared version is not yet published.

**Independent Test**: Independently download every official file and reconcile its name, size and digest with the retained accepted candidate.

**Acceptance Scenarios**:

1. **Given** reviewed merged source and passing checks, **When** publication completes, **Then** all ten accepted files are available on one immutable version tag with concise highlights.
2. **Given** a conflicting tag, changed artifact or incomplete proof, **When** publication is attempted, **Then** it stops without overwriting published authority or asserting success.
3. **Given** the publication candidate, **When** previously elected CPU/NVIDIA qualification runs, **Then** actual devices, complete outcomes, source identity and limitations are retained separately from model-free checks.

### User Story 2 - Read the matching live documentation (Priority: P1)

Readers see v1.3.0 as released on the existing official site with working downloads, schemas, diagrams and historical routes.

**Why this priority**: Downloaded product and authoritative instructions must identify the same release.

**Independent Test**: Compare deployed resources to the exact completed deployment, follow official downloads and inspect the maintained viewport/theme cases.

**Acceptance Scenarios**:

1. **Given** a published release, **When** its documentation is promoted, **Then** the matching version becomes current and prior routes remain available.
2. **Given** uncertain deployment acknowledgement, **When** recovery runs, **Then** only the exact staged identity is reconciled and no success is inferred from an unrelated deployment.

### User Story 3 - Retain auditable delivery (Priority: P2)

The owner receives a durable publication record, completed housekeeping and a reviewed evidence change for final merge.

**Why this priority**: Public claims must remain inspectable after transient workflow artifacts expire.

**Independent Test**: Reconcile source, publication, downloaded bytes, deployment, review and timing records against the final evidence change.

**Acceptance Scenarios**:

1. **Given** the merged slice, **When** housekeeping completes, **Then** the checkout is synchronized, obsolete branch/selector removed and existing completed tracking reconciled.
2. **Given** completed publication proof, **When** the evidence PR is handed off, **Then** checks are green, findings resolved and no more than two review rounds have been triggered.

### Edge Cases

- Squash source identity differs from the reviewed branch despite identical trees.
- Candidate-status documentation cannot stand in for release-status documentation.
- Main moves, model prerequisites drift, artifacts expire or a producer fails.
- Partial release upload, conflicting immutable tag, uncertain Pages push/build or stale CDN content.
- Quoted bot commands can trigger review; publish no trigger text in evidence or routine comments.

## Requirements

### Functional Requirements

- **FR-001**: Verify the merged source and tracking, preserve prior evidence, and keep the release source immutable and distinct from publication records.
- **FR-002**: Qualify the complete release-status candidate with required native, package, installer, documentation, integrity and provenance checks; retain original bytes before publication or rerun.
- **FR-003**: Report preparation, qualification and combined timing with a 1800-second target, 3600-second hard limit, provisional finalization reserve and terminal timestamp authority. Retain overlapping lane timing without inventing a common readiness boundary.
- **FR-004**: Repeat the already elected real CPU/NVIDIA speaker campaign on the exact publication package without new providers, training data or thresholds; retain full results, model identity and cleanup evidence without broad accuracy claims.
- **FR-005**: Publish the accepted ten files unchanged under v1.3.0 with concise highlights ending in the tagged master changelog link. Do not rebuild after acceptance or overwrite conflicting immutable authority.
- **FR-006**: Promote the published documentation to the existing destination with exact deployment identity, historical routes and bounded recovery.
- **FR-007**: Independently verify every published file, deployed documentation resources, official download links and the maintained live rendering cases.
- **FR-008**: Retain source-bound evidence outside workflow retention and record actual publication in internal planning and the master changelog.
- **FR-009**: Open the evidence PR automatically, resolve findings within two review rounds and hand off after green applicable checks. Never include executable review mentions in evidence, quoted text or routine comments.
- **FR-010**: Create no tracking issue, preserve user data/credentials and routing, keep Windows processes hidden/noninteractive, and leave evidence PR merge to the owner.

### Key Entities

- **Release source**: Exact reviewed main commit, tree identity and accepted checks.
- **Publication candidate**: Version, original files, package proofs, release-status documentation and immutable digests.
- **Publication/deployment receipt**: Actual external identity, terminal state, timestamps and independent readback.
- **Delivery record**: Requirement mapping, retained evidence, review count, limitations and owner handoff.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All ten official files independently match the accepted source-bound candidate.
- **SC-002**: Required qualification and elected CPU/NVIDIA execution finish within the hard limit with no missing outcomes or undisclosed fallback.
- **SC-003**: The exact completed deployment passes maintained content, route, download and eight viewport/theme checks.
- **SC-004**: Housekeeping and existing tracking are complete; evidence can be read back outside workflow storage.
- **SC-005**: The evidence PR has green applicable checks, no unresolved findings and at most two review rounds.

## Assumptions

- The selected release source is PR #252 squash 27ddddcd8afe8e44b1dd0c570ae3fd2070130728. Its tree equals the accepted S074 head.
- Existing workflows, local models, credentials and Pages destination remain authoritative. No new runtime behavior or dependency is planned.
- The owner explicitly authorized release publication; routine push, tag, release and documentation operations need no additional confirmation.
- S074 evidence remains historical evidence; fresh publication-package receipts identify their own source and bytes.

## Clarifications

### Session 2026-11-03

- All scope, authority, recovery, timing and completion categories are clear from the owner request and current contracts; no questions remain.
- Release-status candidate preparation is required before publication; pre-merge/candidate-status archives are not substituted.
- The two-round cap is enforced across every posted body, including code blocks and embedded evidence.
