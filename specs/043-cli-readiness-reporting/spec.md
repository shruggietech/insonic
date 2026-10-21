# Feature Specification: CLI readiness and failure reporting

**Feature Branch**: `codex/043-cli-readiness-reporting`
**Created**: 2026-10-21
**Status**: Specified
**Input**: Execute S043 for truthful readiness, local asset budgets and actionable failure reporting, through a reviewed green PR; stop before merge.

## User Scenarios & Testing

### User Story 1 - Identify the failed prerequisite or bytes (Priority: P1)

An operator can distinguish corrupt artifacts and trainer outputs from stale revisions, and knows which logical local asset is missing, unreadable or invalid.

**Independent Test**: Verify valid, same-size corrupt, truncated and missing artifacts; submit deterministic trainer outputs; elect setup using tiny missing/corrupt local assets. Check returned and durable causes without publishing rejected bytes.

**Acceptance Scenarios**:

1. **Given** corrupt or absent managed bytes, **When** verify/materialize runs, **Then** an integrity-specific cause and correction identify the fault without revision-conflict advice.
2. **Given** invalid trainer output, **When** publication validates it, **Then** publication remains rejected with output-integrity or contract detail; genuine stale definitions still report revision conflict.
3. **Given** incomplete local assets, **When** setup or local-model acquisition runs, **Then** the bounded logical filename, role, verification reason and correction agree in human, machine-readable and durable results.

### User Story 2 - Read readiness and budgets as observed facts (Priority: P1)

An operator sees the distinction between supplied configuration, locally verified prerequisites, unavailable prerequisites and unverified remote access. Offline plans separate required bytes from elected transfer bytes.

**Independent Test**: Preview offline assets, absent/resolvable credentials and pinned deterministic trainers without network requests or adapter execution.

**Acceptance Scenarios**:

1. **Given** offline setup, **When** the plan is shown, **Then** elected network bytes are zero and required, verified and missing local totals are separately identified.
2. **Given** an omitted, nonexistent or locally resolvable protected credential, **When** a gated model is previewed, **Then** local reference state is accurate and upstream authorization remains explicitly unverified.
3. **Given** a pinned training-only pipeline, **When** readiness is requested, **Then** executable/support-file checks distinguish unknown from verified unavailable; a usable training-only pipeline does not require recognition or diarization to be elected.

### User Story 3 - End a wait without inventing a startup failure (Priority: P1)

A caller can stop waiting within its elected budget without a false diagnosis of runtime prerequisites or an unsupported assertion about the durable work state.

**Independent Test**: Exercise completed, running and unobserved work against a warmed deterministic runtime and controlled deadline/cancellation boundaries.

**Acceptance Scenarios**:

1. **Given** a caller budget expires during connection/readiness, **When** wait returns, **Then** it reports the wait timeout without claiming independently demonstrated runtime startup failure.
2. **Given** work state was not observed, **When** wait times out, **Then** it makes no assertion that work is still running or completed.
3. **Given** independently observed startup failure or cancellation, **When** wait returns, **Then** its actual cause and existing cancellation behavior remain available.

### Edge Cases

- Equal-size corruption, missing/truncated objects, malformed/unknown trainer fields, valid outputs and genuinely stale definitions.
- Unrelated files, inaccessible local directories, absent tools, missing declared assets, size/hash failures and interrupted inspection.
- Missing, locked, wrong-workspace and resolvable credential references, without exposing secret material or probing an upstream service.
- Changed executable/support bytes, configured-but-unprobed adapters and pipelines electing training alone.
- Completed or running work under small/ordinary budgets, caller cancellation and an independently bounded startup failure.

## Requirements

### Functional Requirements

- **FR-001**: Artifact verification/materialization and trainer output validation MUST distinguish integrity/contract failures from actual revision conflicts and MUST preserve rejection before publication.
- **FR-002**: Local worker/model failures MUST carry a bounded generated reason, logical asset filename/role and corrective action through human, machine-readable and durable work results, distinguishing missing, unreadable, wrong-size, wrong-hash and missing-tool conditions.
- **FR-003**: Offline plans MUST report zero elected network download bytes and separately identify required, verified and missing local asset totals; nominal profile totals MUST be labeled separately.
- **FR-004**: Model previews MUST distinguish supplied, locally resolvable and unverified upstream credential access; missing or wrong-workspace references MUST NOT report access ready. Preview MUST NOT perform network authorization checks or expose secrets.
- **FR-005**: Training readiness MUST distinguish unobserved from proven unavailable, verify bounded pinned executable/support-file prerequisites where possible without running the adapter, and assess training-only pipelines using their elected operations.
- **FR-006**: Work wait MUST distinguish caller wait-budget exhaustion from independently observed runtime-startup failure; unobserved work state MUST remain unknown and cancellation MUST preserve existing behavior.
- **FR-007**: Shared runtime results, human/JSON output, schemas and documentation MUST agree on changed states and failure categories while preserving valid existing behavior and version consistency.
- **FR-008**: Existing packaged CLI qualification MUST execute deterministic regressions for all six defects on disposable profiles/libraries, with owned cleanup and hidden noninteractive Windows children. Targeted checks MUST preserve the current qualification target/hard limit and all existing required integrity checks.
- **FR-009**: New persisted/exposed diagnostic detail MUST use bounded generated fields and MUST NOT leak credentials, arbitrary tool output, environment values or owner-specific filesystem paths.

### Key Entities

- **Failure observation**: Category, generated reason, logical asset role/name, verification stage and correction, with the original internal error identity retained.
- **Readiness observation**: Selected operation, configuration presence, local verification outcome and separately unverified external authorization.
- **Asset budget**: Required, verified, missing and elected network bytes for the selected mode.
- **Wait outcome**: Caller budget/cancellation, independently observed startup state and last observed durable work state.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every valid/corrupt/missing/truncated artifact and trainer-output case returns the expected distinct category, preserves genuine revision conflicts and publishes no rejected output.
- **SC-002**: Every selected local-asset failure exposes its declared logical filename/role and correction consistently in human, machine-readable and durable results, with no protected input leakage.
- **SC-003**: Offline transfer bytes are zero; credential and training readiness matrices distinguish all observed/unobserved states without network requests or adapter execution.
- **SC-004**: Controlled wait cases preserve actual completed/running/unknown state, neutral caller timeouts and independently observed startup errors without fragile exact-millisecond assertions.
- **SC-005**: Required local checks and exact-head hosted qualification pass; all received review findings are answered/resolved within at most two review rounds, with the PR left unmerged for the owner.

## Assumptions

- Close defects #126 and #135-139 through this PR after acceptance. #114 supplies only the relevant fixture subset; #147 remains the audit umbrella.
- The original Windows incident #154, release publication, dependency upgrades and the rest of the full audit corpus are outside S043.
- Existing access/routing authorization and output-integrity checks persist. Readiness reporting adds no new eligibility, approval or per-item review gates.
- Current version remains 1.0.1 until separately authorized release preparation. Additive diagnostic fields are preferred; changed category meanings are documented.
- No critical ambiguity requires owner input: use existing shared contracts and the elected operation/mode as authority. Temporary test outputs are removed; no agent journals or evidence stashes are retained in build or Downloads.
