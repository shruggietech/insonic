# Feature Specification: S010 Canonical annotated import

**Feature Branch**: `codex/010-canonical-annotated-import`

**Created**: 2026-10-09

**Status**: Implemented and CI-qualified; awaiting owner final review and merge

**Input**: Complete #31/#32 through one import boundary, with shared transcript UX from #30 and new canonical admission from #33. Push/PR publication and at most two external review rounds are authorized; merge is not.

## User Scenarios & Testing

### User Story 1 - Preserve annotated imports (Priority: P1)

Import native subtitles or exact versioned Cue JSON with new media or against an existing recording. Preserve incoming assignments and native observations. Skip existing transcripts unless replacement is elected.

**Independent Test**: Import externally attributed documents, compare exact IDs/integers/source bytes, and exercise retry, cancellation and stale replacement.

**Acceptance Scenarios**:

1. Current-version Cue JSON preserves complete and partial assignments, including valid non-UUID local identifiers.
2. Structured native labels with zero flexible assignments produce deterministic recording-scoped untimed participation, without invented acoustic evidence or timing.
3. Default collisions skip successfully; elected transcript replacement publishes one current document and reconciles dependent references.
4. Invalid explicit input fails without silently choosing embedded/generated text.

### User Story 2 - Select local, remote and embedded transcripts (Priority: P1)

Acquire media/transcripts independently, discover embedded tracks and select one inspectably. Explicit valid transcripts win with a gentle unused-track notice.

**Independent Test**: Bounded HTTP fixtures and committed multi-track containers exercise equivalent local/remote documents, MIME disagreement, credentials, language/disposition selection and offsets.

**Acceptance Scenarios**:

1. Separate credentials/budgets cannot leak across hosts; truncated/oversized/HTML responses fail.
2. Embedded selection respects explicit index/language and deterministic defaults without concatenating alternatives.
3. Image subtitles and burned-in captions expose actual OCR/extraction requirements, without fabricated text or unselected engines.

### User Story 3 - Retain canonical audio and current Cue JSON (Priority: P1)

New admission retains canonical audio, captured source facts and one current Cue JSON document. Grouped tracks remain one recording; original video, redundant native subtitles and original local locators are retired from managed data.

**Independent Test**: Verify durable inventories, channel/track/duration/offset preservation and playback for lossy/lossless/stereo/surround/multi-track fixtures, including failures and shared leases.

**Acceptance Scenarios**:

1. Capture metadata/subtitles before managed-input retirement; never delete owner-owned inputs.
2. Suitable audio avoids needless encoding; engine-specific scratch is never the master.
3. All required tracks retain separation, ordering, language, channels, offsets and full duration.
4. Publication is revision/work fenced; accepted retry reconciles receipts and failed work preserves accepted state.

### Edge Cases

Unknown versions, duplicate JSON keys, invalid UTF-8/surrogates, exact integers, bounded expansion, unknown versus zero duration, native conflicting timing, repeated/case/Unicode labels, cross-recording equal IDs, empty documents, negative offsets, missing tools, ambiguous selectors, partial downloads, concurrent edits and legacy managed sidecars.

## Requirements

### Functional Requirements

- **FR-001**: CLI, manifests, runtime and desktop expose consistent input/target/selection/replacement contracts and preserve existing grammar compatibility.
- **FR-002**: Standalone transcript admission targets an unambiguous recording without creating another media record; default transcript collisions skip and replacement requires an explicit election.
- **FR-003**: Validate exact packaged supported versions before any translation and current output before publication; unknown versions report supported versions and never fetch input schema URLs.
- **FR-004**: Preserve exact assignments, arbitrary bounded valid local IDs, cue identity/order, native payload, source envelope and integer instants; equal tokens in different recordings do not merge identities.
- **FR-005**: Auto/native/off derivation preserves existing/partial attribution. Diarize explicitly elects existing configured processing, bypasses recognition for supplied text and never silently changes methods.
- **FR-006**: Structured labels support participation only; trustworthy speaker-specific source bounds are required for timed intervals.
- **FR-007**: Independent source/transcript acquisition enforces credentials, cancellation, byte/time/redirect bounds and actual byte/format validation.
- **FR-008**: Discover all embedded candidates before disposal; explicit valid input wins, otherwise inspectable index/language/disposition selection chooses one supported text track.
- **FR-009**: New admission retains canonical audio and captured facts without original video, redundant native transcript files or durable original local locators.
- **FR-010**: Grouped storage uses one audio-only Matroska (.mka) asset with FLAC tracks; preserve track separation and offsets.
- **FR-011**: Stereo and every larger multichannel/grouped input use FLAC regardless of incoming quality.
- **FR-012**: MP3 routing/encoding and normalization use lossy source-codec classification subject to the FLAC override, suitable MP3 passthrough, V0 encoding otherwise, and format-only normalization without automatic loudness/DSP/upsampling.
- **FR-013**: Preserve full required samples/channels/duration and rational clock maps; model-specific preparation is disposable.
- **FR-014**: Fence publication, retries, dependent-reference invalidation and recoverable reference/lease-aware retirement; protect owner-owned and shared current assets.
- **FR-015**: Preserve affected filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB contracts, portable current records and shared playback.
- **FR-016**: Amend conflicting constitution/docs, help, schema, examples, glossary and changelog together with synchronized release identity.
- **FR-017**: Required CI verifies deterministic orchestration and bounded real media/documents without model loading, downloads or inference.

### Key Entities

- **Recording**: Stable library identity, canonical asset/track map and current document revision.
- **Transcript candidate**: Selected source, exact version, native observations and supplied/derived assignments before acceptance.
- **Admission election**: Frozen source/target/selection/replacement permissions, routing/settings references and observed revisions.
- **Admission receipt**: Imported/replaced/skipped/failed result, stable accepted IDs and bounded diagnostics without original local locators.
- **Canonical asset**: Immutable bytes, codec/container/tool facts and ordered track/channel/time relationships.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every admitted input class has equivalent preservation, collision and replacement behavior in acceptance fixtures.
- **SC-002**: All supplied complete/partial assignments survive auto/native/off; all tested failed/stale/cancelled publication preserves prior accepted state.
- **SC-003**: Required grouped tracks survive with no silent downmix, truncation or record fragmentation.
- **SC-004**: New durable inventory has no original video, redundant native transcript or original local locator; owner-owned inputs remain untouched.
- **SC-005**: Final published head passes affected checks within the ten-minute CI job budget without required model execution.

## Clarifications

### Session 2026-10-09

- Q: Grouped-track container? A: Owner approved one audio-only .mka asset containing FLAC tracks.
- Q: Stereo override? A: Owner approved including ordinary stereo in the mandatory FLAC override.
- Q: MP3 tier and normalization? A: Owner approved lossy-codec classification, suitable MP3 passthrough, V0 encoding and format-only normalization without automatic DSP.

## Assumptions and Scope

- Reuse current acquisition, artifact publication, recording transactions, processing adapters and desktop bridge proportionally.
- Investigate exact historical 1.0.0/1.1.0 contracts; accept only with packaged validation and tested translation.
- Full legacy-audio migration, complete audio replacement, model aliases/acquisition, rosters/matching, training and release promotion remain in their issues. Transcript conversion/retirement necessary here is included.
- Close #31/#32 only after complete acceptance; #29/#30/#33 remain open with precise remaining scope.
- Independent transcript/provenance work may proceed while the final storage policy answer is pending.