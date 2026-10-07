# Feature Specification: S004 Secure media library

**Feature Branch**: `codex/004-secure-media-library`
**Created**: 2026-10-07
**Status**: Implemented; hosted acceptance and external review pending
**Input**: Complete issues #5 and #6 under autopilot, push, publish an official PR, resolve every review with at most two rounds, and stop before owner merge.

## User Scenarios & Testing

### User Story 1 - Persist selected credentials (Priority: P1)

An operator configures credentials once and uses selected adapters after restart without secrets in configuration or diagnostics.

**Why this priority**: Authenticated imports/downloads need usable secret references.
**Independent Test**: Configure, restart, resolve, explicitly replace and delete a credential.

**Acceptance Scenarios**:

1. Native storage persists credentials on Windows, macOS and Linux and reveals only configured/missing/rejected state.
2. Unavailable/locked native services return a fixed status without prompts or plaintext fallback. Explicit encrypted-vault/session alternatives work.
3. Another workspace cannot resolve a workspace-scoped credential; status never retrieves saved values.
4. Catalog-independent setup can provision credentials before an authenticated catalog opens.

### User Story 2 - Register verified downloaded models (Priority: P1)

An operator registers an exact manifest and acquires verified model artifacts through configured storage.

**Why this priority**: Model identities and artifacts are prerequisites for later processing.
**Independent Test**: Register a small multi-file fixture, interrupt/resume acquisition, verify and inspect after restart.

**Acceptance Scenarios**:

1. An exact model version becomes available only after every required artifact matches declared size/digest and has a publication receipt.
2. Wrong digests, incomplete transfers, changed manifests and stale attempts never admit the version.
3. Status exposes provenance/capabilities/references without credentials. Downloaded models require neither speaker nor training lineage.

### User Story 3 - Admit and inspect original media (Priority: P1)

An operator imports audio/video copies or local references individually or through CSV/JSON manifests and immediately inspects original identity, metadata and measured timing.

**Why this priority**: It establishes the library consumed by the subtitle workflow.
**Independent Test**: Import, restart, inspect, retry and relocate fixtures across each supported catalog/storage combination.

**Acceptance Scenarios**:

1. Copies survive input relocation. References detect changed/missing bytes and never delete external files.
2. Raw metadata/stream reports bind stable original digest, extractor identity/version/options and explicit capture state before transforms.
3. Captured, no-metadata, partial, unsupported and failed states remain distinct and do not reject otherwise usable media.
4. One request identity reconciles one admission; explicitly created entries may share immutable bytes. Relocation verifies digest.
5. Supplied subtitle bytes are immutable attachments; duration comes from probing, with unknown distinct from known zero.

### User Story 4 - Preserve dates and replace computed metadata (Priority: P1)

An operator supplies or corrects origination dates and refreshes metadata while preserving original inputs.

**Why this priority**: Date uncertainty and current-result integrity must exist before derived processing.
**Independent Test**: Import date-only/offset/IANA/fold/gap cases, correct one item and refresh computed metadata.

**Acceptance Scenarios**:

1. Per-item values override batch defaults; date-only stays day precision, approximate bounds/conflicts remain explicit, unknown stays unknown.
2. Local timezone/rules are captured once per batch. Unresolved mappings, offset conflicts and DST observations affect only dates, not media admission.
3. Refresh publishes one complete replacement and removes superseded managed computed reports/observations recoverably. Original media, supplied subtitles and owner date evidence remain.

### Edge Cases

- Source mutation during hash/copy/extraction; links; changed/missing references; duplicate manifest keys and relative paths.
- Authentication/TLS/quota failures, interrupted or unknown publication responses, concurrent retries and revoked authority.
- Duplicate/structured/binary metadata, output limits, partial extraction and malformed timing.
- Historical offsets, local-zone mapping failure, folds/gaps, inconsistent offset/zone and integer overflow.
- Recovery after restore and retirement failures without hidden alternative captures.

## Requirements

### Functional Requirements

- **FR-001**: Provide workspace-scoped credential IDs, noninteractive native storage on all three OSes, explicit encrypted-vault/session alternatives, replacement/deletion and status-only display.
- **FR-002**: Resolve credentials for selected storage/catalog/acquisition adapters without values in persisted config/jobs/manifests, errors, URLs, logs, argv or returned status. Never silently route elsewhere.
- **FR-003**: Provide exact downloaded-model manifests, capabilities/version/digest/size checks, organized storage and durable download/publication receipts independent of trained speaker models.
- **FR-004**: Provide shared CLI/runtime import/list/show/raw-metadata/date-correction/refresh/relocation operations consumed by the desktop bridge.
- **FR-005**: Support copy/reference admission, generic acquisition adapters, CSV/JSON manifests and filesystem/S3 plus SQLite/PostgreSQL parity.
- **FR-006**: Capture all available raw embedded metadata/container/stream families from stable originals before transforms with duplicate instances, structured values, payload references, tool provenance and truthful partial/failure states.
- **FR-007**: Preserve original and supplied subtitle identity, detect source changes and bind admission to digest/size/acquisition provenance.
- **FR-008**: Provide versioned owner-first/selected date policies, competing observations, approximate/date-only/unknown precision, local/UTC/IANA/offset zones, captured mapping/rules and explicit fold/gap handling without per-item gates.
- **FR-009**: Preserve optional measured duration, streams/channels and exact source-clock mapping; never infer duration from subtitle coverage.
- **FR-010**: Reconcile durable operations with cancellation/retry and ownership fencing so stale work cannot duplicate admission or overwrite current data.
- **FR-011**: Keep one current computed metadata result, atomically replace references and remove superseded managed bytes/observations while retaining original inputs and owner date evidence.
- **FR-012**: Synchronize authoritative schemas/docs/migrations and verify security, workspace isolation and supported-backend parity.

### Key Entities

- Credential reference: workspace-scoped opaque ID, selected backend, status; value outside catalog state.
- Downloaded model: exact manifest/version/capabilities plus verified artifacts, independent of speaker training.
- Admission: durable operation identity, source byte identity, copy/reference mode, provenance and current metadata.
- Recording date: supplied/captured observation, literal, precision, zone/rules, resolution, conflicts and selection.
- Timing: optional measured duration, stream/channel facts and exact original-clock mapping.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Credential lifecycle/restart fixtures pass on supported OSes; disclosure checks return zero secrets in nonsecret outputs.
- **SC-002**: Every available model artifact has its exact declared size/digest and reconciled receipt; all invalid fixtures remain unavailable.
- **SC-003**: Admission/date/recovery fixtures pass for both catalogs and artifact adapters; one retried request produces one admission.
- **SC-004**: Every refresh leaves one current computed result and no retained superseded report; original/subtitle digests remain unchanged.
- **SC-005**: Both issues have complete acceptance evidence, zero unresolved critical analysis/convergence findings, green affected checks and resolved external discussions before owner handoff.

## Assumptions

- Actual S004 completes roadmap issues #5 (S005) and #6 (S006). Issue #21 governs current metadata/timing but stays open for subtitle assembly with #7.
- Processing (#7/#8), trained speaker models (#15), full desktop controls/installers (#10), graph and release delivery remain later work. Shared bridge parity is included now.
- Existing configured routing persists; no mandatory per-item review is introduced.
- Push and PR are authorized; merge/release are not. Maximum review rounds: initial automatic round plus one requested round.

## Clarifications

### Session 2026-10-07

- Q: Which feature identity is authoritative? A: S004, issues #5/#6, specs/004-secure-media-library and codex/004-secure-media-library.
- Q: What does refresh retain? A: Original inputs and owner date evidence; superseded computed captures are physically retired under #21.
- Q: How is authenticated-catalog setup possible? A: Credential provisioning uses shared catalog-independent bootstrap operations.
