# Feature Specification: Portable catalogs and durable jobs

**Feature Branch**: `codex/s002-portable-catalogs`

**Created**: 2026-10-05

**Status**: Accepted for autopilot implementation

**Input**: Owner-approved S002, implementing GitHub issue #3 and change-aware qualification.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Preserve authoritative workspace state (Priority: P1)

An operator keeps settings, metadata and associations across runtime restarts and catalog transfers, with the same behavior on the default local catalog and the supported server catalog.

**Why this priority**: Durable authority is required before library and pipeline work.

**Independent Test**: Admit records, revise a selection, reopen and transfer the catalog, then compare exact values and relationships.

**Acceptance Scenarios**:

1. **Given** either supported catalog, **When** workspace records and immutable evidence are admitted, **Then** reopening retains identity, exact timestamps, nulls, revision history and scoped associations.
2. **Given** two writers using the same expected revision, **When** both submit changes, **Then** exactly one change succeeds and retrying its operation ID returns the original receipt.
3. **Given** a consistent export, **When** restored into an empty catalog on either backend, **Then** all records, attempts, receipts and outbox ordering match, with active claims fenced.

### User Story 2 - Recover durable work (Priority: P1)

An operator starts, inspects, cancels and retries jobs through the existing shared runtime. Restarting the owner retains completed history and recovers interrupted work without accepting obsolete results.

**Why this priority**: Session-only attempts lose the authority needed for longer operations.

**Independent Test**: Start work, interrupt its owner, restart, inspect history and attempt to commit from the obsolete claim.

**Acceptance Scenarios**:

1. **Given** a running job, **When** a client disconnects, **Then** owner work continues and its accepted result is durable.
2. **Given** an interrupted or expired attempt, **When** recovery claims the job, **Then** a new attempt/generation and recovery receipt preserve the old history; the old owner cannot renew or publish.
3. **Given** explicit cancellation, **When** restarted, **Then** the job stays cancelled until an explicit retry creates a distinct attempt.
4. **Given** a lost acceptance response, **When** the same operation is retried, **Then** no duplicate job or accepted mutation is created.

### User Story 3 - Replay ordered graph changes (Priority: P2)

A projection worker obtains accepted catalog changes in order and records acknowledgements using its current ownership generation.

**Why this priority**: Later graph adapters need an ordered recoverable handoff from catalog authority.

**Independent Test**: Commit several changes, expire a claim, reclaim, reject the obsolete acknowledgement and advance exactly one event at a time.

**Acceptance Scenarios**:

1. **Given** accepted mutations, **When** their target events are read, **Then** sequences are consecutive and carry exact revisions, predecessor, operation identity and payload digest.
2. **Given** an unacknowledged event, **When** another worker claims, **Then** later events cannot bypass it and obsolete generations cannot acknowledge it.

### User Story 4 - Keep qualification relevant (Priority: P2)

A contributor gets integrity checks for affected components without repeating expensive native qualification for prose-only changes.

**Why this priority**: Full qualification should remain meaningful and finish within the project budget.

**Independent Test**: Classify prose, Go/catalog, desktop, lock and workflow changes; confirm conservative fallback and stable check names.

**Acceptance Scenarios**:

1. **Given** documentation-only changes, **When** CI runs, **Then** foundation/docs checks run and unnecessary native artifact operations are skipped explicitly.
2. **Given** runtime, dependencies or qualification changes, **When** CI runs, **Then** applicable core, backend and native qualification runs; declared dependency/version drift fails.

### Edge Cases

- Unknown migration versions, malformed snapshots, invalid UUIDs, out-of-range signed revisions and mismatched timestamp pairs fail without partial writes.
- Workspace-scoped relationships cannot reference another workspace, even when IDs coincide.
- An unavailable selected server catalog fails explicitly without switching to the local catalog.
- Credentials remain opaque references; persisted jobs, receipts, exports and errors contain no resolved credential values.
- Cancellation, lease expiry and acceptance races have a single transactional winner.
- Interrupted shutdown differs from user cancellation; expired claims are never treated as current.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Both SQLite and PostgreSQL MUST provide the same typed catalog operations, portable versioned migrations and workspace-scoped relationship constraints.
- **FR-002**: Settings and selected state MUST use expected revisions; evidence MUST remain append-only with exact integer timestamps and explicit unknown values.
- **FR-003**: Each accepted mutation MUST atomically record a revision and idempotent operation receipt, rejecting reuse with changed intent.
- **FR-004**: Durable jobs MUST retain effective options, immutable attempt identities, cancellation/retry history and recovery receipts across restarts.
- **FR-005**: Ownership MUST use renewable expiring leases and increasing generations; expired, superseded or cancelled attempts MUST NOT publish.
- **FR-006**: Recovery MUST resume elected interrupted qualification work in a fresh attempt, preserving predecessor history and avoiding mandatory human reviews.
- **FR-007**: Accepted graph events MUST allocate consecutive per-target sequences atomically, expose only the next event and fence acknowledgements separately from immutable payloads.
- **FR-008**: The catalog MUST preserve typed metadata, date provenance and schema associations for speaker segments, frozen corpus members, training runs and model artifacts without implementing their pipelines.
- **FR-009**: Consistent export/restore MUST round-trip all authoritative state across both catalogs, reject nonempty destinations and invalidate imported active authority.
- **FR-010**: The selected PostgreSQL profile MUST resolve credentials through the existing opaque-provider boundary, verify server identity by default and avoid credential-bearing error messages or silent fallback.
- **FR-011**: CLI and shared-runtime behavior MUST report durable job history and catalog status; catalog transfer MUST expose a bounded CLI operation and preserve exact integers.
- **FR-012**: Change-aware CI MUST retain stable gates, conservative unknown-change handling, relevant cross-platform/backend acceptance and dependency qualification pin consistency.
- **FR-013**: Public documentation MUST describe demonstrated behavior and remaining scope without exposing slice tracking; versions MUST remain aligned.

### Key Entities

- Workspace/catalog revision, typed settings and backend profile revisions.
- Immutable metadata/date evidence and scoped provenance associations.
- Speaker, segment revision, frozen dataset member, training run and model/artifact association.
- Job, attempt, ownership lease and operation/recovery receipt.
- Projection target, immutable outbox event, claim and checkpoint.
- Versioned consistent catalog snapshot.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: One shared acceptance suite demonstrates all catalog invariants on both supported catalogs, including cross-backend round trips.
- **SC-002**: All accepted state survives reopening; exact nanosecond values above floating-point precision round-trip unchanged.
- **SC-003**: Concurrent revision, cancellation and stale-worker tests admit zero duplicate or obsolete publications.
- **SC-004**: Recovery produces a fresh attempt within ten seconds after its old ownership expires and preserves complete attempt history.
- **SC-005**: All configured PR checks pass within ten minutes each; two review rounds at most are requested and all reported findings are addressed before owner merge handoff.

## Assumptions

- S001 is the merged runtime baseline. Issue #3 retains its original roadmap title while this execution slice is S002.
- Existing qualified dependency pins remain selected; new dependency PRs are assessed separately. Qualification gains a guard against manifest drift.
- Local catalog migrations run at owner startup. A server catalog can be provisioned separately with migration permissions and then opened with runtime-only permissions.
- Artifact operations, encrypted credential persistence, media processing, graph execution and release packaging remain separate slices; their schema/provenance seams are retained here.
- Restoring a snapshot requires an empty selected catalog, avoiding unauthorized destructive replacement.
- Requirements are reviewed under autopilot; no unresolved product questions remain.
