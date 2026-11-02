# Feature Specification: S067 Publication recovery and timing guidance

**Feature Branch**: `codex/067-publication-recovery`

**Created**: 2026-11-01

**Status**: Specified

**Input**: Proceed with S067 (#240 and #241) using Spec Kit and autopilot; push and open the PR, resolve reviews within two rounds, and stop for owner review and merge.

## User Scenarios & Testing

### User Story 1 - Recover an uncertain Pages promotion (Priority: P1)

A release operator can identify the exact attempted deployment after a lost or disagreeing response and determine whether the original accepted documentation deployed without repeating publication.

**Why this priority**: An uncertain response currently requires manual reconstruction of the attempted commit.

**Independent Test**: Inject delayed old-head readbacks, a lost update response and an unrelated writer; inspect durable receipts and the resulting deployment outcome.

**Acceptance Scenarios**:

1. Given accepted original documentation, when promotion reaches its branch update, then durable evidence already identifies the previous and staged commit, original source and tree.
2. Given an old-head or lost response followed by the exact staged head, when bounded reconciliation succeeds, then the exact original tree and completed build are required for success and original observations remain recorded.
3. Given an unrelated branch writer, when readback observes its identity, then promotion fails without force, rollback or another branch update.
4. Given a previously completed exact deployment and retained attempted identity, when read-only reconciliation runs, then it verifies destination, original files, commit ancestry and build without any remote writes.
5. Given persistent disagreement, malformed responses or deadline exhaustion, when promotion fails, then its last durable state and failure category remain available without credentials.

### User Story 2 - Understand actual publication budgets (Priority: P2)

An operator can read current release and website guidance and predict which limit governs each operation.

**Why this priority**: Stale guidance misstates Pages, website, Git and live-verification allowances.

**Independent Test**: Compare every current publication timing claim to maintained policy and executed callers, and build the documentation.

**Acceptance Scenarios**:

1. Given maintained timing policy, when current guidance is audited, then request, Git, phase, wrapper and cleanup ceilings match their executed callers.
2. Given qualification reporting, when guidance is read, then separate preparation, qualification and combined accounting, target 1,800 seconds, hard limit 3,600 seconds and provisional reserve 60 seconds remain distinct from watchdog ceilings.

### Edge Cases

- A successful update loses its response; an old-head read does not prove failure or staleness.
- An exact commit has the wrong parent/tree, or its completed build belongs to another commit.
- The branch stays at its old head, disappears, returns malformed data or changes after an initial exact match.
- Receipt storage fails before mutation; a later failure must not destroy the last complete receipt.
- A retry cannot replace the original attempted receipt or repeat a deployment through the read-only command.

## Requirements

### Functional Requirements

- **FR-001**: Preserve durable source, destination, previous-head, staged-commit, staged-tree and original-archive identities before the branch mutation, and persist observations and terminal failures.
- **FR-002**: Reconcile only bounded old-head/unavailable observations to the exact staged identity under the original enclosing deadline; reject unrelated commits and malformed identities.
- **FR-003**: Require the original complete file inventory, parent relationship, destination policy and exact completed build for recovered success. Never force, roll back or restage during reconciliation.
- **FR-004**: Expose read-only reconciliation using retained attempted evidence and the original candidate; revalidate publication and original bytes, and retain the original failure evidence.
- **FR-005**: Correct current operator timing prose against executed policy without changing qualification acceptance limits or immutable release assets.
- **FR-006**: Preserve existing destination, source, tree, package, publication and security checks. Record sanitized diagnostic categories and validated identities, never credentials or signed URLs.

### Key Entities

- Promotion attempt: original candidate identity, elected destination, prior/staged Git identities, ordered observations, phase and outcome.
- Reconciliation result: independently verified exact deployment linked to the retained attempt.
- Timing guidance: operation, maintained ceiling, enclosing deadline and acceptance authority.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every tested mutation attempt has a readable prior identity receipt, including all injected failures.
- **SC-002**: Delayed/lost-response fixtures recover only the exact original deployment; all concurrent-writer and altered-byte fixtures fail without additional destination mutation.
- **SC-003**: Read-only recovery issues zero remote writes and leaves original failed evidence unchanged.
- **SC-004**: All audited current publication timing claims match executed policy; repository, affected publication and site checks pass.
- **SC-005**: One PR links both issues, has no unresolved actionable reviews after at most two rounds, and green CI before owner merge handoff.

## Assumptions

- The missing original S066 disagreeing response prevents proving a stale-read root cause; fixtures demonstrate supported recovery behavior only.
- Existing publication destinations and authentication remain unchanged. No product feature, release, tag, production deployment or replacement of v1.1.0 assets is included.
- Current-site correction uses the ordinary reviewed website path after merge, with deployment authorization evaluated then.

## Clarifications

### Session 2026-11-01

- Q: Should recovery repeat publication? A: No; recovery is read-only and requires retained exact identities and original bytes.
- Q: Should an arbitrary disagreeing head be retried? A: No; only the elected previous head or unavailable response can be retried. An unrelated valid head is a concurrent-writer refusal.
- Q: Does this authorize release or website deployment? A: The slice authorizes implementation, push and PR, stopping before owner merge. Published bytes stay unchanged.
