# Feature Specification: Speaker models and roster-constrained matching

**Feature Branch**: `codex/013-speaker-models-roster-matching`

**Created**: 2026-10-09

**Status**: Implemented (owner final review and merge pending)

**Input**: Owner-authorized S013, completing #15 and the remaining acoustic matching work in #35, advancing #29 under autopilot with automatic push/official PR and at most two review rounds.

## User Scenarios & Testing

### User Story 1 - Elect training from a current speaker corpus (Priority: P1)

The operator selects a speaker, creates a reusable current-reference dataset and elects a local or hosted training pipeline. The resulting run publishes a validated immutable speaker model without requiring individual sample approvals.

**Why this priority**: Acoustic profiles require independently grounded current audio and reproducible provenance.

**Independent Test**: A deterministic training adapter receives prepared segments from more than one corpus page, returns a model artifact and publishes one exact speaker-associated version; replacing its source before acceptance prevents publication.

**Acceptance Scenarios**:

1. **Given** current user-provided mappings across recordings, **When** a dataset is created and training is elected, **Then** every eligible reference is resolved, preparation exclusions are counted and the run records exact inputs, settings and adapter identity.
2. **Given** selected local or hosted execution, **When** training, cancellation, retry or declared checkpoint resume occurs, **Then** durable jobs preserve attempt provenance and never select another provider or base version implicitly.
3. **Given** a corrected mapping, identity revision or replaced audio/document, **When** pending work completes, **Then** stale preparation cannot publish and owned obsolete preparation is retired while completed model weights retain their own provenance.

### User Story 2 - Discover and retrieve exact model versions (Priority: P1)

The operator lists speaker-associated models, inspects provenance and compatibility, selects a current association and fetches an exact version through configured storage.

**Why this priority**: Trained outputs must be usable and portable rather than orphaned artifacts.

**Independent Test**: List/show/fetch discovers a model through speaker lineage, verifies artifact hashes and writes a portable manifest; hosted-only output explicitly reports unavailable downloadable weights.

**Acceptance Scenarios**:

1. **Given** a completed version, **When** speaker-filtered list/show/fetch is requested, **Then** family, version, dataset/run, originating speaker, current associations, output kind, compatibility, rights and availability are returned.
2. **Given** alias/default retargeting or identity merge/split, **When** an exact version is fetched, **Then** immutable origin remains unchanged and current association discovery reflects the explicit correction.

### User Story 3 - Match existing recording voices within a declared roster (Priority: P1)

The operator runs identity matching over current recording-local voices against compatible profiles belonging to its declared roster. Diarization remains the authority for activity and intervals.

**Why this priority**: Roster declarations must produce useful constrained identification without turning names into acoustic evidence.

**Independent Test**: Deterministic scores exercise empty/missing profiles, one-member unknowns, ambiguous candidates, absent members, multiple local voices matching one identity and preservation of manual corrections.

**Acceptance Scenarios**:

1. **Given** a roster and current local voices, **When** matching runs, **Then** only eligible member profiles are supplied and adapter-specific threshold, ambiguity margin and minimum evidence decide accepted, unknown or ambiguous outcomes.
2. **Given** no roster or unavailable profiles, **When** matching is requested, **Then** explicit diagnostics are returned without a library-wide fallback or fabricated activity.
3. **Given** a roster/profile/identity/audio/document edit or user correction during execution, **When** results return, **Then** stale acceptance is rejected and explicit user mappings remain authoritative.
4. **Given** prior automatic matches, **When** matching alone reruns, **Then** it supersedes applicable automatic mappings without retranscription, hidden historical assignment arrays or self-confirming training feedback.

### User Story 4 - Operate through shared surfaces and portable backends (Priority: P2)

The operator uses CLI or desktop controls to inspect datasets/models and submit training/matching work with the same validation and jobs. Export/import preserves portable lineage without credentials or copied obsolete assignments.

**Independent Test**: Runtime requests and CLI/desktop fixtures produce equivalent accepted work and SQLite/PostgreSQL plus graph projections preserve the same relationships.

**Acceptance Scenarios**:

1. **Given** configured adapters and storage, **When** either surface submits work, **Then** it observes the same durable state, cancellation and bounded diagnostics.
2. **Given** portable state, **When** it is exported/imported, **Then** current references and exact versions retain integrity without transient secrets or signed URLs.

### Edge Cases

- Empty corpus, concurrent pagination edits, duplicate source spans, unusable decode, optional text inputs, overlap and configurable quality exclusions.
- Foreign-workspace references, malicious adapter paths, unsupported output formats, corrupt artifacts, oversized outputs, credential/endpoint failures and unsupported resume.
- Missing current evidence, stale datasets, alias retargets, hosted-only models, partial checkpoints and unavailable objects.
- Empty roster, name-only roster members, insufficient evidence, ambiguous scores, unlisted voices and manual correction races.

## Requirements

### Functional Requirements

- **FR-001**: Dataset creation MUST resolve all pages of current speaker references with stable source/document/identity/mapping authority and a configurable selection recipe; roster membership alone contributes no training evidence.
- **FR-002**: Dataset manifests MUST contain ordered current evidence references, source hashes, settings and digest, without copied assignment lists or subtitle text. Summaries MUST report exclusions, durations and source coverage.
- **FR-003**: Preparation MUST use canonical tracks/source clock, expose decode/duplicate/quality diagnostics, materialize only selected adapter inputs and invalidate/retire obsolete owned inputs on current evidence corrections.
- **FR-004**: Elected local and explicitly configured hosted adapters MUST execute training with declared architecture/output kinds, base identity, settings, limits, credential reference, compatibility and rights declarations; local failure MUST NOT route to hosted execution.
- **FR-005**: Training MUST use durable concurrency/cancellation/retry jobs, exact attempts and declared checkpoint resume compatibility; partial checkpoints MUST NOT appear as completed models.
- **FR-006**: Publication MUST validate shape, integrity and compatibility, publish through configured storage and atomically fence exact current inputs before accepting immutable model versions.
- **FR-007**: Model list/show/fetch MUST support speaker filtering and exact versions, portable manifests, digest validation, explicit hosted retrieval limitations and current revisioned associations without rewriting origin.
- **FR-008**: Matching MUST consume existing local voice activity and only eligible roster-member profiles; no implicit library search, forced member appearance, one-to-one assignment or roster-length diarization count is permitted.
- **FR-009**: Matching MUST apply adapter-specific threshold, ambiguity margin and minimum evidence, return accepted/unknown/ambiguous outcomes and bounded missing-profile diagnostics.
- **FR-010**: Each matching attempt MUST freeze recording/audio/document, roster, identity/profile/version/manifest, settings and adapter/model authority; stale results MUST NOT mutate current mappings.
- **FR-011**: Accepted matching MUST retain bounded provenance outside embedded CueJSON, replace relevant automatic mappings, preserve explicit user corrections and exclude its own predictions from independent enrollment evidence.
- **FR-012**: Matching-only reruns MUST reuse current transcription/diarization evidence and never create new activity or copied assignment histories.
- **FR-013**: CLI and desktop MUST share validated runtime operations and jobs for dataset creation, elected training, model discovery/retrieval, profile associations and recording matching.
- **FR-014**: Filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB MUST retain equivalent portable lineage and constraints; exports MUST exclude credentials, transient URLs and obsolete content.
- **FR-015**: Deterministic contract, race, tenancy, adapter and end-to-end tests MUST verify these behaviors without loading/downloading acoustic models in required CI; actual acoustic quality MUST remain explicitly unmeasured until separately qualified.
- **FR-016**: Authoritative documentation, master schema, desktop/help/glossary and changelog MUST describe shipped behavior and limitations in the same slice; numbered planning remains internal.

### Key Entities

- **Selection recipe and dataset**: current speaker evidence references, applied filters, summary, digest and validity.
- **Preparation and training run**: exact input authority, adapter/base/settings, attempts, phases, checkpoints and owned prepared assets.
- **Model family/version/artifacts**: immutable origin/output and revisioned current speaker associations, declared compatibility/rights/availability.
- **Matching attempt/result**: frozen roster/profile/evidence authority, bounded decision diagnostics and current mapping provenance.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Corpus fixtures exceeding 100 entries yield complete eligible membership with no duplicate source spans or obsolete content copies.
- **SC-002**: Local and hosted adapter fixtures complete preparation, training, validated publication and exact fetch; cancellation/retry/checkpoint and source-edit races cannot publish stale output.
- **SC-003**: All specified roster decision cases and correction races pass, with zero unlisted candidate profiles and zero overwritten explicit user corrections.
- **SC-004**: Required repository/product/site checks and exact PR-head CI pass; every external review thread is answered and resolved within two review rounds.

## Assumptions

- The application has no owner audio and no pre-v1 bulk migration requirement.
- Existing shared runtime, adapters, scheduler, storage and current-reference invalidation are the foundation.
- Generic configured training adapters declare their output architecture and consumers; the runtime does not pretend arbitrary weights are universally compatible.
- Training and matching are elected configured operations, with no mandatory per-item human approval.

## Clarifications

### Session 2026-10-09

- Q: Does roster membership itself enroll a speaker? A: No. Enrollment resolves independently attributed current audio; automatic matching predictions are excluded from independent enrollment by default to avoid feedback.
- Q: What happens when an exact profile/model becomes unavailable or inputs change? A: Diagnose unavailable/stale work and require an explicit new attempt; never substitute a version or candidate scope.
- Q: How are engine-dependent score thresholds chosen? A: The selected matching adapter/profile declares score semantics; the user supplies effective thresholds, ambiguity margin and minimum evidence rather than assuming universal calibrated values.
- Q: What is the publication boundary? A: Automatically publish a PR, handle at most two external review rounds and green exact-head CI, then stop for the owner's final review and squash merge.
