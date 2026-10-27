# Feature Specification: S052 Terminal progress and interruption acceptance

**Feature Branch**: `codex/052-terminal-interruption`

**Created**: 2026-10-27

**Status**: Implemented (owner review pending)

**Input**: Owner authorizes S052 for #147 using Spec Kit/autopilot, automatic push/PR, CI repairs, at most two Codex rounds and no merge. Adopt the bounded terminal campaign recommended in the audit closeout.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Observe real terminal progress (Priority: P1)

An operator waiting on durable work sees changed state, stage and available counts in an interactive terminal. Maintainers can verify that behavior unattended on both supported platforms.

**Why this priority**: Redirected-output checks and a supplied terminal flag do not establish actual terminal delivery.

**Independent Test**: Hold owned deterministic work at a known barrier, observe progress before completion, release it and require the successful final result.

**Acceptance Scenarios**:

1. **Given** an actual terminal and held durable work, **When** a human wait starts, **Then** it emits the correlated running state/stage before the barrier releases and the observed terminal state afterward, without invented percentages.
2. **Given** JSON, quiet or redirected output, **When** the same wait completes, **Then** informational progress is absent and the selected final-output contract remains intact.

### User Story 2 - Interrupt waiting without cancelling work (Priority: P1)

An operator can press Ctrl+C to stop waiting while the same durable operation remains independently observable and can finish.

**Why this priority**: Client lifetime and accepted durable work have different ownership.

**Independent Test**: Deliver Ctrl+C through the owned terminal after a running observation, require natural client exit and an interrupted-wait result, independently observe the same work still running, release it and require success.

**Acceptance Scenarios**:

1. **Given** a running terminal wait, **When** terminal Ctrl+C is delivered, **Then** the client naturally exits nonzero within its enclosing fixture allowance, reports wait interruption and leaves the same work active.
2. **Given** a separate held work item, **When** explicit work cancellation is requested, **Then** its durable state becomes cancelled and a subsequent wait fails with that observed state.
3. **Given** held or failed work, **When** the wait budget expires or observes failure, **Then** timeout/failure remains distinct from interruption and success, retaining the last observed state.

### User Story 3 - Retain trustworthy platform evidence (Priority: P2)

Maintainers receive bounded receipts identifying the executable and terminal mechanism actually exercised, with verified fixture cleanup.

**Why this priority**: Neither simulated tests nor one platform can silently substitute for unexecuted terminal behavior.

**Independent Test**: Run the maintained source and extracted-package qualification on Windows and Linux; reject incorrect identity, missing scenarios, overflow, timeout and forced-exit observations.

**Acceptance Scenarios**:

1. **Given** a completed terminal campaign, **When** its receipt is accepted, **Then** every required case has a natural outcome and exact executable identity, with parent package provenance preserved where applicable.
2. **Given** success, failure or interruption, **When** fixture teardown finishes, **Then** all owned terminal channels, children, runtime and temporary files are released without a visible or focus-stealing window; failed cleanup cannot claim acceptance.

### Edge Cases

- Ctrl+C before initialization is not a valid interrupt proof; delivery follows the correlated running barrier.
- Terminal transport controls are separated from product output; truncated or overflowing captures fail acceptance.
- JSON and quiet output remain controls even when their handles are terminals.
- A timed-out wait leaves durable work active; explicit cancellation has its own evidence.
- A forced client termination is cleanup only and cannot count as natural exit.
- Missing terminal support fails the elected platform scenario explicitly; no pipe-only fallback.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Verify actual terminal human progress for owned durable work on Windows and Linux, including changed running/terminal states and available count detail.
- **FR-002**: Deliver Ctrl+C through the terminal and verify nonzero natural client exit with interrupted-wait meaning after observing running work.
- **FR-003**: Independently prove interruption preserves the same active work identity and its later successful completion; verify explicit durable cancellation separately.
- **FR-004**: Verify JSON, quiet and redirected-output progress suppression plus timeout and observed failed-work outcomes.
- **FR-005**: Integrate the campaign into maintained source and extracted-package acceptance, binding executable bytes and preserving existing package/source provenance.
- **FR-006**: Bound capture size and all waits by the maintained enclosing qualification budget; retain ordinary product deadlines and intentionally short failure clocks.
- **FR-007**: Guarantee hidden noninteractive fixture execution, isolated owned data and natural-outcome evidence. Verify owned resource cleanup on all paths; forced cleanup cannot promote a failed case.
- **FR-008**: Retain compact platform-qualified receipts and mutation/lifecycle regressions, complete final-source CI/reviews, and leave #147 open for remaining audit dimensions.

### Key Entities

- Terminal observation: terminal kind/platform, correlated work, selected output mode, bounded normalized product output, natural exit and elapsed time.
- Campaign receipt: exact executable digest, required scenario outcomes and cleanup proof; inherited package/source identity remains in its parent receipt.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every elected Windows/Linux source/package campaign passes all required scenarios; unavailable terminal support, missing proof or forced termination produces failure.
- **SC-002**: Each interrupted wait exits within its effective fixture allowance while independently observed durable work retains identity and completes after release; explicit cancellation remains cancelled.
- **SC-003**: Receipt mutation and teardown regressions reject absent identity, false success, incomplete cleanup and capture overflow. All elected final-source CI checks and at most two Codex rounds complete before owner handoff.

## Assumptions

- Existing documented wait semantics define intended behavior; fix only demonstrated gaps. This is initially a coverage gap, not an asserted product defect.
- Use owned barrier-controlled model acquisition and existing runtime contracts; no external credentials, operator library, GUI input or actual inference is required.
- The supported Windows/Linux native CI lanes provide actual terminal facilities. Local execution qualifies only the current host.
- Shared test timing remains 1800s target/3600s hard limit after verified readiness, with separate preparation/qualification/combined accounting and terminal timestamp authority.
- Broad acoustic quality, GPU/OCR, external services, power-loss/disk/scale and owner-installed release behavior remain outside this slice.
