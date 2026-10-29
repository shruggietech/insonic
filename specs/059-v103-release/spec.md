# Feature Specification: v1.0.3 patch release preparation

**Feature Branch**: `codex/059-v103-release`
**Created**: 2026-10-29
**Status**: Specified
**Input**: Owner-authorized S059 under Spec Kit autopilot: prepare the recommended patch, automatically push/open its official PR, fix CI/reviews, allow at most two Codex rounds, and stop before owner merge.

## User Scenarios & Testing

### User Story 1 - Obtain one coherent patch candidate (Priority: P1)

A user evaluating the next patch sees the same version in the software, documentation and published data contract, and can understand which reliability fixes it contains.

**Why this priority**: Contradictory versions or release claims prevent trustworthy installation and support.
**Independent Test**: Inspect all maintained version authorities and the patch highlights against the unreleased changes.

**Acceptance Scenarios**:

1. **Given** reviewed fixes after v1.0.2, **When** v1.0.3 is prepared, **Then** current software, documentation and master schema agree on 1.0.3.
2. **Given** a prepared but unpublished patch, **When** users inspect its documentation, **Then** candidate status is explicit and no v1.0.3 publication or fresh acoustic acceptance is claimed.
3. **Given** concise release highlights, **When** the maintainer reads them, **Then** they describe the included reliability fixes and end with the tagged master changelog link.

### User Story 2 - Preserve historical contracts and documentation (Priority: P1)

Existing users retain working versioned documentation, schemas and immutable release records while the next patch receives current evidence requirements.

**Why this priority**: A patch must not silently rewrite the contracts attached to existing releases.
**Independent Test**: Compare historical versioned files and original-release validation with the baseline, then validate new and historical documentation routes.

**Acceptance Scenarios**:

1. **Given** existing released versions, **When** preparation creates the next documentation/schema version, **Then** historical source records, schema identities and published asset authority remain intact.
2. **Given** current and unknown package candidates, **When** their evidence is checked, **Then** modern terminal, import-recovery and storage-failure requirements remain mandatory; exact historical exemptions do not broaden.

### User Story 3 - Review qualified bytes before publication (Priority: P1)

The owner receives one clean, reviewable PR whose platform packages and documentation are qualified, with a precise later release procedure.

**Why this priority**: A green PR is necessary evidence but cannot establish publication or qualify different rebuilt package bytes.
**Independent Test**: Audit exact-source CI identities, package and documentation receipts, review threads and the post-merge instructions.

**Acceptance Scenarios**:

1. **Given** the final PR source, **When** elected checks complete, **Then** all four supported Windows/Linux CLI/desktop variants and applicable installers/documentation pass with exact source, image, inventory and receipt bindings.
2. **Given** hosted visual acceptance, **When** current image-bearing documentation is checked, **Then** source/version provenance and confined fixture cleanup are established without taking owner-desktop focus.
3. **Given** accepted CI and completed reviews, **When** the PR is handed off, **Then** recoverable independent evidence is retained, the tree is clean, and the PR is open and unmerged.
4. **Given** current private Windows runtime directories, **When** ordinary access validates their ownership and permissions, **Then** conforming permissions are not rewritten; changed owned permissions still require successful repair.
5. **Given** later owner merge and publication authorization, **When** the documented release procedure is followed, **Then** a fresh exact-main candidate receives CPU/fresh-restored processing acceptance and the same retained accepted bytes are published and publicly verified.

### Edge Cases

- Version preparation encounters stale current-version links or hard-coded current contract bindings.
- Historical package receipts lack newly introduced evidence; only exact original exemptions apply.
- A screenshot's capture version/source differs from the new documentation version.
- CI or external reviews fail after a push; retain failed evidence before a full rerun.
- A squash merge changes source identity; PR merge artifacts cannot become exact-main release candidates.
- Rebuilding after CPU acceptance changes archive bytes and invalidates that acceptance binding.
- Candidate rendering, release-target rendering and actual publication have distinct status.
- Local qualification must remain hidden/headless; real desktop interaction runs only in isolated hosted acceptance.

## Requirements

### Functional Requirements

- **FR-001**: The prepared software, documentation, master schema and current release bindings MUST agree on version 1.0.3.
- **FR-002**: The changelog and concise highlights MUST describe the included post-v1.0.2 reliability fixes without new unsupported capability claims; highlights MUST end with exactly one exact-tag master changelog link.
- **FR-003**: Historical documentation, schema identities and immutable original release authority MUST remain intact.
- **FR-004**: Candidate status MUST remain explicit; current and unknown receipts MUST retain modern evidence requirements and exact original-release exemptions MUST remain narrow.
- **FR-005**: All four supported native package variants, applicable Windows installers and matching documentation MUST pass existing exact-source qualification and complete inventory/license/receipt validation.
- **FR-006**: Current documentation entry points, rendered contracts/changelog and image/source/version provenance MUST be coherent; hosted visual/interactive checks MUST remain isolated from the owner desktop.
- **FR-007**: Existing product deadlines and maintained qualification policy MUST remain unchanged, including separate preparation, qualification and combined timing.
- **FR-008**: The final official PR MUST have green elected checks, all review findings resolved within at most two Codex rounds, independently retained compact evidence and clean owned-fixture cleanup, and MUST stop before owner merge.
- **FR-009**: The handoff MUST document fresh exact-main candidate creation, real CPU/fresh-restored processing acceptance, retention/publication of those same accepted bytes and subsequent documentation/public verification as post-merge work under publication authorization.

- **FR-010**: Routine access to an already-private Windows directory MUST freshly verify current ownership and exact private permissions without reapplying unchanged permissions. Changed owned permissions MUST still be normalized, and a failed normalization MUST remain an error.

### Key Entities

- **Prepared patch**: Version, source revision, changelog, highlights and explicit unpublished state.
- **Versioned contract set**: Current software/docs/schema authority plus retained historical routes and identities.
- **Qualified candidate**: Exact revision, package/image/archive/inventory identities and required platform receipts.
- **Delivery evidence**: Exact-source CI, terminal timing, review receipts, independent audit and cleanup observations.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All maintained current-version authorities resolve to 1.0.3 and version/schema/integrity checks pass.
- **SC-002**: Every required current documentation page and retained historical route/contract passes its maintained validation, including applicable image provenance and public/offline builds.
- **SC-003**: All elected final-source platform/package/installer/candidate checks pass under existing timing policy, with independently retained artifact and receipt bindings.
- **SC-004**: No unresolved review finding remains, at most two Codex rounds complete, and one clean official PR is handed to the owner unmerged with an executable post-merge release guide.

## Assumptions

- Scope includes the five merged slices after v1.0.2: Pages publication, runtime/wait classification, historical-release verification, interrupted import recovery and storage-failure handling.
- No new feature, backend, dependency upgrade, product deadline, database migration or brand-kit change is intended.
- The current task ends at owner PR review. Merge, tag, real post-merge CPU/restore acceptance, product release and Pages publication are not executed in this phase.
- Existing optional-download/model identities and upstream versions remain independent of the product patch.
- #147 remains the broader audit; the release gets its own linked issue/milestone/Project tracking.

## Clarifications

### Session 2026-10-29

- Autopilot decision: prepare 1.0.3 using the maintained release preparation tooling; do not rebuild old releases or loosen historical evidence.
- Autopilot decision: preserve the owner's merge boundary. Post-merge CPU/restore qualification and publication are a documented continuation, not falsely completed PR tasks.
- Autopilot decision: the built-in requirements-quality checklist is sufficient; no additional reviewer-owned checklist is needed.
