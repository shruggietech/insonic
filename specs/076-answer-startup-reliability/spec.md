# Feature Specification: Answer and startup contract reliability

**Feature Branch**: `codex/076-answer-startup-reliability`

**Created**: 2026-11-08

**Status**: Specified

**Input**: Execute S076 under autopilot, addressing issues [#254](https://github.com/shruggietech/insonic/issues/254) and [#255](https://github.com/shruggietech/insonic/issues/255), prepare v1.3.1, automatically push and open the official PR, satisfy all reviews and CI, and hand off for owner merge. At most one explicitly requested second review round is authorized.

## Clarifications

### Session 2026-11-08

- Autopilot decision: retain the issue's exact model election, frozen rubric and existing product deadlines; do not infer model quality from protocol fixtures.
- Autopilot decision: prepare v1.3.1 and one official PR, with final merge reserved to the owner. Explicit push/PR authorization supersedes the generic pre-push halt.
- Coverage scan: functional scope, identity/lifecycle, operator flow, bounds, security, integrations, failure cases, terminology and completion signals are clear. No critical ambiguity requires an owner question.

## User Scenarios & Testing

### User Story 1 - Obtain a correctly structured grounded answer (Priority: P1)

A user electing native local Ollama receives a complete typed answer contract and actionable bounded repair guidance. A structurally valid response remains subject to exact current source and quotation authority.

**Why this priority**: The reported model completed requests but the answer and repair used wrong citation and limitation types, preventing useful supported output.

**Independent Test**: Repeat the two-source Monday reproduction and all seven frozen evaluation cases with the reported exact model identity, retaining actual results separately from protocol fixtures.

**Acceptance Scenarios**:

1. **Given** two independent sources stating launch is Monday, **When** the reported elected model answers the launch question, **Then** the accepted answer uses the exact current version, source reference objects and string limitations and contains supported Monday evidence.
2. **Given** malformed citation, limitation or version output, **When** the optional single repair runs, **Then** it receives generated field/type expectations and the same source scope, route and enclosing deadline.
3. **Given** invented, withheld, stale or conflicting evidence, **When** a response is evaluated, **Then** unsupported conclusions cannot acquire supported status through type correction, and uncertainty remains explicit.
4. **Given** repair exhaustion, cancellation or transport failure, **When** work terminates, **Then** the existing insufficiency or classified failure behavior and immutable history remain intact.

### User Story 2 - Understand catalog incompatibility at cold startup (Priority: P2)

A user opening a newer unsupported catalog sees its version incompatibility and a useful recovery route without changing accepted catalog authority.

**Why this priority**: Generic startup unavailability hides a known cause even though explicit maintenance identifies it.

**Independent Test**: With exclusive ownership of a disposable catalog, retain accepted records, inject schema marker 999, force cold startup and explicit migration, and compare the marker and records after both refusals.

**Acceptance Scenarios**:

1. **Given** an unsupported newer catalog, **When** a cold client starts its runtime, **Then** the startup report and CLI retain `incompatible_version`, exit class 3, a catalog-specific generated reason and compatible-executable or verified-backup guidance.
2. **Given** that same catalog, **When** explicit maintenance is attempted, **Then** it refuses consistently and neither accepted data nor the version marker changes.
3. **Given** a valid catalog or an unrelated startup failure, **When** startup runs, **Then** valid startup works and unrelated failures retain classified redacted diagnostics.

### Edge Cases

- Missing, unknown, duplicate, mistyped or excessive answer fields; family strings substituted for reference objects; null/object limitations; changed package IDs and version strings.
- Accurate types with invented references, quotes outside visible extracts, conflicts or source changes between attempts.
- Local/hosted route distinctions, existing query/speaker/metadata assistance consumers, oversized requests/responses and a repair sharing the original timeout.
- Cold versus already owned runtimes, corrupted catalog access, invalid startup reports and no provider readiness implied by runtime readiness.

## Requirements

### Functional Requirements

- **FR-001**: Answer generation MUST receive a complete unambiguous output contract defining exact version/package identity, allowed states and claim/support values, citation reference-object fields, string limitations, optional fields and existing bounds.
- **FR-002**: Native local Ollama MUST receive a schema-constrained answer request; other configured adapters MUST retain their documented route/capability behavior and receive the complete contract without silent fallback.
- **FR-003**: A repair MUST receive bounded locally generated field/type/constraint guidance without echoing arbitrary provider values, credentials or additional owner data. Existing zero-or-one repair and shared provider deadline MUST remain unchanged.
- **FR-004**: Strict current source identity, visible quote authority, uncertainty, immutable history, atomic acceptance, cancellation and source-freshness fences MUST remain enforced independently of schema conformity.
- **FR-005**: Qualification MUST repeat the reported actual-provider reproduction and seven frozen cases, retain exact model/configuration/source/corpus identities and complete results, distinguish protocol validity from usefulness, and retain not-applicable metrics when there are zero applicable citations.
- **FR-006**: Known catalog-version incompatibility MUST survive cold runtime startup reporting and CLI envelopes with exit class 3 and bounded catalog-specific recovery guidance.
- **FR-007**: Unsupported catalog refusal MUST preserve accepted records and schema markers; valid controls, explicit maintenance and unrelated classified/redacted startup failures MUST remain covered.
- **FR-008**: The software, master schema and candidate documentation MUST agree on v1.3.1, preserve historical published contracts, and record changed behavior and compatibility effects in the changelog. No catalog schema migration is required solely for these fixes.
- **FR-009**: Delivery MUST complete Spec Kit analysis and convergence, applicable local and hosted checks, retained compact audited evidence, an official issue-linked PR and review resolution, with owner merge retained and no more than two review rounds.

### Key Entities

- **Answer output contract**: Version/package identity, claims, source citations, uncertainty and limitations supplied to the elected provider.
- **Repair guidance**: Bounded generated expectations associated with rejected output, distinct from source evidence and arbitrary provider text.
- **Startup condition**: Generated error code/reason transported through a correlated bounded runtime startup report to client output.
- **Qualification receipt**: Source/executable/model/corpus identity, complete observations, applicable metrics and separate timing phases.

## Success Criteria

### Measurable Outcomes

- **SC-001**: The actual-model two-source reproduction returns a validated supported Monday answer with zero occurrence of the three reported type/version faults; every frozen case has a retained scored outcome.
- **SC-002**: All malformed-output regression cases are rejected or repaired within the elected limit; zero invalid source references or quotes are accepted as supported.
- **SC-003**: Cold startup and explicit maintenance return incompatibility exit class 3 for the newer-catalog fixture, with identical accepted records and marker before/after; the valid-schema control opens successfully.
- **SC-004**: Required functional, integrity, provenance and elected checks pass, with qualification targeting 1800 seconds, bounded by 3600 seconds and the existing 60-second provisional finalization reserve; preparation and combined timing remain separately reported.
- **SC-005**: One official PR covers both issues, all received review comments are addressed, required CI is green, and the final handoff identifies actual limitations and the exact review head without merging.

## Assumptions

- Existing local model election includes the issue's exact qwen2.5:14b identity and synthetic corpus; verify readiness before inference and never substitute a model to manufacture acceptance.
- This slice prepares a patch candidate. Tagging, release publication and owner merge are separate from the explicitly authorized push and PR.
- Dedicated answer GUI controls, generic model tuning, increasing product deadlines and unrelated pipeline work remain outside this bounded patch.
