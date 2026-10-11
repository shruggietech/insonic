# Feature Specification: S019 CLI operator feedback

**Feature Branch**: `codex/019-cli-operator-feedback`

**Created**: 2026-10-11

**Status**: Implemented; owner final review and merge pending

**Input**: Deliver issues #56, #57, #58 and #61 under Spec Kit autopilot, publish an official PR, resolve all review findings within the two-round limit, and stop for owner final review.

## User Scenarios & Testing

### User Story 1 - Understand command results and corrections (Priority: P1)

An operator receives readable results, named conditions and concrete next commands without having to decode internal inventories or encoded documents.

**Why this priority**: Successful admission and truthful failure classification already exist; their meaning must reach the operator.

**Independent Test**: Exercise successful, empty, missing, stale, duplicate, damaged and unavailable conditions with human and machine output.

**Acceptance Scenarios**:

1. Given a library, when showing workspace, doctor, inventories, settings or credentials, then summaries identify the workspace, client version, actual state and relevant next action.
2. Given a missing input, stale revision, duplicate credential, damaged backup or reused source map, when an operation fails, then its correction describes the actual condition and never invents another owner.
3. Given capture or document content, when human output is selected, then content is readable or an exact export/page command is provided; large encoded documents and secrets are not dumped.
4. Given JSON output, when any presentation controls are selected, then the versioned envelope, exact integers, stream and exit contract remain unchanged.

### User Story 2 - Diagnose runtime startup (Priority: P1)

An operator learns why a hidden runtime failed before waiting out an unrelated readiness timeout, and can elect bounded diagnostic logs.

**Why this priority**: Cold credential and profile failures otherwise conceal actionable corrections.

**Independent Test**: Start controlled success, early failure, stalled and wrong-passphrase children, including a parent that exits after readiness.

**Acceptance Scenarios**:

1. Given an early bootstrap failure, when startup is requested, then a classified cause and correction arrive within two seconds of child termination.
2. Given successful startup, when the launching client exits, then detached runtime ownership remains available and all console launches remain hidden.
3. Given elected diagnostic logging, when commands and failures occur, then records retain correlation and generated categories with at most three rotated 1 MiB files and no credential values, protected input, prompts, provider bodies or arbitrary child output.
4. Given stale ownership, a competing owner or a startup timeout, then existing recovery and deadline semantics remain correct.

### User Story 3 - Follow work through completion (Priority: P2)

An operator distinguishes queued, executing, completed, failed and partially useful work and understands that a wait timeout leaves work running.

**Why this priority**: Import and acquisition can take time and retain useful output despite unresolved metadata.

**Independent Test**: Simulate pending/running states, changed stages, completion, partial warnings, failure, interruption, cancellation, retry and a wait deadline.

**Acceptance Scenarios**:

1. Given an interactive human wait, when durable state or meaningful stage/count changes, then bounded progress appears without fictitious percentages.
2. Given redirected or machine output, then no transient progress, cursor controls or diagnostic chatter contaminates result streams.
3. Given an unresolved date in accepted media, then completion remains successful with visible item/date warnings and repair guidance, without a mandatory review gate.
4. Given a wait timeout or interruption, then the last state and show/wait/cancel or retry commands explain what continues and what needs action.

### User Story 4 - Inspect usable model resources (Priority: P2)

An operator inspects exact model identity, incompatibility, acquisition state, file roles, leases and speaker dataset readiness.

**Why this priority**: Models must be inspectable before execution is elected.

**Independent Test**: Compare base and alias inspection, declared/acquiring/available/damaged resources, multi-role files, leases and empty/populated datasets.

**Acceptance Scenarios**:

1. Given incompatible base and alias references, when inspecting both, then target identity and compatibility diagnostics agree; execution selection still refuses incompatible resources.
2. Given a registered but absent or damaged model, when verifying or materializing, then acquisition/verification state and the next command remain available.
3. Given a materialized bundle, then named manifest roles, exact publication identity and lease release instructions accompany local paths.
4. Given an empty dataset, then inspection explains lack of trainability and how to select usable evidence; elected training retains its existing rejection.

### Edge Cases

- Control characters, very large results and arbitrary external messages cannot become terminal controls, log payloads or unbounded human output.
- A broken output/log destination cannot report false command success or leak protected input.
- Quiet mode removes informational human output/progress, retains errors/warnings and does not remove machine result envelopes.
- Startup input is consumed once; detached children neither prompt nor silently reroute to providers.
- Progress and inspection do not modify frozen elections, durable receipts, imported source observations or model manifest identities.

## Requirements

### Functional Requirements

- **FR-001**: Provide task-level human summaries and readable inventory/empty states for the complete admitted command inventory, retaining exact numeric tokens and advanced detail through JSON.
- **FR-002**: Distinguish missing inputs/resources, invalid fields, stale revisions, duplicate credentials, corruption, ownership and dependencies with bounded generated context and concrete supported corrections.
- **FR-003**: Explain stored/locked versus loaded owner-scoped credentials and configured versus verified provider/task state truthfully.
- **FR-004**: Render capture/document content readably or provide an explicit exact-document retrieval/export route without dumping encoded binary content.
- **FR-005**: Retain bounded startup readiness/failure evidence and detect early child failure within two seconds after termination while preserving hidden detached ownership and protected stdin.
- **FR-006**: Expose verbose/debug/quiet and log-level/log-file controls; log only correlation, locally generated categories and bounded status, rotating at 1 MiB with two retained predecessor files.
- **FR-007**: Show meaningful state/stage/count changes during interactive human waits, with no percentage unless a real denominator is available.
- **FR-008**: Show clean/partial/failed/cancelled/interrupted batch summaries and affected-item date warnings with repair commands; wait limits never cancel work.
- **FR-009**: Preserve JSON envelopes, streams, exits, exact integers, redirected output cleanliness and existing one-process polling.
- **FR-010**: Preserve incompatible model inspection targets through aliases while retaining strict execution election.
- **FR-011**: Expose declared/acquiring/available/verified/materialized/damaged context, named manifest file roles and lease identity/release actions.
- **FR-012**: Show dataset trainability and evidence-selection remedies without weakening empty-dataset rejection or adding training approval gates.
- **FR-013**: Align shared runtime results and affected current online/offline documentation while preserving historical released bytes, versions, identities and receipts.
- **FR-014**: Deliver focused regressions, redaction/ownership checks, package qualification and required full checks, publish the authorized PR, satisfy all findings within at most two review rounds, and stop before merge/release.

### Key Entities

- Operator result: operation, bounded readable fields, state, warnings, exact identities and next actions.
- Diagnostic record: correlation, level, generated event/category, state and timestamp, excluding secret and external payloads.
- Startup report: readiness or classified failure tied to the launched child and request.
- Work observation: immutable work identity, state/stage, available counts and item warning references.
- Model inspection: exact target, availability, compatibility, declared role/publication/lease information and dataset trainability.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every admitted operation has bounded human output; audited wording failures have explicit regression coverage and a supported correction.
- **SC-002**: Controlled early child failures are reported within two seconds after termination; successful runtime use survives launching-parent exit.
- **SC-003**: No synthetic secret, arbitrary child/provider payload or terminal control appears in logs or human diagnostics; log retention is bounded to three 1 MiB files.
- **SC-004**: Progress and completion distinguish all specified work states and partial dates without contaminating redirected/JSON streams.
- **SC-005**: Base/alias inspections retain equal target context, model role/lease facts remain inspectable, and empty datasets stay nontrainable.
- **SC-006**: Required local checks and final-head CI pass, every review thread is resolved, and the published PR links all four issues.

## Assumptions

- Scope excludes complete documentation/preset repair (#62/#70), ZIP/MSI installation and first-run/backend/worker setup (#63-#69). Associated current docs are updated as part of this slice.
- 1.0.1 remains the unreleased synchronized software/docs/schema candidate; released 1.0.0 bytes remain immutable.
- Recognition, diarization and model training execution are not part of this slice or ordinary CI qualification.
- Existing route authority, unattended configured work and owner merge/release boundaries persist.

## Clarifications

### Session 2026-10-11

- Human summaries apply to both terminal and redirected human output; transient progress is terminal-only. JSON remains the exact automation interface.
- Quiet suppresses informational human summaries and progress, preserving errors/warnings and all JSON envelopes. Verbose/debug add locally generated diagnostic detail, never raw child/provider output.
- A partial date observation alone is a warning with successful accepted media; failed items keep failed-work semantics.
- Inspection may return incompatible context successfully; execution election remains strict.
