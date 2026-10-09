# Feature Specification: S011 Recording replacement and declared speaker rosters

**Feature Branch**: `codex/011-recording-replacement-rosters`

**Created**: 2026-10-09

**Status**: Specified

**Input**: Complete #30/#33 through explicit stable-recording audio replacement and shared import contracts. Include #35 declared-roster storage, editing, import and replacement lifecycle. Use Spec Kit autopilot, automatic push/official PR, at most two external review rounds, and stop before owner merge.

## User Scenarios & Testing

### User Story 1 - Replace recording audio deliberately (Priority: P1)

A user selects an existing recording and explicitly replaces its audio while keeping its identity and choosing the treatment of its current transcript and declared roster. An omitted replacement election skips rather than overwrites. Candidate preparation does not disturb the accepted recording.

**Why this priority**: Completes the import contract and prevents consumers from following superseded source evidence.

**Independent Test**: Replace an accepted recording with different canonical audio and exercise each transcript election, concurrent edits, delayed work, cancellation and accepted-result replay.

**Acceptance Scenarios**:

1. A resolved media target without audio-replacement permission returns a successful skip and unchanged accepted objects.
2. Elected replacement with current transcript requires explicit keep, clear or replace; unresolved or contradictory elections fail before current publication.
3. Keep requires an explicit assertion that the existing transcript applies and checks known source-clock bounds. Equal duration alone never supplies that assertion. Clear removes the current document; replace validates selected supplied/embedded input before acceptance.
4. Successful replacement preserves the library recording ID, publishes the complete required canonical tracks, increments source authority and reconciles mappings, graph evidence, current segment/corpus preparation and active processing/training work.
5. Any invalid, stale, cancelled or failed replacement preserves prior accepted audio, document, roster and references. An accepted replay reconciles the same receipt without repeating transformations.

### User Story 2 - Declare expected recording speakers (Priority: P1)

A user maintains a recording-specific roster of existing centralized speaker identities independently of transcripts, voice mappings, profiles or acoustic processing. The roster can be created for audio-only recordings and edited through shared CLI and desktop controls.

**Why this priority**: Supplies intentional context required by import and audio-replacement semantics without manufacturing acoustic evidence.

**Independent Test**: Read/add/remove/replace/clear a roster on an audio-only record, prove uniqueness and concurrency, and restore the recording and membership through portable catalog contracts.

**Acceptance Scenarios**:

1. Stable centralized IDs and unambiguous aliases resolve to a unique membership set; ambiguous, missing or foreign identities fail without mutation.
2. Repeated add/remove requests are idempotent; whole-roster replacement is explicit and stale edits cannot overwrite a newer revision.
3. Reading distinguishes undeclared membership from an explicitly declared empty roster. Clear preserves revision authority and records an empty set.
4. Transcript and local speaker mapping edits do not change roster membership; roster edits do not delete accepted mappings or emit Cue JSON attribution.
5. Inactive identities remain visible with lifecycle diagnostics; new elections follow centralized identity rules without inventing unimplemented merge/split behavior.
6. Portable snapshots/restores and graph projections preserve roster relationships while catalog state remains authoritative.

### User Story 3 - Import and update recordings consistently (Priority: P1)

Users provide repeated known-speaker references during new media import or batch work. Shared import, roster editing and replacement controls expose the same meanings, outcomes and diagnostics through CLI, manifests, runtime and desktop.

**Why this priority**: Completes #30's required operation matrix and prevents interface-specific mutation rules.

**Independent Test**: Compare equivalent CLI, manifest and desktop requests, initial roster publication, replacement retain/clear decisions and paged durable receipts.

**Acceptance Scenarios**:

1. Initial canonical admission and selected document/roster accept atomically, including audio-only admissions; failed items never publish orphan memberships.
2. Audio replacement with a declared roster requires explicit retain or clear. Transcript-only replacement preserves membership. An explicit initial/import roster list has documented precedence and cannot silently overwrite a target's roster.
3. Names resolve once at election; queued target/source/document/roster revisions and resolved identities are frozen. Alias changes cannot retarget elected IDs; newer source/document/roster authority produces an actionable conflict.
4. Human and versioned JSON results report imported, replaced, skipped or failed consistently; a partially successful batch retains successes and exposes failures.
5. Core controls provide equivalent desktop behavior, accessible labels, explicit mutation effects, empty/loading/conflict states and no repeated confirmation after concrete elections.

### Edge Cases

Audio-only targets, absent/empty/current transcripts, absent/empty rosters, same bytes but distinct import identity, equal-duration unrelated media, changed grouped track layouts, unknown versus zero audio duration, rational positive/negative offsets, non-UUID local voices, repeated/colliding aliases, inactive identities, duplicate memberships, stale source/document/roster revisions, workspace mismatch, retry after acceptance response loss, shared artifacts, active playback/model leases, cancellation during staging, partial batches and portable restore.

## Requirements

### Functional Requirements

- **FR-001**: Preserve the implemented media/transcript grammar and origin/date, credential, attribution, embedded-selection, preset, result and pagination contracts; expose explicit audio replacement and known-speaker references consistently.
- **FR-002**: Resolve an existing media target unambiguously without creating a second recording. Without explicit audio replacement, return a successful skip before candidate acquisition and preserve all accepted state.
- **FR-003**: Replace audio only on an explicit election while preserving stable recording identity and approved canonical codec/container/sample/track/time policy.
- **FR-004**: Existing transcript handling is explicit keep/clear/replace. Keep requires an affirmative applicability assertion and checks relevant known audio bounds; missing assertion or incompatible known intervals fails. Unknown bounds remain diagnosed uncertainty, never assumed equality or zero.
- **FR-005**: Replace requires a validated selected supplied or embedded document and preserves exact supported Cue JSON attribution/native observations. Invalid explicit input never falls back. Clear leaves no current document at admission; processing runs only through a separately configured/elected stage.
- **FR-006**: Freeze target identity, observed source/document/roster revisions and resolved centralized identities at enqueue/election. Publication is fenced by current revisions and live work authority; retries do not retarget aliases or inherit newer accepted state.
- **FR-007**: Publish verified candidate audio, current document policy and elected roster effects in one atomic current transaction, with durable per-item acceptance receipts and reconciled accepted replay.
- **FR-008**: Source replacement invalidates source-dependent current evidence, automated/local mappings, segment/corpus preparations and stale graph projections; reconcile active processing/training authority before it can publish against superseded data. Preserved documents do not imply acoustic mappings remain valid.
- **FR-009**: Retire obsolete managed artifacts only after acceptance and reference/lease checks. Failures retain previous accepted objects; owner-owned inputs, shared current objects and active leases remain intact. Bound scratch and cancellation through existing acquisition/conversion policies.
- **FR-010**: Store a revisioned intentional roster header and unique recording-to-centralized-speaker membership independently of Cue JSON or processed-recording existence. Reuse stable recording and speaker identity contracts without duplicated names, profiles or assignment arrays.
- **FR-011**: Expose roster read/add/remove/replace/clear with idempotent membership semantics and optimistic concurrency. Distinguish undeclared and explicitly empty rosters. Membership maximum is 1,000 speakers, matching existing context bounds; selectors use the existing 512 UTF-8 byte identity-text bound. Preserve existing ingress and pagination contracts.
- **FR-012**: Resolve references by stable IDs or unique existing aliases under workspace authority. Diagnose inactive/unresolved identities and follow existing lifecycle rules; do not invent unimplemented identity merge/split behavior.
- **FR-013**: Roster membership is intentional context, not attribution or training evidence. Transcript/mapping edits preserve membership; roster edits preserve accepted mappings. No automatic roster expansion, acoustic matching, profile generation or speaker-count inference occurs in this slice.
- **FR-014**: Initial media import accepts repeated speaker references and versioned manifest equivalents, atomically publishing membership with the stable admitted audio-only/annotated recording. Failed imports publish no orphan roster state.
- **FR-015**: Audio replacement with a declared roster requires explicit retain/clear; transcript-only replacement preserves it. Reject contradictory initial membership and replacement-roster elections rather than silently mutating declared context.
- **FR-016**: Include roster records in portable catalog snapshots/restore, SQLite/PostgreSQL operations and applicable LadybugDB/ArcadeDB projection while preserving catalog authority and filesystem/S3 artifact parity.
- **FR-017**: Shared runtime, CLI/help, schemas/manifests, human/JSON receipts and desktop core controls communicate the same elections and failures. Concrete configured elections run without mandatory attended confirmations.
- **FR-018**: Tests use deterministic orchestration and bounded real audio/document fixtures. Required checks never initialize/download/load/infer with acoustic models, and CI jobs retain the ten-minute budget.
- **FR-019**: Amend authoritative docs, constitution, schema/examples, help, glossary and changelog together with synchronized release identity. Remove hypothetical pre-v1 bulk audio migration as a delivery requirement; normal catalog/schema evolution and transcript-version interoperability remain separate concerns.
- **FR-020**: Complete all #30/#33 acceptance criteria, including their existing S010 foundations. Keep #35 open for acoustic matching and #34/#15/#14 for model references, training and release delivery.

### Key Entities

- **Recording replacement election**: Stable target, observed authority, selected audio, transcript policy/applicability assertion and roster policy, frozen before work.
- **Declared roster**: Recording-local revision authority and unique references to centralized speaker identities; independent of acoustic evidence.
- **Replacement candidate**: Verified canonical media, source metadata/time relationships and validated document candidate before current acceptance.
- **Current acceptance receipt**: Stable IDs, accepted revisions and imported/replaced/skipped/failed outcome, permitting recovery without source locators.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every supported import/target/transcript/roster election has consistent behavior through the exposed interfaces and documented operation matrix.
- **SC-002**: Every tested failed, cancelled or stale replacement preserves the prior accepted recording; accepted replay returns the same authority and outcome.
- **SC-003**: All tested audio replacements keep recording identity and required track/channel/time facts while preventing publication from superseded source evidence.
- **SC-004**: Audio-only and annotated recordings support portable revisioned rosters with no duplicated names/profiles/assignment arrays and no membership-derived acoustic claims.
- **SC-005**: Required checks pass without model execution and final published head has green CI and no unresolved review findings within the two-round limit.

## Clarifications

### Session 2026-10-09

- Transcript keep is a configured applicability assertion plus known-bound validation, not automatic alignment inferred from equal duration.
- Explicit roster retain/clear applies to declared empty and nonempty rosters; undeclared targets need no roster policy unless membership is supplied.
- Cleared transcripts remain absent at admission; later elected processing is a separate durable phase.
- Without a prior document, replacement defaults to selected-document replace when a transcript is supplied/discovered, otherwise clear. Initial known-speaker lists apply to new media; existing targets use roster editing plus explicit replacement retain/clear.
- Owner deployment context removes pre-v1 bulk audio migration. Historical fixtures and format interoperability do not establish deployed user data.
- Roster acoustic matching, model acquisition/aliases, training and release promotion remain outside S011; shared identity and current-reference behavior is included.

## Assumptions and Scope

Existing stable recording, centralized identity, artifact, durable work, current-document and graph contracts are the foundation. No independent storage subsystem or alternate transcript authority is introduced. No real user audio migration exists. Completion targets are #30/#33; #35's declared-roster portion is delivered with precise remaining acoustic scope. Push and official PR are explicitly authorized; merge/tag/release are not.
