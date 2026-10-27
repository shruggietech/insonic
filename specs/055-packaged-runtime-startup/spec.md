# Feature Specification: Windows packaged runtime startup diagnosis and repair

**Feature Branch**: `codex/055-packaged-runtime-startup`

**Created**: 2026-10-27

**Status**: Implemented

**Input**: S055 for #191, autonomous implementation, push, PR, CI repair and up to two Codex review rounds. Stop before merging.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Explain a refused runtime launch (Priority: P1)

Maintainers distinguish actual startup producer failures from unavailable ownership or fixture failures without exposing private inputs.

**Why this priority**: The Windows archive campaign stopped before its JSON running barrier; earlier passes do not explain the refusal.

**Independent Test**: Inject a known startup failure and recover correlated evidence after owned fixture cleanup.

**Acceptance Scenarios**:

1. **Given** elected CI terminal qualification, **When** runtime startup fails, **Then** retain the allowlisted producer stage, native numeric cause when available, request/work/startup/session and client identities, observed owner liveness and actual client exit.
2. **Given** failures containing private values, **When** evidence is retained, **Then** omit raw paths, arguments, inputs, endpoints, environment and credentials, preserving the public generated response.
3. **Given** concurrent campaigns, **When** observations are collected, **Then** each belongs to its actual client and request, without adopting another receipt.

### User Story 2 - Repair the demonstrated defect (Priority: P1)

Actual Windows terminal clients can wait for durable work using the live owner, while legitimate hidden startup remains available.

**Why this priority**: Diagnostics alone do not establish repaired operation.

**Independent Test**: A cause-specific regression fails before the repair and passes afterward; corrected Windows archive qualification completes eight terminal cases.

**Acceptance Scenarios**:

1. **Given** cause-complete evidence, **When** a proportional repair is applied, **Then** the focused regression proves corrected behavior and preserves discovery, owner lifetime and startup ownership.
2. **Given** interactive, JSON, quiet and redirected waits, **When** work completes, fails, times out, is interrupted or explicitly cancelled, **Then** existing progress/count, mode suppression, natural exit and durable outcome requirements hold.
3. **Given** campaign failure, **When** cleanup finishes, **Then** the result stays failed and owned clients/runtime/server/terminals/fixtures are released within existing limits.

### User Story 3 - Review exact-source delivery (Priority: P2)

The owner receives an open unmerged PR with final-head CI, retained failed evidence and resolved external findings.

**Independent Test**: Read final head, terminal checks and every review thread; report preparation, qualification and total separately.

**Acceptance Scenarios**:

1. **Given** a failed attempt, **When** a full rerun is needed, **Then** retain a compact independent failure audit first.
2. **Given** green checks and at most two Codex rounds, **When** handed off, **Then** the PR stays unmerged and merged-main acceptance remains pending owner action.

### Edge Cases

- Failure precedes readiness or log creation; numeric native cause is unavailable.
- Owner exits after readiness, discovery becomes stale, or a client starts a competing owner.
- Child exit precedes final output drain; cleanup also fails.
- Receipts are stale, malformed, concurrent, missing or oversized.
- Error wrappers contain private data; enclosing deadline expires.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Diagnose #191 using bounded correlated observations before owned cleanup; separate established causes from unknowns.
- **FR-002**: Retain allowlisted producer stage/native numeric cause without private values or arbitrary error text.
- **FR-003**: Preserve public startup responses, ordinary startup/operation deadlines, detached owner lifetime and hidden noninteractive child creation.
- **FR-004**: Repair a demonstrated cause with a failing-before/passing-after regression. A passing repetition alone is insufficient.
- **FR-005**: Preserve all eight actual-terminal cases, running barriers, progress/counts, suppression, durable continuation/cancellation, natural exits and cleanup.
- **FR-006**: Bind source/package/executable identity; isolate all fixtures and evidence from owner libraries and unrelated campaigns.
- **FR-007**: Retain 1800-second target, 3600-second hard limit after verified preparation, 60-second finalization reserve and separate phase/combined accounting. Do not inflate clocks for this functional failure.
- **FR-008**: Complete relevant checks and exact-head CI, retain failures before full reruns, resolve findings within two Codex rounds and stop before merge.

### Key Entities

- **Startup observation**: Generated producer stage, numeric native cause availability/value, platform and client identity.
- **Terminal failure evidence**: Executed digest, request/work/startup/session identities, observed owner/client state, bounded generated events and cleanup outcome.
- **Qualification attempt**: Exact revision, original artifact identities, terminal outcomes, readiness boundary and separate elapsed phases.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Injected failure yields attributable startup cause evidence after fixture removal; private sentinels are absent.
- **SC-002**: Demonstrated cause has a failing-before/passing-after regression, and corrected Windows desktop archive qualification completes all eight cases without relaxed assertions/deadlines.
- **SC-003**: Final applicable checks pass and actionable findings are resolved within two review rounds; PR remains unmerged.
- **SC-004**: Acceptance reports bind actual source/image identities and preparation, qualification and combined time.

## Assumptions

- Main failure is durably retained on PR #190. Public `startup_launch` does not identify its native cause.
- #185 and published v1.0.2 requalification are out of scope; #147 remains broad tracking.
- Native installer/UI tests run on disposable CI. Local tests use owned fixtures and hidden console children.
- Push, PR and review comments are authorized. Owner retains merge authority; merged-main verification follows later.

## Clarifications

### Session 2026-10-27

- Requirements are clear; unknown causes are research tasks, not owner scope questions.
- Choose cause-driven diagnosis over unconditional retry, larger clocks or removed barriers; preserve public responses and keep CI evidence separate.
