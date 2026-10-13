# Feature Specification: Backend configuration and local query assistance

**Feature Branch**: `codex/022-backend-service-setup`

**Created**: 2026-10-13

**Status**: Implemented and qualified; final owner review and merge pending

**Input**: S022 under autopilot for [#67](https://github.com/shruggietech/insonic/issues/67) and [#68](https://github.com/shruggietech/insonic/issues/68). Push and official PR authorized; maximum two review rounds; hand off after green CI, without merging.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Configure the three backend roles (Priority: P1)

A user selects artifact storage, operational catalog and graph projection through matching CLI and desktop controls, using local defaults or any supported alternative without hand-editing configuration.

**Why this priority**: Backend selection must be usable before a guided first-run journey can orchestrate it.

**Independent Test**: Initialize each supported backend combination from a complete plan; inspect the persisted selections and exercise the relevant adapters.

**Acceptance Scenarios**:

1. **Given** a fresh location, **When** defaults are selected, **Then** filesystem, SQLite and LadybugDB are initialized without an API key.
2. **Given** elected alternative backend profiles and credentials, **When** initialization is requested, **Then** credentials are available before the first catalog connection and exact selected profiles have generated identities and revisions.
3. **Given** existing accepted data, **When** an invalid or failed replacement is submitted, **Then** the active profiles and accepted data remain intact; changing populated storage follows explicit migration/backup operations.

### User Story 2 - Inspect readiness before provisioning (Priority: P1)

A user preflights complete profile plans and receives specific corrections without altering existing databases or uploading library contents.

**Independent Test**: Probe successful and denied fixtures, invalid fields, TLS identity failures, deadlines and cancellation, and compare target state before/after.

**Acceptance Scenarios**:

1. **Given** six supported backend choices, **When** preflight runs, **Then** syntax, reachability, transport, authentication, compatibility and untested capabilities are independently reported.
2. **Given** remote PostgreSQL plaintext disguised as local transport, **When** configuration is inspected or submitted, **Then** it cannot silently be accepted as local; verified TLS remains the default and compatibility consequences are documented.
3. **Given** a cancelled or timed-out probe, **When** results are returned, **Then** current profiles remain unchanged and no credential value enters output or saved plans.

### User Story 3 - Discover and elect local services (Priority: P2)

A user discovers service candidates and models from package declarations, configured endpoints, loopback defaults, environment hints and known installations, then deliberately selects an endpoint and model.

**Independent Test**: Discover running/stopped/missing and multiple service fixtures with custom endpoints; verify metadata-only requests and unchanged configured routes.

**Acceptance Scenarios**:

1. **Given** candidate endpoints, **When** discovery runs, **Then** source/time and installed, running, reachable, task-compatible, configured and tested states remain distinct.
2. **Given** an external hint or a hosted model advertised by a loopback server, **When** candidates are shown, **Then** actual local/cloud/unknown routing is shown and nothing is silently adopted.
3. **Given** failed, oversized, redirecting or stale metadata, **When** a probe finishes, **Then** its bounded result explains the correction without prompts, library data or secret disclosure.

### User Story 4 - Use native Ollama query assistance (Priority: P2)

A user elects a compatible Ollama model, asks for a query proposal and optionally runs the validated read-only query through the existing exploration workflow.

**Independent Test**: Use deterministic native protocol fixtures to translate valid proposals, reject malformed results and execute only the existing validated query operations.

**Acceptance Scenarios**:

1. **Given** configured local Ollama, **When** assistance is requested, **Then** no fabricated API key is required, endpoint/model/route are inspectable and existing query budgets and validation remain effective.
2. **Given** an explicitly elected synthetic task test, **When** it runs, **Then** possible model loading is explained, no library content is sent and success records tested status for that exact selection.
3. **Given** missing models, authentication rejection, busy/rate limits, protocol mismatch or malformed responses, **When** assistance fails, **Then** actionable redacted errors are returned without silent rerouting.

### Edge Cases

Concurrent profile revisions, lost save responses, preexisting destinations, credentials unavailable before a remote catalog exists, database schema absent versus incompatible, read permissions without writes, SQLite on network storage, custom CA and non-loopback plaintext, service restarts, OLLAMA_HOST without scheme, missing binaries, cloud models served locally, model capability absence, redirected probes, cancellation during metadata and inference, oversized JSON and provider text resembling an arbitrary database query.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Complete versioned plans MUST cover filesystem root; S3 endpoint/bucket/prefix/region/addressing/authentication; SQLite path; PostgreSQL host/port/database/schema/TLS/CA/credential; LadybugDB path; ArcadeDB endpoint/database/credential/dialect.
- **FR-002**: CLI and desktop MUST expose equivalent shared configuration/preflight/provisioning operations and useful field errors, retain local defaults and generate profile IDs/revisions.
- **FR-003**: Credentials MUST be registered through protected input before the initial catalog connection, referenced opaquely, and absent from plan files, arguments and diagnostic output. Registration MUST NOT imply authentication success.
- **FR-004**: Bounded cancellable preflight MUST distinguish syntax, reachability, TLS, authentication, schema/target compatibility and untested capabilities without writes, migrations or library upload.
- **FR-005**: Explicit provisioning MUST be separate from preflight; rejected plans MUST preserve active configuration/data and revision history. Existing-data relocation MUST remain the established migration/backup workflow.
- **FR-006**: Local PostgreSQL transport MUST resolve to actual local transport, or intentionally remote plaintext MUST be named accurately. Verified identity MUST remain default. SQLite WAL placement guidance MUST be specific and not restrict remote artifact backends.
- **FR-007**: Discovery MUST inventory package resources, configured endpoints, loopback defaults, OLLAMA_HOST hints and known installations/PATH candidates with discovery source/time, preserving manual/custom endpoint choices.
- **FR-008**: Metadata probes MUST have deadlines, cancellation and response caps, send no prompts/library contents, and use shared runtime operations.
- **FR-009**: Installed, running, reachable, task-compatible, configured and tested states MUST remain separate. Actual local/cloud/unknown model routes MUST be inspectable; adoption and hosted-model use MUST require election.
- **FR-010**: A versioned native Ollama adapter MUST translate into the existing proposal contract and preserve aggregate query budgets, parameter validation and validated read-only execution.
- **FR-011**: Local Ollama MUST work without a fabricated key. Explicit configured authentication MUST be honored for custom endpoints without exposing values or changing routes.
- **FR-012**: An optional synthetic task test MUST explain model-loading effects, use synthetic input only and identify the endpoint/model tested.
- **FR-013**: Missing/stopped services, incompatible protocol/tasks, unavailable models, authentication rejection, timeouts, busy/rate limits and malformed/oversized/redirected results MUST produce useful redacted corrections.
- **FR-014**: Current schemas, CLI help and official documentation MUST describe actual behavior, advanced migration boundaries and remaining audio-adapter/setup work accurately. Frozen 1.0.0 documentation remains unchanged.
- **FR-015**: Required security, redaction, revision, cancellation and backend integration checks MUST pass with normal repository/site checks; CI MUST retain its ten-minute job budget and no attended pre-release gate is added.
- **FR-016**: Windows child processes MUST stay hidden/noninteractive. PR review rounds MUST never exceed two; every finding requires a response and resolution before exact-head green-CI handoff.

### Key Entities

- **Backend plan**: Version, role-specific profile options, opaque credential references and expected revision.
- **Preflight report**: Checked time, individual readiness dimensions, corrections and capabilities not exercised.
- **Service candidate**: Endpoint, installation evidence, discovery sources/time, observed version/model capabilities and routing.
- **Assistance selection**: Exact adapter protocol, elected endpoint/model/route, authentication reference and revision.
- **Synthetic test result**: Selection identity, checked time, task outcome and redacted correction.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All six supported backends have equivalent CLI/desktop plans and independently testable readiness results.
- **SC-002**: Failed, cancelled and stale-revision configuration attempts preserve 100% of active selections and accepted data in acceptance fixtures.
- **SC-003**: Every remote plaintext case is explicitly identified; no credential value appears in saved plans or diagnostics across tested failure paths.
- **SC-004**: Discovery/configuration/native assistance succeeds for local fixtures without a key and without silently selecting any external destination or hosted model.
- **SC-005**: Every network operation respects its aggregate deadline, response cap and cancellation; malformed proposals never execute as arbitrary queries.
- **SC-006**: All mapped issue acceptance requirements, automated checks and review findings are satisfied before handing off the PR for merge.

## Assumptions

- S022 supplies configuration/discovery operations for #66; guided first-run orchestration, managed workers/models, MSI and installation screenshots remain separate issues.
- Existing adapter/credential/query machinery is reused; alternate backend configuration does not promise implicit migration of populated workspaces.
- Optional write-test objects are omitted: readiness explicitly reports write capabilities as untested until explicit provisioning or real operations.
- Deterministic service/protocol fixtures qualify contracts; no claim of real-model inference is made without separate evidence.

## Clarifications

### Session 2026-10-13

- Q: How should plaintext PostgreSQL be represented? A: Keep verified TLS as default; local is genuinely local transport; explicitly named insecure remote transport is allowed without pretending to be local. Existing mislabeled remote-local configurations receive a correction rather than silently changing routes.
- Q: Does configuration relocate existing data? A: Fresh workspace provisioning and preflight are supported; populated backend changes retain the established explicit migration/backup workflow.
- Q: Does discovery elect a model or infer audio readiness? A: No; endpoint/model/route election is explicit, and S022 implements query assistance only.
- Q: How are write permissions established? A: Non-mutating probes report them untested; explicit provisioning or normal elected operations establish actual results.

### Clarification refinements after research

Lost provisioning responses reconcile against the identical generated plan identity and content; differing retry input conflicts. Preflight/network budgets are at most 30 seconds, with at most 16 candidates, 128 models, 1 MiB per metadata response and a bounded detail-probe count. Readiness claims describe their observed evidence. Local transport is a literal loopback, pinned localhost or Unix socket; named remote plaintext is insecure. Network errors use fixed redacted correction reasons.
