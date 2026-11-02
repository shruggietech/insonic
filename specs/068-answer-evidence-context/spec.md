# Feature Specification: S068 Answer evidence and context

**Feature Branch**: `codex/068-answer-evidence-context`

**Created**: 2026-11-01

**Status**: Specified

**Input**: Owner-authorized S068, existing issues #220, #221, #222 and #223 under milestone 48. No new tracking issue. Push and PR creation authorized; at most two external review rounds; owner retains final merge.

## User Scenarios & Testing

### User Story 1 - Inspect current evidence for a question (Priority: P1)

A user scopes a question to their library and obtains an inspectable evidence package spanning relevant recordings, with exact current source references and visible limitations.

**Why this priority**: Every later answer needs resolvable evidence and trustworthy source boundaries.

**Independent Test**: Retrieve a known multi-recording fixture with text, concept, entity, speaker and time constraints; inspect the selected cues, connecting relationships, provenance and omitted evidence.

**Acceptance Scenarios**:

1. **Given** relevant statements in multiple recordings, **When** bounded retrieval runs, **Then** the package includes the qualifying spans, selection paths and exact current document/cue identities within every elected budget.
2. **Given** stale graph projections, missing documents or untimed cues, **When** retrieval resolves candidates, **Then** stale evidence is excluded or diagnosed and unknown timing remains unknown.
3. **Given** cancellation or exhausted budgets, **When** work stops, **Then** its partial or cancelled state and continuation/omission information are explicit and it is not represented as a completed answer.

### User Story 2 - Distinguish independent support from repeated evidence (Priority: P1)

A user can inspect why evidence was ranked, grouped or excluded without repeated imports and derivative assertions inflating corroboration.

**Why this priority**: Confidence must follow provenance rather than repeated text.

**Independent Test**: Compare exact duplicates, known alternate encodings, overlapping cues, quoted speech, independent agreement and disagreement in a fixed evidence snapshot.

**Acceptance Scenarios**:

1. **Given** repeated imports or overlapping derivative assertions, **When** ranking runs, **Then** shared source families count once for independent support and the reasons remain inspectable.
2. **Given** similar wording from independent sources or conflicting statements, **When** ranking runs, **Then** the distinct sources and contradictions remain visible; text similarity alone never proves common origin.

### User Story 3 - Prepare bounded, reusable context (Priority: P2)

A user prepares a source-linked context package for an elected provider budget and can inspect its hierarchy, cache reuse and omissions.

**Why this priority**: Long libraries cannot be pasted into a provider prompt without loss of control or traceability.

**Independent Test**: Build cue, section, recording and cross-recording layers at small and large budgets, repeat unchanged requests, and replace a source or configuration before repeating.

**Acceptance Scenarios**:

1. **Given** more evidence than fits, **When** context is assembled, **Then** output stays within its declared token and byte budgets and identifies omitted or condensed evidence with paths to underlying cues.
2. **Given** a prior cached result, **When** source, speaker/date identity, scope, model/provider, prompt contract or configuration changes, **Then** incompatible cache entries are not reused.
3. **Given** corrupt cached content or a condensation that conflicts with source text, **When** it is read, **Then** the package diagnoses or rejects it; condensed text never becomes a new independent source.

### User Story 4 - Preserve citation and persistence authority (Priority: P2)

Users and later answer generation share a versioned question, evidence, claim and citation contract with inspect/export and invalidation semantics.

**Independent Test**: Round-trip supported, unsupported, conflicting, partial and cancelled result envelopes, including large identifiers and absent source timing; mutate source authority and inspect freshness.

**Acceptance Scenarios**:

1. **Given** a claim envelope, **When** it is validated, **Then** every supported claim resolves to current evidence or is rejected; unsupported and uncertain claims are explicitly marked.
2. **Given** a saved evidence/context package, **When** authoritative source identity changes, **Then** inspection reports stale authority and export retains the historical identity without pretending it is current.

### Edge Cases

Empty libraries and scopes; missing projections; disconnected relationships; graph lag; incomplete pagination; duplicate/overlapping sources; unknown lineage; conflicting dates and speakers; identifiers above JavaScript's exact-integer range; Unicode budget boundaries; corrupt caches; concurrent source changes; cancellation; credentials and private original paths; unavailable configured providers.

## Requirements

### Functional Requirements

- **FR-001**: Define versioned question scope, filters, retrieval/context budgets, elected provider configuration, evidence sets, answer claims/citations and partial/unsupported/conflicting/cancelled outcomes (#220).
- **FR-002**: Bind every evidence package and citation to current recording/document/cue identity, catalog/graph freshness, exact known source times and effective query/model configuration; preserve unknown values (#220, #221).
- **FR-003**: Retrieve and expand evidence across recordings using bounded selection, neighbor cues and relationship traversal, with pagination, selection paths and one aggregate work budget; do not silently scan the whole library without bounds (#221).
- **FR-004**: Preserve equivalent normalized behavior across supported backends, with direct query/search remaining available independently (#221).
- **FR-005**: Group proven common source families and overlapping evidence without using text equality as proof of lineage; keep unknown independence explicit (#222).
- **FR-006**: Rank reproducibly with inspectable relevance, independence, coverage and contradiction components, retaining independent corroboration and conflicting evidence (#222).
- **FR-007**: Assemble cue/section/recording/cross-recording context within explicit token and byte budgets, report truncation/omission and retain underlying source references (#223).
- **FR-008**: Cache derived context only for matching evidence, scope, source/document/speaker/date, model/provider/configuration and summary-contract identity; detect corruption and prohibit stronger or independent authority for summaries (#223).
- **FR-009**: Persist, inspect and export packages with explicit freshness/invalidation, cancellation and incomplete-result semantics; accepted transcripts, mappings and original inputs remain unchanged (#220).
- **FR-010**: Use configured unattended execution, existing routing/credentials and backend authority; prevent credential/private-location disclosure and cross-workspace evidence or cache reuse. Update authoritative contracts/help/docs and meaningful tests with implementation (#220-#223).

### Key Entities

- Question request: question, scope/filters, elected configuration and explicit budgets.
- Evidence package: current evidence spans, source families, relationships, ranking components, exclusions, freshness and effective configuration.
- Citation/claim: resolvable package member reference, exact known source range, faithful source text, support state and uncertainty.
- Context package: bounded hierarchical layers and source links, byte/token accounting, limitations and cache identity.
- Saved package: immutable historical result plus current validity, request identity and inspect/export semantics.

## Success Criteria

- **SC-001**: Every selected citation in the controlled multi-recording fixtures resolves to its exact current authority; forged/stale references cannot pass as current.
- **SC-002**: Every retrieval/context fixture stays within its declared aggregate and output budgets, including Unicode and repeated expansion, with visible omissions.
- **SC-003**: Duplicate and overlapping inputs never increase independent-source counts; independent and conflicting sources remain inspectable with reproducible ordering.
- **SC-004**: All incompatible or corrupt cache fixtures prevent reuse; unchanged requests demonstrate valid reuse without new source authority.
- **SC-005**: Supported-backend, restart/export/freshness and cancellation fixtures pass with no source mutation, credential leakage or cross-workspace reuse.

## Assumptions and scope boundaries

This slice delivers evidence/context preparation and the shared future answer envelope. Answer synthesis and provider citation repair (#224), complete durable answer-generation workflows (#225), and milestone-wide model evaluation/release qualification (#226) remain their existing issues. Preparation must be usable and inspectable through shared runtime/CLI contracts; this boundary does not defer any acceptance in #220-#223. Provider-based free-form paraphrasing is not required to satisfy context condensation when deterministic source-preserving condensation meets the contract. Real model quality claims require separately elected inference evidence. GUI-specific controls may follow with the documented CLI-first exception. No release publication, new tracking issue or owner merge is included.

## Clarifications

### Session 2026-11-01

Autopilot coverage scan: all functional, domain, interaction, quality, dependency, failure, constraint, terminology and completion categories are clear. No user questions required. Extractive, source-preserving context condensation is selected over introducing free-form synthesis before #224; cache keys still bind elected model/provider/configuration and contract identity. Supported backend parity and existing routing remain required.
