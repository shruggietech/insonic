# Feature Specification: Original release revalidation

**Feature**: S056 | **Issue**: #185 | **Created**: 2026-10-27 | **Status**: Specified

Restore read-only verification of the original published v1.0.1 packages using their exact release authority. Later verification requirements must not invalidate authentic historical evidence or become optional for current candidates.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Revalidate the original publication (Priority: P1)

A maintainer selects the complete original v1.0.1 candidate and its exact source and receives a verified result without rebuilding, executing historical code or changing published bytes.

**Independent test**: Validate all four original product archives, the MSI and both documentation archives against the original manifest and selected source.

**Acceptance scenarios**:

1. Given the authentic original candidate and matching complete source, verification accepts its original receipt and MSI formats without demanding later terminal or setup-EXE evidence.
2. Given only restored schemas, missing source files, changed source or a different source revision, verification refuses historical acceptance.

### User Story 2 - Reject altered history (Priority: P1)

A maintainer must receive a failure when any original artifact, receipt, source, inventory, signing declaration or installer table changes, even if local checksums have been recomputed.

**Independent test**: Mutate each authority boundary, including a failed Windows first-use receipt and an MSI table byte, and require rejection.

### User Story 3 - Preserve current release requirements (Priority: P1)

Maintainers verifying current or unknown later candidates continue to require their full modern evidence and installer format.

**Independent test**: Remove current terminal, setup or installer lifecycle evidence, or attempt to reuse historical identity with another candidate, and require rejection.

### Edge Cases

Reject duplicate assets, missing members, changed manifest/checksum files, source/receipt version disagreement, incomplete source, symlinks replacing required source files, and receipts claiming historical identity with different bytes. Preserve the existing v1.0.0 historical platform contract. Historical files are data, including build helpers; they never supply executable validation logic.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Select the original-v1.0.1 compatibility profile only for version 1.0.1, source 136f5c0307a6fce00ba8240ef69c907a944cd8c0, its complete exact source tree and the original immutable manifest. Unknown identities retain modern verification.
- **FR-002**: Verify every tracked selected-source file against trusted historical source authority, including recipes, pins, helpers, schemas, worker metadata, licenses and branding. Missing or changed bytes fail; no historical code executes.
- **FR-003**: Require all applicable original package qualification, Windows first-use, distribution, signing and source/inventory/archive integrity checks. The original profile does not require evidence introduced after publication.
- **FR-004**: Verify the original MSI and its qualified payload receipt/inventory against immutable published byte identities and applicable original lifecycle evidence. Changed MSI tables, embedded payloads or declarations fail without using the current MSI model as the historical format.
- **FR-005**: Apply the same profile authority across candidate loading, published-candidate retrieval and public verification. A nested verifier must not infer a weaker profile from a version string alone.
- **FR-006**: Retain all modern terminal, coordinated setup, interactive/transaction/documentation and source requirements for current and unknown candidates. Preserve existing v1.0.0 handling.
- **FR-007**: Revalidation is read-only with respect to original tags, releases, assets and selected source. Downloaded/staged data and all child processes follow existing bounded, hidden, noninteractive execution rules.
- **FR-008**: Document the historical verification contract and report checked identities, performed checks, unverified runtime behavior and qualification timing truthfully.

### Key Entities

- **Historical profile**: Trusted release version, source revision/tree, manifest and installer byte identities and original qualification rules.
- **Selected source**: Complete immutable tracked file contents; never executable verifier authority.
- **Candidate**: Original manifest, checksums, four product archives, MSI and two documentation archives.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The complete authentic original publication passes maintained read-only verification without asset changes or historical code execution.
- **SC-002**: Every specified negative authority/qualification case fails, including changed MSI tables and failed first-use.
- **SC-003**: Current evidence-omission regressions and existing historical compatibility checks pass without widened exemptions.
- **SC-004**: Required local checks and exact-head CI pass; compact audited evidence and at most two resolved Codex review rounds precede owner handoff.

## Assumptions

Only the original published v1.0.1 identity needs this additional compatibility profile. New historical releases require a reviewed profile. This slice does not publish a product release, change the software version, alter original assets or establish installed-product runtime behavior.

## Clarifications

### Session 2026-10-27

Autopilot resolved the implementation choice: anchor the historical exception to the original manifest/MSI and complete source identity. A generic version-based exemption or execution of old verifier code would weaken authority. No owner decision remains unresolved.
