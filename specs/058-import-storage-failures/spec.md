# Feature Specification: Local import storage-failure recovery

**Feature Branch**: `codex/058-import-storage-failures`
**Created**: 2026-10-28
**Status**: Implemented; source acceptance and bounded reviews passed, awaiting owner merge
**Input**: Owner-authorized S058 for #147 under Spec Kit autopilot, including automatic push/PR, CI fixes, at most two Codex review rounds, and a stop before owner merge.

## User Scenarios & Testing

### User Story 1 - Refuse incomplete import authority (Priority: P1)

A library owner encounters exhausted or failing local storage during import. The operation must report failure rather than accept incomplete audio or capture metadata, and unrelated accepted recordings must remain usable.

**Why this priority**: Partial storage success must not become accepted library authority.
**Independent Test**: Induce bounded write, short-write and persistence failures at distinct staging/publication boundaries and independently inspect retained bytes and catalog state.

**Acceptance Scenarios**:

1. **Given** an isolated library with a previously accepted recording, **When** a new source cannot be fully staged, **Then** no new recording is accepted, original source bytes and the prior recording remain unchanged, and a failed outcome is observable.
2. **Given** staged source bytes, **When** canonical or capture-report publication cannot finish, **Then** no recording is admitted with incomplete or unverified artifacts, and cleanup does not remove unrelated accepted bytes.
3. **Given** an uncertain final publication outcome, **When** durable bytes exist but persistence confirmation fails, **Then** existing recovery authority is retained instead of treating that uncertainty as permission to discard completed bytes.

### User Story 2 - Complete one import after storage recovers (Priority: P1)

After write access or capacity returns, the owner can retry through existing work/publication contracts and obtain exactly one verified import.

**Why this priority**: Transient storage failure must remain recoverable without manual catalog repair.
**Independent Test**: Restore the controlled failure boundary and retry; independently compare accepted bytes, metadata/date provenance, publication identity and attempt history.

**Acceptance Scenarios**:

1. **Given** a failed or pending import, **When** the storage condition clears and the existing retry/reconciliation contract runs, **Then** exactly one correct recording becomes accepted with verified canonical/report bytes and preserved source provenance.
2. **Given** changed source bytes or corrupt surviving candidates, **When** recovery is attempted, **Then** the existing refusal/fencing rules remain effective and previously accepted authority is unchanged.

### User Story 3 - Trust bounded platform evidence (Priority: P2)

Maintainers can identify which storage failures were injected and which were observed through actual filesystem operations.

**Why this priority**: A synthetic disk-full error must not be represented as physical disk-exhaustion qualification.
**Independent Test**: Validate compact receipts from disposable Windows/Linux source and affected extracted-package checks, including a negative mutation matrix.

**Acceptance Scenarios**:

1. **Given** a bounded fixture, **When** a real filesystem publication is obstructed then restored, **Then** executed failure/recovery, image/platform identity and owned cleanup are required for acceptance.
2. **Given** injected short-write or disk-full evidence, **When** a receipt describes it, **Then** its injection layer and exercised boundary are explicit; missing or mislabeled evidence cannot establish actual filesystem exhaustion.

### Edge Cases

- Failure occurs after a partial write, during flush/final publication, or after completed bytes exist.
- Cleanup itself fails, cancellation races failure, or a prior attempt still owns a fence.
- The failed import shares storage with an unrelated accepted recording.
- Failure clears before retry; a repeated recovery observes already committed authority.
- Changed source or corrupt metadata/canonical candidates cannot inherit the prior operation's authority.
- Platform limitations prevent an actual capacity-limited filesystem fixture; injected evidence remains accurately labeled without claiming physical exhaustion.

## Requirements

### Functional Requirements

- **FR-001**: Import MUST reject incomplete staging, canonical and capture-report bytes before recording acceptance.
- **FR-002**: Storage failures MUST preserve unrelated accepted recordings, existing references and immutable owner source bytes.
- **FR-003**: Failure outcomes MUST remain observable through existing shared runtime/work results without false success or a new approval gate.
- **FR-004**: Completed but uncertain publications MUST retain sufficient existing authority for reconciliation; incomplete owned stage bytes MUST be cleaned up or explicitly retained with an observable cleanup failure.
- **FR-005**: After the failure clears, existing retry/reconciliation MUST produce one verified admission with correct bytes, metadata/date provenance and attempt authority.
- **FR-006**: Changed-source, corrupt-candidate, stale-attempt, immutable publication and isolation rules MUST remain effective.
- **FR-007**: Maintained coverage MUST include injected disk-full and short-write/persistence failures plus actual bounded filesystem refusal/recovery on Windows/Linux source and affected extracted-package paths.
- **FR-008**: Evidence MUST distinguish fault injection, actual filesystem errors and unexecuted physical exhaustion, binding executed image/platform, exercised boundaries, observed outcomes and owned cleanup.
- **FR-009**: Fixtures MUST remain confined and size-bounded, use hidden noninteractive children, and never exhaust an owner host volume or touch owner libraries.
- **FR-010**: Existing operation deadlines and qualification timing MUST remain intact. Current evidence requirements MUST preserve immutable published-package compatibility without weakening unknown/current candidate checks.

### Key Entities

- **Storage-failure observation**: Exact failing boundary, injected or operating-system origin, failure/uncertainty outcome and retry disposition.
- **Import authority**: Original source identity, durable work/attempt fence, canonical/report publications and accepted recording.
- **Qualification receipt**: Executed image/platform, controlled failure/recovery cases, immutable package bindings and cleanup observations.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every maintained storage-failure case prevents incomplete recording acceptance and preserves the independently hashed sentinel recording and original source.
- **SC-002**: Every cleared failure case completes one verified admission through normal recovery or explicit retry, with no duplicate accepted recording.
- **SC-003**: Required platform contexts report complete failure/recovery and cleanup evidence; omitted, altered, stale or mislabeled receipts fail validation.
- **SC-004**: Final source checks and all elected CI results pass within existing timing policy, with compact independently retained evidence and at most two completed Codex review rounds before owner review.

## Assumptions

- This slice advances #147; it is initially a coverage gap and claims product defects only after reproduction.
- Scope is local import staging/publication and admission. Backup/restore capacity, live object-service quotas, broad scale, real inference and physical power-loss durability remain separate.
- Existing publication/work recovery rules define retries and ordinary failures; no new product fault switch, persistent capacity service, migration or mandatory review is intended.
- Owner merge, release/tag/publication and owner-installed qualification remain outside this slice.

## Clarifications

### Session 2026-10-28

- Autopilot decision: cover injected storage-capacity/short-write failures and actual confined filesystem refusal/recovery separately. A real host-volume exhaustion campaign is unnecessary and would violate fixture confinement; evidence must state the boundary it actually exercised.
- Autopilot decision: preserve ordinary observed-failure retirement and uncertain-publication reconciliation rather than requiring every failed import to reuse candidate IDs. Reuse is required only where existing authority permits it.
