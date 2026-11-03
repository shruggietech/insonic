# Feature Specification: Grounded speaker enrollment and unattended application

**Feature Branch**: `codex/073-grounded-speaker-enrollment`

**Created**: 2026-11-02

**Status**: Implemented; qualification and external review in progress

**Input**: S073, existing issues #230, #231 and #232 in milestone 49 under #211. Implement using Spec Kit and autopilot, publish the PR automatically, resolve at most two review rounds, and leave final review and merge to the owner. No new tracking issue.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Enroll from independently grounded clips (Priority: P1)

An operator uses independently supported contextual identities to select clean source-clock clips and enroll a speaker profile without individual clip approvals.

**Why this priority**: Grounded context must become useful enrollment evidence while preventing a model prediction from certifying itself.

**Independent Test**: Prepare and enroll a dataset containing independent contextual support; inspect every accepted and excluded clip and the resulting evidence/source/model/device lineage.

**Acceptance Scenarios**:

1. **Given** a current, independently supported contextual decision, **When** enrollment is elected, **Then** eligible source-clock clips enter the existing dataset and enrollment flow with their exact evidence class and lineage.
2. **Given** match-only, circular, stale, duplicate, mixed-speaker, overlapping, noisy or insufficient clips, **When** preparation runs, **Then** technical exclusions explain the outcome and excluded clips do not contaminate the profile.
3. **Given** a completed profile, **When** supporting identity or source evidence changes, **Then** current usability becomes stale while immutable profile artifacts and their historical lineage remain retrievable.

### User Story 2 - Apply and retract correctable decisions (Priority: P1)

An operator applies configured identity decisions through existing recording-to-speaker mappings and corrects or retracts them without rewriting transcript content.

**Why this priority**: Every consumer must agree on the same current identity authority.

**Independent Test**: Race an automatic decision against an owner correction, replay the operation after an unknown commit result, and inspect dependent datasets, profiles and graph projections.

**Acceptance Scenarios**:

1. **Given** a grounded decision and a current revision, **When** application commits, **Then** the existing mapping authority records the decision evidence and source revision exactly once.
2. **Given** a newer accepted mapping or owner correction, **When** stale automatic work attempts publication, **Then** it cannot overwrite that mapping or make an obsolete profile current.
3. **Given** an applied decision, **When** it is retracted or its supporting source/document/identity changes, **Then** dependent current state and recoverable projections are invalidated without deleting owner input or immutable historical models.
4. **Given** an interrupted commit, profile publication, graph delivery or cleanup, **When** work retries, **Then** it reconciles the existing outcome idempotently on every supported backend.

### User Story 3 - Run the configured workflow unattended (Priority: P2)

An operator configures discovery, inspection, resolution, application and elected enrollment once, then invokes or reruns the workflow through normal shared operations and saved pipelines.

**Why this priority**: The complete behavior must be usable without assembling private intermediate state or approving each sample.

**Independent Test**: Run installed command-line journeys for supported, unknown and ambiguous identities; interrupt, restart, correct and repeat them with the same configuration.

**Acceptance Scenarios**:

1. **Given** an already configured route and elected application/enrollment, **When** work runs, **Then** the complete flow proceeds without per-voice or per-clip approvals and records durable progress.
2. **Given** any output mode, **When** an operator inspects results, **Then** hypotheses, independently supported identities and acoustic matches are visibly distinct, with concise actionable human summaries and complete versioned machine output.
3. **Given** unavailable evidence, cancellation or provider failure, **When** the attempt terminates, **Then** accepted mappings/profiles remain intact, diagnostics explain recovery, and no unselected external research or fallback occurs.

### Edge Cases

- A contextual decision depends indirectly on a profile trained on that same identity claim.
- A correction arrives between selection, preparation, enrollment and publication.
- An operation committed but its response was lost; a repeated request must not duplicate mappings or erase a correction.
- A document is replaced, a candidate becomes contradicted, or a source is revised after application.
- Different recordings share labels/names; library and recording ownership remain enforced.
- A partial workflow completes application before enrollment fails; accepted results remain recoverable and later work resumes against current authority.
- Retraction targets an older automatic decision after the owner has replaced its mapping.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Independently supported contextual evidence MUST qualify for enrollment alongside manual attribution, with explicit evidence class and complete lineage; match-only and circular evidence MUST NOT qualify itself.
- **FR-002**: Enrollment MUST preserve existing duration, channel, overlap, signal and duplicate controls, configurable technical thresholds, source-clock mapping and visible exclusion reasons.
- **FR-003**: Profiles MUST record exact evidence, dataset, source, model and device identity; embedding enrollment MUST remain distinct from base-model parameter training.
- **FR-004**: Application MUST use existing revisioned recording-to-speaker mappings, recording decision provenance without rewriting source transcripts or maintaining a competing assignment store.
- **FR-005**: Stale decisions MUST NOT overwrite newer accepted mappings or owner corrections. Automatic reruns MUST preserve manual corrections.
- **FR-006**: Retraction, contradiction, source/document replacement and correction MUST invalidate dependent selections, prepared datasets, current profile authority and graph projections through recoverable existing paths while retaining immutable history and owner inputs.
- **FR-007**: Commit uncertainty, profile publication, outbox delivery and cleanup retries MUST reconcile idempotently on SQLite/PostgreSQL and applicable storage/graph alternatives.
- **FR-008**: Shared runtime and CLI requests plus saved pipeline configuration MUST cover discovery, inspection, resolution, application, elected enrollment and reruns without individual sample approval gates.
- **FR-009**: Human and versioned machine outputs MUST distinguish hypotheses, independent contextual support and acoustic predictions, retaining manual correction commands and actionable error/recovery guidance.
- **FR-010**: Work MUST reuse durable scheduling, cancellation/retry/restart, encrypted credential references and explicit configured routing; failures MUST NOT silently reroute or erase accepted mappings/profiles.
- **FR-011**: Supported filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB behavior, source/document authority and hidden noninteractive Windows child execution MUST remain intact.
- **FR-012**: Affected schema contracts, CLI help, authoritative documentation and changelog MUST agree. Verification MUST include installed CLI journeys, integrity/isolation/failure cases and CPU/NVIDIA enrollment policy; deterministic checks MUST remain distinct from separately elected real-engine quality evidence.

### Key Entities *(include if feature involves data)*

- **Grounded decision**: Current source-bound resolution, independent supporting evidence, target identity and input revision.
- **Applied mapping**: Existing recording/voice-to-speaker assignment with revision, origin and decision provenance.
- **Enrollment snapshot**: Immutable accepted/excluded clips, evidence lineage, source-clock coordinates and technical configuration.
- **Profile version**: Immutable dataset/model/device result with separately determined current usability.
- **Configured workflow attempt**: Durable operation with explicit stage elections, accepted outcomes, retry/cancellation state and diagnostics.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every eligible independently supported fixture completes configured application and enrollment with zero per-item approvals; all match-only/circular fixtures are excluded with explanations.
- **SC-002**: Every correction/retraction/concurrency scenario preserves the newest accepted identity and makes all affected dependent current state stale without altering owner inputs or immutable model bytes.
- **SC-003**: Every injected commit/publication/delivery/cleanup interruption either reconciles exactly one accepted result or reports an actionable terminal failure without authority loss.
- **SC-004**: Installed command-line journeys cover success, unknown, ambiguity, correction, cancellation, restart and repeat execution with equivalent outcomes across supported backend variants.
- **SC-005**: Required checks pass with retained evidence. Qualification targets 1800 seconds after readiness and respects the 3600-second hard limit, with preparation and combined timing reported separately.

## Assumptions

- S072 discovery/resolution and existing enrollment, mapping, work and backend infrastructure are the starting contracts; this slice extends them.
- Dedicated GUI controls may follow advanced CLI operations; the shared desktop operation bridge remains aligned and the documented lag remains explicit.
- Full known-truth corpus quality qualification and release readiness remain #233. This slice neither publishes a release nor claims acoustic accuracy from protocol fixtures.
- The owner has explicitly authorized push, official PR creation and up to two review rounds; merge remains owner-controlled.

## Clarifications

### Session 2026-11-02

- Q: Does contextual support replace manual authority? A: No. It is a distinct evidence class; explicit owner corrections win, and evidence integrity replaces no approval requirement.
- Q: Can a partial workflow roll back accepted mappings on enrollment failure? A: No. Preserve accepted stages and resume using current revision checks.
- Q: Does this slice claim full acoustic quality or publish a release? A: No. Deterministic behavior and elected engine/device verification are distinct; corpus qualification remains #233.

Autopilot coverage scan: functional, data, interaction, quality, dependency, failure, constraint, terminology and completion categories are clear; no unresolved owner question.
