# Feature Specification: S006 Configurable pipelines, speakers and terminology

**Feature Branch**: `codex/006-configurable-speaker-pipelines`

**Created**: 2026-10-07

**Status**: Specification review

**Input**: Owner-authorized S006 combines GitHub #8 and #9. Their S008/S009 titles are roadmap labels; the actual slice, directory and branch use 006. Automatically push and open an official PR, address every review within at most two rounds, then hand off before merge.

## User Scenarios & Testing

### User Story 1 - Configure and run selected processing (Priority: P1)

A maintainer saves a readable Local, Connected or Custom pipeline, inspects capabilities and effective stage settings, and processes a recording through selected models/routes. Credentials use existing encrypted setup. Each stage can rerun independently without item approval.

**Why this priority**: Routing and effective settings are prerequisites for recognition context and repeatable corrections.

**Independent Test**: Save/inspect each preset, execute deterministic local/hosted adapters, override one stage, and verify the accepted result and elected route.

**Acceptance Scenarios**:

1. **Given** an elected pipeline, **When** a stage runs, **Then** its adapter/model/endpoint/options are explicit and only that route executes.
2. **Given** incompatible capabilities or provider failure, **When** processing is attempted, **Then** an operation-specific error occurs without provider/device fallback.
3. **Given** a current result, **When** one stage reruns, **Then** compatible unaffected evidence is reused and fenced replacement retires obsolete outputs/references.
4. **Given** a definition changes after submission, **When** work executes, **Then** elected effective settings remain bound to the job.

### User Story 2 - Maintain speaker continuity (Priority: P1)

A maintainer gives a known person names/aliases, maps recording-local voices, corrects mappings and selects that speaker's current audio across the library. Equal names do not merge people automatically.

**Why this priority**: Known-person identity must stay distinct from acoustic voices and subtitle observations.

**Independent Test**: Map different recording-local UUIDs to one person, correct a mapping and select current evidence without changing either embedded document.

**Acceptance Scenarios**:

1. **Given** two recordings, **When** voices map to one person, **Then** recording-local UUIDs remain separate while the external mapping provides continuity.
2. **Given** a correction, **When** accepted, **Then** subtitle bytes/assignments stay unchanged, dependent preparation is invalidated and selection reflects current mapping.
3. **Given** current and superseded evidence, **When** selecting speaker audio, **Then** only current source/cue/local-voice references contribute without copied assignments.
4. **Given** overlap or untimed participation, **When** selecting evidence, **Then** overlap is explicit and no clip interval is invented.

### User Story 3 - Apply specialized recognition context (Priority: P2)

A maintainer manages active terms, variants, languages/context and optional speaker links. Relevant names/aliases join terms as deterministic recognition hints. A configured vocabulary rerun replaces current text.

**Why this priority**: Context assists recognition without becoming evidence of who spoke.

**Independent Test**: Compile hints within a declared budget, pass them to an adapter fixture and inspect effective context provenance and current replacement.

**Acceptance Scenarios**:

1. **Given** duplicate/inactive/scoped variants, **When** context compiles, **Then** ordering/inclusion are deterministic and omissions are diagnosed.
2. **Given** a context budget, **When** exceeded, **Then** deliberate truncation and the effective context digest are reported.
3. **Given** concurrent edits, **When** elected work runs, **Then** its resolved context is used without changing inputs midway.

### User Story 4 - Inspect automatic quality diagnostics (Priority: P2)

A maintainer configures automatic diarization and speaker-correlation diagnostics, inspects results and runs configured work without a sample-review queue.

**Why this priority**: Quality measurements and document correctness are separate facts.

**Independent Test**: Deterministic boundary/overlap/empty/mapping scenarios prove defaults, thresholds, optional disabling and mandatory document validation.

**Acceptance Scenarios**:

1. **Given** defaults, **When** assembling results, **Then** automatic quality diagnostics are enabled and separate from document validation.
2. **Given** changed diagnostic settings, **When** processing runs, **Then** effective settings are recorded and schema/timing validation remains mandatory.

### Edge Cases

- Missing credentials, malformed capabilities, HTTP errors/redirects, cancellation, excessive response sizes and sensitive provider errors.
- Unsupported hints/models, exhausted budgets, empty selection, duplicate names/aliases, inactive/scoped terms and concurrent revisions.
- Source/document replacement during selection or work, removed local UUIDs, overlap, unknown speakers, untimed participation and no usable speech.
- Historical migration/snapshot restore must not revive obsolete assignments; supported catalog/storage alternatives remain equivalent.

## Requirements

### Functional Requirements

- **FR-001**: Persistent readable Local, Connected and Custom definitions support inspection, revision-aware updates and capability checks.
- **FR-002**: Resolve selected definition plus explicit stage overrides into effective adapter/model/endpoint/device/options/limits.
- **FR-003**: Implement usable local and configured hosted recognition/diarization adapters with declared protocol/capabilities and no silent fallback.
- **FR-004**: Reuse encrypted credential setup; persist references only and keep resolved values out of definitions/jobs/status/errors/receipts.
- **FR-005**: Bind work to effective configuration and context; later defaults cannot change in-flight inputs.
- **FR-006**: Preserve independent reruns, fenced current replacement, physical retirement, reference reconciliation and recovery.
- **FR-007**: Manage known-speaker names/aliases and explicit recording-local mapping corrections without rewriting embedded documents.
- **FR-008**: Select library-wide speaker audio through current references and exact source timing, deduplicating repeated evidence and reporting overlap/untimed participation.
- **FR-009**: Manage revisioned active terms, spelling, variants, language/context and explicit optional speaker/alias links.
- **FR-010**: Compile deterministic deduplicated hints within adapter budgets, diagnose omissions/unsupported hints, record effective digest and pass hints to recognition.
- **FR-011**: Configurable automatic diarization/correlation diagnostics default on, remain separate from document validity and add no manual gate.
- **FR-012**: Mapping/result changes invalidate dependent corpus/preparation; completed models retain noncontent lineage only.
- **FR-013**: Shared CLI/runtime operations and desktop bridge implement the behavior; dedicated GUI controls remain #10.
- **FR-014**: Preserve SQLite/PostgreSQL and filesystem/S3 parity, bounded operations, migrations and recoverable snapshots.
- **FR-015**: CI/required checks never invoke inference engines, initialize/load models, download weights or require engine results. Deterministic adapters, committed licensed clips, decoding and Cueson are permitted.
- **FR-016**: Complete Spec Kit analysis before implementation, reconcile documentation/contracts and resolve every received review within the two-round cap before handoff.

### Key Entities

- **Pipeline definition**: Preset/revision/stage configuration, credential references and quality settings.
- **Effective processing selection**: Elected configuration/context identity/settings for one job, without credentials or copied assignments.
- **Known speaker and alias**: Stable person identity, names, language/scope and provenance.
- **Term**: Stable revisioned spelling/variants/language/context/active state and optional identity links.
- **Compiled context**: Effective hints, digest and omission diagnostics within capability budget.
- **Current speaker selection**: Current recording/document/cue/local UUID references resolving mapped original audio, without stored copied intervals.

## Success Criteria

### Measurable Outcomes

- **SC-001**: All three presets can be configured/inspected/used; incompatible capabilities fail before affected work starts.
- **SC-002**: Local/hosted deterministic paths prove elected routing, independent reruns, cancellation and no fallback.
- **SC-003**: One speaker can be selected across two recordings; correction changes selection without changing document digests.
- **SC-004**: Equivalent context yields identical hints/digest; unsupported/omitted hints are reported.
- **SC-005**: Concurrent-edit/replacement scenarios reject stale inputs and obsolete references.
- **SC-006**: Required checks remain inference-free and each hosted CI job stays below ten minutes.
- **SC-007**: Both target issues meet scope, affected checks pass and every review is addressed before merge handoff.

## Assumptions

- Reuse S005 current-document, secret, model, artifact and durable-work contracts.
- Existing media support deterministic acceptance; real inference is explicit optional maintainer work outside CI.
- Hosted endpoints are operator-selected. Local protocol fixtures do not authorize sending real library media to external services during development.
- Full desktop packages, graph projections, similarity/embedding engines and training stay in separate delivery contracts. Explicit corrections, aliases, current speaker selection and terms are included.

## Clarifications

### Session 2026-10-07

- Routing: configured hosted recognition/diarization use declared versioned adapter capabilities. A selected operation unsupported by an adapter fails; configuration inspection does not trigger inference or model loading.
- Concurrency: save effective pipeline/context settings at submission; saved definition changes affect future work. Current source/document fencing remains mandatory at acceptance.
- Quality: configurable measurements never replace schema/timing correctness or require manual sample approval.
- Scope: speaker continuity is explicit mappings, names/aliases, current corpus selection and correlation diagnostics. Similarity/embedding inference and dedicated GUI panels retain separate delivery boundaries.
- Verification: deterministic local hosted fixtures suffice for required acceptance; real inference and real external media submission are not CI or publication requirements.
