# Feature Specification: S005 Current correlated subtitles

**Feature Branch**: `codex/005-correlated-subtitles`
**Created**: 2026-10-07
**Status**: Specified; design and task analysis complete
**Input**: Complete #21 and #7 with committed audio/video fixtures, maintainer-only real-engine qualification, authorized push/PR, at most initial plus one requested review round, and owner merge handoff.

## User Scenarios & Testing

### User Story 1 - Assemble current annotated subtitles (Priority: P1)

An operator combines supplied SRT/VTT and recording-local speaker results into one finalized Cue JSON document embedded in the master recording.
**Why this priority**: Establish one subtitle/assignment authority before engine integration.
**Independent Test**: Assemble either real fixture with supplied subtitles and deterministic turns, inspect and export.
**Acceptance Scenarios**:

1. Preserve supplied subtitle bytes; validate the finalized embedded document with the exact current upstream local schema.
2. Timed assignments exist only inside that document; other records reference document/cue evidence and known speakers use an external recording/local-UUID mapping.
3. Multiple/overlapping voices, untimed participation and cue/media bounds follow the upstream contract. Diagnose unrepresentable intervals without inventing cues or timestamps.
4. Measured duration distinguishes unknown from zero; preserve exact higher-resolution original-clock mapping.
5. Cue JSON export preserves annotations; SRT/VTT omission diagnostics are retained and strict mode refuses publication before lossy output.

### User Story 2 - Process audio/video locally (Priority: P1)

An operator extracts mapped audio, uses supplied subtitles or local recognition, and runs local diarization without per-item approval.
**Why this priority**: Deliver a usable local subtitle pipeline from the media library.
**Independent Test**: Explicit maintainer tooling runs real elected engines on the committed media and records model/settings, resource use and text/speaker/timing diagnostics.
**Acceptance Scenarios**:

1. Mapped audio retains source identity/time/channel mapping; supplied subtitles skip unnecessary recognition.
2. Actual local diarization assigns recording-local UUIDs and reports automatic quality diagnostics by default, separately from document validation. Another recording receives different local UUIDs.
3. Missing/invalid models, unsupported capabilities and local failure produce actionable outcomes without silent hosted fallback or fabricated output.
4. No-speech/unusable timing remain explicit; CLI/runtime and desktop bridge share core operations.

### User Story 3 - Replace results and recover work (Priority: P1)

An operator independently reruns transcription or diarization, corrects mappings and recovers interrupted work with one current result.
**Why this priority**: Prevent stale assignment authority and unnecessary retranscription.
**Independent Test**: Rerun annotations alone, interrupt/retry, correct a mapping and verify source identity plus physical retirement.
**Acceptance Scenarios**:

1. Diarization-only rerun preserves subtitle text and skips transcription; transcription replacement reconciles affected evidence.
2. Atomic current replacement physically retires superseded derived artifacts/rows through recoverable cleanup; temporary candidates cannot become retained alternatives.
3. Expired/cancelled/concurrent attempts cannot replace accepted state; retry reconciles publication without duplicates.
4. Mapping correction leaves text unchanged; downstream query/playback/export/training references resolve current evidence or are explicitly invalidated without copied assignments/frozen alternatives.
5. SQLite/PostgreSQL and filesystem/S3 have equivalent migration, snapshot, replacement, recovery and workspace-isolation contracts.

### User Story 4 - Reproducible inexpensive qualification (Priority: P1)

A maintainer uses redistributable committed media for deterministic checks and elects real-engine qualification outside CI.
**Why this priority**: Exercise real media paths without expensive CI inference.
**Independent Test**: Clean ordinary checkout includes audio/video and runs deterministic checks; only explicit maintainer tooling runs actual engines.
**Acceptance Scenarios**:

1. Commit a lossless speech sample and short dialogue video directly under tests/fixtures/media, without LFS or test-time media fetches; aim below 10 MiB combined.
2. Retain asset license/attribution, creator/source/rendition revision, hashes/sizes, measured streams/duration, crop/time mapping and reproducible adaptation commands, plus independently checked reference text/subtitles/turn expectations.
3. No CI/required check invokes transcription/diarization, initializes/loads their models, downloads weights or requires engine results, even if cached/small. Bounded media extraction/probing and actual Cueson operations are allowed.
4. Opt-in maintainer engine testing is outside CI and merge requirements, records bounded resource use and truthful diagnostics, and never injects reference text as generated recognition output.
5. Deterministic constructed cases cover overlap, silence, untimed participation, unknown duration and integer boundaries absent from natural clips.

### Edge Cases

Malformed subtitle/adapter output; unsupported schema; out-of-cue/media intervals; integer overflow/rounding; duplicate/cross-workspace IDs; unknown/zero duration; no speech/cues; multiple source tracks; source mutation; partial adapter failure; stale/cancelled commit; cleanup retry; stale snapshot; downstream invalidation; strict native-export refusal; unavailable/unintelligible fixture rendition.

## Requirements

### Functional Requirements

- **FR-001**: Pin exact Cueson v1.2.0 packages/schema, validate locally and remove superseded Cueson support/pins/references from all maintained files.
- **FR-002**: Admit supplied/generated SRT/VTT through Cueson while preserving source inputs and embedding one finalized current document in the master record.
- **FR-003**: Store recording-local UUIDs/assignments only in embedded Cue JSON; maintain external known-speaker mappings and reference-only consumers.
- **FR-004**: Preserve measured duration and exact source-clock mapping; deliberately project milliseconds and diagnose unrepresentable intervals.
- **FR-005**: Implement mapped audio, actual local recognition/diarization and shared CLI/runtime/desktop routes with bounded hidden noninteractive execution and no silent provider fallback.
- **FR-006**: Enable automatic diarization quality diagnostics by default, distinct from document validity; report no-speech/unusable timing without fabrication/manual approval.
- **FR-007**: Independently rerun transcription/diarization and correct mappings without unnecessary recognition or subtitle rewriting.
- **FR-008**: Atomically replace current results, physically retire superseded managed bytes/rows, reconcile downstream references and recover cleanup without archives/alternatives.
- **FR-009**: Use durable fenced work and cancellation/restart/retry, preserving workspace isolation and backend parity.
- **FR-010**: Export current Cue JSON; expose native omissions and strict refusal before publishing lossy output.
- **FR-011**: Commit two licensed real media fixtures with attribution, hashes, provenance, measured properties and checked test references.
- **FR-012**: Exclude inference/model loading/weight downloads from all CI and merge requirements; provide opt-in maintainer qualification with measured evidence.
- **FR-013**: Reconcile master schemas, migration/restore, internal planning, public docs and legacy retention conflicts while preserving release-version alignment.
- **FR-014**: Publish a PR closing #21/#7, resolve all findings, request at most one second round, and stop before merge/release with green affected checks.

### Key Entities

Current recording/embedded document; cue/source mapping; recording-local speaker UUID; external known-speaker mapping; mapped audio; local adapter/model/settings; fenced processing work; retirement obligation; export diagnostic; licensed fixture manifest; maintainer qualification receipt.

## Success Criteria

- **SC-001**: Both media fixtures yield valid current documents via supplied/deterministic checks, with zero duplicated persisted assignments and explicit timing diagnostics.
- **SC-002**: Independent rerun/recovery preserves originals/subtitles, leaves one current result and removes superseded managed results after cleanup.
- **SC-003**: Opt-in maintainer tooling exercises actual elected engines and distinguishes measured behavior/limits from deterministic evidence.
- **SC-004**: Required checks pass without transcription/diarization/model loading/weight downloads; clean ordinary Git checkout supplies both fixtures.
- **SC-005**: Exact dependency/document/media and supported-backend qualification, critical convergence and external reviews are complete before owner PR handoff.

## Assumptions

- Literal S005/specs/005-correlated-subtitles/codex/005-correlated-subtitles maps to #21/#7; #7's S007 roadmap title remains unchanged.
- Reuse S004 media/artifact/catalog/model/secret contracts. General presets/hosted adapters (#8), broad speaker/term UX (#9), full GUI/installers, graph exploration and training engines remain later work; required current-reference/mapping integrity is included.
- Owner #21 overrides retained alternate transcripts/frozen assignment copies. Preserve source media/subtitles and provenance, without stale derived copies.
- Candidates are CC BY 3.0 Speech 12dB s16.flac and a short Sintel dialogue excerpt; exact rendition/window and measured properties are implementation qualification.
- Push/PR authorized; initial automatic plus one manual review request maximum; no extra security request or merge/release.

## Clarifications

### Session 2026-10-07

- CI exclusion applies to testing, not configured product processing. Real engine testing is opt-in maintainer work outside CI/merge requirements.
- Fixture reference assignments are test oracles, never a second product assignment store.
- CI may probe/extract real media and execute Cueson, using supplied/deterministic adapter results without loading models.
- Legacy retention conflicts follow #21: migrate to one current embedded document/external mappings and remove stale derived copies.

- Timing policy: validate positive nonnegative raw nanosecond intervals and known media bounds; intersect them with cue coverage, then project inward using ceil(start_ns/1000000) and floor(end_ns/1000000). Diagnose rounded, clipped, uncovered and collapsed portions. Omit collapsed timed assignments without inventing untimed participation; untimed participation requires explicit cue/local-UUID evidence. Unknown duration omits media_timing, measured zero remains explicit. No representable cues yields a null document with the explicit no-speech/no-timed-subtitles state.
