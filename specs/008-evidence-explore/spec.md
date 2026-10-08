# Feature Specification: S008 Source-backed evidence and Explore

**Feature Branch**: `codex/008-evidence-explore`

**Created**: 2026-10-08

**Status**: Specified

**Input**: Owner-authorized S008 targets #11 and #12. Deliver source-backed evidence and queries, then all-media timelines, saved definitions and accessible graph views. Push/publish an official PR, address every finding across at most two review rounds, and stop before owner merge. No release promotion is authorized.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Search current evidence (Priority: P1)

Publish current cues and elected assertion extraction, search words/speakers/time, and follow results to original sources on either supported backend.

**Why this priority**: Exploration must be grounded in current evidence.

**Independent Test**: Import licensed fixtures, assemble supplied cues/turns, map a speaker, publish/query both backends without inference.

**Acceptance Scenarios**:

1. **Given** current cues and an elected extractor, **When** bounded overlapping complete-cue windows are processed, **Then** accepted assertions cite actual cues, exact intervals and provenance; invalid output is diagnosed while text search remains available.
2. **Given** pending publication, **When** interrupted/replayed/taken over, **Then** atomic ordered facts/receipts/checkpoints advance, replay is harmless and stale owners cannot commit.
3. **Given** matching backend fixtures, **When** portable word/speaker/media/time/evidence operations paginate, **Then** ordered results, identities and freshness match.
4. **Given** native text and typed parameters, **When** dialect/read-only validation runs, **Then** supported exploration executes or incompatibility returns without modifying evidence.

### User Story 2 - Explore dates and original clocks (Priority: P1)

Browse the whole library by origination date, retain undated/approximate entries, open recording-relative evidence and play original intervals.

**Why this priority**: S007 playback becomes discoverable across the library.

**Independent Test**: Populate multiple pages of dated/approximate/undated media; filter/navigate by keyboard and seek current evidence.

**Acceptance Scenarios**:

1. **Given** more entries than a page, **When** a calendar is filtered, **Then** all matching media remain reachable with visible uncertainty and no substituted import/file dates.
2. **Given** current cues/voices/mappings/segments/assertions, **When** a recording opens, **Then** original clocks remain distinct from calendar dates and playback uses current-document bindings.
3. **Given** date correction/result replacement, **When** refreshed, **Then** dates change without rewriting cue clocks and obsolete references disappear without alternative timelines.

### User Story 3 - Save queries and inspect graph views (Priority: P2)

Save/revise/rerun queries against current evidence; inspect force-directed relationships or an accessible table and persist layouts separately.

**Why this priority**: Reusable investigation must not copy prior processing results.

**Independent Test**: Save/edit/reopen portable/native definitions, switch backend compatibility, inspect provenance and expand bounded graph/table results.

**Acceptance Scenarios**:

1. **Given** a valid query, **When** saved/edited with expected revision, **Then** immutable versions survive restart, conflicts fail and no transcript/assignment copies are saved.
2. **Given** an incompatible native definition, **When** reopened, **Then** it stays inspectable with an explanation and supports a compatible revision without unsupported execution.
3. **Given** results beyond view limits, **When** rendered, **Then** omitted counts/expansion, labels/provenance and keyboard table are available; coordinates never change facts.

### Edge Cases

- Empty libraries, untimed cues and unknown voices return explicit useful results.
- Negation, modality, quotes and conditions remain source statements rather than established truth.
- Timeout/rejected assertion/graph outage/uncertain commit have observable outcomes without fallback or approval queues.
- Replacement during extraction/publication/pagination/playback cannot expose stale evidence.
- Overlap deduplicates by source identity; multi-speaker cues never create copied assignment stores.
- Quoted/commented native text, parameters and oversized input cannot bypass read-only/workspace boundaries.
- Date precision/zones/approximate bounds, concurrent edits and incompatible definitions remain explicit.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Chunk current complete cues in bounded overlapping windows with speaker transitions and exact source references without modifying text/clocks.
- **FR-002**: Elected extractors return subject/relation/object, polarity/modality/conditions and supporting cues; validate shape/ownership/time bounds before automatic acceptance, leaving transcript search usable on failure.
- **FR-003**: Preserve source/quoted attribution and extractor/tool/settings provenance; deduplicate overlap without collapsing differing conditions/negation.
- **FR-004**: Both backends project current media/assets/raw metadata/date selections, documents/cues, speakers/mappings, terms, assertions, segments and existing model lineage using stable IDs; forbid copied assignments and superseded results.
- **FR-005**: Publication supports ordered recoverable events, fencing, atomic fact/receipt/checkpoint, exact replay, correction/rebuild and unknown-outcome reconciliation with disclosed freshness.
- **FR-006**: Portable media/text/word/term/speaker/time/model/evidence/graph operations use typed filters, deterministic ordering/pagination and equivalent behavior on both backends.
- **FR-007**: Native queries declare dialect/typed parameters, enforce read-only exploration, reject unsupported features actionably and expose capabilities/explanation/freshness.
- **FR-008**: Shared CLI/runtime contracts expose extraction, publication/status/rebuild, queries/timelines, saved definitions and layouts; GUI wraps them.
- **FR-009**: Calendar covers all media with selected origination precision/zones/approximate bounds/undated groups, filtering, zoom/pan and keyboard navigation independent of Library pagination.
- **FR-010**: Recording-relative views show current cues/voices/mappings/segments/assertions on original clocks with source playback/provenance.
- **FR-011**: Saved queries retain immutable typed portable/native versions and schema/backend/capability validation with CAS; incompatible definitions stay inspectable; separate layouts contain no prior-result snapshots.
- **FR-012**: Force-directed views include labels/filtering/provenance/truncation/expansion and an equivalent accessible table.
- **FR-013**: Document/mapping replacement refreshes or invalidates assertions/relationships/results/timelines/playback; date correction never changes original clocks.
- **FR-014**: Preserve workspace isolation, opaque credentials, parameterized values, bounded requests and hidden noninteractive Windows execution.
- **FR-015**: Required checks cover deterministic supplied cues, real pinned graph operations, replay/fencing/correction/parity, CLI/runtime/UI/accessibility/package integration; never run transcription/diarization, initialize/load models or download weights; hosted jobs stay below ten minutes.
- **FR-016**: Contracts/docs/version 0.0.0 remain aligned/self-contained; reconcile conflicting pinned-result prose with #21; tracking evidence stays internal.

### Key Entities *(include if feature involves data)*

- **Evidence reference**: Current document/cue, original source/interval and current mapping identity.
- **Assertion**: Validated proposition/evidence and elected provenance, representing a source statement.
- **Projection event/checkpoint**: Ordered catalog identity/digest and fenced atomic graph receipt.
- **Query definition/result**: Portable operation/native dialect, typed input, current evidence and freshness.
- **Saved query revision**: Immutable definition/compatibility with optimistic revision.
- **Timeline entry**: Origination uncertainty or original-clock evidence.
- **Graph view**: Current nodes/relationships, omitted counts and separate layouts.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Every assertion/result resolves current evidence; replacement leaves zero obsolete cue/assignment references available.
- **SC-002**: Common operation identities/order/values/pagination match on both backends including correction/recovery.
- **SC-003**: More than one Library page is fully discoverable by calendar/undated/recording navigation with keyboard playback and graph/table inspection.
- **SC-004**: Saved definitions/layouts survive restart, reject concurrent conflicting edits and contain zero copied processing documents/assignments.
- **SC-005**: All required repository/native/backend/UI/docs checks pass with no model execution/initialization/weights and each hosted job below ten minutes.

## Assumptions

- Reuse S007 desktop/playback, #21 current Cue JSON/mappings, catalog/outbox and licensed fixtures; graph production adapters/Explore are new.
- Common parity does not imply arbitrary native dialect compatibility.
- Extraction uses only explicitly elected adapters/routes; deterministic responses qualify orchestration without required inference.
- #13 assistance, #14 release/full portability and #15 training engines follow later; existing lineage remains queryable.
- Explicit push/PR authorization fulfills pre-push authorization. Owner retains merge.

## Clarifications

### Session 2026-10-08

- Q: What does saving preserve? A: Immutable portable/native definitions and separate layouts; reruns resolve current evidence, never obsolete transcript/assignment snapshots (#12/#21).
- Q: Does extraction/publication failure block text exploration? A: No; source search stays usable and freshness discloses state.
- Q: What is native parity? A: Common operations are portable; native text validates the selected dialect.
- Q: Does CI run inference? A: No. Deterministic responses qualify extraction/current-cue orchestration.