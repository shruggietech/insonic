# Feature Specification: Windows desktop startup and Exit reliability

**Feature Branch**: `codex/038-windows-webview-startup`

**Created**: 2026-10-19

**Status**: Owner-approved diagnostic checkpoint; final validation in progress

**Input**: Run S038 under autopilot for #154, push/open a PR, resolve up to two review rounds, and stop before owner review and merge.

## User Scenarios & Testing

### User Story 1 - Understand a failed native launch (Priority: P1)

A maintainer receives the actual native startup error and enough bounded evidence to distinguish a failed launch, failed native action, cancellation, timeout and output truncation without reconstructing unrelated CI failures.

**Why this priority**: The current failure loses the stream containing the native error, preventing an evidence-led correction.

**Independent Test**: A controlled child writes its meaningful failure on standard output and stack context on standard error; the parent retains both and accurately classifies the result.

**Acceptance Scenarios**:

1. **Given** a native child fails before its action, **When** qualification reports it, **Then** both output streams, the stage/action, exit code and truncation/deadline state remain available in bounded fixture diagnostics while successful machine output stays valid.
2. **Given** cancellation, timeout or excess output, **When** the parent returns, **Then** its failure classification and retained evidence describe that condition without reporting successful qualification.

### User Story 2 - Observe startup accurately (Priority: P1)

A maintainer can repeat real native launches on disposable Windows, compare retained historical bytes with a separately identified source GUI, and distinguish observed failures from an unreproduced historical cause.

**Why this priority**: The historical controller error was discarded. Bounded observations and retained evidence are needed before a supported startup correction can be selected.

**Independent Test**: Disposable Windows observations use distinct historical/source executables and actual native menu, close and Alt+F4 fixtures. Synthetic source tests cover output flooding, exact compiler environments, historical source authority and cleanup failure propagation.

**Acceptance Scenarios**:

1. **Given** an intermittent historical startup failure, **When** bounded focused comparisons do not reproduce it, **Then** record the actual outcomes and unknown cause without claiming a startup repair; #154 remains open.
2. **Given** repeated actual native fixtures with and without a workspace, **When** diagnostics and complete delivery qualification run, **Then** preserve native actions, normal GUI completion, playback release and durable-work assertions, keeping diagnostic success separate from package authority.
3. **Given** historical and derived diagnostic inputs, **When** source builds, child capture and cleanup run, **Then** preserve historical bytes, select exact source/compiler identity, bound streams while reading, exclude parent credentials from compiler children and confine cleanup to owned resources.

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
- **FR-003**: Provide bounded focused disposable Windows observations with distinct exact incident/source identities; record demonstrated facts and unreproduced cause explicitly. Keep #154 open for actual HRESULT/cause recovery and its supported correction.
- **FR-004**: Exercise bounded repeated actual Windows native menu, close and Alt+F4 diagnostics and preserve complete existing no-workspace/workspace/playback qualification, normal GUI exit and durable-work assertions. New cold/warm profile lifecycle qualification remains deferred under #154.
- **FR-005**: Confine diagnostic files/processes to owned disposable resources, join concurrent workers before removal, bound retained streams during reads and use exact compiler environments without parent credentials. Keep console children hidden/noninteractive and preserve production startup/profile/security behavior.
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
- **SC-002**: Focused disposable Windows evidence identifies actual input/tool revisions and repeat outcomes; historical bytes remain untouched, source observations use a separate GUI, and bounded-reader/compiler/source-authority regressions pass. Unreproduced startup cause remains explicit.
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

### Owner scope decision (2026-10-19)

After 124 focused native Exit child launches passed without recovering the controller error, the owner selected: "Deliver diagnostic checkpoint; keep #154 open." This supersedes the original requirement to demonstrate/correct the startup cause within S038. The checkpoint repairs lost diagnostics and validates the investigation boundary. Actual HRESULT/cause recovery, its supported startup correction and new cold/warm owned-profile lifecycle qualification remain tracked in #154. Passing PR diagnostics or delivery is not proof that the original intermittent defect is repaired.
