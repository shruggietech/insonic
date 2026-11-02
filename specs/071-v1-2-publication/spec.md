# Feature Specification: v1.2.0 publication and independent verification

**Feature Branch**: `codex/071-v1-2-publication`

**Created**: 2026-11-02

**Status**: Specified

**Input**: Owner-authorized S071 following merged S070, including publication, automatic push and official PR, at most two review rounds and owner-controlled final PR merge. No new tracking issue.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Obtain the accepted release (Priority: P1)

A user downloads v1.2.0 built from the accepted merged source, with all required supported packages and consistent versioned documentation and contracts.

**Why this priority**: Preparation has completed; users need verified official product downloads.

**Independent Test**: Independently retrieve every published file and compare its identity, size and digest with the accepted retained candidate.

**Acceptance Scenarios**:

1. **Given** green accepted merged source, **When** the publication candidate is qualified, **Then** all required supported package and installer checks pass and publication-specific documentation identifies the same source/version.
2. **Given** the complete accepted candidate, **When** publication occurs, **Then** the exact unchanged bytes become the official release at an immutable matching tag with concise highlights and a changelog link.
3. **Given** the official release, **When** an independent readback downloads all files, **Then** the complete inventory, sizes and digests match the retained candidate.

### User Story 2 - Read matching live documentation (Priority: P1)

A reader sees the published v1.2.0 documentation, working product downloads and retained historical documentation at the existing website.

**Why this priority**: Release downloads and instructions must describe the same product.

**Independent Test**: Verify the exact deployed snapshot, all maintained content/resource checks and eight landing-page rendering cases.

**Acceptance Scenarios**:

1. **Given** a verified published release, **When** documentation is promoted, **Then** the original accepted documentation archive is deployed to the existing destination without rebuilding or removing historical routes.
2. **Given** completed deployment, **When** independent public verification runs, **Then** source/version, content, resources and product links agree and sampled desktop/mobile light/dark rendering passes.
3. **Given** an uncertain publication response, **When** recovery inspects remote state, **Then** only exact matching retained identities establish completion; conflicting state fails without an inferred success or silent rollback.

### User Story 3 - Review a durable release record (Priority: P2)

The owner receives a green evidence PR with durable verification records, resolved reviews and precise remaining limitations.

**Why this priority**: A successful publication must remain independently auditable after temporary workflow artifacts expire.

**Independent Test**: Inspect retained receipts, their readback integrity, final PR checks and every review thread.

**Acceptance Scenarios**:

1. **Given** completed publication and verification, **When** evidence is recorded, **Then** it distinguishes product source, tooling source, candidate acceptance, public asset readback and live deployment.
2. **Given** automatic review and any findings, **When** delivery finishes, **Then** all actionable findings are resolved within at most two rounds, applicable final CI is green and the PR remains for owner merge.

### Edge Cases

- Changed main, conflicting tags, missing files or invalid provenance block publication until resolved against explicit source authority.
- A candidate-status documentation build is not publication-specific documentation.
- Interrupted draft upload cannot establish public availability; uncertain Pages updates reconcile only to the exact staged commit and inventory.
- Rebuilding the same source may change archive bytes; accepted publication bytes must remain unchanged.
- Expired workflow artifacts cannot replace retained evidence. Capture original failed outcomes before any full rerun.
- Qualification target overruns are advisory; functional, integrity, provenance, elected checks and the hard limit remain mandatory.
- Missing security review, signing or real-model evaluation remains explicitly absent rather than inferred from other checks.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Publish version 1.2.0 from accepted S070 merged source, preserving source/version consistency across product, documentation and master contracts.
- **FR-002**: Independently qualify a complete publication-target candidate across Windows/Linux x86-64 CLI and desktop packages, Windows MSI/setup and required release checks. Preserve accepted bytes unchanged.
- **FR-003**: Retain complete candidate identity, original files, artifact digests and terminal qualification evidence before publication or another full run. Record preparation, qualification and combined timing under the existing 1800-second target, 3600-second hard limit and provisional 60-second reserve policy.
- **FR-004**: Publish the immutable matching tag and verified complete official asset set with concise highlights ending in a changelog link; interrupted or conflicting state must not be reported as successful publication.
- **FR-005**: Promote the original verified release documentation to the existing website, retaining historical routes and the exact deployment identity without changing routing or configuration.
- **FR-006**: Independently download every official asset, compare its bytes with the retained candidate, and verify complete maintained public content/resources plus eight landing rendering cases.
- **FR-007**: Retain compact source-bound audited receipts outside workflow artifact retention, recording performed checks and unverified signing, model-quality or security-review claims separately.
- **FR-008**: Deliver Spec Kit artifacts and publication records in one official PR with final applicable CI green, every actionable review resolved within at most two rounds, and owner-controlled final merge.
- **FR-009**: Create no tracking issue and make no unrelated speaker-discovery, dependency, version or product behavior changes. Existing credentials and authorized publication destinations persist.

### Key Entities

- **Accepted source**: Reviewed merged revision, release version and successful prerequisite checks.
- **Publication candidate**: Complete immutable assets, source/version, documentation target and qualification receipts.
- **Official release**: Immutable tag, published complete asset inventory and concise highlights.
- **Deployment**: Exact parented destination revision, original documentation identity and completed build.
- **Verification record**: Independent downloads, content/render receipts, timing and explicit limitations.
- **Evidence PR**: Spec Kit artifacts, review rounds, final checks and owner merge boundary.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All ten official release files match the accepted candidate by name, size and digest, with four portable packages and both Windows installer forms.
- **SC-002**: All required release checks pass within the hard qualification limit; any target overrun is separately reported.
- **SC-003**: Live documentation identifies v1.2.0 and the exact accepted deployment, all maintained content/resource checks pass, and all eight sampled landing cases pass.
- **SC-004**: Retained source-bound evidence can be independently read back and verified outside temporary workflow storage.
- **SC-005**: Final PR has no unresolved actionable review findings within two rounds, green applicable checks and a concrete owner handoff.

## Assumptions

- Accepted product source is S070 squash 61a5ba35e2b388965ac316412e5a380e6e99b258; its main CI has passed. S071 planning/evidence commits do not replace product source.
- The owner authorized the recommended release publication and independent verification scope, including the existing documentation destination; no new permission gate is needed for routine steps.
- No new real-model CPU/NVIDIA evaluation or signing configuration is elected. Existing deterministic and native required qualification remains mandatory.
- Speaker-discovery issues #211 and #227-#233 remain outside this slice. No tracking issue is created.

## Clarifications

### Session 2026-11-02

No critical ambiguity requires an owner question. Scope, identities, lifecycle, recovery, integrations, quality requirements, terminology and completion criteria are clear under the accepted recommendation, constitution and existing release contracts.
