# Feature Specification: Saved import presets and executable documentation

**Feature Branch**: `codex/020-import-presets-executable-docs`

**Created**: 2026-10-11

**Status**: Specified

**Input**: S020, GitHub issues #70 and #62, owner-authorized autopilot through official PR and at most two review rounds.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reuse deliberate import choices (Priority: P1)

An operator saves destination, date and optional processing choices, then selects them by name or ID when importing media. CLI and desktop use the same saved choices.

**Why this priority**: A selected preset currently acts as an ignored label, so operators cannot rely on their intended workflow.

**Independent Test**: Create an admission-only preset, select it for a synthetic recording and inspect the effective destination and date choices.

**Acceptance Scenarios**:

1. **Given** a saved preset, **When** it is created, listed, shown, updated and deleted from either interface, **Then** identity, revision, name and nonsecret configuration agree across interfaces and persist after restart.
2. **Given** explicit overrides and a selected preset, **When** a direct import, refresh or JSON/CSV manifest is admitted, **Then** built-in defaults, preset choices, manifest defaults, item choices and explicit request overrides apply in that order; absent fields do not erase inherited choices.
3. **Given** a missing, deleted, ambiguous or incompatible selection, **When** admission is requested, **Then** the affected item fails before work is queued, with an actionable diagnosis and no silent fallback.

### User Story 2 - Keep elected work stable (Priority: P1)

An operator elects an existing configured pipeline through a preset and can inspect queued work, resume or retry it without a later preset edit changing the original choice.

**Why this priority**: Reusable imports must launch real work and preserve execution provenance.

**Independent Test**: Import synthetic audio with a configured fixture pipeline, edit/delete the preset after enqueue and retry the durable work.

**Acceptance Scenarios**:

1. **Given** a usable configured pipeline, **When** a preset import succeeds, **Then** it queues the elected processing work using existing runtime execution and exposes effective choices and queued job IDs.
2. **Given** work already queued, **When** the preset or pipeline changes or disappears, **Then** retries retain the admitted preset identity/revision and frozen effective configuration.
3. **Given** unavailable processing dependencies, **When** import is requested, **Then** readiness diagnostics explain the missing dependency; no fictitious processing success or unselected external route is reported.
4. **Given** no selected preset or a deliberately admission-only preset, **When** import is requested, **Then** existing explicit import behavior remains available with no mandatory processing or per-item review.

### User Story 3 - Follow release-matched examples (Priority: P1)

An operator follows current introductory and advanced documentation and receives the stated result using an extracted package.

**Why this priority**: Unsupported commands and incomplete payloads prevent first successful use.

**Independent Test**: Execute documented workspace, import, transcript, work, speaker/model and preset examples against packaged binaries with synthetic fixtures.

**Acceptance Scenarios**:

1. **Given** current documentation, **When** examples are run in their documented order, **Then** command groups, flags, required inputs, IDs and revisions are valid and the stated effects are demonstrated.
2. **Given** the same current page online and offline, **When** it is read, **Then** capability, date uncertainty and mapping concurrency descriptions agree with the runtime.
3. **Given** immutable historical release pages, **When** current corrections publish, **Then** historical archives remain intact and current publication status is stated once in the index with aligned software/schema/documentation versions.

### Edge Cases

- Duplicate names are rejected; a legacy ID/name collision is diagnosed rather than guessed.
- Stale update/delete revisions cannot overwrite another edit; unknown preset formats fail before enqueue.
- Explicit false/zero values, destination lists, date fields and pipeline opt-out retain their presence semantics.
- Mixed manifest rows preserve existing partial-result boundaries; invalid rows never acquire durable work.
- Credentials remain references, never values in presets, work snapshots, diagnostics or documentation.
- Missing optional processing tools preserve actionable readiness and already-authorized routing.
- Catalog migrations and backup/restore preserve presets on both supported catalog backends.
- Desktop forms expose loading, empty, conflict and error states, labeled controls and keyboard operation.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Provide versioned saved import preset create/list/show/update/delete through shared CLI and desktop operations, with stable identity, positive revision and unambiguous names.
- **FR-002**: Persist nonsecret destination, date/time interpretation and optional configured pipeline choices with the catalog and its portable backups on every supported catalog backend.
- **FR-003**: Resolve preset ID/name before affected admission work is queued; reject missing/deleted/ambiguous/incompatible selections with supported next actions.
- **FR-004**: Apply deterministic presence-aware precedence: built-in defaults < selected preset < manifest defaults < manifest item < explicit request overrides. Preserve existing behavior when no preset is selected and document any intentional merge change.
- **FR-005**: Freeze preset identity/revision, effective import options and elected pipeline configuration in durable work. Later edits/deletion cannot change retries.
- **FR-006**: Queue elected processing with the established execution runtime, resolve pipeline/model dependencies at admission, expose queued job IDs and never silently route to an unelected provider.
- **FR-007**: Support intentional admission-only presets and actionable processing readiness failures without adding per-item approval requirements.
- **FR-008**: Align direct, refresh, JSON/CSV manifest and desktop import behavior, including explicit clearing and date provenance.
- **FR-009**: Replace unsupported current public command examples with minimal complete executable procedures, including ID/revision retrieval, expected results and task-oriented explanations.
- **FR-010**: Reconcile current source, online and offline documentation with implemented capabilities, partial-date warnings and recording/mapping concurrency semantics; preserve immutable prior-release archives and version alignment.
- **FR-011**: Qualify documented examples against extracted CLI and desktop packages using synthetic/licensed fixtures, and fail automated qualification when required command grammar or effects drift.
- **FR-012**: Preserve existing authentication, catalog/workspace boundaries, protected credentials, source integrity and precise date representation. Do not add unrelated eligibility or review gates.

### Key Entities

- **Saved import preset**: Catalog-owned identity, revision, name, format version and nonsecret reusable choices.
- **Effective import selection**: Preset provenance plus choices after deterministic overrides, attached to admitted durable work.
- **Elected processing work**: Frozen configured pipeline and dependency references connected to the admitted recording.
- **Executable documentation procedure**: Release-matched steps, fixture inputs, captured identities/revisions and observable expected effects.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every documented preset lifecycle action gives equivalent persisted results in CLI and desktop.
- **SC-002**: All direct/refresh/JSON/CSV precedence cases and invalid selections produce the specified destination/date/work outcomes, with zero work queued for invalid affected items.
- **SC-003**: Every retry in the edit/delete regression scenarios retains the original selection and effective workflow.
- **SC-004**: Every qualified current introductory/advanced procedure runs against extracted packages and produces its documented result.
- **SC-005**: Current online/offline capability and command descriptions contain zero unsupported examples; historical archive integrity checks pass.

## Assumptions

- Existing configured pipelines and shared durable execution are reused. Guided initial setup (#66), managed workers (#69), package installation prose (#63) and unrelated issues remain separate slices.
- Presets are catalog-scoped, matching pipeline ownership and existing catalog portability.
- Name uniqueness and optimistic revision checks follow established editable configuration behavior.
- The current unreleased 1.0.1 candidate is the documentation and contract target; shipped 1.0.0 archives are immutable.
- Owner authorization covers publication and two review rounds, ending at final review/merge handoff.
