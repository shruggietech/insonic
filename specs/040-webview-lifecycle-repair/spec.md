# Feature Specification: Windows desktop lifecycle repair

**Feature Branch**: `codex/040-webview-lifecycle-repair`

**Created**: 2026-10-19

**Status**: Accepted for investigation and repair

**Input**: Owner-directed post-merge housekeeping, followed by full deconstruction and repair of recurring Windows desktop startup and shutdown failures in #154.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reach a usable desktop consistently (Priority: P1)

An operator opens a fresh or previously used desktop, with or without a selected library, and reaches usable controls without an unexplained blank or stalled window.

**Why this priority**: Startup failures prevent every desktop operation and repeatedly block package qualification.

**Independent Test**: Exercise fresh and reused desktop profiles with and without a library, retain the startup phase evidence, and verify the existing readiness deadlines and actual frontend interaction.

**Acceptance Scenarios**:

1. **Given** a fresh desktop profile and no selected library, **When** the operator opens the desktop, **Then** the document, application controls and native actions become ready within existing qualification deadlines.
2. **Given** a reused profile with a selected library and playback setup, **When** the operator opens the desktop, **Then** the same readiness contract holds without altering durable library data.
3. **Given** a failed launch, **When** qualification records it, **Then** the first incomplete boundary is distinguishable from controller creation, asset processing, native response completion and frontend readiness failures, without exposing content or credentials.

### User Story 2 - Close the desktop and release its resources (Priority: P1)

An operator closes through File/Exit, the window close control or Alt+F4. The desktop and its owned browser processes finish naturally while separately owned durable work remains available.

**Why this priority**: Surviving browser processes prevent deterministic maintenance and can contaminate later launches.

**Independent Test**: Exercise all three actual native actions, verify their callbacks, host exit and live owned-process release, then verify shared work remains available.

**Acceptance Scenarios**:

1. **Given** a ready desktop, **When** each supported native exit action occurs, **Then** the original action and callback are observed, the host exits normally and all owned browser processes release within the existing deadline.
2. **Given** durable shared-runtime work and desktop playback, **When** the desktop exits, **Then** desktop playback handles release and independently owned work survives.
3. **Given** a timeout or cleanup failure, **When** forced cleanup is required, **Then** the case remains failed and its pre-cleanup observations remain authoritative.

### User Story 3 - Receive an evidence-backed repair (Priority: P1)

The maintainer can explain the failing contract and the resulting correction, using a bounded reproduction and exact source/package evidence, without substituting more passing repetitions for a cause.

**Why this priority**: Repeated speculative changes and full-pipeline retries have consumed substantial time and cost without resolving every signature.

**Independent Test**: Review the source responsibility map, failure inventory, rejected hypotheses, controlled failing regression and corrected behavior, followed by the normal candidate qualification gates.

**Acceptance Scenarios**:

1. **Given** retained incident records, **When** a cause is proposed, **Then** its evidence, counterexamples, applicable failure signatures and unresolved limits are stated independently.
2. **Given** an experiment proposal, **When** it is executed, **Then** it records a falsifiable prediction, control, fixed stop condition, actual result and next decision before another run is elected.
3. **Given** a repair candidate, **When** final verification completes, **Then** the tested source and package identities match, all required checks pass within 660 seconds, and #154 closes only if its acceptance scope is demonstrated.

### Edge Cases

- Fresh versus reused profiles, no library versus selected library, cold launch versus concurrent CLI work.
- Controller failure before navigation, an accepted navigation with no completed document, and a returned document handler with an unfinished native response.
- Delayed or failed UI dispatch, failed response setters, incomplete asynchronous work, native callbacks after shutdown begins and partial initialization.
- A zero-exit desktop with live owned browser descendants; stale job membership or unobservable processes must not be misrepresented as a live-process result.
- Native menu cancellation, focus restoration and reopening before Exit; no callback substitution or ambient input targeting.
- Runtime ownership, profile locks, linked native interfaces and clean setup/maintenance must remain distinct from release gates and source acquisition failures.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Preserve the completed S039 implementation and verified squash-merge identity while performing housekeeping.
- **FR-002**: Maintain one failure inventory separating controller creation, pre-document startup, UI-operation join, menu consumption and browser-release signatures; connect signatures only with demonstrated evidence.
- **FR-003**: Map each responsible layer and its relevant startup/response/shutdown paths to exact source versions and primary framework contracts before changing behavior.
- **FR-004**: Preserve the existing readiness, native-input, action/callback, natural-release and source/package identity requirements and deadlines.
- **FR-005**: Every behavior change MUST address a demonstrated contract violation and have a regression that fails before correction and passes afterward.
- **FR-006**: New observations MUST distinguish handler return, final response publication and asynchronous completion; observation alone MUST NOT count as a repair.
- **FR-007**: Failure evidence MUST be bounded and limited to fixed stages, statuses, counters, timings, function names and verified owned-process metadata; exclude credentials, media, URLs, request/response bodies and private paths.
- **FR-008**: Test launchers MUST remain hidden and noninteractive; actual focus/input experiments MUST run only in a disposable hosted desktop, and unshown local native regressions MUST not steal focus.
- **FR-009**: Each hosted experiment MUST have a written hypothesis, expected discriminating outcomes, control and stopping rule; do not repeat an identical experiment merely because it passed or rerun full packaging to collect startup evidence.
- **FR-010**: Shutdown MUST release the owned controller and retained interfaces on their owning thread, complete or reject pending work explicitly, and preserve independently owned jobs.
- **FR-011**: Natural-release verification MUST inspect only the launched ownership tree and retain failure observations before forced cleanup.
- **FR-012**: Update #154 with material findings and independently distinguish proven, rejected, candidate and unresolved conclusions.
- **FR-013**: Preserve third-party source identity, patch allowlists, notices and corresponding-source packaging whenever dependency bytes change.
- **FR-014**: Complete root, affected native/product and site verification and the configured candidate checks on the final source, targeting the constitutional 660-second budget through scope and caching.
- **FR-015**: Keep release version 1.0.1 synchronized; no release, product installation or site publication is part of this repair.
- **FR-016**: Do not trigger a third external review on completed PR #156 or change its historical native-CI exception. A repair remains a separate reviewable change.

### Key Entities

- **Failure signature**: The first demonstrated failed boundary, exact source/run/attempt and pre-cleanup observations; does not imply a shared root cause.
- **Responsibility map**: Ordered ownership and callback transitions among the product host, desktop framework, native binding, operating system and browser.
- **Experiment record**: Prediction, control, bounds, immutable inputs, observations, conclusion and next decision.
- **Repair evidence**: A reproduced violated contract, failing regression, corrected behavior and final qualification identities.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All five retained failure-signature classes have a source-mapped explanation or an explicitly unresolved boundary; no signature disappears from the record.
- **SC-002**: Every delivered behavior correction has a reproduced failure before correction and a passing meaningful regression afterward.
- **SC-003**: The corrected desktop passes cold/warm, with/without-library and all three actual native-exit cases with natural process release under unchanged deadlines; repetitions supplement the causal regression and do not replace it.
- **SC-004**: Final configured checks are green on the exact repair head and complete within 660 seconds; inherited S039 exceptions do not establish repair qualification.
- **SC-005**: The issue and repair report identify every performed check, observed result and remaining unverified condition, with zero claim that a finite passing sample proves absence of a rare failure.

## Assumptions

- S039 merged as c9ed48d682e255f9e03f9d4ffd1ebbb9801f9c20; its product tree exactly matches the accepted final head.
- The owner's renewed instruction authorizes investigation and repair of #154 and supersedes the earlier S039 diagnostic stop for this new work. It does not authorize merging a future repair PR or publishing a release.
- S040 is the next unused sequential slice. The scope is #154; unrelated CLI and dependency-update issues remain separate.
- Primary framework documentation and exact pinned sources govern the audit. Newer releases are compared for relevant fixes, without assuming an upgrade resolves the incident.

## Clarifications

### Session 2026-10-19

- No owner questions are needed: scope, intended behavior, evidence standards, hidden execution and merge/publication boundaries follow the renewed request and current project authority.
- The historical S039 native-CI exception applies only to that completed handoff. S040 requires actual repair evidence and normal passing gates.
