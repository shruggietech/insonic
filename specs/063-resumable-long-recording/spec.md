# Feature Specification: Resumable long-recording processing

**Feature Branch**: `codex/063-resumable-long-recording`

**Created**: 2026-10-31

**Status**: Implemented; owner review and merge pending

**Input**: S063 under autopilot for GitHub issues [#212](https://github.com/shruggietech/insonic/issues/212), [#213](https://github.com/shruggietech/insonic/issues/213), [#214](https://github.com/shruggietech/insonic/issues/214), [#215](https://github.com/shruggietech/insonic/issues/215), and [#216](https://github.com/shruggietech/insonic/issues/216). Push and open a PR automatically, resolve findings within at most two review rounds, and stop for owner final review and merge.

## User Scenarios & Testing

### User Story 1 - Process long recordings with accurate boundaries (Priority: P1)

An operator processes selected canonical tracks through the existing configured workflow. Bounded windows produce one transcript retaining exact source timing and available word evidence.

**Why this priority**: Memory and decoded scratch must not grow with complete recording duration.

**Independent Test**: Process offset multitrack fixtures with uneven durations and boundary speech, then inspect samples, cues, provenance, and peak scratch.

**Acceptance Scenarios**:

1. **Given** tracks with unequal offsets and lengths, **When** planning, **Then** deterministic nonoverlapping cores cover all elected samples with explicit bounded context.
2. **Given** overlap duplicates and genuine repetitions later in time, **When** stitching, **Then** duplicate observations appear once and genuine repetitions remain.
3. **Given** compressed audio, **When** extracting windows, **Then** inward sample projection agrees with trusted full decode and canonical bytes remain unchanged.
4. **Given** no word timing from a recognizer, **When** stitching, **Then** timing limitations and ambiguous overlap remain explicit rather than invented.

### User Story 2 - Resume after interruption (Priority: P1)

An operator retries an interrupted job, retaining verified windows without accepting partial or incompatible output.

**Why this priority**: A crash must not lose all long-job progress.

**Independent Test**: Inject interruption at write, verification, checkpoint acceptance, and final publication; compare resumed invocation counts and output with an uninterrupted run.

**Acceptance Scenarios**:

1. **Given** completed verified windows and one partial window, **When** retrying, **Then** only incomplete windows execute again.
2. **Given** changed source, document, model, or identity-bearing settings, **When** resuming, **Then** incompatible reuse is rejected.
3. **Given** an expired owner, **When** it writes after a new attempt starts, **Then** it cannot accept checkpoints or current output.
4. **Given** cancellation or backup/restore, **When** retrying applicable work, **Then** checkpoint identity and bytes survive under the documented retention policy across supported backends.

### User Story 3 - Recover within resource policy (Priority: P2)

An operator uses elected CPU/NVIDIA policy with finite pressure adjustment and fallback, preserving original source, model, route, and progress.

**Why this priority**: Resource exhaustion needs an explainable bounded outcome.

**Independent Test**: Inject resource failures and inspect attempts, pressure, devices, reasons, deadlines, and checkpoint reuse.

**Acceptance Scenarios**:

1. **Given** resource exhaustion, **When** compatible pressure reduction is available, **Then** recovery is finite and recorded before allowed fallback.
2. **Given** explicit CPU selection, **When** retrying, **Then** execution stays CPU.
3. **Given** allowed device fallback, **When** resuming, **Then** verified windows remain reusable and source/model/provider routing never changes.

### Edge Cases

- Empty/short input, exact boundaries, short final windows, nonzero/negative starts, uneven tracks, and integer overflow.
- Mono/stereo/grouped tracks, simultaneous speech, punctuation, multilingual text, truncated/ambiguous overlap, and absent word timing.
- Changed worker output, corrupt or forged checkpoints, cross-recording substitution, expired leases, cancellation during decode, interrupted artifact/catalog writes, source replacement, and restore.
- Device unavailability, CUDA OOM, repeated exhaustion, expired deadlines, and corrupt readiness evidence.

## Requirements

### Functional Requirements

- **FR-001**: Freeze a versioned reproducible plan identifying canonical sources/tracks/channels, rational offsets, source samples, core/context windows, model/adapter, language/context, device policy, and resource limits (#212).
- **FR-002**: Cover all elected intervals deterministically, including uneven tracks and final partial windows. Document units, defaults, limits, and precedence; preserve existing short-recording requests and command family (#212).
- **FR-003**: Bind plans/checkpoints to source/document/configuration identities and attempt authority. Only explicitly compatible actual-device changes under the same frozen policy may reuse output (#212, #215, #216).
- **FR-004**: Decode lazily with memory/scratch bounded by window size and concurrency. Avoid whole-recording PCM allocation and repeated full downloads when range access or reusable staging exists (#213).
- **FR-005**: Preserve canonical bytes and source sample mapping. Cancellation, decode failure, and lease loss clean owned temporary output (#213).
- **FR-006**: Retain available word/time evidence and reconcile overlaps deterministically without text-only deletion of genuine repeats, fabricated timing, or merging independent-track speech (#214).
- **FR-007**: Accept one current validated transcript with contributing source/window provenance for every cue. Reconciliation identity includes verified worker output; ambiguous overlap stays explicit (#214).
- **FR-008**: Persist complete verified window artifacts/manifests with plan/output digests, attempt/generation, and completion state. Partial/uncommitted output is never reusable (#215).
- **FR-009**: Recover interrupted artifact/catalog publication, reject stale owners, expose progress, and reuse existing durable retry/cancellation authority (#215).
- **FR-010**: Include checkpoint metadata/bytes in applicable backup/restore for supported storage/catalog combinations. Cancellation retains verified windows for explicit retry and retires partial scratch (#215).
- **FR-011**: Bound active windows, batch pressure, and worker resources using actual capability/readiness; apply finite compatible pressure adjustment and allowed fallback (#216).
- **FR-012**: Record requested policy, actual device, adjustments, and fallback reasons per window/attempt, preserving source/model/provider identity and enclosing deadlines (#216).
- **FR-013**: Preserve unattended configured execution, protected credentials, hidden noninteractive Windows children, and filesystem/S3, SQLite/PostgreSQL, LadybugDB/ArcadeDB support.
- **FR-014**: Synchronize runtime, JSON contracts, CLI help, public docs, and changelog. Separate deterministic verification from elected real-engine observations.

### Key Entities

- **Processing plan**: Immutable identity and ordered source windows under frozen configuration.
- **Window**: Source-sample core and bounded context for a selected track/channel, mapped to the source clock.
- **Window checkpoint**: Verified artifact/digest bound to plan, source, work lineage, and publishing attempt.
- **Reconciled transcript**: One accepted current document with contributing windows and explicit uncertainty.
- **Execution observation**: Actual device, adjustment, failure classification, and finite recovery history.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Planning fixtures cover 100% of elected samples exactly once in cores, with zero gaps or double ownership.
- **SC-002**: Tenfold fixture-duration growth under the same window/concurrency settings does not exceed the configured live decoded scratch bound.
- **SC-003**: Boundary fixtures preserve every distinct expected word, remove only duplicate observations, and retain valid source intervals for every cue.
- **SC-004**: Restart tests rerun zero verified compatible windows; every partial/corrupt/stale checkpoint is rejected or recomputed without duplicate final acceptance.
- **SC-005**: Every injected resource failure terminates within finite attempt/deadline policy with actual-device evidence.
- **SC-006**: Required maintained checks pass; elected engine observations identify candidate, model, device, input, and limitations separately.

## Assumptions

- Existing durable work and current-source/current-transcript authority remain authoritative.
- Coverage diagnostics, targeted repair, and milestone release recommendation (#217-#219) follow this slice; epic #209 remains open.
- Hosted routing remains explicitly selected. Local failure does not elect a provider.
- This slice authorizes push/PR publication but no tag, release, or merge.

## Clarifications

### Session 2026-10-31

- Q: Should cancellation retain verified windows? A: Yes, retain verified output for explicit retry and retire partial scratch; cancelled final output is not accepted.
- Q: Does fallback change plan identity? A: Actual device may change only within frozen requested policy; source, model, boundaries, and output semantics remain identity-bearing.
- Q: Is a separate job system needed? A: No, extend existing processing and durable-work contracts.
- Q: Do deterministic fixtures establish model accuracy? A: No, elected actual-engine acceptance is reported separately.
