# Feature Specification: Contextual speaker discovery and resolution

**Feature Branch**: `codex/072-contextual-speaker-discovery`

**Created**: 2026-11-02

**Status**: Specified

**Input**: Owner-authorized S072 covering existing issues #227, #228 and #229, with Spec Kit/autopilot, automatic PR publication, at most two review rounds and owner-controlled final merge. No new tracking issue.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Inspect grounded identity hypotheses (Priority: P1)

A user can inspect why a recording-local voice might correspond to a known person, without confusing a mentioned name, a roster membership, a prior assignment or an acoustic prediction with independent identification evidence.

**Why this priority**: Extraction and resolution require one current, explainable evidence contract.

**Independent Test**: Persist and inspect source-linked hypotheses for an existing recording, then change the source or an owner correction and verify that stale support cannot be used.

**Acceptance Scenarios**:

1. **Given** introductions, quoted names and roster context, **When** evidence is represented, **Then** each observation retains its distinct evidence class, current source/cue/local-voice references, provenance and uncertainty.
2. **Given** equal names or repeated observations, **When** hypotheses are inspected, **Then** people are not merged by spelling and duplicate or dependent support is not counted as independent.
3. **Given** an earlier acoustic prediction, **When** its derived evidence is reused, **Then** its lineage remains dependent and it cannot confirm itself or claim independently grounded enrollment eligibility.
4. **Given** a corrected mapping or replaced source/document, **When** prior hypotheses are inspected or consumed, **Then** stale content is rejected or invalidated without rewriting accepted source text or restoring an obsolete assignment.

### User Story 2 - Discover candidates from elected context (Priority: P1)

A user elects a configured provider to extract evidence-cited speaker identity candidates from the current transcript, local voice observations and selected roster/context. Unknown identities remain usable.

**Why this priority**: This supplies useful contextual candidates that the current acoustic/manual workflow lacks.

**Independent Test**: Run discovery through the shared application interface with a deterministic structured provider and inspect the resulting supported, quoted, uncertain and rejected observations.

**Acceptance Scenarios**:

1. **Given** a self-introduction and relevant elected context, **When** discovery runs, **Then** it proposes a source-linked candidate while preserving alternatives and abstention.
2. **Given** third-person mentions, quotations, role changes or duplicate names, **When** discovery runs, **Then** a mention is not automatically assigned to the current speaking voice and ambiguities remain explicit.
3. **Given** malformed, foreign or fabricated references, **When** a provider returns candidates, **Then** invalid output fails before publication; source-controlled identities and intervals remain authoritative.
4. **Given** configured routing, budgets and credentials, **When** discovery succeeds, fails or is cancelled, **Then** only elected data reaches the elected provider, no per-voice approval is added and diagnostics preserve accepted mappings/profiles.

### User Story 3 - Resolve and explain candidates (Priority: P1)

A user obtains reproducible identity decisions combining contextual candidates, canonical identities/aliases and available independently grounded acoustic evidence, with reasons for ambiguity, conflict or abstention.

**Why this priority**: Candidate extraction becomes an end-to-end useful discovery workflow only when users can inspect a defensible resolution.

**Independent Test**: Resolve fixed evidence/configuration repeatedly and compare decisions and explanations, then inject contradiction, circular lineage and concurrent correction cases.

**Acceptance Scenarios**:

1. **Given** genuine independent support and applicable identity context, **When** resolution runs, **Then** it reports matched, supported-contextual, ambiguous, conflicting or unknown outcomes with evidence classes and alternatives.
2. **Given** contradictions, close acoustic alternatives or repeated/dependent support, **When** resolution runs, **Then** uncertainty remains visible and repetition cannot inflate a decision or turn context into acoustic verification.
3. **Given** fixed current evidence and decision rules, **When** resolution is repeated, **Then** outcomes and reasons are reproducible; newer owner corrections supersede dependent hypotheses.
4. **Given** completed discovery and resolution, **When** the user reads human or machine output, **Then** both distinguish observations, contextual support and acoustic matches, with actionable failure/staleness diagnostics and no implicit mapping or profile mutation.

### Edge Cases

- Empty or missing transcript/voice/context; unknown people; multilingual aliases; several people sharing a name; multiple voices in one cue; untimed participation.
- Self-introductions versus quoted or third-person statements, changing roles, contradictory rosters and partial introductions.
- Forged recording/cue/voice/person references, provider-generated times, oversized output, prompt-like source text and unauthorized provider routing.
- Duplicate recordings/cues, circular model lineage, derived predictions presented as corroboration and missing acoustic provenance.
- Source/document/audio/roster/alias/profile/mapping changes during extraction or resolution; repeated requests, lost responses, cancellation and unavailable providers.
- Workspace isolation, both supported catalog backends and existing storage/graph alternatives; portable restoration must not resurrect stale evidence.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Represent versioned contextual hypotheses separately from recording-local voices, known-person mappings and acoustic predictions, retaining current source/cue/voice identity, names/aliases, uncertainty, alternatives and contradictions (#227).
- **FR-002**: Distinguish self-identification, roster/context association, mentioned/quoted people, independent owner mappings and acoustic predictions; record provenance and independence relative to the prediction being evaluated (#227).
- **FR-003**: Allow automatically supported contextual evidence under an explicit independence contract without requiring individual manual approval; prediction-derived or circular support cannot certify itself (#227, #229).
- **FR-004**: Extract bounded structured candidates through a replaceable elected provider using only configured current transcript, local voice and elected roster/context; preserve existing credential protection, routing and aggregate budget/cancellation behavior (#228).
- **FR-005**: Validate every returned reference and citation against admitted current evidence; provider-created identities/times cannot override source authority. Reject foreign, malformed, oversized or stale results before publication (#228).
- **FR-006**: Preserve abstention, alternatives, multilingual/alias observations and relational context. A third-person mention or quoted introduction cannot by itself identify the current voice (#228).
- **FR-007**: Compare candidates with explicit canonical identities/aliases, roster facts and eligible independently grounded acoustic evidence without merging people solely by spelling or equating evidence strength (#229).
- **FR-008**: Produce reproducible matched, supported-contextual, ambiguous, conflicting and unknown decisions under explicit configurable rules, retaining explanations, competing candidates and provenance (#229).
- **FR-009**: Deduplicate source support and reject circular/dependent corroboration. Contradictory context and close acoustic alternatives remain explicit; confidence alone is not identity proof (#227, #229).
- **FR-010**: Respect current authority and owner corrections at publication and consumption. Source/document/mapping/context changes invalidate dependent hypotheses and prevent stale concurrent results from superseding current state (#227, #229).
- **FR-011**: Provide a usable shared-runtime CLI discovery, evidence inspection and resolution path with complete versioned machine output and concise human summaries; keep desktop shared-operation access aligned and document any advanced-control lag.
- **FR-012**: Preserve accepted mappings, profiles, transcripts and source media on success, retry, failure or cancellation. This slice creates discovery/decision evidence; application/retraction and enrollment remain with #230-#232.
- **FR-013**: Maintain workspace isolation, repeatable request behavior, supported backend parity and current-reference recovery/portable-state integrity for new persisted evidence. No implicit external research or provider fallback.
- **FR-014**: Update affected contracts, schemas, CLI help, authoritative documentation and changelog with meaningful deterministic positive/negative acceptance coverage. Keep protocol correctness separate from any unelected model/acoustic-quality claim.
- **FR-015**: Deliver one coherent S072 PR closing #227-#229, with no new tracking issue, green applicable checks, every review finding addressed within at most two rounds, retained evidence and owner-controlled final merge. No release or unrelated feature change is included.

### Key Entities

- **Current evidence snapshot**: Exact recording, document, cue, local voice, mapping, roster and identity/model authority elected for discovery.
- **Identity observation**: Evidence class, current references, asserted name/role/relationship, source provenance and independence lineage.
- **Hypothesis**: A possible identity for a local voice with alternatives, uncertainty and supporting/contradicting observations.
- **Discovery election**: Provider/configuration, admitted data scope, limits and repeatable request identity.
- **Resolution**: Configured decision rules, current inputs, outcome, alternatives and explainable reasons.
- **Invalidation**: A current-authority change making prior hypotheses unusable while retaining allowed content-free diagnostics.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All maintained known-truth scenarios for introductions, mentions, quotations, aliases, duplicate names, contradictions, independent agreement and circular/duplicate support yield their expected candidate/decision classes.
- **SC-002**: Every published hypothesis and decision cites current admitted evidence; all malformed, foreign and stale-reference acceptance cases fail before publication without changing accepted mappings, profiles or source text.
- **SC-003**: Repeated resolution of identical evidence/configuration produces identical outcomes/reasons after retry and reopening across supported catalog backends; corrections invalidate dependent prior results.
- **SC-004**: A configured user completes discovery, inspection and resolution through the shared command interface without per-item approvals; unknown/ambiguous outcomes remain usable and failures have actionable explanations.
- **SC-005**: Required deterministic, integration, documentation and applicable platform checks pass; every actionable review is resolved within two rounds and the final PR remains for owner merge.

## Assumptions

- The accepted recommendation and live issue bodies settle S072 scope. No new issue, product release or milestone-completion claim is authorized here.
- Existing manual/acoustic matching, rosters, source-linked evidence, configured providers and credential storage are reused. Hypotheses never become a parallel mapping authority.
- Minimal discovery/inspection/resolution commands are included for end-to-end utility; complete enrollment/application orchestration and saved pipeline integration remain #230-#232.
- Automatically grounded contextual support is allowed by this contract. Extending actual training dataset eligibility occurs in #230 and must reconcile its existing manual-only implementation then.
- Required deterministic tests establish contracts and recovery behavior. Real-engine/model-quality evaluation remains explicitly elected and distinct; the milestone qualification child #233 retains that broader acceptance scope.

## Clarifications

### Session 2026-11-02

No critical ambiguity requires an owner question. Scope/behavior, entities/lifecycle, interaction, quality, integrations, failure handling, constraints, terminology and completion criteria are clear from #227-#229, the approved recommendation and current contracts. Implementation choices and bounded numeric limits are resolved in the plan under autopilot. The author requirements checklist remains 16/16 with no changed markers.
