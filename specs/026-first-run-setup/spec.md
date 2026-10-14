# Feature Specification: Optional, resumable first-run setup

**Feature Branch**: `codex/026-first-run-setup`

**Created**: 2026-10-13

**Status**: Specified

**Input**: S026, complete issue [#66](https://github.com/shruggietech/insonic/issues/66) under authorized autopilot.

## User Scenarios & Testing

### User Story 1 - Reach a first import with local defaults (Priority: P1)

A new user chooses an editable library location, accepts local defaults and reaches the normal import screen without needing an account or model download.

**Why this priority**: The installed application should explain a usable starting point.

**Independent Test**: Start with no library, create the proposed location, complete or skip optional steps and import a fixture through either client.

**Acceptance Scenarios**:

1. **Given** a fresh client, **When** setup opens, **Then** it suggests an editable writable location outside the application installation and explains control files, media/model storage and disposable cache.
2. **Given** local defaults, **When** the user continues, **Then** the saved library identity and ordinary import route work without an API key.
3. **Given** a user who chooses Set up later, **When** they enter the normal application, **Then** setup does not block ordinary commands.

### User Story 2 - Resume without duplicating accepted work (Priority: P1)

A user saves a step, closes a client, then resumes from either client with accepted settings and durable work intact.

**Why this priority**: Downloads and configuration can outlive a window or fail independently.

**Independent Test**: Save, retry a request after an unknown response, reopen, change settings concurrently and inspect the reconciled state.

**Acceptance Scenarios**:

1. **Given** saved choices, **When** another client resumes, **Then** it observes the same revision, steps and effects.
2. **Given** an unknown save outcome, **When** the same request is retried, **Then** it returns the accepted state without duplicate profiles or jobs.
3. **Given** a stale draft, **When** a save races an accepted revision, **Then** it reports a conflict and preserves the newer choice.
4. **Given** active work, **When** setup is deferred or a stage is skipped, **Then** accepted configuration and running work remain intact.

### User Story 3 - Elect optional capabilities and inspect readiness (Priority: P2)

Users configure supported backends, local processing or connected services through existing operations, then see what is usable and what needs attention.

**Why this priority**: A configuration declaration alone does not establish runtime usability.

**Independent Test**: Exercise local defaults, an alternate backend plan, a failed dependency and an elected connected service without automatic routing.

**Acceptance Scenarios**:

1. **Given** optional stages, **When** a user selects a capability, **Then** the existing plan/preflight, protected credential and durable preparation controls are available.
2. **Given** missing dependencies, **When** the final summary is opened, **Then** each affected task names its corrective action while ready import remains available.
3. **Given** a credential form, **When** a friendly name and secret are entered, **Then** only its opaque reference can persist in setup and authentication remains a separate observation.

### Edge Cases

- A proposed directory is unwritable, belongs to an existing library or is inside installation bytes.
- A response is lost after commit; requests are retried across client restart.
- Settings or backend profiles change while a draft is open.
- A user skips processing while model acquisition is active.
- A secret, signed URL or unknown arbitrary field is supplied to a setup draft.
- Setup is reopened after completion, a library moves, or a portable copy is restored.
- Work fails, is cancelled or succeeds after the setup client closes.

## Requirements

### Functional Requirements

- **FR-001**: Offer optional setup on fresh entry and allow explicit reopening through Settings and noninteractive commands.
- **FR-002**: Present five logical stages: workspace/location, backends, processing, connected services and first import, with visible progress, Back, Skip, Retry and Set up later.
- **FR-003**: Suggest an editable user-writable library location outside installation bytes; distinguish control/catalog files, large artifacts and disposable cache, with observed or explicitly unknown space estimates.
- **FR-004**: Persist versioned nonsecret choices, draft revision, stage states and configuration observations through shared operations. Completed, skipped, pending and needs attention must be distinct.
- **FR-005**: Save through revision checks and stable request identity; reconcile unknown outcomes and reject stale writes without overwriting accepted state.
- **FR-006**: Reuse existing library initialization and identity; never overwrite an existing library or silently replace configured routes.
- **FR-007**: Orchestrate existing supported backend, worker, model, service discovery and import capabilities; setup must not create a parallel configuration engine.
- **FR-008**: Keep local defaults independent of accounts and processing. Remote routing, model downloads and first import require the user's explicit election through the normal operations.
- **FR-009**: Supply friendly task-specific protected credential entry and automatically generated opaque references; never persist credential values in setup state, command arguments or diagnostics.
- **FR-010**: Derive readiness from current shared capability observations, retain configuration revisions and distinguish registered, configured, authenticated and usable states.
- **FR-011**: Deferring/cancelling the journey preserves accepted settings and ongoing work; work cancellation/retry uses existing durable work controls.
- **FR-012**: Provide accessible headings, step progress, keyboard navigation, focus on changed steps and actionable announced errors; forms remain usable at increased scaling.
- **FR-013**: End with concrete import/settings/jobs actions and exact unresolved prerequisites; ordinary operations stay available regardless of setup completion.
- **FR-014**: Keep current online/offline documentation, command help and machine contracts aligned, including removal of stale managed-processing claims.

### Key Entities

- **Setup state**: Version, library identity, revision, current stage, status and bounded nonsecret choices.
- **Setup stage**: Logical group, completion/skip state, observed configuration revision and corrective actions.
- **Effect reference**: Accepted profile, model or work identity retained through normal authoritative operations.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Both clients can reach a first fixture import using local defaults with zero account registrations or model downloads.
- **SC-002**: Replaying each tested accepted save produces zero duplicated profiles or jobs, including after client restart.
- **SC-003**: Every tested stale write preserves the newer state and exposes a corrective conflict response.
- **SC-004**: Every optional stage can be skipped and reopened without blocking ordinary import or changing accepted routes.
- **SC-005**: Credential sentinel values appear zero times in persisted setup state and public diagnostic responses.
- **SC-006**: Keyboard checks cover all steps, focus changes, error correction and completion; automated product/site checks pass.

## Assumptions

- S022 backend/service capabilities and S025 worker/model readiness are merged dependencies, not new engines to implement.
- Windows and Linux x86-64 clients share behavior; native credential and dialog integration retains platform ownership.
- S026 excludes MSI authoring, launcher wording approval, installer screenshots, release tagging and merging. Those remain #65/#64.
- Existing configured work remains authorized. No new per-item approval gate is introduced.

## Clarifications

### Session 2026-10-13

- Q: Does setup completion gate ordinary operations? A: No; the owner requires an optional, resumable journey.
- Q: Does setup cancellation cancel accepted downloads or configuration? A: No; defer preserves effects and explicit work cancellation uses Jobs.
- Q: Does S026 include MSI labels and installer screenshots? A: No; those remain the later #65/#64 bundle.

Coverage: functional scope, state lifecycle, integration, error behavior, accessibility and completion are clear. Implementation choices are resolved in planning. Requirements checklist remains 8/8. No unresolved owner decision.
