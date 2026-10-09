# Feature Specification: S012 Model references and acquire-on-demand

**Feature Branch**: `codex/012-model-references-acquisition`

**Created**: 2026-10-09

**Status**: Implemented; owner final review and merge pending

**Input**: Owner-authorized S012 completes #34 after merged S011. Specify, implement, verify, push and publish an official PR; satisfy CI and every review finding, request at most one second round, then stop for owner final review and squash merge.

## Clarifications

### Session 2026-10-09

- Q: Which aliases and exact-reference syntax avoid ambiguity? A: Lowercase ASCII aliases of 1-64 characters, beginning with a letter, then letters/digits/dot/underscore/hyphen; UUID inputs remain base IDs, `base:UUID` and `speaker:UUID` are explicit kinds. Hosted targets carry explicit provider configuration. Tombstone revisions prevent remove/recreate conflicts.
- Q: Which discovery boundary is supported? A: Explicit revisioned workspace sources expose bounded pinned-manifest catalogs, including mutable selector names resolved once to complete immutable bundle declarations. Custom manifests remain supported. Direct Hugging Face transport is not advertised without its separate adapter.
- Q: How do unavailable models affect work? A: Freeze before enqueue, use shared acquisition work IDs and scheduler dependencies. Waiting consumers use no worker slot; dependent cancellation does not cancel other consumers' acquisition.
- Q: Which compatibility declarations preserve existing installations? A: Optional explicit adapter/task/runtime/audio metadata; legacy exact role layouts infer only established supported adapters without changing accepted digests.
- Q: Does reference support claim matching or training delivery? A: Preserve and inspect exact trained lineage and hosted handles; full matching/enrollment/training remains #35/#15.

Coverage: Functional scope, domain identity, UX, reliability, access security, dependencies, edge cases, constraints, terminology and completion signals are clear. Five routine decisions are answered under autopilot; no owner clarification is needed.

## User Scenarios & Testing

### User Story 1 - Choose an exact model through a short reference (Priority: P1)

An operator creates a short workspace alias, inspects its exact version and compatibility, and selects it. Multiple versions coexist. Retargeting changes future elections only.

**Why this priority**: Predictable selection enables reproducible automatic work.

**Independent Test**: Register two tiny declared bundles, alias either one, perform concurrent updates and resolve exact references without acquiring files or loading models.

**Acceptance Scenarios**:

1. **Given** two versions, **When** an alias is created, **Then** it resolves to one immutable version, manifest and operation.
2. **Given** an elected job, **When** its alias is retargeted, **Then** original execution and retry retain the originally selected identity.
3. **Given** a stale alias revision or ambiguous name, **When** it is changed or resolved, **Then** a conflict or ambiguity is reported without arbitrary selection.

### User Story 2 - Acquire missing models before processing (Priority: P1)

An operator submits processing using a declared compatible model. Missing files acquire automatically; verified completion enables the elected work. Acquisition and inference are separate visible phases.

**Why this priority**: Missing files should not require manual preflight downloads.

**Independent Test**: Submit concurrent jobs against a synthetic source, interrupt acquisition, retarget aliases, restart/retry and supply deterministic processing results.

**Acceptance Scenarios**:

1. **Given** a compatible registered bundle without bytes, **When** processing is submitted, **Then** exact identity is frozen and durable acquisition precedes inference.
2. **Given** concurrent consumers, **When** they run, **Then** acquisition is shared and incomplete bytes never become available installations.
3. **Given** failed/cancelled acquisition, **When** retry is elected, **Then** the original bundle is recovered without model or route substitution.
4. **Given** an available installation, **When** it is selected, **Then** verified artifacts are reused and materialized with leases.

### User Story 3 - Discover declared models and understand compatibility (Priority: P2)

An operator discovers declared catalog entries or supplies a custom manifest. Inspection distinguishes operation, runtime/format, complete bundle identity and availability. Unsupported models receive an actionable diagnosis.

**Why this priority**: Downloaded weights do not automatically become executable adapter input.

**Independent Test**: Discover bounded fixture catalogs, freeze mutable selector resolution and reject wrong task/runtime/roles before downloading or inference.

**Acceptance Scenarios**:

1. **Given** unknown shorthand without a source, **When** resolved, **Then** a declared source is required instead of a guessed URL.
2. **Given** a mutable source selector, **When** resolved, **Then** immutable upstream revision where supplied and complete file hashes/sizes/roles are pinned.
3. **Given** incompatible architecture/task/layout, **When** elected, **Then** incompatibility is diagnosed before inference without a curated-model restriction.
4. **Given** a trained speaker version or hosted handle, **When** referenced, **Then** kind, lineage and actual retrieval capabilities remain explicit without fabricated weights.

### User Story 4 - Operate and move the workspace consistently (Priority: P2)

CLI and desktop expose the same selection and acquisition behavior. Portable state retains aliases and exact relationships across supported backends.

**Why this priority**: Shared runtime behavior and portability are required product contracts.

**Independent Test**: Exercise CLI/runtime/desktop journeys and populated snapshot migration/restore with model references and relationships.

**Acceptance Scenarios**:

1. **Given** registered unavailable models, **When** viewed, **Then** declarations, availability and compatibility are distinct.
2. **Given** exported reference state, **When** restored, **Then** alias revisions and model relationships survive without secrets or cache paths.
3. **Given** import, direct processing or saved pipelines, **When** selecting a reference, **Then** shared resolution and diagnostics agree.

### Edge Cases

- Remove/recreate must fence old alias revisions; case, UUID collision, empty and overlong names need explicit rules.
- Source/tag/pipeline/alias updates cannot alter elected retries.
- Missing roles/hashes, checksum mismatch, interrupted downloads and unavailable credentials remain visible.
- Waiting parents cannot starve acquisition workers. Parent cancellation cannot cancel another consumer's acquisition.
- Stale publication is fenced; corrupt available artifacts are verified/recovered rather than trusted by filename.
- Hosted-only targets and unavailable trained outputs never claim local executable weights.
- Legacy accepted manifests and populated prior snapshots retain explicit compatible evolution.

## Requirements

### Functional Requirements

- **FR-001**: Support exact immutable IDs/version references and bounded workspace aliases with kind/operation, case and collision rules.
- **FR-002**: Provide revisioned create/list/show/retarget/remove aliases with concurrency and remove/recreate fencing; historical elections remain immutable.
- **FR-003**: Resolve installed records, configured sources/catalogs and custom manifests without guessed URLs, curated-only models or first-match ambiguity.
- **FR-004**: Freeze exact model kind/ID/version, manifest digest, upstream revision where available, all file roles/hashes/sizes, adapter contract/version and settings before acquisition.
- **FR-005**: Distinguish registered/acquiring/failed/available/incompatible outcomes and validate task, architecture/runtime, roles and relevant audio requirements before inference.
- **FR-006**: Ensure missing elected bundles through durable verified acquisition before dependent processing and deduplicate concurrent acquisition.
- **FR-007**: Preserve frozen identity through restart/retry, independently cancel dependents, avoid worker starvation and fence stale publication.
- **FR-008**: Reuse verified artifacts, recover missing/corrupt bytes, retain leases and preserve durable trained/profile data from temporary cache lifecycle.
- **FR-009**: Preserve configured source/credential routing and transport rules; keep secrets out of payload/status and prohibit silent model/provider substitution.
- **FR-010**: Share reference grammar/identity/availability across import, processing, pipelines, CLI, runtime and desktop; retain existing model operations.
- **FR-011**: Preserve base versus trained-version lineage and hosted-only reference identity. Matching/training consumers gain this foundation without a claim their engines are delivered.
- **FR-012**: Preserve filesystem/S3, SQLite/PostgreSQL and affected graph parity, populated catalog migrations and prior snapshot compatibility.
- **FR-013**: Update authoritative docs, executable examples/help, offline/desktop help, schemas, glossary and changelog together; keep release versions aligned and planning internal.
- **FR-014**: Verify checksums/roles/resume/cancel/concurrency/tenancy/recovery using synthetic bundles and deterministic results. Required checks never download real weights, load models or run acoustic engines/training.

### Key Entities

- **Model reference**: Exact kind/identity or alias, operation and unambiguous syntax.
- **Alias**: Revisioned workspace name targeting one immutable record.
- **Declared source**: Explicit catalog definitions with upstream identity and opaque credential references.
- **Resolved election**: Exact bundle/adapter identity independent of future convenience-reference edits.
- **Acquisition dependency**: Shared durable verified bundle work with independently cancellable consumers.
- **Speaker model version**: Existing family/version/run/corpus/artifact or hosted-handle lineage.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every accepted reference resolves uniquely; all stale mutation cases reject conflicting edits.
- **SC-002**: Every restart/retry case retains originally selected identity after alias/source edits.
- **SC-003**: Concurrent consumers share one acquisition; incomplete/cancelled/checksum-invalid bundles never become available.
- **SC-004**: All unsupported selections fail before inference without unselected routing.
- **SC-005**: CLI/runtime/desktop journeys agree and portable round trips preserve new relationships.
- **SC-006**: Affected integrity/product checks and final-head CI pass without real models; all reviews are handled within the two-round cap.

## Assumptions

- Reuse model registration/acquisition, durable work, leases and catalog storage.
- Completes #34 and advances #29. Full acoustic matching (#35), enrollment/training delivery (#15) and release promotion (#14) remain separate.
- Keep UUID inputs compatible; established legacy role layouts may prove compatibility without rewriting accepted manifests.
- Explicit configured catalog protocol supports noncurated discovery. Direct provider integrations requiring other transport protocols are not falsely advertised.
- Providers return immutable revisions where possible; complete file hashes still pin bundles when upstream identity is unknown.
- Push/PR authorization is explicit. Merge/release remains outside this slice.
