# Feature Specification: S035 HTTP evidence extraction and provider diagnostics

**Feature Branch**: codex/035-http-adapter-contracts
**Created**: 2026-10-17
**Status**: Draft
**Input**: Run S035 under autopilot for #146, #127 and #130, publish a PR, handle at most two review rounds, stop before owner merge.

## User Scenarios & Testing

### User Story 1: Reach the elected extraction provider (Priority: P1)

A configured operator submits HTTP assertion extraction using the documented identifier. The request reaches the selected provider and publishes only assertions backed by the selected current source cues and authoritative clocks. Existing requests using the older HTTP spelling continue to work.

**Why this priority**: #146 prevents the configured HTTP extraction capability from working at all through the CLI.

**Independent Test**: An installed CLI submits both supported spellings to a deterministic loopback provider and commits valid cited assertions with unchanged source clocks.

**Acceptance Scenarios**:

1. Given current recording evidence and an explicitly configured provider, when extraction is elected with either supported spelling, then work reaches that provider and records equivalent canonical adapter provenance.
2. Given unknown, foreign-media or other-chunk cue references, when provider output is validated, then invalid assertions are excluded and bounded diagnostics explain rejection according to existing chunk publication semantics.
3. Given unknown response members or attempted clock overrides, when output is received, then it cannot overwrite authoritative source clocks or publish the invalid provider output.
4. Given stale revision or document identity, when extraction is submitted, then the existing conflict fence still prevents publishing stale assertions.

### User Story 2: Understand provider failures (Priority: P2)

An operator can distinguish authentication, missing service/model, throttling/unavailability, connection failures, invalid provider responses, deadline expiry and deliberate cancellation. Equivalent outcomes have useful bounded explanations across the generic HTTP and Ollama adapters.

**Independent Test**: Deterministic providers exercise 401/403, 404, 429/503, malformed/truncated/oversized responses, redirects, connection refusal, deadline expiry and user cancellation through both adapters.

**Acceptance Scenarios**:

1. Given an otherwise valid request, when the provider returns invalid output, then the explanation identifies a provider response fault rather than invalid user input.
2. Given an expired provider deadline, when the operation ends, then diagnostics distinguish that timeout from user cancellation.
3. Given provider secrets or arbitrary response contents, when any failure is reported, then they are absent from human/machine diagnostics and logs.

### User Story 3: Explain the actual output bound (Priority: P2)

An operator receives an operation-specific explanation when artifact materialization, native query results or provider response bytes exceed the elected bound, retaining the existing machine error category.

**Independent Test**: A 25-byte artifact with a 24-byte bound, a native result above the row/byte ceiling and an oversized HTTP proposal each retain output_limit and provide the relevant correction.

**Acceptance Scenarios**:

1. Given a materialization byte ceiling, when the selected artifact exceeds it, then the explanation identifies materialization bytes and the elected bound.
2. Given a native result row/byte ceiling, when output exceeds it, then the explanation identifies query results and suggests narrowing the query.
3. Given a provider response ceiling, when a response exceeds it, then the explanation identifies provider response bytes and the configurable limit.

### Edge Cases

Legacy aliases, omitted/default extraction choices, chunk overlap and duplicate citations, cross-media identities, source clock precision, unknown output fields, headers followed by a slow or truncated body, caller cancellation racing a deadline, redirects, unavailable credentials and response bodies containing dummy secret markers all require deterministic coverage. Optional configurations must not elect new routes or fallback providers implicitly.

## Requirements

### Functional Requirements

- **FR-001**: Public extraction input and runtime normalization MUST accept canonical insonic-http plus legacy http, normalize to one canonical identity and preserve cue-statement defaults (#146).
- **FR-002**: Extraction MUST publish only current, source-owned, current-chunk citations with authoritative clocks; malformed/unknown-field/clock-spoof outputs MUST remain rejected under existing chunk diagnostic and revision semantics (#146).
- **FR-003**: Generic HTTP and Ollama assistance MUST distinguish provider authentication, missing resources, retryable service failure, transport, malformed response, timeout and caller cancellation with bounded stable reasons and useful corrections (#127).
- **FR-004**: Artifact materialization, native query results and provider response ceilings MUST retain output_limit while identifying the correct bound and recovery (#130).
- **FR-005**: All diagnostic surfaces MUST exclude credentials, arbitrary provider bodies, private source text and unsafe endpoint details; existing route, redirect, size and secret-handling controls MUST remain in force.
- **FR-006**: CLI schemas, generated contextual help, shared runtime and GUI consumers MUST agree; preserve supported backend parity and existing exact numeric/source identity behavior.
- **FR-007**: Installed Windows/Linux CLI qualification MUST exercise successful extraction plus adversarial citations, provider failures and operation-specific limits using deterministic generated/licensed fixtures without inference engines or owner media.

### Key Entities

- Extraction election: adapter identity/contract, provider/model, current recording/document fence and chunk policy.
- Provider output: bounded structured assertions/proposals referencing current owned evidence.
- Operator condition: existing machine category plus stable bounded reason and correction.
- Maintained qualification fixture: deterministic selected media, multi-chunk cues, loopback providers and expected durable results.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Both supported extraction spellings reach a fixture provider and commit the expected assertions; all foreign/unknown/other-chunk citations and attempted clock overrides are excluded.
- **SC-002**: Every provider failure scenario has the intended classification in both adapters; caller cancellation and deadline expiry remain distinguishable.
- **SC-003**: All three output-bound reproductions retain their machine category and report the actual bounded operation in human and machine output.
- **SC-004**: Dummy credential/body/source markers never appear in error messages or diagnostic logs in the regression matrix.
- **SC-005**: Required repository, affected product/site and exact-head hosted checks pass, all findings from at most two review rounds are addressed, and the PR remains unmerged for the owner.

## Assumptions and Scope

Exactly #146, #127 and #130 are in scope. #114 supplies maintained-fixture guidance but remains broader open work; #147 remains the audit index. Other CLI, desktop/installer, training/artifact corruption and Unicode defects remain outside this slice. No release, merge, auto-merge, new production provider routing or owner-media/inference run is included. The owner explicitly authorizes push and PR publication, superseding the autopilot skill's default pre-push halt.

## Clarifications

### Session 2026-10-17

- Autopilot decision: keep insonic-http canonical and accept http as a compatibility alias, rather than break the documented older input or retain two provenance identities.
- Autopilot decision: preserve existing per-chunk rejection/publication and stale revision fencing; the slice does not redefine malformed evidence as authoritative or introduce mandatory per-item review.
- Autopilot decision: add bounded reasons while preserving output_limit and existing compatible Ollama reasons; never echo raw provider bodies as diagnostic detail.
