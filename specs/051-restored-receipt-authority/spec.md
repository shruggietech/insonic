# Feature Specification: Restored receipt recording authority

**Feature Branch**: `codex/051-restored-receipt-authority`

**Created**: 2026-10-27

**Status**: Specified

**Input**: Owner-authorized S051 for #180, under #147: reject incomplete restored CPU recording authority, revalidate retained real evidence, reconcile S050 records and deliver an unmerged PR through CI and at most two Codex review rounds.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Refuse incomplete recording authority (Priority: P1)

A maintainer auditing restored CPU processing needs every recording and revision to be explicit so missing evidence cannot establish acceptance.

**Why this priority**: Missing source and destination identifiers currently compare equal, and a missing source revision receives an invented baseline.

**Independent Test**: Remove or corrupt required recording identities/revisions in otherwise complete restored evidence and require rejection; keep valid restored evidence accepted.

**Acceptance Scenarios**:

1. **Given** complete restored evidence, **When** a source recording identifier is absent, null, malformed, noncanonical or duplicated (including enrollment-only records), **Then** acceptance refuses that evidence.
2. **Given** complete restored evidence, **When** a source revision is absent, null, boolean, noninteger or nonpositive, **Then** acceptance refuses it without supplying a default.
3. **Given** valid source authority, **When** a restored identifier differs or a destination revision is not strictly newer than an explicit prior revision at least as new as the source revision, **Then** acceptance refuses that lineage.

### User Story 2 - Preserve legitimate evidence and its history (Priority: P2)

A maintainer can validate the retained real CPU result with the corrected checks while preserving the meaning of historical source-only receipts and the exact execution attribution.

**Why this priority**: Stronger refusal must not discard valid prior evidence or turn revalidation into a claimed new inference run.

**Independent Test**: Recover the retained receipt, verify its digest, validate it unchanged, and keep the existing source-only positive fixture passing.

**Acceptance Scenarios**:

1. **Given** the retained successful CPU receipt, **When** its exact bytes and frozen references verify, **Then** corrected acceptance passes and records its original package/harness identity, timing, eight fresh destination works, twelve retained history entries and cleanup.
2. **Given** a historical source-only receipt, **When** validated, **Then** its original contract remains applicable without restored-only authority requirements.
3. **Given** S050 records written before completion, **When** reconciled, **Then** measured results, owner merge, review scope and remaining work are stated accurately without retroactively inventing an unmerged final handoff.

### Edge Cases

- Both source and destination omit the same identifier, or share the same malformed value.
- Duplicate source identities involve only enrollment records, which have no fresh destination case.
- Text, lists, objects, booleans and floating-point revisions must not pass integer checks or cause an unrelated exception.
- A newer explicit prior revision is valid; unchanged, regressing or malformed destination revisions are refused.
- Historical raw bytes, compressed retention and reference identities disagree; do not rewrite evidence to manufacture a match.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Extended restored acceptance MUST require canonical recording identifiers for every source case, with no duplicates across its complete source inventory.
- **FR-002**: Extended restored acceptance MUST require explicit positive integer source revisions, excluding booleans and coercion/defaults.
- **FR-003**: Restored recording identifiers MUST equal their source authority, and explicit integer revisions MUST satisfy source revision <= prior revision < new revision.
- **FR-004**: Missing and malformed required authority MUST produce the maintained validation refusal; valid restored evidence MUST continue passing.
- **FR-005**: Historical source-only receipt acceptance MUST retain its existing meaning and must not acquire restored-only requirements.
- **FR-006**: Verification MUST recover and hash-check retained real evidence, validate its unchanged contents and preserve exact execution identities, phase timing and scope limits, distinguishing revalidation from new inference.
- **FR-007**: S050 records MUST accurately distinguish its completed campaign/CI/repair, outstanding validation work and owner merge before final convergence/handoff; S051 MUST close its own convergence gaps.
- **FR-008**: Delivery MUST pass relevant local checks and elected final-head CI, resolve every received review finding within two Codex rounds and provide an unmerged PR. Temporary verification inputs MUST be owned and removed after durable evidence retention.

### Key Entities

- **Source recording authority**: canonical recording identifier and explicit accepted revision for each source case, including enrollment records.
- **Restored lineage**: matching recording identity and explicit prior/new revisions bound to fresh terminal work.
- **Retained receipt**: immutable original CPU evidence with its digest, source/package/harness/reference identities and measured outcomes.
- **Verification record**: dated distinction between executed campaign, later evidence revalidation, CI, review and owner merge.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Both demonstrated omissions and every enumerated identity/revision mutation are rejected; valid restored and historical source-only positive controls pass.
- **SC-002**: One exact retained real receipt passes corrected validation with unchanged digest and execution attribution; all eight fresh destination and twelve preserved history entries remain accounted for.
- **SC-003**: S050 verification is reconciled, S051 convergence has no remaining implementation gaps, applicable checks pass and the owner receives an unmerged PR with resolved review findings.

## Assumptions

- This slice changes internal evidence validation and records; product inference, storage, routing, schemas, release version and ordinary deadlines keep their existing contracts.
- The retained CPU campaign already passed against a clean package/harness tree identical to merged S050. Revalidation is sufficient for this validator-only change; no new inference result is implied.
- #147 remains open for broader installed-product and quality coverage.

## Clarifications

### Session 2026-10-27

- Autopilot decision: validate the entire source inventory, including enrollment cases; otherwise duplicate or missing identities could escape the destination subset.
- Autopilot decision: retain exact canonical identifier format and integer types rather than normalize or coerce malformed evidence.
- Autopilot decision: preserve S050's original task history and add dated reconciliation; its owner merge preceded final convergence/handoff, so those historical events must not be fabricated.
