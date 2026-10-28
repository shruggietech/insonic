# Feature Specification: Import interruption recovery

**Feature Branch**: `codex/057-import-interruption-recovery`

**Created**: 2026-10-27

**Status**: Implemented and verified; awaiting owner merge

**Input**: S057 for #147, following the approved recommendation for actual abrupt runtime termination/restart around import publication and catalog acceptance. Use Spec Kit autopilot, push/open the PR, resolve at most two Codex rounds, and stop before merging.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recover an interrupted import (Priority: P1)

An operator loses the runtime after it durably publishes an import report but before admitting the recording. Restart recovers the pending operation with original media and captured metadata intact.

**Why this priority**: Reconstructing recovery state alone does not demonstrate behavior across a real process boundary.

**Independent Test**: Terminate an owned disposable runtime at an observed publication/acceptance boundary, prove it exited, restart it, and inspect recovered bytes, recording identity, provenance and attempts.

**Acceptance Scenarios**:

1. **Given** a published report and no accepted recording, **When** its runtime ends abruptly and a replacement resumes the same operation/source bytes, **Then** exactly one recording is admitted and original/report bytes and captured metadata/date provenance remain valid.
2. **Given** successful recovery, **When** the operation is inspected and reconciled again, **Then** interrupted/replacement attempts remain distinguishable and no duplicate recording or current report is admitted.

### User Story 2 - Refuse mismatched recovered metadata (Priority: P1)

A source changes between interruption and recovery. Previously captured observations must never describe replacement bytes.

**Why this priority**: Recovery must preserve the binding between bytes and captured observations.

**Independent Test**: Change source bytes after actual runtime loss, restart, require refusal, and inspect publication/catalog authority.

**Acceptance Scenarios**:

1. **Given** a report bound to an interrupted source, **When** the source changes before restart, **Then** replacement bytes cannot inherit that report or become an accepted recording under it.
2. **Given** refused recovery or corrupted publication bytes, **When** accepted recordings and unrelated work are inspected, **Then** their bytes/provenance/authority remain unchanged and integrity failure cannot become successful recovery.

### User Story 3 - Trust repeatable qualification (Priority: P2)

Maintainers distinguish actual process recovery from reconstructed state, terminal-client interruption and physical power-loss claims.

**Why this priority**: Acceptance must remain reviewable across supported source/package paths.

**Independent Test**: Validate receipts from Windows/Linux source and extracted CLI/desktop-package CLI runs; reject incomplete boundaries, identities, outcomes or cleanup.

**Acceptance Scenarios**:

1. **Given** supported qualification, **When** recovery completes, **Then** receipts bind the executed image/platform/package to the observed boundary, actual termination, distinct replacement and independently checked result.
2. **Given** a missing checkpoint, early exit, stalled recovery, mismatched receipt or cleanup failure, **When** qualification ends, **Then** it fails without manufacturing a successful receipt.

### Edge Cases

- The selected runtime exits early, never reaches the checkpoint, or another process occupies its identity.
- A replacement starts before ordinary attempt expiry; existing fencing remains authoritative.
- Source content changes with the same length, or report/publication bytes are corrupted.
- Recovery fails after publication; bytes/history remain inspectable without stale acceptance.
- An earlier enclosing deadline or cancellation interrupts qualification.
- Paths include spaces/Unicode; unrelated runtimes/files and owner libraries are outside fixture ownership.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Qualification MUST observe durable report publication before recording admission, abruptly terminate the actual owned runtime, prove termination, and start a distinct replacement. Fabricated state, client Ctrl+C, graceful shutdown or an arbitrary sleep MUST NOT substitute.
- **FR-002**: Same-byte recovery MUST admit exactly one recording for the original operation, preserve source/report bytes and raw metadata/date provenance, retain distinguishable interrupted/replacement attempts, and remain duplicate-free on repeated reconciliation.
- **FR-003**: Changed-source recovery MUST refuse stale report admission, including equal-length substitution, without changing existing accepted recordings or unrelated work.
- **FR-004**: Success MUST require independent post-restart catalog/publication/byte checks rather than command exit alone. Publication corruption MUST remain an integrity failure.
- **FR-005**: Checks MUST cover Windows/Linux source and both extracted package variants using isolated owned profiles, workspaces and inputs. Owner installations/production libraries MUST NOT be changed.
- **FR-006**: Intervention MUST remain internal to qualification without an ordinary product command, new user recovery gate or changed production deadline/lease. Windows children MUST remain hidden/noninteractive.
- **FR-007**: Cleanup MUST cover success, checkpoint failure, interruption, replacement failure, timeout and cancellation. Only fixture-owned paths/processes may be removed/stopped. Successful receipts require final runtime/resource release and owned fixture removal.
- **FR-008**: Receipts MUST identify executable/platform/source or package, scenario inputs, observed boundary, terminated/replacement runtime identities, attempts, publications/admission and cleanup. Missing, altered, stale or incomplete evidence MUST reject qualification and package/release acceptance.
- **FR-009**: Nested checks MUST share existing enclosing test budgets and preserve separate preparation/qualification/combined accounting. Existing functional/integrity/provenance/elected checks remain required. Process-interruption evidence MUST NOT claim physical power-loss durability or unexecuted acoustic/installed-product behavior.

### Key Entities

- **Interrupted import**: One operation with captured source/report identities and no accepted recording at the boundary.
- **Runtime attempt**: A process/session and fenced work attempt, distinguishable before/after replacement.
- **Recovery observation**: Independently read publication bytes, recording authority, provenance and attempts.
- **Qualification receipt**: Source/package-bound scenario observations and final owned-resource cleanup.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All six source/package contexts demonstrate abrupt replacement and same-byte recovery with exactly one recording and complete provenance.
- **SC-002**: Equal-length changed-source and corrupted-publication negatives never produce falsely accepted recovery.
- **SC-003**: Maintained negatives detect missing checkpoints, premature exit, wrong identity, timeout and cleanup failures without affecting unrelated files/processes.
- **SC-004**: Exact final PR-source CI passes every elected check; compact independently audited receipts remain outside workflow storage. All actionable comments are addressed within at most two Codex rounds, and the PR stays unmerged for owner review.

## Assumptions

- #147 stays open as the broader audit; this slice covers one bounded recovery dimension.
- Filesystem/SQLite/LadybugDB defaults define this campaign. Existing alternative-backend support and fixtures remain required.
- Reuse existing capture recovery and attempt fencing; repair demonstrated defects proportionally rather than hiding them behind retries.
- Physical power loss, disk exhaustion, broad scale, owner-installed behavior and acoustic quality remain separate verification dimensions.
- Push and PR publication are explicitly authorized. Merge/tag/release publication are outside this turn.

## Clarifications

### Session 2026-10-27

No critical ambiguities require owner input. Use the agreed publication/acceptance boundary, disposable automated fixtures and ordinary production leases. Choose deterministic boundary observation rather than timed termination, with intervention confined to qualification.
