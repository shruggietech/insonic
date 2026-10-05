# Feature Specification: Native platform and shared-runtime foundation

**Feature Branch**: `codex/s001-native-runtime-foundation`

**Created**: 2026-10-05

**Status**: Specified

**Input**: Run S001 under autopilot, delivering GitHub issues #1 and #2 together. Push and open the official PR, resolve every review, allow at most two external Codex review rounds, and stop for the owner merge.

## User Scenarios & Testing

### User Story 1 - Open one workspace from both clients (Priority: P1)

A user opens a workspace through the CLI or desktop shell. Both clients use the same operations and one workspace writer, with readable errors for incompatible or unavailable workspaces.

**Independent Test**: Initialize a temporary workspace, discover it from a child directory, connect two clients and prove they see one owner and matching operation results.

**Acceptance Scenarios**:

1. **Given** an initialized workspace, **When** a second client connects, **Then** it reaches the existing owner instead of starting a competing writer.
2. **Given** concurrent initialization or owner startup, **When** clients race, **Then** exactly one succeeds in ownership and existing workspace configuration remains intact.
3. **Given** malformed configuration or an unsupported protocol, **When** a client connects, **Then** it receives a typed error without credentials or source content.
4. **Given** no connected clients or running work, **When** the idle interval expires, **Then** the owner exits and releases ownership.

### User Story 2 - Run, cancel and retry bounded work (Priority: P1)

A caller starts a configured operation, follows attempt status, cancels it or retries it. Closing a client does not cancel active work. A stale attempt cannot publish or complete a newer attempt.

**Independent Test**: Start a bounded test operation, disconnect, reconnect, cancel, retry and reject a completion from the superseded generation.

**Acceptance Scenarios**:

1. **Given** a running attempt, **When** the client disconnects, **Then** the runtime retains the operation until completion or explicit cancellation.
2. **Given** a cancelled or failed attempt, **When** retry is requested, **Then** a new attempt generation is created and older results are rejected.
3. **Given** a noninteractive invocation, **When** an action cannot proceed, **Then** it returns a typed failure without a prompt or silent backend change.

### User Story 3 - Qualify native dependency behavior (Priority: P1)

Maintainers can reproduce the same pinned dependency and platform checks on Windows, macOS and Linux before downstream application work relies on them.

**Independent Test**: Run bounded platform qualification on each supported OS and retain an exact dependency identity, architecture, build and execution receipt.

**Acceptance Scenarios**:

1. **Given** the pinned dependencies, **When** qualification runs, **Then** the subtitle executable validates/renders a real supplied track and embedded catalog/graph probes execute real reads and writes.
2. **Given** configured bounded services, **When** alternative-backend probes run, **Then** catalog transactions, immutable object publication and native graph queries prove feasibility for each alternative.
3. **Given** either desktop or CLI entry, **When** it requests the qualification operation, **Then** the common runtime contract returns the same result.
4. **Given** installed offline help, **When** network access is unavailable, **Then** its entrypoint and local assets remain available.

### Edge Cases

- Competing writers, abrupt owner death, lingering connections, corrupted metadata and unsupported versions.
- Workspace paths containing spaces or Unicode, symlink aliases and discovery outside a workspace.
- Unauthorized local IPC callers, oversized/trailing requests, unsupported operations and client deadlines.
- Secret-bearing process output, URL/query credentials, failed child start, cancellation and bounded output exhaustion.
- Locked or absent native secret service, unsupported native architectures and unavailable dependencies.

## Requirements

### Functional Requirements

- **FR-001**: Provide platform-appropriate private configuration, cache, data and runtime paths, explicit workspace selection and ancestor discovery.
- **FR-002**: Initialize workspace identity/configuration atomically without overwriting existing files, reject unsupported versions and serialize owner access across processes.
- **FR-003**: Connect CLI and desktop callers to one versioned shared operation boundary using current-user-only local IPC. Installation must not expose an application network listener.
- **FR-004**: Start the runtime on demand without visible console windows or interactive prompts, retain active work across client disconnection, and exit after a bounded idle interval.
- **FR-005**: Define configuration, adapter health/capabilities, catalog, artifact and secret-reference boundaries for downstream slices, preserving all required backend choices without claiming their full implementation here.
- **FR-006**: Define and exercise attempts, cancellation, retry and generation checks. Report runtime restart loss explicitly; durable recovery belongs to issue #3.
- **FR-007**: Return versioned machine output, readable human output, stderr diagnostics and distinguish invalid input, incompatible versions, unavailable dependencies and operation failures.
- **FR-008**: Execute configured child programs without a shell, using literal arguments, closed noninteractive input, bounded output, explicit cancellation and hidden Windows creation. Diagnostics must not return child output or secret values.
- **FR-009**: Pin exact dependency versions/revisions and native artifact digests and record supported architectures. Prove real dependency execution on all three OSes, including desktop bridge and offline assets.
- **FR-010**: Qualify native credential-store availability without revealing or persisting credential values, reporting absent/locked services as capabilities. Credential persistence and encrypted fallback implementation belong to issue #5.
- **FR-011**: Qualify alternative catalog, object-storage and graph operations on bounded configured fixtures. Failures must not silently select another backend.
- **FR-012**: Retain acceptance evidence separately for issues #1 and #2. Do not close either issue on scaffolding, cross-compilation alone or missing platform evidence.

### Key Entities

- **Workspace**: Stable identity, contract version, selected backend profiles and private state location.
- **Owner**: One OS-enforced workspace lock and authenticated local endpoint with bounded lifetime.
- **Operation**: Versioned request ID, operation name, typed arguments and result/error.
- **Attempt**: Job ID, increasing generation, status and cancellation state scoped to one runtime session.
- **Adapter contract**: Versioned identity, capabilities, health, typed operations and opaque credential references.
- **Qualification receipt**: OS/architecture, exact dependency identities, verified hashes and real operation outcomes.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All workspace concurrency, discovery, version, attempt ownership and unauthorized-client acceptance cases pass on Windows, macOS and Linux.
- **SC-002**: Both clients return equivalent results through the shared operation contract; no core operation exists only in the desktop shell.
- **SC-003**: Qualification receipts include real executable/catalog/graph operations and offline-help/desktop checks on each of the three OSes.
- **SC-004**: No test credential appears in diagnostics, serialized status, child launch arguments or source-controlled receipts.
- **SC-005**: Repository, documentation and affected product checks pass; affected native CI jobs each finish within ten minutes.

## Assumptions

- Preserve the approved CLI-first architecture and backend roles. No media processing, full catalog persistence, full artifact storage or encrypted credential registry is claimed by this foundation.
- Qualification exercises bounded maintainer fixtures rather than real user media, credentials or paid model calls.
- The owner authorized push, PR creation and review responses; owner squash merge remains the final handoff.

## Clarifications

### Session 2026-10-05

- Q: How are the original issue codes retained? A: S001 delivers #1 (roadmap S001) and #2 (roadmap S002); original issue history and downstream dependencies remain intact.
- Q: Are attempts durable here? A: This slice proves session-scoped ownership and interfaces; portable durable persistence/recovery is explicitly #3.
- Q: What completes native qualification? A: Executed native checks on all three OSes, never cross-compilation alone. Hosted checks run after the authorized push.
- Q: Does secret-store qualification implement credential persistence? A: No; exercise availability/status without real credentials, reserving complete persistence and fallback for #5.
