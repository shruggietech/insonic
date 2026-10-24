# Feature Specification: Comprehensive test timeout expansion and qualification validation

**Feature Branch**: `codex/047-test-timeout-expansion`

**Created**: 2026-10-24

**Status**: Implemented; hosted acceptance and review pending

**Input**: S047 for [#172](https://github.com/shruggietech/insonic/issues/172). Expand all maintained test allowances, automatically publish the PR, complete CI and at most two Codex review rounds, and stop before merge.

## User Scenarios & Testing

### User Story 1 - Let slow healthy checks complete (Priority: P1)

Maintainers need healthy cold-start and hosted-runner checks to survive their former short cutoffs without an earlier nested timeout defeating the expanded allowance.

**Independent Test**: A healthy delayed operation succeeds past the former bound; a noncompleting operation still expires.

**Acceptance Scenarios**:

1. **Given** a slow healthy check, **when** it crosses the former bound, **then** it continues through the expanded allowance and returns immediately when complete.
2. **Given** nested readiness, child and matrix timers, **when** the full path runs, **then** enclosing limits accommodate all sequential cases and cleanup.
3. **Given** verified preparation, **when** qualification takes more than 30 minutes but at most 60 minutes, **then** elapsed time alone produces an advisory; functional failures still reject acceptance.

### User Story 2 - Retain failure and cancellation truth (Priority: P1)

Maintainers need larger allowances to distinguish delayed completion from a stall without losing cancellation, diagnostics or owned cleanup.

**Independent Test**: Delayed success, expiry, cancellation and forced cleanup retain distinct outcomes and effective bounds.

**Acceptance Scenarios**:

1. **Given** a never-completing check, **when** its expanded limit expires, **then** it fails with bounded diagnostics and owned cleanup.
2. **Given** earlier explicit cancellation, **when** it arrives, **then** cancellation remains authoritative.
3. **Given** packaged help startup, **when** its stages execute, **then** monotonic progress and completion evidence distinguish success before/after the former 35-second bound from continued failure.
4. **Given** forced termination, **when** cleanup finishes, **then** no natural-release success is claimed.

### User Story 3 - Apply consistent allowances everywhere (Priority: P2)

Maintainers need local, CI, native, package, release, documentation, diagnostic and real-processing checks to follow one documented policy with complete exception accounting.

**Independent Test**: A checked inventory detects stale successful-execution bounds, mismatched units and shorter parents.

**Acceptance Scenarios**:

1. **Given** any maintained test entry point, **when** inspected, **then** its timer owner, units, origin, caller and parent are known.
2. **Given** a short deadline/cancellation regression fixture, **when** policy grows, **then** its injected clock remains short while the harness receives headroom.
3. **Given** an ordinary product operation, **when** run outside qualification, **then** its configured deadline and cancellation semantics remain unchanged.

### Edge Cases

- Sequential matrices reserve for every case and cleanup; inner allowances cannot extend an earlier absolute parent deadline.
- Preparation, queue, finalization and warm reuse retain separate elapsed accounting.
- Invalid policies, inconsistent units and override validation maxima must not silently restore old limits.
- Buffered stderr arrival is not a frontend stage timestamp.
- Historic runs retain their original policy and outcome.
- A startup failure may remain under larger bounds and stays tracked in #154.

## Requirements

### Functional Requirements

- **FR-001**: Inventory and increase successful-execution allowances across all maintained root, native, frontend, browser, docs, package, installer, backend, preparation, transfer, release, diagnostic and elected local real-processing test paths. Justify every retained shorter bound.
- **FR-002**: Qualification MUST use a 1,800-second advisory target, 3,600-second hard limit and 60-second provisional finalization reserve after verified readiness. Outer watchdogs permit bounded cleanup/evidence without changing acceptance accounting.
- **FR-003**: Help qualification MUST allow at least 180 seconds, full GUI qualification 600 seconds, native Exit child capture 120 seconds and its repeated matrix 1,800 seconds. Increase enclosing values when cumulative paths require it.
- **FR-004**: Short successful native/frontend readiness and UI waits MUST receive 60-120 seconds as appropriate, with consistent watchdog/child/matrix bounds. Common 180-second test subprocess defaults grow to 600 seconds; already-longer processing allowances do not decrease.
- **FR-005**: Source-artifact and Windows-producer joins MUST allow 900 and 1,800 seconds respectively, bounded by the original shared phase deadline. Preparation jobs allow at least 60 minutes and qualification watchdogs at least 65 minutes.
- **FR-006**: Checks return immediately on observed completion/failure. Earlier cancellation, bounded output, timeout classification, natural release assertions and owned cleanup remain mandatory.
- **FR-007**: Meaningful regressions MUST distinguish delayed success beyond the former bound, expanded-bound expiry, earlier cancellation and forced-cleanup failure without long sleeps.
- **FR-008**: Packaged help MUST retain monotonic elapsed stage/progress/completion observations and effective bounds sufficient to classify the recurrence; infer no unobserved HRESULT or common cause.
- **FR-009**: Policy, governance, current documentation, consumers, validation maxima and workflows change together. Historic records stay truthful.
- **FR-010**: Functional, security, backend, source/run/attempt/package identity and elected checks remain required. Native menu/close/Alt+F4, durable work, playback cleanup and natural owned-browser release stay genuine assertions.
- **FR-011**: Ordinary product deadlines remain unchanged unless a concrete shared dependency needs a documented proportional change. Short injected failure fixtures remain distinguishable from execution allowances.
- **FR-012**: Exact-head full Windows/Linux elected CI, isolated cold preparation and a full same-source warm comparison MUST complete with independent terminal evidence. Passing repetition does not establish a causal cure of #154.
- **FR-013**: Finish implementation, required verification, automatic publication and at most two Codex rounds; resolve actionable review threads and stop before merge. Main acceptance follows a later owner-approved merge.

### Key Entities

- **Timing policy**: Named allowances, explicit units, finalization reserve and enclosing relationships.
- **Deadline inventory**: Owner, old/new allowance, caller, platform, origin, parent and exception.
- **Qualification observation**: Source/run/package identity, effective bounds, elapsed stages and terminal outcome.

## Success Criteria

- **SC-001**: Every maintained testing entry point has an inventoried owner or justified exception, with no unaccounted shorter parent.
- **SC-002**: Delayed success passes beyond its former bound; expiry, cancellation and forced cleanup remain distinct.
- **SC-003**: Required exact-source hosted checks and cold/warm comparison complete with independent receipts and unchanged substantive assertions.
- **SC-004**: Help is classified as success before/after its former bound or continued failure with observed progress, without unsupported incident closure.
- **SC-005**: Published PR has green applicable checks and resolved actionable findings through at most two rounds and remains unmerged.

## Assumptions

- Adopt #172's proposed defaults under authorized autopilot, increasing enclosing allowances when cumulative paths require it.
- Increased runner cost and later reporting of genuine stalls are accepted tradeoffs.
- Owner desktop/input, installed packages and libraries remain outside this testing change.
- #154 and #147 remain independent incident/audit records.

## Clarifications

### Session 2026-10-24

- Expand inner and outer timers together; changing only job limits is insufficient.
- Keep injected failure clocks and normal product deadlines distinct from test harness bounds.
- Main acceptance occurs after merge. This slice stops at an unmerged PR with completed exact-head CI and reviews.
