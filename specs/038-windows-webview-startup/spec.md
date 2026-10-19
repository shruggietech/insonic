# Feature Specification: Windows desktop startup and Exit reliability

**Feature Branch**: `codex/038-windows-webview-startup`

**Created**: 2026-10-19

**Status**: Draft

**Input**: Run S038 under autopilot for #154, push/open a PR, resolve up to two review rounds, and stop before owner review and merge.

## User Scenarios & Testing

### User Story 1 - Understand a failed native launch (Priority: P1)

A maintainer receives the actual native startup error and enough bounded evidence to distinguish a failed launch, failed native action, cancellation, timeout and output truncation without reconstructing unrelated CI failures.

**Why this priority**: The current failure loses the stream containing the native error, preventing an evidence-led correction.

**Independent Test**: A controlled child writes its meaningful failure on standard output and stack context on standard error; the parent retains both and accurately classifies the result.

**Acceptance Scenarios**:

1. **Given** a native child fails before its action, **When** qualification reports it, **Then** both output streams, the stage/action, exit code and truncation/deadline state remain available in bounded fixture diagnostics while successful machine output stays valid.
2. **Given** cancellation, timeout or excess output, **When** the parent returns, **Then** its failure classification and retained evidence describe that condition without reporting successful qualification.

### User Story 2 - Start and exit repeatedly (Priority: P1)

The desktop starts reliably across repeated automated cold and warm launches, then responds to its real menu, window close and Windows Alt+F4 while preserving independently owned work and releasing GUI playback resources.

**Why this priority**: A controller startup failure currently blocks delivery before the Exit action can be tested.

**Independent Test**: Disposable Windows qualification observes repeated actual packaged launches, native actions, normal process completion, playback release and surviving durable work.

**Acceptance Scenarios**:

1. **Given** the failing native boundary, **When** a focused diagnostic reproduces it, **Then** the native error and a demonstrated cause are recorded before selecting the correction; assumptions remain explicitly unproven.
2. **Given** repeated cold and warm fixture launches with and without a workspace, **When** the actual menu, ordinary close and Windows Alt+F4 routes run, **Then** each GUI terminates normally and required playback/durable-work assertions pass without a callback substitute or forced-kill success.
3. **Given** a fixture-specific profile or child process, **When** cleanup runs, **Then** it affects only owned resources and cannot overwrite or remove an operator profile or unrelated process.

### User Story 3 - Preserve trustworthy delivery (Priority: P2)

A maintainer can accept a fully qualified repair without reducing installer recovery, source identity or publication checks.

**Why this priority**: A green individual probe cannot replace complete exact-byte delivery qualification.

**Independent Test**: The complete elected workflow verifies matching sources/assets, every required delivery lane and refusal of incomplete evidence within the existing time budget.

**Acceptance Scenarios**:

1. **Given** a failed, missing or foreign qualification proof, **When** delivery joins results, **Then** candidate/publication remains blocked and the primary failed scope is identified.
2. **Given** the repaired exact PR head, **When** required local and hosted checks complete, **Then** all elected gates pass within 660 seconds and the PR stops unmerged for owner review.

### Edge Cases

- Native errors emitted on either stream; nonzero exit with empty output; cancellation versus child timeout; bounded output exhaustion.
- Controller failure before readiness, successful GUI exit with surviving browser descendants, profile release delays, repeated same-profile launches and unrelated processes sharing an executable name.
- Unknown/foreign/redirected fixture paths, absent evidence, already closed resources, simultaneous source/package probes on separate runners.
- Partial proof artifacts and secondary missing joined receipts after a primary failure.

## Requirements

### Functional Requirements

- **FR-001**: Retain bounded failed-child stdout and stderr plus stage/action, exit code, truncation and cancellation/timeout state in fixture diagnostics; preserve successful JSON and exclude protected inputs.
- **FR-002**: Cover the lost-stdout defect and result classifications with meaningful failing-first regressions.
- **FR-003**: Recover the native controller error, demonstrate the cause of the startup failure, and apply a proportional correction supported by that evidence.
- **FR-004**: Exercise actual repeated cold/warm Windows native menu, close and Alt+F4 launches, including no-workspace and workspace/playback cases; preserve normal exit and durable-work survival assertions.
- **FR-005**: Confine any profile/process lifecycle changes to demonstrably owned resources, observe release before reuse/removal, keep console children hidden/noninteractive, and preserve production security configuration.
- **FR-006**: Preserve source/asset binding, ZIP/MSI/setup/recovery gates, complete evidence inventories, and rejection of failed/missing/foreign proof.
- **FR-007**: Keep CI within 660 seconds through targeted diagnosis and efficient existing checks; do not bypass native actions, retry blindly, or inflate deadlines to claim a fix.
- **FR-008**: Keep source diagnostics, packaged execution, PR results and actual post-merge main results distinct. Stop before merge; actual main confirmation belongs to authorized post-merge follow-up.

### Key Entities

- **Native child observation**: Stage/action, separate bounded streams, native result, cancellation/deadline and truncation state, readiness and ownership observations.
- **Owned launch resources**: A fixture's profile path/process identities and observed release lifecycle, independent from operator state.
- **Delivery proof**: Existing immutable source/run/attempt/asset identity and successful checks; failed diagnostics confer no publication eligibility.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A failed child with error-only stdout and stack-only stderr retains both, and all classified negative cases remain failures.
- **SC-002**: Focused disposable Windows evidence identifies the native error and cause; its correction passes a relevant regression and repeated cold/warm native launch routes.
- **SC-003**: Every elected native Exit scenario preserves independently owned work and releases GUI playback resources with no fallback success.
- **SC-004**: No operator profile, unrelated process or publication integrity gate is changed by qualification.
- **SC-005**: Required repository/affected checks and complete exact-head hosted delivery pass in at most 660 seconds; up to two review rounds are handled with every finding resolved before unmerged owner handoff.

## Assumptions

- Source baseline is d44d95b6e97d741a32cbf87323bf78c824c3155b; issue #154 describes the observed main failure, not a proven profile collision.
- The Windows native GUI and installer probes run only on acknowledged disposable CI; no owner-desktop GUI or MSI installation is required.
- No release, merge, product policy change or installer architecture expansion is authorized by this slice.

## Clarifications

### Session 2026-10-19

- Q: Is profile collision the required fix? A: No. Preserve the native error and demonstrate the cause first; select the correction from evidence.
- Q: How does post-merge confirmation fit the stop boundary? A: Verify the exact PR head now, document actual main verification as a post-merge follow-up, and do not merge autonomously.
- Q: Are extra full-CI reruns a diagnostic strategy? A: No. Use a focused disposable Windows probe before complete qualification.
