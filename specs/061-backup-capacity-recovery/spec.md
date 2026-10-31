# Feature Specification: Backup/restore capacity-failure recovery

**Feature Branch**: `codex/061-backup-capacity-recovery`

**Created**: 2026-10-31

**Status**: Implemented and reviewed; final branch-revision verification tracked in PR #201

**Input**: S061, continuing #147: exercise backup/restore capacity failures, repair demonstrated gaps, preserve accepted authority, and prove recovery through source and packaged paths.

## Clarifications

### Session 2026-10-31

- Autopilot clarification found no critical ambiguity. Existing request identity, backup pins, restore fencing and platform contracts govern retries; testing must distinguish native error injection from actual confined filesystem observations. All ten specification-quality criteria remain satisfied.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recognize an incomplete backup (Priority: P1)

An operator creating a portable backup receives an actionable failure when the destination cannot accept more bytes. An incomplete bundle must not be reported as verified or replace an existing backup.

**Why this priority**: A successful backup is a claim that recoverable authority and its required bytes are present.

**Independent Test**: Fail each backup write boundary, inspect completion and source integrity, restore capacity, and retry the same request.

**Acceptance Scenarios**:

1. **Given** accepted source data, **When** object, catalog or manifest output cannot complete, **Then** creation fails, no completed bundle is admitted, and accepted source authority remains unchanged.
2. **Given** a write or flush failure, **When** cleanup also fails, **Then** the failure remains observable, the primary capacity cause remains identifiable, and residual owned staging is not mistaken for a completed backup.
3. **Given** recovered destination capacity, **When** the same request is retried, **Then** exactly one complete bundle is verified and capture pins reconcile under existing rules.

### User Story 2 - Recover an interrupted restore (Priority: P1)

An operator restoring into an independently configured empty workspace can retry after capacity returns without losing recovery identity or activating an incomplete destination.

**Why this priority**: Restored authority must remain fenced until its bytes and catalog are accepted.

**Independent Test**: Interrupt destination publication, progress persistence and activation, then retry the original bundle and profiles.

**Acceptance Scenarios**:

1. **Given** a verified self-contained bundle, **When** destination bytes or progress cannot be persisted, **Then** restore fails without activating incomplete authority and preserves the bundle bytes.
2. **Given** catalog restore committed but activation failed, **When** another bundle or changed profile attempts takeover, **Then** existing restore fencing rejects it; an identical retry can reconcile after capacity returns.
3. **Given** an occupied destination or existing output, **When** a new operation is attempted, **Then** existing authority is preserved and an ordinary conflict remains distinguishable from capacity failure.

### User Story 3 - Trust bounded acceptance evidence (Priority: P2)

A maintainer can distinguish controlled fault injection, actual confined filesystem observations and ordinary successful backup/restore execution in qualification receipts.

**Why this priority**: Automated evidence must describe the failures actually exercised and the subsequent recovery.

**Independent Test**: Validate receipts from Windows and Linux source and packaged contexts, including failure classification, authority checks, retry and owned cleanup.

**Acceptance Scenarios**:

1. **Given** a bounded failure campaign, **When** qualification completes, **Then** every required case has an explicit result and checks the intended failure and recovery rather than accepting any nonzero exit.
2. **Given** a receipt missing a required case or integrity check, **When** aggregation runs, **Then** it rejects incomplete evidence.

### Edge Cases

- Failure during open, write, short write, flush, close, final publication, progress replacement or activation.
- Disk full and quota errors wrapped by operating-system or shared contract errors.
- Cleanup fails after an earlier capacity failure; no original error is silently discarded.
- Existing destination, symlink/traversal input, tampered recovery progress, reference-only pin lifetime and cancellation retain their existing contracts.
- An interrupted operation leaves recoverable pins or progress; retry must not duplicate accepted objects or bypass ownership.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Backup/restore MUST retain an actionable `insufficient_space` cause for recognized disk-full and quota failures through wrapped errors, without exposing paths, credentials or raw provider responses.
- **FR-002**: A short or incomplete write MUST fail before its output becomes verified authority. Flush and close errors MUST remain failures.
- **FR-003**: Failed backup creation MUST preserve accepted source data and existing output. Incomplete owned staging MUST be cleaned when possible; cleanup failure MUST remain observable alongside the primary cause.
- **FR-004**: Restore MUST preserve verified bundle bytes, existing destination authority and recovery fencing across publication, progress and activation failures.
- **FR-005**: After capacity returns, an identical ordinary retry MUST complete correctly with existing request, pin, progress and activation reconciliation rules. Changed inputs MUST NOT inherit prior authority.
- **FR-006**: Capacity classification MUST NOT change normal conflicts, checksum errors, cancellation, unsupported capabilities or admission authorization.
- **FR-007**: Qualification MUST cover bounded backup and restore failure boundaries and successful retry on Windows/Linux, with explicit source/package identities and required case validation.
- **FR-008**: Receipts MUST distinguish injected failures from actual filesystem observations, retain integrity and cleanup results, and use the maintained preparation/qualification/combined timing contract.

### Key Entities

- **Backup bundle**: Manifest, portable catalog and immutable bytes bound to one backup request and source capture.
- **Restore attempt**: Bundle and destination identity, retained publication progress and fenced catalog/configuration activation.
- **Failure receipt**: Exercised boundary, observation method, classified cause, integrity observations, retry outcome and owned cleanup result.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every required failure case refuses false completion and preserves all checked source, bundle and pre-existing destination identities.
- **SC-002**: Every recognized disk-full or quota case reports `insufficient_space`; short writes and cleanup failures cannot produce success.
- **SC-003**: Each recoverable case completes an identical retry after the fault clears and verifies restored accepted bytes and recovery authority.
- **SC-004**: Required Windows/Linux source and package receipts pass strict validation, with no owned runtime or temporary-directory leak and no missing required cases.

## Assumptions

- S061 is a bounded continuation of #147; it does not publish another release or merge its PR.
- Shared runtime changes serve CLI and desktop callers. Dedicated desktop backup controls remain separately scoped.
- Existing supported storage/catalog alternatives retain parity; fixture injection does not claim live-provider quota or physical disk-exhaustion qualification.
- Tests use owned fixtures and confined failure mechanisms. Exhausting the owner's host volume, power-loss experiments, broad scale and new inference campaigns are outside this slice.
- The owner has authorized push, PR publication and at most two Codex review rounds; final merge remains owner-controlled.
