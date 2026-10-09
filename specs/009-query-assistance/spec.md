# Feature Specification: S009 Optional query assistance

**Feature Branch**: `codex/009-query-assistance`
**Created**: 2026-10-08
**Status**: Draft
**Input**: Owner-authorized S009, issue #13. Suggestions through elected local/hosted adapters, optional validated read-only auto-run and direct querying independent of assistance.

## Clarifications

### Session 2026-10-08

Autopilot decisions: configuration is explicit per request or saved workspace setting; saving never contacts a provider. Suggestions are the default mode. An elected auto-run executes only a fully validated proposal through the ordinary current query operation. Local and hosted services implement a documented versioned adapter contract; no ambient provider or model selection. Provider receives schema/capabilities and the prompt, never library result excerpts unless explicitly elected. Workspace configuration and results remain bound to their workspace. No mandatory per-suggestion approval is introduced.

## User Scenarios & Testing

### User Story 1 - Obtain an inspectable suggestion (Priority: P1)

A user selects a local or hosted assistant and asks a question about their library. They receive the proposed query, explanation and model/provider identity without running that query by default. They can edit, run or save it through ordinary exploration.

Acceptance scenarios:
1. Given configured assistance, requesting a proposal returns a complete current query definition, typed parameters, explanation and provenance without executing it.
2. Given native or normalized output, invalid shape, disallowed native operations and a wrong backend dialect are rejected before query execution.
3. Given disabled assistance or a provider failure, direct querying remains available with an actionable fixed error.

### User Story 2 - Elect bounded read-only auto-run (Priority: P1)

A user elects automatic read-only execution and receives the shown proposal with current results under configured limits.

Acceptance scenarios:
1. Given an elected auto-run mode, a valid proposal runs once through shared querying; its visible result retains current source identity and clocks.
2. Given rejected output, cancellation, timeout, oversized data or incompatible dialect, no generated query executes.
3. Given prompt-like instructions in supplied data, the generated output remains untrusted and cannot alter execution mode, routes, credentials or query limits.

### User Story 3 - Configure and use assistance across interfaces (Priority: P2)

A user saves their elected adapter/model/route and uses the same settings from CLI and desktop. Configuration inspection sends no requests. Results show enough context to inspect and edit the generated definition.

Acceptance scenarios:
1. Given saved settings, restart preserves them and a concurrent edit fails with a revision conflict instead of overwriting another choice.
2. Given a pending desktop request, editing the prompt/settings or leaving the route prevents stale results from appearing as the current proposal.
3. Given no settings, the user can configure assistance or continue direct exploration without an account or model download.

### Edge Cases

Empty/oversized prompts; non-JSON, trailing or unknown output fields; invalid typed parameters; wrong backend; unsupported native functions; redirect/authentication failure; oversized/slow responses; disabled settings; expected-revision conflicts; concurrent graph/profile changes; sparse or unavailable projection; current recording replacement; late UI completion; unmount and double submission. Explicitly elected excerpts must come from bounded current results and preserve exact integer clocks.

## Requirements

### Functional Requirements

- **FR-001**: Assistance is optional and direct query operations remain independent of its availability.
- **FR-002**: Users can inspect/save revisioned workspace settings for elected local/hosted adapter, model, endpoint, opaque credential reference, mode, limits and optional result context; saving/inspection does not execute a model or contact a provider.
- **FR-003**: Only the elected route/model/credential executes. Failures never select a fallback. Responses and diagnostics do not expose credential values or provider error bodies.
- **FR-004**: The assistant receives a bounded prompt plus selected backend schema, capabilities and declared dialects. Source/result excerpts are excluded by default and transmitted only when explicitly elected.
- **FR-005**: Default suggestion mode returns a complete editable query, typed parameters, explanation, validation and provider/model/context provenance without running the proposal.
- **FR-006**: Validate normalized/native definitions and typed parameters; native text must satisfy existing read-only rules and the selected backend dialect before execution. Invalid or unsupported output cannot run.
- **FR-007**: Explicit auto-run uses ordinary current query execution under configured and existing maximum limits. Generated output cannot alter route, mode, authorization or limits.
- **FR-008**: Runtime response, prompt, excerpt, duration and result limits are finite; cancellation and late results cannot become accepted current UI state.
- **FR-009**: CLI and desktop share operations; desktop exposes configuration, prompt, mode, visible proposal, diagnostics and current results with edit/run/save actions and accessible controls.
- **FR-010**: Preserve exact source identifiers/clocks and current-result rules; assistance retains no alternate transcript, assignment list or persisted result snapshot.
- **FR-011**: Check workspace identity and settings revision; concurrent configuration/profile changes are diagnosed rather than executing on an unintended target.
- **FR-012**: Required checks use deterministic provider replies and real existing query/backend paths, never model loading, inference or weights. Publish self-contained matching contracts/help and implementation evidence.

### Key Entities

- Assistance settings: revisioned workspace election and limits, with opaque credentials.
- Assistance request: prompt, explicit settings election/overrides and optional current query context.
- Assistance proposal: query definition/typed parameters, explanation and bounded validation/provenance.
- Assistance result: proposal plus optional current read-only results and diagnostics, never a persisted alternate document.

## Success Criteria

- **SC-001**: Every valid suggestion can be inspected, edited and run through the existing direct query path in both interfaces; suggestion mode performs zero proposal executions.
- **SC-002**: Every invalid/unsupported generated query in acceptance fixtures performs zero executions, and each valid auto-run performs one bounded read.
- **SC-003**: Configuration survives restart; concurrent changes and cross-workspace requests fail without route substitution.
- **SC-004**: Disabled/failing assistance leaves direct queries functional; every stalled fixture terminates within its elected limit.
- **SC-005**: Required checks, published contracts and mounted platform journeys pass with zero model initialization, inference or weight downloads.

## Scope and assumptions

Scope includes issue #13 and S008 integration. Training (#15), backups/acquisition/release (#14), provider-specific model installations and a public release are separate work. Existing provider protocol can be implemented by local or hosted services; a generic API is not silently considered compatible. Version remains 0.0.0. Owner authorizes publication and bounded two-round review protocol; halt at the final squash merge ritual.
