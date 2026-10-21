# Feature Specification: Maintained CLI regression corpus and audit coverage

**Feature Branch**: `codex/044-cli-regression-corpus`
**Created**: 2026-10-21
**Status**: Implemented; hosted acceptance and review tracked in PR #167
**Input**: S044 autopilot kickoff, following the recommended fixture inventory and coverage-ledger slice. Primary issue #114; audit reconciliation #147.

## User Scenarios & Testing

### User Story 1 - Locate reproducible maintained fixtures (Priority: P1)

A maintainer can identify the fixture, origin, generator and semantic expectations behind a qualification scenario without reconstructing prior audit scratch files.

**Why this priority**: Reusable, attributable inputs are the foundation for reliable coverage.
**Independent Test**: Validate the complete fixture inventory and demonstrate deterministic generation of selected bounded inputs.

**Acceptance Scenarios**:
1. **Given** a clean checkout, **When** the fixture inventory is inspected, **Then** each fixture has a unique identity, source or generator, provenance/license and meaningful expected values or byte checks.
2. **Given** a stale reference, duplicate identity, escaping path or changed pinned asset, **When** inventory validation runs, **Then** it fails with a bounded diagnosis.

### User Story 2 - Understand actual command and option coverage (Priority: P1)

A maintainer can inspect coverage for every registered command and accepted option, and distinguish successful workflows, intentional failures and behavior that was not exercised.

**Why this priority**: Help/parser acceptance alone cannot establish operational success.
**Independent Test**: Compare the coverage ledger with the current command/option authority and run deterministic scenarios against a relocated installed package.

**Acceptance Scenarios**:
1. **Given** a current package, **When** qualification completes, **Then** every current command and accepted option has an explicit coverage disposition linked to its scenario and fixture, including aliases and global controls.
2. **Given** only a classified missing-prerequisite invocation, **When** the ledger is emitted, **Then** it records expected failure and cannot claim the successful operation ran.
3. **Given** a newly added option, removed scenario, unknown fixture or mismatched package identity, **When** coverage is checked, **Then** drift or unsupported claims are rejected.

### User Story 3 - Preserve bounded package qualification and truthful audit status (Priority: P2)

A maintainer can run the existing installed-package qualification from spaces/Unicode paths, retain its actual observations in CI and remove owned temporary resources on success or failure.

**Why this priority**: Coverage improvements must remain useful on supported platforms without creating an unbounded parallel harness.
**Independent Test**: Exercise targeted missing coverage in the existing Windows/Linux source and package paths, then inspect cleanup and audit dispositions.

**Acceptance Scenarios**:
1. **Given** owned disposable profiles, libraries and loopback providers, **When** selected scenarios finish or fail, **Then** servers, credentials, durable synthetic work, leases and owned runtime/image handles are released and scratch outputs are deleted.
2. **Given** a passing deterministic run, **When** audit status is reconciled, **Then** exact package execution is distinguished from parser checks, synthetic lifecycle checks and unverified inference/service/owner-installed behavior.

### Edge Cases

- Zero/missing/duplicate fixture or scenario IDs; stale command/option references; renamed aliases; global options and value-bearing options.
- Escaping or symlinked fixture references, BOM/mojibake, non-ASCII paths, bounded byte/character limits and invalid generated input.
- An expected failure changes its code/reason, an assertion never ran, a helper raises before cleanup, or a runtime/lease remains live.
- External authenticated providers, real model inference, unavailable hardware and non-elected services remain explicitly unverified.

## Requirements

### Functional Requirements

- **FR-001**: Maintain one compact versioned fixture inventory with unique IDs, sources/generators, provenance/license, bounded generation and checksum or semantic expectations; retain only reusable fixtures and scenario mappings in source.
- **FR-002**: Coverage MUST account for every current registered command and accepted option, including aliases/global controls, using the application's current command authority rather than a second independently maintained command list.
- **FR-003**: Coverage MUST distinguish observed success, observed expected failure and unverified behavior, identify the actual scenario/assertion and fixture behind a claim, and reject drift, unknown references, duplicate identities and unsupported success claims.
- **FR-004**: Executed coverage MUST bind to the actual executable, companions, platform and package/source identity. Static inventory/parser/schema checks MUST NOT be reported as execution of the corresponding operation.
- **FR-005**: Reuse existing semantic-help, media, HTTP, model, trainer, integrity, readiness and wait scenarios; fill demonstrated gaps and compose shared generators proportionally instead of creating a parallel qualification framework or duplicating existing matrices.
- **FR-006**: Installed qualification MUST cover owned spaces/Unicode paths and relevant human/quiet/verbose/debug/JSON output, streams, exit categories, exact values and durable causes through meaningful existing or targeted scenarios.
- **FR-007**: Fixtures MUST contain only constructed or already licensed committed inputs and dummy credentials. Coverage output MUST be bounded and exclude owner media, real credentials, arbitrary tool output and owner-specific local paths.
- **FR-008**: Windows children MUST be hidden and noninteractive. All temporary profiles, libraries, provider servers, dummy registrations, work, leases and image handles MUST have owned success/failure cleanup; do not retain agent journals, raw run transcripts or scratch archives in build or Downloads.
- **FR-009**: Preserve all elected functional/integrity/provenance checks and the current readiness boundary, 660-second advisory target, 900-second hard limit and fifteen-second provisional reserve. Report preparation, qualification and total time separately, using terminal timestamps for final claims.
- **FR-010**: Document inventory use and explicit verification limits; reconcile #147 with current closed defects and executed coverage without claiming its unverified services, inference, hardware or historical Windows incident are repaired.

### Key Entities

- **Fixture**: Stable ID, purpose, source/generator, provenance/license and byte/semantic expectations.
- **Scenario**: Stable ID, existing qualification owner, fixture references, command/option relationships and assertions.
- **Coverage observation**: Command/option, scenario, actual outcome and verification boundary, bound to an executed artifact identity.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every maintained fixture/scenario reference validates; corrupt, escaping, duplicate and stale cases fail meaningful regressions.
- **SC-002**: Every current command and accepted option has an explicit validated disposition; every executed-success claim has a corresponding successful asserted observation.
- **SC-003**: Supported installed package paths pass targeted deterministic regressions from spaces/Unicode paths; cleanup succeeds on normal and failed execution without owner-desktop GUI activity.
- **SC-004**: Required local checks and exact-head hosted qualification pass within the current hard limit; received review findings are resolved within at most two Codex rounds and the PR is handed off unmerged.
- **SC-005**: Audit reconciliation and maintainer guidance distinguish actual execution, synthetic contracts and all remaining unverified boundaries without preserving ephemeral scratch reports.

## Assumptions

- #114 is the primary deliverable; #147 receives coverage/status reconciliation and retains any outstanding verification limits. #154, dependency upgrades and release publication are outside S044.
- Existing package qualification and its current 168-command guidance are the starting point; command counts may evolve and must be derived rather than frozen.
- Keep version 1.0.1 and existing configured routes/access authorization. Add no per-item or attended owner verification gate.
- No critical clarification is needed: classify unavailable/non-elected capabilities honestly, preserve mandatory elected checks and reuse the current qualification paths.
