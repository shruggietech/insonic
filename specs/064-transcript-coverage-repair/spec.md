# Feature Specification: Transcript coverage and targeted repair

**Feature Branch**: codex/064-transcript-coverage-repair

**Created**: 2026-10-31

**Status**: Implemented, qualified and reviewed; final closeout CI receipts are recorded in PR #236 before owner merge

**Input**: S064 under autopilot for [#217](https://github.com/shruggietech/insonic/issues/217), [#218](https://github.com/shruggietech/insonic/issues/218) and [#219](https://github.com/shruggietech/insonic/issues/219). Publish an official PR automatically, address every finding within at most two external review rounds, and stop for owner final review and merge.

## User Scenarios & Testing

### User Story 1 - Understand transcript coverage (Priority: P1)

An operator inspects an accepted recording and sees which selected source intervals have text, observed speech without text, uncertain evidence or suspicious repetitions. Independent tracks remain separate and every metric names its denominator.

**Why this priority**: Repair must address observable gaps without treating silence, music or uncertain model output as proof of failure.

**Independent Test**: Inspect known-truth recordings with unequal tracks, gaps, repeats, quiet activity and model disagreement; compare report intervals and denominators to their source evidence and verify accepted state is unchanged.

**Acceptance Scenarios**:

1. **Given** two unequal source tracks and speech missing text on only one, **When** coverage is requested, **Then** the report names that track and exact bounded interval without charging the other track.
2. **Given** silence, music, low-volume speech or absent activity evidence, **When** coverage is inspected, **Then** observed silence, absent audio and unknown speech coverage remain distinct, with no invented speech or timestamps.
3. **Given** corroborated duplicate boundary text, genuine repeated utterances and conflicting model observations, **When** configured diagnostics run, **Then** reasons and evidence remain visible and genuine repetitions are not automatically rejected.
4. **Given** a recording changes during inspection, **When** the report is returned, **Then** it identifies the exact source and document snapshot it evaluated and never mutates current authority.

### User Story 2 - Repair bounded regions without losing accepted work (Priority: P1)

An operator explicitly chooses source regions or elects diagnosed-region repair. Configured workers reprocess only the bounded target and permitted context, compare the candidate with the accepted transcript, and commit a replacement only when the documented criteria improve.

**Why this priority**: Long recordings should not require full retranscription for local errors, and failed repair must not destroy accepted work.

**Independent Test**: Inject one known defect, repair it through the durable workflow, verify unchanged out-of-scope cues and source mappings, then repeat under cancellation, interruption and concurrent replacement.

**Acceptance Scenarios**:

1. **Given** a diagnosed gap or repetition, **When** configured repair runs, **Then** only selected windows and declared context execute, with source/track/model identity preserved and no per-region approval.
2. **Given** an explicit interval intersects an existing cue, **When** the repair scope is normalized, **Then** the declared reconciliation margin covers complete affected cues and all cues outside it retain their original identity, text, timing and voice data.
3. **Given** a candidate with no measurable improvement or worsened validity/coverage, **When** acceptance is evaluated, **Then** the previous document and mappings remain current and a bounded decision receipt explains rejection.
4. **Given** cancellation, publication failure, stale ownership, changed source or a concurrent document/mapping correction, **When** repair attempts acceptance, **Then** no stale candidate overwrites accepted state and retries reconcile the exact original intent.
5. **Given** interruption after a verified target window, **When** the same repair work resumes, **Then** compatible completed target windows are reused; unrelated windows are never retranscribed.
6. **Given** a saved repair policy and automatic diagnosed-region selection, **When** the configured repair operation executes, **Then** it honors finite attempt/region/resource budgets and terminates with accepted, unchanged or actionable failure status without changing providers.

### User Story 3 - Qualify the complete long-recording workflow (Priority: P2)

A maintainer runs a reproducible unattended journey from admission and window processing through interruption, coverage, repair, final retrieval and cleanup, then evaluates milestone release readiness.

**Why this priority**: The complete workflow must be demonstrated across supported platforms and backends before the milestone is considered complete.

**Independent Test**: Execute the maintained qualification journey and documented CLI examples against the candidate, retain auditable evidence and prepare a compatibility-based release recommendation.

**Acceptance Scenarios**:

1. **Given** known-truth fixtures, **When** the complete journey executes, **Then** source intervals, cue timing, bounded resources, checkpoint integrity, targeted changes and cleanup are verified together.
2. **Given** required Windows/Linux and alternative-backend checks, **When** acceptance completes, **Then** preparation, qualification and combined times are retained separately with terminal timestamps and meaningful failure receipts.
3. **Given** the accepted milestone candidate, **When** the release packet is prepared, **Then** it identifies compatibility changes, a proposed semantic version, aligned release-version requirements, concise changelog-linked highlights and remaining evidence limits; it does not claim a published release.

### Edge Cases

- Empty transcripts, no accepted document, absent or stale source mapping, supplied transcripts whose native clock extends beyond audio, negative/positive source origins and final partial windows.
- Independent coincident tracks, downmix versus explicit channels, unavailable audio streams, overlapping requested ranges and ranges crossing cue boundaries.
- Missing acoustic evidence, contradictory no-speech results, legitimate repetition, identical text at separate times and malformed evidence.
- No repairable regions, exhausted finite budgets, changed configurations, stale document/source/owner generations, cancellation before/after publication and unknown commit outcomes.
- Rejected candidates and temporary publications must not become alternate accepted transcripts or leak retained transcript bytes.
- Large reports, bounded report pagination/output, integer overflow, zero-duration intervals and source intervals outside the elected media.

## Requirements

### Functional Requirements

- **FR-001**: Coverage MUST bind its recording, source identity/revision, document digest/revision, selected track/channel, exact intervals, evidence basis and effective tolerances.
- **FR-002**: Coverage MUST report selected audio duration, accepted cue coverage, observed speech coverage and uncovered observed speech separately for each track; unavailable denominators MUST be explicit.
- **FR-003**: Coverage MUST distinguish missing audio, observed silence, unknown acoustic evidence and speech missing text, without treating diarizer silence or confidence as unquestionable truth.
- **FR-004**: Diagnostics MUST expose bounded gap, repeated-text and boundary-inconsistency regions with generated reasons and provenance; genuine nonoverlapping repetitions and uncertain/music/quiet observations MUST not be blindly rejected.
- **FR-005**: Coverage MUST be available through human and versioned machine-readable CLI/shared runtime output with configurable tolerances; inspection MUST not start inference or alter accepted documents, identities or mappings.
- **FR-006**: Repair MUST support explicit source ranges and automatic selection from current diagnostics, bind exact source/document and effective configuration, and use finite target, context and attempt budgets.
- **FR-007**: Repair MUST reuse bounded window preparation and compatible durable checkpoints. It MUST not run inference on unaffected windows or silently switch models/providers.
- **FR-008**: Repair MUST declare its normalized reconciliation scope, preserve out-of-scope cue identity, text, timing, native fields and source/voice mappings (structural ordinal reindexing and document summary updates are permitted), and retain canonical sources and supplied native document clocks.
- **FR-009**: Candidate acceptance MUST require a valid document and a documented strict improvement in diagnosed coverage/anomalies without worsened protected metrics. Insufficient evidence or an unchanged/non-improving candidate MUST preserve current authority with a decision receipt.
- **FR-010**: One accepted repair MUST atomically replace the current Cueson document under revision/owner and speaker-mapping fences, invalidate dependent current mappings/datasets/evidence consistently, and reconcile cleanup without deleting source inputs.
- **FR-011**: Cancellation, failure, stale evidence, changed source or owner and unknown commit outcomes MUST preserve or reconcile exact accepted authority; retries MUST be idempotent for the original elected intent.
- **FR-012**: Configured repair MUST run unattended through existing work/model/credential routing. Saved policies and per-request elections MUST remain inspectable; no per-region approval queue or new external route is introduced.
- **FR-013**: Reports and compact repair decision receipts MUST retain bounded before/after metrics, selected scopes, actual attempts, reasons and candidate/current identity while retiring rejected/superseded temporary transcript bytes.
- **FR-014**: The complete admission, processing, interrupted resume, coverage, targeted repair, retrieval and cleanup journey MUST have maintained deterministic qualification and executable CLI examples across applicable platform/backend paths.
- **FR-015**: The milestone packet MUST evaluate compatibility and propose a semantic version and concise release highlights, keep software/docs/master-schema release alignment explicit, and distinguish tested behavior from acoustic/hardware evidence and publication.
- **FR-016**: Required checks MUST preserve current qualification timing and retention rules, hidden Windows child execution, canonical source integrity and all supported backend contracts.

### Key Entities

- **Coverage report**: An immutable observation of an exact current recording snapshot, selected source tracks, named metrics, evidence limitations and suspicious regions.
- **Coverage region**: A bounded source interval, track identity, generated reason, supporting cue/activity/window references and evidence strength.
- **Repair election**: Frozen source/document identity, explicit or diagnosed targets, pipeline/model selection, effective policy and finite budgets.
- **Repair candidate**: Temporary validated target output merged with unaffected accepted cues; never a second current document.
- **Repair decision**: Compact before/after measurements, scopes, attempts, candidate identity and acceptance/rejection reason tied to durable work.
- **Milestone qualification packet**: Reproducible fixture results, compatibility assessment, timing/integrity receipts and release recommendation.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Known-truth coverage fixtures identify every injected uncovered-speech and duplicate-boundary defect with exact track/range evidence, with zero destructive changes from inspection.
- **SC-002**: Silence, music, low-volume, unknown-evidence and genuine-repetition controls produce no fabricated speech evidence or automatic rejection based solely on confidence/silence.
- **SC-003**: Accepted targeted repairs eliminate the elected fixture defect while preserving 100% of out-of-scope cue identities/content/timing/source associations and making zero inference calls for unrelated windows.
- **SC-004**: Every injected failure/cancellation/stale-generation boundary preserves the previous accepted document or reconciles the one proven accepted result; retry never accepts corrupt or mismatched checkpoints.
- **SC-005**: Each repair terminates within its configured finite attempt/region limits, returns a bounded decision receipt and requires zero new per-region approvals.
- **SC-006**: The maintained complete workflow and documented CLI examples pass on applicable Windows/Linux and backend checks; terminal qualification receipts satisfy mandatory checks and the 3,600-second hard limit, with 1,800-second target overruns reported as advisory.
- **SC-007**: The release recommendation names a compatibility-appropriate proposed version and all unverified evidence; no tag, merge or release publication is implied.

## Assumptions

- S063 (#212-#216) and #205 are merged prerequisites. This slice completes the remaining child deliverables of milestone 47; #210/#211 and their children remain separate.
- Current source/document and external speaker mapping authority are reused. Repair is additive and opt-in; existing normal processing retains its behavior.
- Real-model CPU/NVIDIA acoustic acceptance remains separately elected outside deterministic required CI; fixture correctness is not acoustic accuracy.
- The user explicitly authorizes push, official PR publication and at most one requested review follow-up after the initial round. Owner merge and release publication remain separate.

## Clarifications

### Session 2026-10-31

- Q: Does coverage inspection perform new inference? A: No. It reports persisted current evidence and explicit unknowns; repair alone executes elected workers.
- Q: What qualifies a candidate as improved? A: A strict reduction in diagnosed target defects under the same source/evidence/tolerance basis, with no worsened protected coverage/validity metric; otherwise retain the current document.
- Q: How are cue-boundary intersections handled? A: Normalize and expose a bounded scope covering whole intersecting cues and permitted context, then preserve all cues outside that scope.
- Q: Does automatic repair need a new approval queue? A: No. An elected saved policy or request executes diagnosed targets under finite budgets using existing route authority.
- Q: Does milestone qualification authorize a release? A: No. It produces the release recommendation and evidence packet; publication and merge remain owner-controlled.

### Design clarification 2026-10-31

Cueson requires each cue ordinal to equal its array position. Targeted insertion/removal therefore permits ordinal reindexing and root summary updates, while preserving all unaffected semantic cue fields. Compact text-free acoustic observations must be captured before cue assembly because speaker attributions alone omit speech in transcript gaps. Speaker mappings can change without changing the recording revision, so repair acceptance compares their frozen digest transactionally.
