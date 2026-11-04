# Feature Specification: Speaker quality and release qualification

**Feature Branch**: `codex/074-speaker-release-qualification`

**Created**: 2026-11-03

**Status**: Specified

**Input**: Proceed with S074 (#233/#211) using Spec Kit and autopilot, including real CPU/NVIDIA qualification. No new tracking issue. Push/open the PR automatically, resolve reviews within two rounds, and hand off for owner review and merge after green CI.

## User Scenarios & Testing

### User Story 1 - Understand measured speaker quality (Priority: P1)

A maintainer can reproduce contextual identity and independent enrollment evaluation on conversations with known voices, rather than infer model accuracy from protocol fixtures.

**Why this priority**: Remaining milestone acceptance requires observable identity quality and truthful limitations.

**Independent Test**: Run a versioned evaluation corpus and reconcile every expected/observed case with the retained summary.

**Acceptance Scenarios**:

1. **Given** introductions, repeated names, quotations, contradictions and unknown identities, **When** configured discovery and resolution run, **Then** precision, coverage, unresolved/ambiguous outcomes and errors are reported separately with supporting references.
2. **Given** enrollment and held-out matching clips with known provenance, **When** real engines enroll and match profiles on CPU and NVIDIA, **Then** actual device, model/configuration, source identity and matching outcomes are retained without treating protocol outputs as acoustic accuracy.
3. **Given** dependent predictions or duplicated source evidence, **When** enrollment eligibility is evaluated, **Then** circular evidence is rejected and repetition cannot supply independent support.

### User Story 2 - Trust the complete configured workflow (Priority: P1)

An operator can run the configured speaker workflow unattended and correct an identity without stale profiles or mappings silently surviving.

**Why this priority**: Measurements are useful only when the installed product preserves evidence and correction authority.

**Independent Test**: Run installed discovery/application/enrollment and correction/recovery journeys on supported backend combinations.

**Acceptance Scenarios**:

1. **Given** a configured provider and independent evidence, **When** the workflow runs, **Then** accepted mappings and enrollment complete without per-voice or per-clip approval.
2. **Given** an owner correction, contradiction or source replacement, **When** affected work resumes or matching runs, **Then** current authority is respected, stale evidence is excluded and immutable historical outputs remain inspectable.
3. **Given** cancellation, restart or failed enrollment, **When** recovery is requested, **Then** accepted work is reconciled without duplicate evidence or unauthorized provider rerouting across default and alternative backends.

### User Story 3 - Review a qualified release candidate (Priority: P2)

The owner receives a compatibility-reviewed candidate with synchronized software, documentation and schema versions, complete evidence and an official reviewed PR.

**Why this priority**: This closes the existing milestone and makes subsequent publication concrete.

**Independent Test**: Validate retained candidate source, version, inventory, documentation and terminal qualification receipts.

**Acceptance Scenarios**:

1. **Given** accumulated speaker features and compatibility findings, **When** release preparation runs, **Then** the proposed version reflects the change, published history remains intact and software/docs/master schema agree.
2. **Given** all mandatory and elected checks, **When** the final PR is handed off, **Then** applicable CI is green, every finding is addressed, at most two review rounds have been requested and evidence limits are explicit.

### Edge Cases

- Equal names, quoted or contradictory introductions, absent identities and close acoustic alternatives.
- Empty metric denominators, failed engine runs, missing prerequisites and GPU fallback.
- Mixed speakers, duplicate clips and source lineage crossing enrollment/evaluation partitions.
- Retraction during enrollment, stale profiles, restart after unknown commit outcomes and portable tooling mismatch.
- Feature edits to historical documentation, missing new-version screenshots and incomplete advanced CLI metadata.
- Terminal CI failure or time overrun; audited evidence precedes a full rerun.

## Requirements

### Functional Requirements

- **FR-001**: Retain a reproducible synthetic or licensed corpus manifest with known local voices, independent enrollment/evaluation partitions, case expectations, exact content identity and provenance (#233).
- **FR-002**: Evaluate contextual discovery and resolution across introductions, name collisions, quotations, contradictions, unknown identities and independent support (#233).
- **FR-003**: Report discovery precision/coverage, unresolved/ambiguous outcomes, circular-evidence rejection and acoustic matching separately, including counts, undefined denominators, exact configuration and observed errors (#233).
- **FR-004**: Execute elected real enrollment/matching on CPU and NVIDIA. Retain actual device attempts and fallback; CPU fallback alone cannot satisfy NVIDIA qualification (#233).
- **FR-005**: Reject duplicate/dependent evidence as independent corroboration and demonstrate that false predictions cannot certify their own enrollment (#233/#211).
- **FR-006**: Exercise installed unattended workflow, correction, retraction, cancellation, retry and restart through shared authority on default and supported alternative backends (#211).
- **FR-007**: Preserve owner inputs, current evidence, credentials, configured routes and historical immutable outputs. Use owned disposable qualification workspaces and hidden noninteractive processes (#211).
- **FR-008**: Select a compatibility-appropriate release version and align software, public documentation, rendered contracts and master schema while preserving published history (#233).
- **FR-009**: Document actual CLI capabilities, GUI lag, model-quality limitations and behavior. Keep internal evidence separate; concise release highlights end with the changelog link (#233).
- **FR-010**: Run root and affected product/site/package checks plus elected real-engine qualification. Separate preparation/qualification/combined accounting, target 1800 seconds and enforce the 3600-second hard limit with finalization reserve and evidence retention (#233/#211).
- **FR-011**: Push and open the official PR automatically, handle every finding within two requested review rounds, and hand off after applicable terminal CI acceptance and issue/epic acceptance reconciliation (#211).
- **FR-012**: Create no tracking issue, merge no PR and publish no release/tag or production documentation. Existing issues close only when demonstrated acceptance warrants it (#233/#211).

### Key Entities

- **Evaluation corpus**: Source identity, license/provenance, local voice truth, expectations and independent evidence partitions.
- **Evaluation result**: Corpus/model/configuration/device identity, counts, metrics, errors, timing and terminal state.
- **Release candidate**: Aligned version, source revision, immutable assets, compatibility findings and audited receipts.
- **Milestone packet**: Existing acceptance criteria mapped to evidence, limitations and PR/review state.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every corpus case has one accounted terminal outcome or explicit error; aggregate counts reconcile with individual observations.
- **SC-002**: Separate real CPU and NVIDIA receipts identify actual execution and held-out outcomes, with no unreported fallback or fabricated quality claim.
- **SC-003**: Deterministic independence, authority, correction and recovery assertions pass across required backend journeys; circular fixtures yield zero independent enrollment eligibility.
- **SC-004**: Software/docs/master schema share one proposed version, published history is intact and the complete candidate meets integrity/provenance checks.
- **SC-005**: Final applicable CI is green, all review findings are handled within two rounds and #233/#211 have complete evidence mappings without another issue.

## Assumptions

- Merged S072/S073 are dependencies; their protocol fixtures do not establish acoustic quality.
- 1.3.0 is the initial minor-release proposal for additive speaker features; compatibility research determines the final version.
- The owner explicitly elects CPU and NVIDIA qualification. Existing local models/providers may be reused after readiness/identity verification; no implicit external routing.
- Report measured performance and limitations without inventing universal accuracy thresholds or tuning against held-out truth.
- This slice prepares the candidate; publication follows separate authorization after merge.

## Clarifications

### Session 2026-11-03

- Autopilot resolves routine choices from #233/#211 and current contracts; no owner clarification remains.
- Real acoustic and model-generated contextual outcomes are distinct from deterministic contract tests.
- Synthetic transcript semantics use a disclosed PCM carrier without speech. Acoustic evaluation uses independently partitioned licensed human speech. Neither small corpus establishes broad conversational generalization.
- Explicit push/PR authorization supersedes the skill's default pre-push pause. Owner merge and the two-round review cap remain.
