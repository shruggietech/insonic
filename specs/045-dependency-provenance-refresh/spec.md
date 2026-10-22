# Feature Specification: Dependency refresh and provenance qualification

**Feature Branch**: `codex/045-dependency-provenance-refresh`

**Created**: 2026-10-22

**Status**: Implementation and two Codex rounds complete; acceptance and final-head checks recorded in PR #169

**Tracking**: #168, milestone 29, Project 7.

**Input**: S045 autopilot kickoff: refresh dependency PRs #157, #158 and #159 on current main; automatically push and publish one replacement PR, complete CI and at most two Codex review rounds, and stop before merge.

## User Scenarios & Testing

### User Story 1 - Adopt the pending compatible updates (Priority: P1)

A maintainer receives one reviewable update containing the pending maintainer-tool, runtime and documentation dependency changes without losing the fixes already merged to main.

**Why this priority**: Three stale update PRs currently have failed checks and cannot establish current package readiness.

**Independent Test**: Compare the resulting dependency selection against all three original PRs and current main, then verify that each intended update is present and unrelated product or native dependency choices remain unchanged.

**Acceptance Scenarios**:
1. **Given** the three pending dependency updates, **When** S045 is prepared, **Then** all intended package updates and their required transitive resolutions are included with exact source identities.
2. **Given** a superseded bot PR, **When** the replacement is qualified, **Then** tracking links explain its disposition and retain the original failed checks as history.

### User Story 2 - Keep dependency provenance authoritative (Priority: P1)

A maintainer can reproduce dependency preparation and package notices from the selected source and verify local patched dependencies against their declared upstream sources.

**Why this priority**: The runtime update currently fails upstream-source verification; accepting an inconsistent dependency record would weaken package evidence.

**Independent Test**: Prepare and verify the selected dependencies, then change or remove an upstream identity and confirm rejection rather than an integrity bypass.

**Acceptance Scenarios**:
1. **Given** locally patched runtime dependencies, **When** dependency selection changes, **Then** upstream versions, checksums, source archives and patch identities agree with the resolved dependency graph.
2. **Given** a missing or mismatching source checksum, **When** preparation or collection runs, **Then** it fails with a useful bounded cause.

### User Story 3 - Qualify the complete refreshed candidate (Priority: P2)

The owner receives a complete Windows/Linux candidate whose source, packages, documentation and maintained CLI coverage passed the elected checks, with review findings resolved and the PR still unmerged.

**Why this priority**: Passing lockfile checks alone does not establish that updated dependencies work in the installed product.

**Independent Test**: Execute the existing local and hosted qualification, inspect exact-source receipts and terminal timing, and verify every received review thread.

**Acceptance Scenarios**:
1. **Given** the refreshed source, **When** ordinary full CI completes, **Then** every elected functional, integrity, provenance and package check succeeds with exact-source evidence.
2. **Given** a review finding or CI failure, **When** it is addressed, **Then** the final head is requalified and findings are replied to and resolved within at most two Codex rounds.

### Edge Cases

- Stale bot changes omit current local-patch provenance or remove still-required upstream checksums.
- Dependency resolution introduces an unintended downgrade, replacement, toolchain change or unrelated upgrade.
- A documentation capture came from an earlier valid revision; source history and frozen capture bytes retain their existing verification contract.
- Cold prerequisites, failed source producers, unexpected skips, dirty sources or mismatched run/attempt artifacts cannot become accepted candidates.
- Expected platform-specific non-election remains distinct from an omitted required check.

## Requirements

### Functional Requirements

- **FR-001**: Include the compatible exact updates proposed by #157-#159 and the transitive changes required by resolution; account for every intended update against current main.
- **FR-002**: Keep current application behavior, locally patched desktop dependencies, native pins, brand bytes, supported platforms and product version unless a demonstrated dependency compatibility repair is necessary and documented.
- **FR-003**: Dependency manifests, resolved selections, upstream checksums, local patch provenance and packaged sources/notices MUST agree. Missing, stale or mismatching identities MUST fail verification.
- **FR-004**: Repair the demonstrated runtime provenance failure without removing checksum verification, broadening cache identity, fabricating receipts or disabling security/audit checks.
- **FR-005**: Reuse the existing documentation, source-native, installed-package, backend fixture and CLI-corpus checks; preserve ordinary Windows setup, interactive, maintenance, archive and exact-byte joins.
- **FR-006**: Preserve verified prerequisite readiness, the 660-second advisory qualification target, 900-second hard limit, fifteen-second provisional reserve and terminal timestamp authority. Report preparation, qualification and combined elapsed time, including queue time.
- **FR-007**: Use hidden noninteractive Windows execution and owned disposable resources; remove temporary workflow files and retain canonical CI evidence without journals or raw scratch output in build or Downloads.
- **FR-008**: Maintain Spec Kit, linked GitHub tracking and a concise changelog entry; supersede the three original update PRs only after their intended changes are represented by a qualified replacement.
- **FR-009**: Automatically push and publish the S045 PR, address every received review comment, request no more than a second Codex review round, and stop before merging.
- **FR-010**: Keep audit #147 and historical incident #154 open for their outstanding criteria; dependency qualification does not establish real inference quality or the historical incident cause.

### Key Entities

- **Dependency selection**: Exact direct versions and resolved transitive versions for a package context.
- **Upstream source identity**: Module version/checksum and the relation between pristine source and a local patch.
- **Qualified candidate**: Exact source/run/attempt, verified assets, elected results and separate phase timings.
- **Superseded update**: Original PR, intended version changes and replacement tracking link.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every intended update from the three pending PRs has an explicit included or technically justified disposition; no unrelated downgrade or missing provenance is accepted.
- **SC-002**: Valid selected dependencies pass preparation and source verification, while missing or changed upstream identities fail meaningful regressions.
- **SC-003**: Required local checks and exact-head full hosted qualification pass, with terminal qualification within 900 seconds and all phase times recorded.
- **SC-004**: Every received review finding is resolved within at most two Codex rounds, original PR dispositions are reconciled, and the final PR remains unmerged for the owner.

## Assumptions

- One consolidated replacement PR is preferable to independently rebasing three stale branches and repeating the shared package campaign; original bot PRs remain open until replacement qualification.
- Adopt the requested versions, not an unbounded latest-version sweep. Preserve version 1.0.1 and Go 1.27.1.
- Release publication, owner-desktop GUI execution, new model acquisition, broad inference coverage and root-cause closure of #154 are outside this slice.

## Clarifications

### Session 2026-10-22

- Q: Separate refresh PRs or a consolidated slice? A: One consolidated replacement under S045, preserving attribution and closing superseded PRs only after qualification.
- Q: Does automatic publication need another approval? A: No. The owner explicitly authorized push and PR creation; merge remains owner-controlled.
- Q: May provenance checks be bypassed to accept the update? A: No. Reconcile the selected graph and upstream identities and retain rejection checks.
