# Feature Specification: S003 artifact storage

**Feature Branch**: `codex/s003-artifact-storage`

**Created**: 2026-10-06

**Status**: Implemented; hosted CI and external reviews pending

**Input**: Implement issue #4 through filesystem and generic S3 storage, verified immutable publication, catalog-backed recovery and guarded retirement. The actual session code is S003; issue #4 retains its original roadmap label S004. S001 completed #1/#2 and S002 completed #3.

## User Scenarios & Testing

### User Story 1 - Publish verified durable bytes (Priority: P1)

A workspace user stores a file in the selected storage profile and receives an immutable artifact identity and location receipt. Both supported storage choices provide the same observable behavior.

**Why this priority**: Media import and model downloads need durable bytes before recording their associations.

**Independent Test**: Publish empty, binary and multipart-sized files, retrieve full and ranged bytes, and compare size and digest on each store.

**Acceptance Scenarios**:

1. **Given** a configured store, **When** publication succeeds, **Then** the final readable bytes match the receipt before a catalog location becomes available.
2. **Given** a lost response or interrupted attempt, **When** the user retries/reconciles, **Then** exact key/version and digest determine the outcome without overwriting previously admitted bytes.
3. **Given** an expired or cancelled publisher, **When** it finishes external I/O, **Then** it cannot admit an available catalog location under stale authority.

### User Story 2 - Read and materialize with leases (Priority: P2)

A configured operation reads a bounded range or obtains a verified seekable local materialization whose lifetime is recorded. Storage credentials are resolved by opaque references.

**Why this priority**: Tools require seekable inputs without making temporary cache paths permanent identities.

**Independent Test**: Materialize from both stores, retain a live lease through a cleanup attempt, then release it and observe eligibility after grace.

**Acceptance Scenarios**:

1. **Given** an available artifact, **When** materialized, **Then** the returned local file has verified bytes and a renewable/releasable lease.
2. **Given** missing credentials, corruption, quota failure or cancellation, **When** accessed, **Then** the affected operation reports a redacted error without switching providers or admitting partial bytes.

### User Story 3 - Retire only unreferenced objects (Priority: P2)

A workspace user explicitly retires an eligible location while preserving catalog provenance. Reference, lease and publication admission share the retirement authority.

**Why this priority**: A separate reference check followed by deletion races with new users of those bytes.

**Independent Test**: Race retirement against reference/materialization admission and retry an uncertain delete on its exact immutable object.

**Acceptance Scenarios**:

1. **Given** retention references or active leases, **When** retirement is requested, **Then** deletion is refused.
2. **Given** an eligible location past grace, **When** atomically claimed, **Then** new references/materializations are blocked until exact deletion is reconciled and historical availability is retained.

### Edge Cases

- Zero bytes, large streaming inputs, invalid ranges and split Unicode filenames.
- Traversal, Windows reserved/ambiguous names, symlinks and a managed root replaced during access.
- Duplicate publication, cancellation, lost catalog commit response, expired ownership, reconnect and delayed remote visibility.
- Multipart upload with unknown completion, pending upload abort and exact-version deletion retry.
- Changed storage profile revision, unversioned S3 providers and multipart ETags that are not content digests.
- Leased or retained originals, metadata reports, training snapshots and model artifacts are durable, never disposable cache.

## Requirements

### Functional Requirements

- **FR-001**: Provide equivalent filesystem and generic S3 storage behavior through shared capability, stat, range, immutable publication, materialization, verification and guarded deletion contracts.
- **FR-002**: Stream input, compute the application SHA-256 and size, stage privately, and verify readable final bytes before admitting available catalog references. Provider checksums and ETags remain separate observations.
- **FR-003**: Preserve operation/attempt identity and exact immutable key/version across interruption, unknown responses, reconciliation, resumable multipart work and explicit abort. Stale/cancelled authority cannot admit results.
- **FR-004**: Use conditional creation where qualified, otherwise unique nonreused publication keys. Never overwrite an admitted artifact with different bytes.
- **FR-005**: Record renewable materialization leases, verify seekable cache files and enforce configured capacity/bounds. Incomplete files are never returned as usable inputs.
- **FR-006**: Atomically claim retirement after checking retention and live leases plus grace; block new admission until deletion is reconciled. Delete only the claimed key/version/generation and retain retired/missing provenance.
- **FR-007**: Preserve workspace/profile scope, reject path escape and secret-bearing storage configuration, and resolve credentials only through the existing opaque secret-provider interface. Responses/logs/snapshots never expose credential values.
- **FR-008**: Expose shared CLI/runtime publication, inspection, verification, materialization/lease and retirement/recovery operations with machine-readable contracts. Document advanced CLI controls preceding desktop controls.
- **FR-009**: Preserve catalog export/restore consistency including lifecycle state, expiring imported authority and validating receipt/record relationships on both catalog backends.
- **FR-010**: Prove both storage adapters with shared positive/failure/security/concurrency tests and a pinned S3-compatible fixture. Run repository checks, affected site/product checks and three-OS CI within existing timeouts.

### Key Entities

- Artifact identity: immutable digest, byte length and observations.
- Location/publication receipt: selected profile revision, exact immutable key/version, verification method and attempt history.
- Retention reference: durable association that prevents retirement.
- Materialization lease: location identity, owner/generation, expiry and verified disposable local path.
- Retirement claim: exclusive generation, eligibility time, exact object and reconciled outcome.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every successful publication in the acceptance matrix yields matching full bytes, digest and size on both storage choices.
- **SC-002**: All injected interrupted/uncertain publication cases either reconcile to the verified object or remain explicitly pending without an available catalog reference.
- **SC-003**: Every stale-authority, cross-workspace, retained/live-lease and path-escape negative scenario refuses the prohibited admission/deletion.
- **SC-004**: Empty and multipart-sized files complete the same publish/read/materialize/retire journey without whole-file memory buffering.
- **SC-005**: All required automated gates pass before the owner merge handoff, with at most two external review rounds.

## Assumptions

- The selected provider and routing are owner-configured. No per-item manual review is introduced.
- Encrypted/native credential persistence, model registry, library/media processing and portable byte backups remain their existing subsequent issues. This slice provides their required artifact interfaces.
- Fixture qualifications describe demonstrated capabilities, not universal guarantees of every S3-labelled provider.
- Existing dependency pins remain unless an affected change is qualified and justified.
