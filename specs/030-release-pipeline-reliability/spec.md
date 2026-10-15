# Feature Specification: Release pipeline reliability

**Feature Branch**: `codex/030-release-pipeline-reliability`

**Created**: 2026-10-15

**Status**: Specified

**Input**: Resume S030 under autopilot, push the official PR, resolve at most
two review rounds, verify green CI, then stop for owner final review and merge.

## User Scenarios & Testing

### User Story 1 - Recover transient bootstrap downloads (Priority: P1)

A maintainer runs qualification or release preparation without manually
restarting the pipeline for a transient toolchain-action download.

**Why this priority**: Several attempts failed before checkout while acquiring
the same pinned setup-go action, despite later successful qualification.

**Independent Test**: Simulate two transient failures followed by the approved
archive. Confirm bounded recovery, approved bytes and preserved cache behavior.

**Acceptance Scenarios**:

1. **Given** a transient download failure, **When** bootstrap retries, **Then**
   it verifies the same approved revision before executing it.
2. **Given** corrupt content, unsafe members or an unapproved redirect, **When**
   acquisition finishes, **Then** execution is refused without partial output.
3. **Given** an older release source, **When** current automation qualifies it,
   **Then** bootstrap tooling comes from the workflow's trusted exact revision
   and the selected product source remains unchanged.

### User Story 2 - Verify published release and deployed docs (Priority: P1)

A maintainer runs repeatable, read-only verification after publication and
documentation promotion and obtains explicit identity and content evidence.

**Why this priority**: Publication, successful deployment and matching live
responses are separate claims that need consistent verification.

**Independent Test**: Verify a complete fixture, then corrupt its tag, asset
inventory, checksum, deployment identity, alias and served resource separately.

**Acceptance Scenarios**:

1. **Given** a published candidate and matching docs, **When** verification runs,
   **Then** it confirms version, source, complete inventory, original archive
   bytes, latest entry points and referenced public resources.
2. **Given** a matching build marker but stale page or resource, **When**
   verification runs, **Then** it identifies the mismatching path and fails.
3. **Given** network failure, **When** bounded recovery is exhausted, **Then**
   verification fails without publishing, rebuilding or moving tags.

### Edge Cases

- Permanent access failures are not retried as transient outages.
- Truncated, oversized, duplicate, linked or escaping archive members fail.
- Credentials never accompany public downloads or appear in diagnostics.
- Historical product sources may predate new automation scripts.
- Provider failures before checkout remain an explicit residual limitation.
- Release and docs identity must remain coherent during concurrent deployment.

## Requirements

### Functional Requirements

- **FR-001**: Acquire approved Go and Node setup actions with at most three
  attempts and a total acquisition deadline of 60 seconds per action.
- **FR-002**: Require immutable revision and content digest, bounded archive
  sizes, confined regular members and atomic availability before execution.
- **FR-003**: Preserve setup inputs, normal action execution and cache post
  steps on Windows and Linux qualification hosts.
- **FR-004**: Historical source qualification uses bootstrap tooling from the
  workflow revision, independently of selected product source.
- **FR-005**: Provide a reusable read-only verifier for an explicit published
  release and original verified documentation archive.
- **FR-006**: Verify tag/source/version, complete asset names, sizes and digests,
  checksum inventory and original publication-target documentation.
- **FR-007**: Compare live documentation entry pages, latest aliases, schemas,
  navigation targets and referenced local resources with original archive bytes.
- **FR-008**: Verify live content after successful exact Pages promotion; expose
  the same verifier for an explicitly configured HTTPS site.
- **FR-009**: Bound requests, redirects, response sizes and total verification
  time; reject credential-bearing and unapproved destinations.
- **FR-010**: Preserve qualification, provenance, source-check, same-attempt
  artifact and security gates. Final-head CI remains within 660 seconds.
- **FR-011**: Document proof levels, limits, recovery and provider limitations;
  keep slice evidence in internal planning.

### Key Entities

- Approved action: repository, immutable revision, digest and size bound.
- Bootstrap attempt: number, deadline, outcome and atomic destination.
- Publication identity: repository, tag, version, source and full inventory.
- Verification receipt: selected identity, checked paths, proof levels and time.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Transient acquisition succeeds within three attempts; permanent,
  corrupt and exhausted fixtures fail within the documented bound.
- **SC-002**: Windows and Linux qualification, including action cache post
  steps, pass on the exact final PR head within 660 seconds.
- **SC-003**: Every identity, inventory or content mismatch fixture fails;
  read-only live verification confirms the existing v1.0.1 publication and docs.
- **SC-004**: Publication verifies live docs after exact deployment without
  rebuilding original archives or weakening publication gates.

## Assumptions

- Existing v1.0.1 tag, release and deployed documentation remain unchanged.
- Earlier main CI repair is complete; S030 adds reliability tooling.
- New landing-page issue #101 belongs to a separate slice.
- The owner authorizes push, official PR and at most two review rounds.

## Clarifications

### Session 2026-10-15

- Q: Retry complete workflows or action acquisition? A: Acquire pinned actions
  at step time. Never retry application failures or mix attempt-bound artifacts.
- Q: Download every binary again for each docs check? A: Reuse the existing
  verified candidate. Report public binary identity as API metadata evidence;
  original docs and live responses are downloaded and hashed.
