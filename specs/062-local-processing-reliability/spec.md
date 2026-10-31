# Feature Specification: NVIDIA-first local processing and CLI reliability

**Feature Branch**: `codex/062-local-processing-reliability`

**Created**: 2026-10-31

**Status**: Specified

**Input**: Owner-authorized S062, issues #202, #203, #204, #205, #206, #207 and #208. Complete the combined local-processing and operator-feedback milestone, publish a PR, resolve CI/reviews and stop before merge. At most two external Codex review rounds, including the automatic initial round.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Reliable accelerated local processing (Priority: P1)

An operator uses available NVIDIA acceleration for recognition, diarization, speaker enrollment and matching, with automatic CPU recovery when that local device cannot execute. Explicit CPU selection remains CPU.

**Why this priority**: One shared policy makes the existing local processing workflow usable on both accelerated and CPU-only hosts.

**Independent Test**: Exercise each operation with available acceleration, absent hardware, missing/incompatible libraries and bounded resource failure; inspect requested policy, attempted devices, actual device and fallback cause.

**Acceptance Scenarios**:

1. **Given** verified usable NVIDIA prerequisites, **when** a new local request omits its device, **then** execution uses CUDA and records that fact.
2. **Given** unavailable/incompatible CUDA or CUDA resource exhaustion, **when** an automatic or CUDA request executes, **then** one CPU attempt may complete under the original deadline with the same source/model and an observable reason.
3. **Given** explicit CPU or previously frozen CPU work, **when** execution or retry occurs, **then** no CUDA attempt is made.
4. **Given** a usable selected Windows environment, **when** readiness runs, **then** it resolves the same elected native libraries as execution without loading models.

### User Story 2 - Honest transcription and actionable failures (Priority: P1)

An operator receives an honest no-speech outcome for exact zero-signal media and useful corrections for invalid inputs or unavailable prerequisites, while prior accepted data remains intact on failures.

**Why this priority**: Invented speech and generic failures undermine accepted recording authority and make recovery difficult.

**Independent Test**: Exact silence, quiet speech, noise/music and conflicting recognition/diarization fixtures; unsupported languages and typed worker failures through every output mode.

**Acceptance Scenarios**:

1. **Given** exact digital silence, **when** recognition runs on either device policy, **then** invented text is not accepted and an explicit no-speech outcome retains timing/provenance.
2. **Given** nonzero quiet speech or model disagreement, **when** diarization reports no speech, **then** valid recognition is not erased solely by that observation; contradictions are diagnosed.
3. **Given** an unsupported known language, **when** work is requested, **then** admission identifies the language field and correction without queueing immutable bad work.
4. **Given** device/dependency/input failure, **when** the work is inspected, **then** bounded generated reasons survive human, verbose, debug and JSON output without raw engine text or private inputs.

### User Story 3 - Concise usable CLI guidance (Priority: P2)

An operator sees one task-specific outcome and valid next actions, and can follow shipped documentation without encountering nonexistent commands or adapters.

**Why this priority**: Complete local workflow delivery includes reliable instructions and intelligible results.

**Independent Test**: Golden output and installed/package example journeys for completed/empty/missing work, unknown commands and artifact lifecycle refusals.

**Acceptance Scenarios**:

1. **Given** finished work, **when** normal output renders, **then** it has no duplicate summary, empty Error field or irrelevant wait/cancel action; complete JSON and useful warnings remain available.
2. **Given** a missing work ID or unknown command, **when** help is rendered, **then** it names executable discovery/help commands instead of reusing nonexistent resources or placeholder routes.
3. **Given** retained, leased or grace-period-bound artifacts, **when** retirement is refused, **then** the actual lifecycle cause and applicable action are explained.
4. **Given** current processing instructions, **when** their changed commands run against a packaged CLI, **then** they are accepted and accurately identify shipped adapters and device behavior.

### Edge Cases

- Partial CUDA output, CUDA OOM, CPU fallback failure, cancellation and exhausted enclosing budgets never publish partial work or start unlimited retries.
- Model corruption, invalid timing/input, invalid embedding and unknown engine failures never justify blind device retry.
- Historical frozen explicit device choices remain valid; new defaults do not retarget accepted work or change hosted routes.
- A zero-result transcript must obey the current Cueson contract; if empty documents cannot be accepted, retain prior authority and report a documented no-speech result.
- Quiet nonzero samples, music, noise and conflicting stage results do not become a blanket quality rejection.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: New omitted local device selections MUST prefer CUDA; explicit CPU MUST remain CPU. Recognition, diarization and single/batch speaker embedding consumers MUST share this policy (#202, #204).
- **FR-002**: Recoverable CUDA device, native-dependency and resource failures MUST permit at most one CPU retry under the original operation deadline with unchanged source/model/provider authority; unrelated errors and cancellation MUST NOT cause fallback (#202, #206).
- **FR-003**: Work/results MUST retain requested policy, attempted device(s), actual device and generated fallback cause. Previously accepted work and model/profile compatibility MUST remain intact (#202, #204).
- **FR-004**: Readiness MUST use the elected verified environment's native-library resolution and distinguish absent hardware, missing/incompatible libraries and unsupported compute capability without model construction (#203).
- **FR-005**: Exact digital silence MUST NOT accept invented words; recognition/diarization disagreement MUST be visible without using diarizer silence alone to remove real text (#205).
- **FR-006**: Known unsupported language values MUST fail early with field-specific correction. Bounded typed worker failures MUST remain actionable through durable work and all output modes, with raw stderr/private content excluded (#206).
- **FR-007**: Normal human output MUST show one concise outcome, preserve warnings and useful readback, omit empty error fields and select state/resource-appropriate actions. Detailed provenance remains available through verbose/debug/JSON (#207).
- **FR-008**: Artifact retirement refusal MUST distinguish retained reference, active lease, grace period and true revision conflict without weakening lifecycle rules (#207).
- **FR-009**: Current schemas, CLI help, shared desktop/runtime callers, managed defaults and authoritative documentation MUST agree. Changed documentation commands MUST execute against source/package fixtures; only shipped adapter functionality may be described as available (#208).
- **FR-010**: Source/document integrity, encrypted credential references, configured unattended behavior and supported filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB parity MUST be preserved. Console children remain hidden/noninteractive.
- **FR-011**: Required deterministic checks and separately elected real-engine evidence MUST be distinguished. Preparation, qualification and combined time remain separate with the existing 1,800-second target, 3,600-second hard limit and finalization reserve.

### Key Entities

- **Device policy**: Immutable requested device preference, finite attempts and resulting device/fallback observation.
- **Worker failure**: Generated bounded category, affected field/prerequisite and applicable correction without raw input.
- **Processing quality observation**: Exact zero-signal or stage disagreement tied to source intervals and existing provenance.
- **Operator outcome**: Task/result state, warnings, lifecycle reason and valid follow-up routes.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All four local inference consumers pass available-device, CPU-only, explicit-CPU and finite-fallback cases, retaining exact input/model identities.
- **SC-002**: Exact zero-signal fixtures never publish fabricated words; nonzero/disagreement fixtures preserve valid text and expose discrepancies.
- **SC-003**: Every listed invalid-language, device/dependency and lifecycle refusal case has an applicable generated correction and preserves accepted data.
- **SC-004**: Every changed example command executes successfully in its declared source/package fixture, and all listed golden-output cases preserve complete machine readback.
- **SC-005**: All seven issues have traced implementation and verification evidence; actual engine observations state device/model/corpus limits separately from deterministic checks.

## Assumptions

- The current owner-approved NVIDIA-first policy supersedes the older CPU-default/no-fallback requirement. Existing saved explicit CPU choices and frozen work remain CPU.
- CUDA preference falls back to CPU on classified CUDA failures, including explicit CUDA requests; no new hosted routing or model substitution is authorized by fallback.
- Long-recording windows/repair and contextual identity/answer epics remain separate slices. The isolated ArcadeDB timing observation is investigated diagnostically without inventing a reproduced backend defect.
- Push and official PR publication are explicitly authorized. One additional Codex review request is allowed after the automatic first round. Merge, tagging and release remain owner-controlled.

## Clarifications

### Session 2026-10-31

- Q: Does explicit CUDA permit fallback? A: Yes, the approved issue requests CPU recovery when CUDA cannot execute; explicit CPU remains strict, and requested CUDA stays in provenance.
- Q: Does a diarizer no-speech result invalidate recognition? A: No. Only verified exact zero-signal prevents fabricated recognition; disagreement is diagnostic.
- Q: Should legacy work be silently upgraded? A: No. Normalize new omitted choices only; retain frozen device/model/input identities.
