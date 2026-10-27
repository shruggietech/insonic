# Feature Specification: Restored-library CPU processing acceptance

**Feature Branch**: `codex/050-restored-cpu-processing`

**Created**: 2026-10-26

**Status**: Specified

**Input**: Owner-authorized S050 for #147: extend the maintained real CPU journey through fresh processing after self-contained restore; deliver an unmerged PR under autopilot with at most two Codex review rounds.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Process restored media on destination workers (Priority: P1)

A library owner restores accepted media and profiles without access to the originating store, prepares compatible destination workers and performs fresh recognition, diarization and speaker matching.

**Why this priority**: Existing acceptance proves restored data retrieval but leaves fresh restored inference unverified.

**Independent Test**: Restore an isolated real CPU library, withdraw its source store, prepare destination prerequisites and process restored canonical media while comparing identities and retained history.

**Acceptance Scenarios**:

1. **Given** a verified self-contained backup and unavailable source store, **When** restoration and destination setup finish, **Then** recognition and diarization prerequisites become usable through the existing preparation contract.
2. **Given** restored media and verified retrieved speaker profiles, **When** fresh processing and matching complete, **Then** their new outputs bind the restored recording, elected models, destination workers and prior speaker identity.
3. **Given** prior accepted results and completed work, **When** destination processing publishes new results, **Then** retained canonical media, profiles and completed history remain unchanged, the current transcript may be superseded under its existing contract, and new results have distinct traceable lineage.

### User Story 2 - Diagnose and recover destination prerequisites (Priority: P2)

The owner can understand missing destination prerequisites and recover through configured setup, preserving model identities and existing authorization.

**Why this priority**: Missing host-local environments after a portable restore are expected; readiness must distinguish them from corrupted accepted data.

**Independent Test**: Exercise missing/changed destination files, explicit local-model recovery and cancellation with controlled negative fixtures before successful processing.

**Acceptance Scenarios**:

1. **Given** absent or changed destination bindings, **When** readiness or processing is attempted, **Then** an actionable bounded result identifies the unavailable prerequisite without altering accepted authority or silently switching devices/providers.
2. **Given** a missing local acquisition input, **When** a new model preparation is explicitly elected, **Then** one shared successor is linked to the original work and original history remains intact.
3. **Given** destination preparation or an ephemeral inspection is cancelled, **When** its cancellation completes, **Then** owned work stops, fresh incomplete state is not promoted, and prior accepted data remains available.

### User Story 3 - Audit reproducible restored inference (Priority: P2)

Maintainers receive compact source-bound evidence for restored inference and cleanup, with a clear distinction between measured CPU outcomes and untested dimensions.

**Why this priority**: Successful orchestration and readable restored data alone cannot establish inference acceptance.

**Independent Test**: Reject receipts with missing, fabricated or inconsistent destination stages, lineage, identity, timing or cleanup facts; validate a real complete-package campaign.

**Acceptance Scenarios**:

1. **Given** a complete exact-source package and frozen reference corpus, **When** the real campaign finishes, **Then** its receipt proves fresh destination recognition, diarization and matching and independently validates their measured outcomes.
2. **Given** failed, cancelled or superseded attempts, **When** subsequent work succeeds, **Then** prior failures remain separately attributed and compact evidence survives outside the workflow run before any full rerun.

### Edge Cases

- Source store and original workspace/control paths are unavailable after restoration.
- Destination paths differ and include Windows aliases, spaces or Unicode.
- Existing compatible environments may be reused only after verification; source host paths are not restored as execution authority.
- Restored current results can satisfy reuse checks; the campaign must prove a fresh inference run rather than accepting reused output.
- Destination setup or inspection can fail, time out or be cancelled without changing prior accepted data.
- Training may remain unavailable without an elected current dataset; its readiness is recorded without asserting architecture-specific training.
- Finally blocks cannot guarantee cleanup after forced process termination; report residuals truthfully.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Acceptance MUST restore from a verified self-contained backup while the original source store remains unavailable through fresh destination inference.
- **FR-002**: Acceptance MUST prepare or verify compatible destination recognition and diarization workers using ordinary configured setup, without introducing a product-wide preparation or approval gate.
- **FR-003**: Acceptance MUST execute fresh recognition and diarization on restored canonical media and fresh matching using restored/retrieved speaker profiles; reuse-only observations MUST NOT satisfy fresh inference.
- **FR-004**: Acceptance MUST verify new output lineage, unchanged model/device elections, destination worker identities and recording/speaker associations.
- **FR-005**: Retained canonical media, metadata, profiles/model outputs and completed work history MUST remain verifiable after new results publish. Current transcripts and derived artifacts MAY be superseded/retired under the existing retention contract; provenance MUST retain their identities/digests.
- **FR-006**: Missing/changed destination dependencies and explicit local-model recovery MUST preserve actionable diagnostics, original history and one stable successor identity, without silent routing or device fallback.
- **FR-007**: Cancellation/earlier deadlines MUST stop owned ephemeral work or follow existing durable cancellation semantics without promoting incomplete preparation or damaging accepted data.
- **FR-008**: The maintained receipt MUST independently reject absent, contradictory or mismatched destination stages, outcomes, identities, lineage, source withdrawal and cleanup claims.
- **FR-009**: Verification MUST retain exact package/source/harness/reference identities and separate preparation, qualification and combined timing under existing policy, plus compact audited failure/success evidence outside workflow runs before full reruns.
- **FR-010**: The campaign MUST use isolated owned fixtures, hidden noninteractive child processes, verified cleanup and bounded redacted diagnostics; it MUST NOT modify owner libraries or installed product state.

### Key Entities

- **Restored authority**: preserved recording/media/document/profile identities, accepted artifact bytes and completed work history.
- **Destination preparation**: selected role, environment identity, verified receipt, setup work and readiness outcome.
- **Fresh destination result**: new work/output identity linked to restored recording, retained history, frozen models/devices and destination environment.
- **Campaign receipt**: package/source/corpus identity, stage outcomes, reference comparisons, lineage, timing, explicit limits and cleanup facts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: One clean complete-package Windows CPU campaign completes fresh restored recognition, diarization and speaker matching with original source storage unavailable and all required evidence checks passing.
- **SC-002**: Every sampled retained canonical media/profile/history entry remains verifiable; every new sampled result has traceable new work and preserved model/recording identity, with supersession measured according to the current-transcript contract.
- **SC-003**: All controlled missing/changed dependency, recovery, cancellation and malformed-evidence cases produce their expected outcome without changing prior accepted authority.
- **SC-004**: All required local and elected CI checks pass, timing and owned cleanup are accounted for, all individual review findings are resolved within two review rounds, and the owner receives an unmerged PR.

## Assumptions

- This is a bounded increment of broad #147; broad acoustic accuracy, GPU/OCR, authenticated external routes, terminal interaction, interruption/disk/scale and installed-release observations remain distinct.
- Existing public setup/restore/processing contracts govern behavior; implement proportional repairs only for demonstrated gaps.
- Use the existing frozen public reference corpus and verified read-only model inputs and ordinary pinned managed worker preparation. No new route or credential election is implied.
- Record training readiness truthfully; speaker embedding enrollment is not base-model retraining.

## Clarifications

### Session 2026-10-26

- Autopilot decision: run genuinely fresh destination work, rather than count restored accepted results as inference evidence.
- Autopilot decision: retain existing setup elections and portable receipt binding; do not serialize source host environments into backups.
- Autopilot decision: keep #147 open, use bounded child #178 for S050, and stop before owner merge.
- Source-contract clarification: preservation covers canonical media, metadata, profiles, provenance and completed history. Existing one-current-document rules permit superseded transcript rows and mapped derivatives to retire; S050 does not introduce alternate transcript retention.
