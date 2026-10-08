# Feature Specification: S007 Desktop delivery

**Feature Branch**: `codex/007-desktop-delivery`

**Created**: 2026-10-07

**Status**: Clarified

**Input**: Owner-authorized desktop delivery targeting GitHub #10 (historical roadmap label S010), with automatic push/PR, at most two review rounds, and owner merge boundary.

## User Scenarios & Testing

### User Story 1 - Manage a workspace and library (Priority: P1)

Create/open a workspace, import audio/video and inspect original identity, raw metadata, normalized recording dates and competing date observations. Correct dates through the shared operation and retain source bytes and provenance. Relocate originals only after identity verification.

**Independent Test**: Import both committed media fixtures through desktop controls, read matching state through CLI and verify source identity.

**Acceptance Scenarios**:

1. Given no workspace, creating or opening one visibly selects its identity for subsequent operations.
2. Given imported media, selecting details shows raw metadata, date precision/zone/provenance and current derived results.
3. Given an unresolved date, explicit correction records the choice without inventing an instant or rewriting media.

### User Story 2 - Configure and operate processing (Priority: P1)

Manage Local/Connected/Custom pipelines, speaker names/aliases/mappings, terms, models and settings. Assemble/process/rerun recordings, inspect current results and export subtitles. Monitor durable jobs and act on interrupted or failed work.

**Independent Test**: GUI edits read back through CLI; deterministic processing fixtures exercise current-result publication and durable cancellation/retry.

**Acceptance Scenarios**:

1. Saving/inspecting a pipeline reports elected stages, context and quality settings with no silent fallback.
2. Correcting speakers, mappings or terms refreshes current evidence without copied assignment arrays.
3. Jobs show state/stage/progress/failure actions; cancellation/retry target the selected job; closing the desktop preserves runtime work.
4. Settings and credentials use shared validation and protected input; existing secret values are never displayed.

### User Story 3 - Play current source-linked evidence (Priority: P1)

Play selected audio/video and seek current cues or timed speaker spans to original source positions. Refresh after corrections/replacement; obsolete evidence is unavailable.

**Independent Test**: Committed audio/video plays from authorized source bytes, cue selection seeks original time, stale/arbitrary-file requests fail.

**Acceptance Scenarios**:

1. Selecting a current cue/timed speaker span opens the correct original asset and position.
2. Missing originals require identity-verified relocation; unavailable codecs have an actionable documented preview path.
3. Current-result replacement invalidates old playback/evidence selections; untimed participation never invents a seek interval.

### User Story 4 - Install with matching help (Priority: P1)

Install/extract a native GUI-with-CLI package on Windows, macOS or Linux, use packaged companions and open matching offline docs.

**Independent Test**: Each platform package contains working sibling GUI/CLI, exact companion tools/libraries, notices and help; extracted-package smoke qualification passes.

**Acceptance Scenarios**:

1. Packaged executables work without development-checkout paths and declare platform prerequisites.
2. Help works without networking, contains matching schemas/changelog and excludes the landing page.
3. Core workflows support keyboard access, labels, visible focus, readable contrast, reduced motion and theme preference.

### Edge Cases

- Runtime failures and revision conflicts are actionable and never show optimistic success as persisted state.
- Large lists use bounded pagination; old asynchronous replies cannot overwrite newer selection/workspace.
- Playback ranges, relocation mismatch, expired references and replaced evidence fail without arbitrary file access.
- Desktop closure does not cancel jobs; explicit cancellation remains available.
- Missing package dependencies/help and unqualified architectures fail qualification instead of claiming support.

## Requirements

### Functional Requirements

- **FR-001**: Workspace create/open and Library import/list/detail/relocation use explicit workspace identity.
- **FR-002**: Raw metadata/date provenance and owner date corrections use shared CLI/runtime operations.
- **FR-003**: Current recording assembly/processing/inspection/rerun/native subtitle export have dedicated controls.
- **FR-004**: Saved pipeline create/update/list/inspect exposes presets, stage overrides and optional automatic quality diagnostics.
- **FR-005**: Speaker CRUD/aliases/current mappings/paginated evidence preserve local UUIDs and embedded document authority.
- **FR-006**: Terminology editing and effective recognition-context preview are available.
- **FR-007**: Jobs list/detail/stage/progress/cancel/retry expose durable state and actionable recovery, independent of GUI lifetime.
- **FR-008**: Settings, models and credential configuration use shared validation and protected input without returning existing secret values.
- **FR-009**: Scoped playback of authorized original/documented proxy audio/video supports ranges and original-source cue/speaker seeking.
- **FR-010**: Refresh resolves current evidence and invalidates superseded references; no copied assignment store or alternative-run selector exists.
- **FR-011**: Core logic is shared with CLI parity; capability matrix names dedicated GUI controls and explicit advanced-operation lag.
- **FR-012**: Qualify GUI-with-CLI packages on Windows/macOS/Linux, including exact companion inventory/hashes/notices and matching offline help. Model weights remain optional separate downloads.
- **FR-013**: Use governed tokens/components and Wails shell ownership; keyboard, labels, focus, contrast, reduced motion and accessible tables are required.
- **FR-014**: Required CI never invokes transcription/diarization engines, loads their models, downloads weights or requires their results. Deterministic orchestration and real media decoding/document fixtures remain required.
- **FR-015**: Preserve configured routing authorization; introduce no mandatory per-item review or silent fallback.

### Key Entities

- Workspace selection: identity/location and session-bound requests.
- Desktop view state: ephemeral selections/request generations; catalog remains authoritative.
- Playback reference: bounded access to current source bytes/time, invalidated by relevant changes.
- Package inventory: release/revision/platform, included files/hashes/notices/help identity.
- Capability matrix: shared operation, CLI route, GUI flow and declared advanced lag.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All four journeys pass controlled fixtures; GUI edits match CLI with no duplicate business store.
- **SC-002**: All three platform packages build and pass extracted-package smoke checks; included dependencies have identity and notices.
- **SC-003**: Original-time playback/seek works; stale/relocation-mismatched/arbitrary-file access is rejected.
- **SC-004**: Required jobs finish below ten minutes with zero model inference/loading/weight downloads.
- **SC-005**: Every mature in-scope operation has a usable dedicated GUI flow; advanced lag is explicit.

## Assumptions

Shared runtime/media/pipeline/speaker contracts are implemented dependencies. Graph/exploration (#11/#12), query assistance (#13), backup/release promotion (#14) and training (#15) remain outside this slice. Build and qualify packages; do not tag/publish releases or promote docs. Architecture support follows executed native qualification.

## Clarifications

### Session 2026-10-07

- Q: Does #10's historical S010 label change the requested slice? A: No. S007, directory 007-desktop-delivery and branch codex/007-desktop-delivery agree.
- Q: Are graph/training screens required now? A: No. Mature implemented operations receive dedicated controls; future capabilities remain explicit.
- Q: Is model inference required for desktop qualification? A: No. Deterministic callbacks/fixtures prove orchestration; inference is maintainer-only.
- Q: Is an official release authorized? A: No. Package qualification and reviewed green PR are the stop boundary before owner merge.
