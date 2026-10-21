# Feature Specification: Verified CI readiness and elected acceptance

**Feature Branch**: `codex/041-ci-readiness-acceptance`

**Created**: 2026-10-20

**Status**: Authorized bounded Windows fixture calibration; final-head and post-merge acceptance pending

**Issue**: [#161](https://github.com/shruggietech/insonic/issues/161). Milestone 25 is assigned; the existing Project 7 item remains In Progress pending post-merge acceptance.

**Input**: Owner-directed Windows reliability repair, beginning with issue #161. Existing native incident #154 remains a separate outcome.

## Clarifications

### Session 2026-10-20

- Q: Does the 660-second requirement include cold preparation? A: Separate preparation and qualification, report both and total (owner decision).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reach qualification after cold preparation (Priority: P1)

A maintainer changing a source recipe or running with an empty or evicted cache receives verified prerequisite preparation before package consumers begin. A warm cache accelerates preparation without becoming an acceptance assumption.

**Why this priority**: Current consumers abandon valid cold producers after 180 seconds, preventing delivery tests from running.

**Independent Test**: Simulate a valid producer taking 1,342 seconds through the actual orchestration graph. Consumers must start after readiness, while the existing short transfer wait retains its bound.

**Acceptance Scenarios**:

1. **Given** exact cached inputs, **When** verification succeeds, **Then** qualification consumes current-run verified transfers.
2. **Given** missing, evicted or wrong-identity cached inputs, **When** preparation is needed, **Then** cold stages finish before package consumption begins.
3. **Given** failed or cancelled preparation, **When** downstream jobs are considered, **Then** consumers do not run and acceptance fails.
4. **Given** corrupt or stale transfers, **When** restoration occurs, **Then** existing identity and digest checks reject them.

### User Story 2 - Make merge acceptance represent elected delivery (Priority: P1)

A maintainer receives one required acceptance result that rejects any elected failed, cancelled or unexpectedly skipped check, including candidate verification. Prose-only changes explicitly record non-election.

**Why this priority**: Foundation and documentation alone do not establish package acceptance.

**Independent Test**: Enumerate full, prose-only and maintenance scopes and inject every non-success result into every elected job.

**Acceptance Scenarios**:

1. **Given** full election, **When** any required result is absent or unsuccessful, **Then** the aggregate fails.
2. **Given** documentation-only election, **When** all documentation checks pass and native checks are explicitly not elected, **Then** the aggregate accepts that limited scope.
3. **Given** an already exceeded qualification budget, **When** the aggregate evaluates results, **Then** acceptance fails despite successful individual checks.

### Edge Cases

- New compiler identities, PR versus main cache scopes, unavailable or cancelled producers, and rerun attempts.
- Warm and cold reusable paths include intentionally skipped internal jobs; only a verified terminal readiness result authorizes consumption.
- Documentation-only scope has no elected dependency preparation.
- Queue time, finalization and incomplete API inventories cannot be silently discarded.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Package consumers MUST require successful verified prerequisite readiness for the current revision/run/attempt.
- **FR-002**: Absent or invalid cache inputs MUST follow preparation or fail explicitly; cache availability MUST NOT establish correctness.
- **FR-003**: Preserve exact recipe/compiler/source validation, archive hashes, receipt rehashing, producer success, package joins and current-run/attempt transfer checks.
- **FR-004**: Preserve all native input, ownership, concurrent workload and outer product deadlines. The owner-authorized 2026-10-21 convergence may calibrate only the per-operation Windows fixture UI completion join from two to ten seconds; these latency policies are explicitly different.
- **FR-005**: One always-running aggregate MUST evaluate every elected top-level check, candidate and explicit non-election. Failures, cancellations and unexpected skips MUST reject acceptance.
- **FR-006**: Existing bootstrap configuration MUST name the aggregate as a required check; hosted application remains a separately authorized publication operation.
- **FR-007**: Dependency preparation and qualification MUST be reported separately with combined elapsed duration. Qualification MUST finish within 660 seconds, with final terminal timestamps authoritative and visible finalization margin.
- **FR-008**: Resolved source toolchain packages MUST be recorded as observation, without claiming a compiler display string fully pins the environment.
- **FR-009**: Regression coverage MUST prove scheduling behavior for warm/cold/missing/failed/cancelled prerequisites without extending the 180-second helper deadline or weakening transfer checks.

### Key Entities

- **Preparation**: Current-run source verification or cold compilation, ending in verified readiness.
- **Qualification**: Elected integrity, documentation, runtime, backend, package and candidate checks after preparation.
- **Acceptance receipt**: Exact revision, run/attempt, elected results, preparation/qualification/combined elapsed times, evaluation timestamp and finalization margin.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A producer taking longer than 180 seconds never starts a consumer before readiness in deterministic orchestration tests.
- **SC-002**: Every elected non-success result rejects aggregate acceptance; expected non-election remains explicit.
- **SC-003**: Qualification rejects elapsed time above 660 seconds and reports preparation and combined elapsed time without relabelling historical runs.
- **SC-004**: Exact-candidate cold and warm hosted results retain provenance and final timestamps before issue #161 is closed.

## Windows qualification convergence (2026-10-21)

### User Story 3 - Qualify a measured native fixture allowance (Priority: P1)

A maintainer observes the actual archive child desktop/input context and created WebView2 environment, compares two- and ten-second per-operation UI joins on the same instrumented executable, and receives complete final candidate acceptance without dropping native behavior.

**Acceptance scenarios**:

1. Given either arm, the original twelve-case cold/warm no-workspace/workspace sequence runs until its first failure, retaining successful and failed observations with monotonic boundaries and exact identity.
2. Given a ten-second arm that completes later than two seconds, native input, shutdown and natural release still pass inside unchanged outer limits before a calibration conclusion is supported.
3. Given two passing arms whose joins all finish within two seconds, the experiment is inconclusive about the earlier failure; adopting ten seconds is an explicit permitted test-policy choice, subject to complete final qualification.
4. Given any remaining functional or outer deadline failure, the run remains failed and a proportional correction must address the observed boundary. A targeted production/binding behavior correction receives a separate next-available dependent Spec Kit slice in this PR; broader redesign is outside this authorization.

- **FR-010**: Observe process session, window station flags/name, owned UI desktop/input state, foreground/visible/enabled ownership and optional API errors at preparation and before native input. Unknown optional diagnostic fields MUST NOT create product eligibility rules.
- **FR-011**: Capture the created environment BrowserVersionString/getter HRESULT once on its owner thread, separately from available-runtime lookup and environment-creation status, without enabling navigation-gating CDP. Retain runner, executable/package and dependency identities plus successful and failed operation/stage timings.
- **FR-012**: Run one sequential two-second baseline cohort and one ten-second comparison cohort with identical instrumentation, equivalent separate cold/warm profile histories, native input and the same assembled executable. Retain the first failure; no unchanged retry-until-green.
- **FR-013**: Ten seconds MAY be adopted as the explicit fixture policy after the bounded comparison and complete final-head qualification. Preserve five-second window search, eight-second readiness waits, twelve-second watchdog, eighteen-second child capture/release, hundred-second exit matrix, and 660-second qualification. Preparation failure MUST prevent input. Freeze final source before acceptance; no third hosted Codex review, merge, release, branch deletion or issue closure.

- **SC-005**: Retained baseline/comparison evidence identifies each effective allowance, all observed operation boundaries, desktop/runtime identity or explicit observation errors, every executed case and the first failed boundary; no diagnostic result substitutes for candidate acceptance.
- **SC-006**: The final ordinary PR run passes all elected package/runtime/documentation/integrity checks and all twelve archive native cases with natural release, with exact-attempt terminal qualification at most 660 seconds and phase/queue accounting. Prior-head compiled-media campaigns remain scoped historical evidence; final-head cold/warm and exact-main closure evidence stay tracked before #161 closure without imposing an unconditional repeat for this fixture-only repair.

## Assumptions

- The original slice addressed readiness/acceptance. The 2026-10-21 owner amendment adds bounded fixture observations/calibration. Broader native recovery remains separate; a directly demonstrated targeted behavior correction is permitted as a distinct dependent slice within PR #162.
- Existing verified source stages and same-run transfers provide the smallest mechanism; cross-run artifact import is unnecessary.
- No publication, hosted experiment, merge, release or incident closure is inferred from session guidance.

## Hosted convergence clarification (2026-10-20)

The owner authorized pushing S041, opening its PR, monitoring and fixing CI, stopping before merge. Separately timed preparation may compile standard and external Go library dependencies under the exact consumer environment. It must exclude every main-module package, executable and test. Optional cache warming cannot replace a verified same-run/attempt handoff. At that stage qualification retained all original product builds, native cases and timing allowances. The explicit 2026-10-21 amendment above supersedes only the two-second fixture join policy.
