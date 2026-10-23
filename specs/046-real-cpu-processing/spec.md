# Feature Specification: Real CPU processing and speaker matching

**Feature Branch**: `codex/046-real-cpu-processing`

**Created**: 2026-10-23

**Status**: Specified

**Input**: Owner-authorized S046 autopilot, extending the maintained packaged CLI qualification for #147 with real CPU execution, independently checked references, automatic PR publication and at most two Codex review rounds. Stop before owner merge.

## User Scenarios & Testing

### User Story 1 - Prepare and process real audio (Priority: P1)

A maintainer runs the current complete package in a disposable library, prepares its declared CPU environments and models, imports licensed audio, and receives persisted transcript cues and speaker activity from actual engines.

**Why this priority**: Deterministic adapters establish orchestration contracts but cannot establish actual model execution or the current package's acoustic outputs.

**Independent Test**: Prepare the declared environments and exact models through the packaged CLI, process a short reference fixture, and compare its persisted results with independent references.

**Acceptance Scenarios**:

1. **Given** an isolated profile and complete current package under a path containing spaces and Unicode, **When** managed CPU preparation and exact model acquisition complete, **Then** task readiness identifies the accepted bytes and real processing publishes a current validated document without global environment changes.
2. **Given** a verified local model snapshot or complete offline dependency bundle, **When** reuse is elected through the normal commands, **Then** the accepted model/environment retains the verified identity and a later real CPU operation succeeds.
3. **Given** missing or corrupt selected assets, **When** preparation or invocation is attempted, **Then** it fails with its original actionable cause, publishes no successful output, and does not silently change devices or routes.

### User Story 2 - Measure speaker and transcript outcomes (Priority: P1)

A maintainer can distinguish actual recognition and speaker-matching behavior from fixture orchestration, using transcript text and speaker identities established independently of model predictions.

**Why this priority**: Earlier real CPU measurements cover a small historical corpus; one matching journey returned unknown and did not establish successful identification or broad accuracy.

**Independent Test**: Enroll at least two reference speakers from enrollment-only clips, evaluate disjoint held-out clips plus an unseen speaker and a two-speaker sequence, and retain the reference comparison for each observed decision.

**Acceptance Scenarios**:

1. **Given** licensed, hash-bound source clips with independent transcripts and speaker labels, **When** the fixture corpus is generated, **Then** enrollment and evaluation sources do not overlap and every derived sample retains a reproducible source/time mapping.
2. **Given** independently attributed enrollment evidence, **When** real profile enrollment, artifact retrieval and held-out matching finish, **Then** exact originating speaker/model/evidence identities survive and each matched, unknown or ambiguous result is compared with the frozen reference.
3. **Given** actual generated transcript and activity, **When** metrics are calculated, **Then** word errors and speaker decisions include their denominators, settings, timing and reference limitations; a poor acoustic result cannot be presented as correct identification.

### User Story 3 - Preserve results through query and restore (Priority: P2)

A maintainer can search real generated cues, resolve playback positions, retrieve the enrolled profile, and restore the library without the original disposable library remaining available.

**Why this priority**: A successful engine subprocess alone does not establish useful accepted library behavior.

**Independent Test**: Compare source-linked query, timeline, model retrieval and artifact identities before and after portable restore, then verify owned runtime shutdown and scratch removal.

**Acceptance Scenarios**:

1. **Given** accepted real CPU results, **When** search, timeline and playback operations run, **Then** their cue/recording/time references resolve against the current accepted audio and document.
2. **Given** a complete portable backup, **When** restore runs with the original library unavailable, **Then** the selected result and model identities are preserved, destination host prerequisites are explicit, and completed history does not become synthetic new execution.
3. **Given** an operation fails or is cancelled, **When** the run ends, **Then** the receipt identifies the failing stage and observed outcome, owned processes and temporary files are cleaned, and owner inputs remain unchanged.

### Edge Cases

- Truncated, altered or mislabeled source/model/dependency bytes; overlapping enrollment and held-out source intervals.
- Missing reference text, empty predicted text, unknown speakers, merged/split predicted voices and uncertain turn boundaries.
- A selected model returns an incorrect or unknown identity even though lifecycle execution succeeds.
- Restart or retry after preparation failure, stale work identity, model retrieval mismatch and unavailable original workspace during restore.
- Missing protected model access, unavailable device/library, stream overflow and child deadline expiry.
- A partial receipt, old package or old model receipt must not qualify the current source.

## Requirements

### Functional Requirements

- **FR-001**: Extend the maintained qualification tools and fixtures; execute the complete current packaged CLI and exact companions, with package/source/inventory identities recorded before execution.
- **FR-002**: Exercise real managed CPU environment preparation and exact model acquisition, including verified offline/local reuse through ordinary product commands. Existing authorized routes and protected references apply; no fabricated credentials, implicit hosted fallback or global installation is permitted.
- **FR-003**: Provide a small reproducible, licensed corpus with independently sourced transcripts and speaker labels, at least two enrollment identities, disjoint held-out speech for each, one unseen-speaker case and a constructed two-speaker sequence. Retain source hashes, provenance, license and sample/time mappings.
- **FR-004**: Run actual recognition, diarization, enrollment and matching. Deterministic mocks remain explicitly separate and cannot satisfy real-engine acceptance.
- **FR-005**: Compute transcript edit counts and denominators plus per-case expected/observed speaker decisions and aggregate correct/incorrect/unknown/ambiguous counts. Describe reference timing limits and do not claim a diarization error rate without adequate independent speech-boundary annotations.
- **FR-006**: Freeze corpus roles, model identities, device and matching settings before inference. Never alter references or lower matching thresholds after observing outputs to manufacture success. Acoustic quality measurements and functional/integrity acceptance remain separate, with incorrect predictions and limitations retained.
- **FR-007**: Assert persisted cue/audio/source authority, query/timeline/playback positions, speaker-associated profile/artifact retrieval and portable restore after the original library is unavailable.
- **FR-008**: Bound real operations and child output, hide noninteractive Windows children, isolate all profiles/libraries, preserve first failures, and clean owned processes/files on both success and failure. Owner media, installed package state and protected credentials must remain unchanged.
- **FR-009**: Produce a compact versioned receipt binding source/package/tool/model/environment/corpus identities, selected settings, actual stages, comparisons, resource measurements and unexecuted claims. Missing, stale, synthetic or incomplete evidence cannot yield real acceptance. Retain canonical evidence without raw private logs or generated junk in the build directory.
- **FR-010**: Regressions for evaluation, provenance, isolation, cleanup and real/synthetic mode separation must run without inference in ordinary CI. Real inference remains an explicit maintainer election outside required CI, using the existing guard. Preserve every elected required check and terminal timing policy.
- **FR-011**: Repair demonstrated integration defects proportionally, retaining meaningful failing regressions. Do not convert intrinsic statistical model limitations into undocumented product restrictions or claim a broad accuracy guarantee.
- **FR-012**: Update linked issue/milestone/project state and self-contained maintainer documentation. Keep #147 open for other unverified dimensions and #154 open for its unresolved historical cause. Automatically publish the PR, satisfy CI and at most two Codex rounds, then stop before merge.

### Key Entities

- **Reference corpus**: Licensed immutable source clips, independent transcript/speaker references, enrollment/evaluation roles and reproducible derivations.
- **Real acceptance receipt**: Exact execution identity, stage outcomes, measurements, reference comparisons, cleanup disposition and explicit limits.
- **Speaker case**: Enrollment evidence, held-out source identity, expected speaker and observed match decision with frozen settings.
- **Accepted library result**: Current recording/audio/document authority, query/playback positions, enrolled model version/artifact and restored equivalents.

## Success Criteria

### Measurable Outcomes

- **SC-001**: One complete current Windows package executes every real stage in FR-002, FR-004 and FR-007 and produces a source-bound receipt with no omitted stage or silent fallback.
- **SC-002**: Every evaluation clip has independently established references and zero source-interval overlap with enrollment. Each observed transcript and speaker decision has a retained comparison and denominator.
- **SC-003**: Pre/post-restore comparisons preserve the asserted result/model identities and positions; altered, stale or missing evidence is rejected by meaningful regressions.
- **SC-004**: The run records explicit operation/resource bounds and cleanup, leaves no owned runtime or scratch behind, and changes no owner library or global environment.
- **SC-005**: Required automated checks pass on the final PR head under the 660-second advisory / 900-second hard qualification policy, separately reporting preparation and total time. Every received finding is addressed within two Codex rounds and the PR remains unmerged for owner review.

## Assumptions

- Windows CPU is the initial real-engine acceptance host. Existing Windows/Linux deterministic package/backend coverage remains mandatory; unexecuted acoustic platforms, GPU, authenticated hosted inference and architecture-specific weight retraining are not claimed.
- Speaker profile enrollment uses the existing supported embedding adapter and does not claim to update base-model parameters.
- Statistical accuracy is measured against frozen independent references. This slice establishes repeatable current-package execution and honest baseline measurements, not universal recognition/identification quality thresholds.
- Public fixture and declared model/dependency acquisition are authorized by this slice. Reuse already accepted exact local model snapshots where access is required; unresolved upstream access is reported rather than invented.
- Product/docs/schema remain 1.0.1 unless an actual contract change requires an explicitly justified coordinated version change. Tags, releases, website publication and owner-desktop GUI execution are not part of this slice.
