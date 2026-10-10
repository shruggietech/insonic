# Feature Specification: Portable backups and release readiness

**Feature Branch**: `codex/014-portable-backups-release-readiness`

**Created**: 2026-10-09

**Status**: Specified

**Input**: Owner-authorized S014 under autopilot, issues #14 and remaining #29. Push and official PR are authorized; merge, tags, release and deployment remain owner actions.

## User Scenarios & Testing

### User Story 1 - Recover a complete workspace (Priority: P1)

An operator backs up accepted authority and restores to an empty workspace on any supported backend, independently of the original artifact store.

**Why this priority**: Catalog-only export cannot recover audio or trained model bytes.

**Independent Test**: Disconnect the source store after backup; retrieve all restored authoritative bytes and relations with identical hashes.

**Acceptance Scenarios**:

1. **Given** canonical multitrack audio, current Cue JSON, mappings, rosters and durable models, **when** backed up and restored across supported backend choices, **then** bytes, timing, raw metadata, uncertainty and exact lineage survive.
2. **Given** corruption, missing bytes, unsupported schema or a nonempty target, **when** restore runs, **then** no target authority is replaced.
3. **Given** superseded runs and engine scratch, **when** managed backup runs, **then** current authority and durable provenance remain while scratch, local original paths and redundant subtitles are excluded.
4. **Given** graph materialization, **when** restored, **then** stale checkpoints are reset and accepted evidence can rebuild the selected graph backend.

### User Story 2 - Manage backup dependencies (Priority: P1)

An operator explicitly chooses reference-only backup, sees its original-store dependency and releases retained references when deleting it.

**Why this priority**: Lightweight backups must not silently lose referenced bytes through cleanup.

**Independent Test**: Pin a reference-only backup, attempt retirement, release the backup and repeat retirement.

**Acceptance Scenarios**:

1. **Given** reference-only mode, **when** complete, **then** exact immutable objects and source dependencies are recorded and lifetime references are pinned.
2. **Given** interruption, **when** retried, **then** no incomplete bundle is advertised and no competing authority is activated.
3. **Given** configured remote access, **when** exported, **then** secrets are excluded and target access uses independently configured credentials.

### User Story 3 - Assemble publishable packages (Priority: P1)

A maintainer builds native CLI and GUI-with-CLI packages, complete corresponding sources/notices, checksums and exact-version documentation from one revision.

**Why this priority**: Existing development archives lack complete bundled-library source material.

**Independent Test**: Relocate packages on Windows/Linux/macOS, exercise real media operations and reject changed assets or mismatched versions/revisions.

**Acceptance Scenarios**:

1. **Given** each platform, **when** assembled, **then** dependency sources and build inputs are complete without reducing supported decoding contracts.
2. **Given** configured signing, **when** packaging runs, **then** signing precedes final hashes; absent configuration is represented without a signed claim.
3. **Given** version, revision or byte mismatch, **when** release validation runs, **then** publication fails.
4. **Given** published release success, **when** documentation promotion runs, **then** only matching immutable documentation is promoted; failed publication cannot promote it.

### User Story 4 - Complete coherent tracking (Priority: P2)

Maintainers reconcile canonical taxonomy and final contracts while preserving historical issue ownership and state.

**Why this priority**: Bootstrap must not restore obsolete requirements or reopen completed issues.

**Independent Test**: Reconcile edited/closed historical fixtures and new outcome fixtures.

**Acceptance Scenarios**:

1. **Given** historical edited and closed issues, **when** bootstrap runs, **then** bodies/states/project values remain intact and new outcomes use canonical taxonomy.
2. **Given** final import/model/storage implementation, **when** contracts are audited, **then** documentation/schema/help agree without new per-item reviews, bulk conversion or acoustic CI gates.

### Edge Cases

Concurrent cleanup and mutations; duplicate bytes; empty catalog; future schemas; wrong credentials; interruption; symlink/archive traversal; missing reference-store objects; trained lineage with superseded outputs; signing failure; post-signing byte changes; mismatched tag/commit/version; partial release uploads.

## Requirements

### Functional Requirements

- **FR-001**: Provide complete self-contained backup/restore with consistent catalog revision and verified immutable artifact manifest.
- **FR-002**: Preserve canonical tracks/channels/timing, early raw metadata, current embedded Cue JSON, external mappings, rosters, exact model versions and durable trained lineage/artifacts.
- **FR-003**: Exclude transient inference inputs, original local locators, redundant subtitles and superseded derived results without invalidating retained provenance.
- **FR-004**: Support filesystem/S3, SQLite/PostgreSQL and graph rebuilding on LadybugDB/ArcadeDB with equivalent integrity and authority checks.
- **FR-005**: Validate schema, bytes, hashes and relations before atomic empty-target activation; reject overwrite and silent downgrade.
- **FR-006**: Provide explicit reference-only backups with exact object identities, recorded source dependency and lifetime retention pins.
- **FR-007**: Make interruption retryable without advertising incomplete bundles or activating competing catalogs; fence conflicting writes during transfer.
- **FR-008**: Exclude credentials/secrets from bundles/status/errors and use separately configured destination access.
- **FR-009**: Expose shared human/machine CLI backup contracts with aligned GUI controls or documented advanced CLI availability.
- **FR-010**: Assemble Windows/Linux/macOS CLI and GUI-with-CLI packages with complete corresponding sources, build inputs and notices for bundled dependencies, preserving supported media input behavior.
- **FR-011**: Bind immutable manifests/checksums to software/docs/schema version, exact revision, capabilities, package bytes, sources and signing disposition; reject mismatches.
- **FR-012**: Apply configured signing before final hashes; fail signing errors and never claim unconfigured signing/notarization.
- **FR-013**: Implement complete draft-first release publication with concise changelog-linked highlights and matching documentation promotion only after successful publication.
- **FR-014**: Align bootstrap/forms to canonical type, priority, effort and areas, preserving edited histories, closed states, project fields and compatibility labels.
- **FR-015**: Reconcile public contracts, help, glossary, changelog and schema; preserve older versioned surfaces and align all release versions.
- **FR-016**: Verify integrity, isolation, interruption and backend parity in required CI jobs below ten minutes; exclude real model engines, weights and acoustic inference.

### Key Entities

- **Backup**: Identity, mode, workspace/catalog revision, snapshot, manifest, completion proof and dependency.
- **Backup artifact**: Exact immutable locator/version, digest, byte count, relationships and optional copied bytes.
- **Retention reference**: Workspace ownership and backup lifetime pin.
- **Release manifest**: Version, source revision, capabilities, source inventory, package hashes and signing evidence.
- **Tracking taxonomy**: Canonical new-outcome labels with preserved historical issue ownership.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every retained self-contained artifact is retrievable with identical bytes after source-store loss.
- **SC-002**: All supported backend alternatives pass restore integrity and graph rebuild checks with zero replaced target authority on rejection.
- **SC-003**: Normal cleanup cannot delete reference-only backup dependencies before explicit release.
- **SC-004**: All three native platforms produce relocated working CLI/GUI packages with complete source inventories and verified hashes.
- **SC-005**: All changed-asset, version and revision mismatch fixtures fail before publication/promotion.
- **SC-006**: Historical edited/closed fixtures remain intact and new issues/forms carry canonical taxonomy.
- **SC-007**: Required CI jobs finish within ten minutes without real acoustic inference or model initialization.

## Assumptions

- Existing acquisition, subtitle, training and local/hosted adapter implementations are reused.
- Graph checkpoints reset on restore; accepted catalog evidence rebuilds materialization.
- Self-contained backup is default; reference-only is explicit and lifetime-pinned.
- Current version remains 0.0.0 until an owner-selected release change. Signing credentials and documentation destination are configuration, not invented identities.
- No public tag/release/deployment or final merge occurs in S014; publication machinery is validated with fixtures and candidate artifacts.
- The owner has no deployed media: no pre-v1 bulk conversion. Catalog evolution and transcript interoperability remain supported.
- Audio policy remains stereo/multitrack FLAC, audio-only Matroska independent tracks, eligible lossy mono MP3 passthrough/V0, no automatic DSP/upsampling.
