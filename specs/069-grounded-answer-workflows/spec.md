# Feature Specification: Grounded answer workflows

**Feature Branch**: `codex/069-grounded-answer-workflows`

**Created**: 2026-11-02

**Status**: Specified

**Input**: S069, issues #224, #225 and #226 under #210. Implement using Spec Kit and autopilot, no new tracking issue, automatically push and open a pull request, resolve at most two review rounds, and stop for owner review and merge after green CI.

## User Scenarios & Testing

### User Story 1 - Inspect grounded synthesis (Priority: P1)

An operator asks a question about current recordings using a configured local or hosted synthesis route and receives claims with independently resolvable evidence, exact quotations or explicitly identified paraphrases, and visible uncertainty.

**Why this priority**: An answer without trustworthy evidence cannot satisfy the approved outcome.

**Independent Test**: Use a bounded fixture provider and an existing evidence package. Inspect claims, current cue identities and exact source times without changing recordings.

**Acceptance Scenarios**:

1. **Given** independent and duplicated sources, **When** a provider returns supported claims, **Then** the answer preserves source-family counts and exact authoritative citations without treating copies as independent support.
2. **Given** fabricated, foreign, stale or shifted references, **When** synthesis is validated, **Then** invalid claims are rejected or excluded with diagnostic reasons and cannot be reported as a verified complete answer.
3. **Given** disagreement, insufficient evidence or source text containing instructions, **When** the provider proposes a conclusion, **Then** uncertainty remains expressible and source text cannot change the selected route, authority or execution policy.

### User Story 2 - Run and recover unattended answer work (Priority: P1)

An operator requests, waits for, inspects, exports, cancels and retries answers through the CLI or a saved pipeline using existing configured credentials and routes.

**Why this priority**: Long operations must survive normal client exits and failures without losing provenance or publishing stale results as current.

**Independent Test**: Exercise fresh and saved requests with a controlled provider, fail and retry work, restart a worker, cancel work, and invalidate evidence before export on supported storage/catalog/graph combinations.

**Acceptance Scenarios**:

1. **Given** a configured route, **When** an answer is requested, **Then** work exposes retrieval, context, synthesis and validation progress, effective budgets and route identity without a per-answer approval.
2. **Given** a failed, interrupted or ambiguously published attempt, **When** the operator inspects or retries it, **Then** persisted work and verified artifacts determine the outcome; retries retain the previous attempt and never silently change routes.
3. **Given** a cancelled request or changed source authority, **When** execution completes or an answer is read, **Then** cancellation and stale/unknown status remain visible and the answer cannot silently become current.
4. **Given** synthesis is unavailable, **When** direct evidence queries are requested, **Then** those queries remain independently usable.

### User Story 3 - Assess quality and release readiness (Priority: P2)

A maintainer evaluates cross-recording answers against a versioned public synthetic corpus and receives measured retrieval, independence, citation and support results, retained failure cases and a compatibility/release recommendation.

**Why this priority**: Protocol correctness and plausible prose are different claims, and the milestone needs an auditable readiness decision.

**Independent Test**: Run deterministic cases without inference and optionally elect a configured model evaluation on the same corpus. Inspect metrics, thresholds, model/configuration identity, failures and proposed release version.

**Acceptance Scenarios**:

1. **Given** duplicate, contradictory and withheld evidence, **When** evaluation runs, **Then** duplicate support never improves independence scores and unsupported questions produce limited answers under explicit criteria.
2. **Given** an elected real model, **When** evaluation completes, **Then** the report names its model/configuration and measured limitations independently of deterministic acceptance.
3. **Given** complete milestone qualification, **When** readiness is reported, **Then** the packet states compatibility impact, a proposed semantic version, software/docs/master-schema alignment and concise highlights ending in a changelog link.

### Edge Cases

- Empty retrieval, partial graph readiness, exhausted retrieval/context/provider budgets, unknown source times and oversized/malformed provider output.
- Repeated citations, cross-workspace references, modified quotations, invented cue IDs, changed revision metadata and authority changes during validation or publication.
- Conflicting claims and exact quotes versus paraphrases; structurally valid evidence references do not by themselves prove semantic entailment.
- Provider refusal, timeout, transport failure, bounded repair exhaustion, rotated credential values and configuration changes between admission and execution.
- Worker/client restart, cancellation before or after artifact storage, duplicate execution, publication response loss and retry after stale evidence.
- Export and status must not expose credential values, original private paths or unaudited provider diagnostics.

## Requirements

### Functional Requirements

- **FR-001**: Synthesis MUST use an explicitly selected existing local or hosted route and a bounded structured request/response with claim-level evidence references. It MUST NOT silently reroute or add per-answer approval.
- **FR-002**: Validation MUST check package membership, workspace, current source/document authority, quotation fidelity and source-time boundaries against authoritative cues. Provider-supplied identities and times are untrusted.
- **FR-003**: Exact quotations MUST remain distinguishable from paraphrases. Supported claims require citations; insufficient, conflicting and unverifiable answers MUST retain explicit reasons. Structural validation MUST NOT be presented as semantic entailment proof.
- **FR-004**: Provider output, execution duration, claim/citation counts and repair attempts MUST have explicit finite limits. Repair MUST use the same elected route, retain diagnostics and stop on cancellation, authority change or budget exhaustion.
- **FR-005**: Existing immutable sources, duplicate-family accounting, retrieval/context bounds and direct evidence operations MUST remain usable and unmodified by synthesis.
- **FR-006**: Shared runtime and CLI MUST support requesting, waiting, inspecting, exporting, cancelling and retrying durable answer work. Persisted work MUST expose retrieval/context/synthesis/validation phases, effective route/budgets, evidence counts and limitations in complete JSON and concise human output.
- **FR-007**: Saved pipeline execution MUST use existing encrypted credential resolution and configured routing. Persisted requests, results, logs and exports MUST contain no credential values or original private source paths.
- **FR-008**: Restarts, duplicate execution, cancellation and uncertain publication MUST reconcile using durable work and verified artifact/catalog authority. A retry MUST be traceable to its prior attempt and MUST NOT overwrite history or silently change route/configuration identity.
- **FR-009**: Freshness MUST be checked before publication and at inspection/export. Changed or unresolvable evidence MUST be visibly stale or unknown and MUST NOT be represented as a current answer.
- **FR-010**: Answer artifacts MUST preserve resolvable citations, source and configuration provenance, validation diagnostics and bounded provider-attempt information across filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB.
- **FR-011**: A versioned synthetic or appropriately licensed corpus MUST contain independent recordings, duplicates, contradictions, unsupported questions and withheld-evidence cases with known expected evidence.
- **FR-012**: Evaluation MUST separately measure retrieval coverage, independent-source counts, citation validity, quotation faithfulness and answer support. Each metric MUST state its denominator, pass criterion and retained failure examples. Deterministic regression MUST require no real inference.
- **FR-013**: Real-model evaluation MUST be explicitly elected, identify model/configuration and report actual metrics and limitations. An unelected or unavailable model evaluation MUST remain explicitly unverified and MUST NOT be replaced by fixture results described as model quality.
- **FR-014**: CLI help, schemas, authoritative public documentation and changelog MUST agree with implemented behavior. Advanced CLI capabilities may precede GUI controls when documented. Public docs MUST omit slice/task evidence links.
- **FR-015**: The release-readiness packet MUST recommend a semantic version from compatibility impact and describe aligned software, versioned docs and master schema. S069 authorizes a PR, not a release publication or merge.
- **FR-016**: Qualification MUST cover installed CLI success and failure/recovery journeys on supported backends, security/tenancy boundaries and affected package/site checks, under the existing preparation, 1,800-second target, 3,600-second hard limit and terminal-receipt rules. No attended desktop gate is introduced.
- **FR-017**: Planning MUST map directly to #224-#226 without creating a tracking issue. External review requests MUST total at most two rounds and every finding MUST receive a disposition before owner handoff.

### Key Entities

- **Answer request**: Question, selected scope, evidence budgets and elected route/configuration identity.
- **Answer attempt**: Durable execution identity, lifecycle, phase/progress, parent retry, limitations and publication outcome.
- **Grounded answer**: Immutable claims, quote/paraphrase distinction, authoritative evidence references, validation and freshness status.
- **Evaluation case/report**: Versioned evidence, expected support and uncertainty, separately measured metrics, failures and optional exact model identity.
- **Release-readiness packet**: Compatibility assessment, proposed version, required/elected verification evidence and concise release highlights.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every accepted deterministic citation resolves to elected current evidence with faithful exact quotes and valid known source timing; every adversarial fabricated/foreign/stale/shifted reference is rejected or diagnosed.
- **SC-002**: All maintained answer journeys complete without per-answer interaction on each supported backend, including cancel/restart/retry/unknown-publication and stale export cases.
- **SC-003**: Duplicate corpus additions do not increase independent support counts; withholding all supporting evidence produces no accepted supported answer in the deterministic corpus.
- **SC-004**: Every evaluation report states all metric denominators, thresholds and failure cases and clearly separates protocol results from any elected inference-quality results.
- **SC-005**: All required automated gates pass for the final PR head; all external findings are resolved within at most two review rounds, with a retained release recommendation and no new tracking issue.

## Assumptions

- The merged evidence request, retrieval, ranking, context and cache contracts satisfy dependencies #220-#223; S069 verifies their integration rather than replacing them.
- Configured credentials and local/hosted routes already establish authorization. No new external route is elected by the request itself.
- Release preparation/publication follows the recommendation in a separately authorized slice; current software/docs/master schema remain aligned until then.
- No model-quality result is claimed without an explicitly elected model run. Deterministic fixtures are synthetic and contain no private recordings.

## Clarifications

### Session 2026-11-02

- Q: Does the request authorize release publication or merge? A: No. It authorizes implementation, push and an official PR, with owner review/merge after green CI. The release deliverable is a compatibility and readiness recommendation.
- Q: Is provider-free fixture evaluation sufficient evidence of real-model quality? A: No. It establishes protocol/provenance behavior; optional elected model evaluation is separately identified and measured.
- Q: Must answer execution prompt on each item or recover by switching provider? A: No. Existing configured authorization persists, and failures retain the selected route and uncertainty.
- Q: Should new issues be created for this work? A: No. Existing #224-#226 carry scope; parent #210 closes only if its complete milestone criteria are actually met.
