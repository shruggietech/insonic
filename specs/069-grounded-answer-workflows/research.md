# Research: Grounded answer workflows

Research completed 2026-11-02 by provider, durable-work and evaluation research tasks required by speckit-plan. All research was read-only; no model inference was elected.

## Shared synthesis transport

**Decision**: Extract bounded structured generation from assistance HTTP/Ollama adapters, preserving Propose as a query-specific wrapper. Keep answer-specific contracts in internal/answers without importing assistance (catalog already imports answers).
**Rationale**: Existing normalization, encrypted credential lookup, no redirects, classified redacted failures, cancellation, CI inference prohibition and Ollama local/cloud detection already govern elected routes. Reusing these avoids a second security implementation. Preserve 128 KiB serialized provider request and 512 KiB maximum response; reject oversized input explicitly, with no hidden truncation. Align answer model metadata with the existing 512-character model allowance.
**Alternatives considered**: New answer HTTP client duplicates sensitive logic. A query proposal repurposed as an answer conflates contracts and validation.

## Visible evidence and validation

**Decision**: Send a versioned answer-synthesis-v1 request containing only the bounded context's cue extracts and their authoritative references/family identities. Use typed quote/paraphrase claims. Require package membership, actually transmitted extract membership, exact supporting quote fidelity and current catalog authority. A quote claim must equal one of its citation quotes. Paraphrases remain provider assertions whose semantic quality is separately evaluated.
**Rationale**: Context can omit package items or truncate their text. The older envelope validator intentionally accepts full package evidence, so synthesis needs a stricter visible-evidence layer without breaking standalone validation. Provider output never selects routes/tools or mutates sources.
**Alternatives considered**: Sending all package text defeats context budgets. Treating valid citations as entailment gives unsupported confidence.

## Execution and bounded repair

**Decision**: One primary provider call plus at most one elected repair (default one), both under the same configured provider deadline (default 30s, cap60s). Repair only structured/citation failures and include bounded locally generated reasons, never arbitrary failed output. All answer execution shares a 180s maximum product deadline including up to60s retrieval,60s synthesis and bounded validation/publication. Claims cap128, citations cap128 per claim and128 distinct references overall; retained result cap1MiB. Provider response digests and generated diagnostics retain attempt evidence.
**Rationale**: Bounded correction helps malformed output without recursive loops or provider fallback. Transport failures remain actionable durable failures. Existing caller cancellation always wins.
**Alternatives considered**: Independent timeouts for each repair silently expand the elected budget; unbounded repair is inappropriate.

## Durable attempts and publication

**Decision**: Add answers.request work with a frozen normalized answer request, configuration, pipeline revision when selected, request digest, optional parent work ID and stable answer publication ID. Reuse the scheduler, leases, work wait/show/cancel and immutable artifact service. answers.retry creates a new linked work identity with the original frozen route; generic work.retry delegates for this kind. Save compact candidate publication metadata before publishing; recovery reconciles actual bytes before generating again. Final catalog acceptance checks work owner/generation/cancellation, current source epoch and available hash-verified publication in one transaction, and retains an idempotent receipt.
**Rationale**: Generic retry clears Work.Result and cannot establish attempt history. Artifact availability alone cannot establish accepted answer work. PostgreSQL catalog writes serialize on the workspace row and SQLite uses the existing write transaction, allowing an answerEpochTx helper to close the source-check/acceptance race.
**Alternatives considered**: Cache pointer reuse has no work fence. A separate job engine duplicates existing durable infrastructure. Raw answer bytes in Work.Result violate reference-only journal design.

## Pipeline configuration and readback

**Decision**: Add optional answer_synthesis configuration to existing pipeline definitions, with answer-only and combined definitions, Local/Connected/Custom route consistency and explicit inspection capability. Submission chooses a provided configuration, an elected saved pipeline, or the saved assistance configuration; ambiguous simultaneous selections fail. Freeze the election. Existing credential IDs resolve at execution, never persist values. Readiness remains configuration-only unless explicitly probed. answers.show/export accepts retained evidence package or grounded answer artifact identities, preserving existing package output. Work views report route/model/configuration digest, budgets/phases/counts and publication references only.
**Rationale**: Recognition/diarization and speaker-training definitions must not misclassify answer-only pipelines. Existing endpoint/model choices and authorization persist.
**Alternatives considered**: A parallel answer-settings store would duplicate existing routing. Silently changing routes at retry violates approved behavior.

## Evaluation and qualification

**Decision**: Version a public synthetic corpus with independent, duplicate/overlap, conflicting, unsupported, withheld and source-instruction cases. Keep reference rubrics out of prompts. Report retrieval coverage, source-family count, citation validity, exact quote fidelity, expected proposition coverage, unsupported claims and required uncertainty separately. Deterministic thresholds are 100% required coverage/validity/fidelity/uncertainty, exact family counts and zero unsupported claims. Null denominators are explicitly not applicable. A separately elected installed-CLI model runner reports exact model/configuration/corpus digests and conservative frozen-rubric limitations; unrecognized paraphrases are failures, not assumed support.
**Rationale**: Protocol checks cannot establish model quality. No applicable answer-model evaluation election was found, so report real-model quality as not elected unless explicitly elected later. Fixtures cannot masquerade as inference.
**Alternatives considered**: Token overlap or fluent prose is not a reliable answer-support metric. An unrestricted model judge adds another unapproved route.

**Decision**: Extend installed package CLI qualification for fresh/saved/partial/fail/retry/cancel/export/stale journeys with deterministic loopback providers, and add an installed Linux alternative-backend campaign against PostgreSQL/S3/ArcadeDB alongside default backend coverage. Register corpus ownership and executable help substitutions.
**Rationale**: Existing runtime backend tests do not prove installed CLI parity. Owned fixture workspaces may clear CI markers solely for deterministic loopback fixtures; required CI continues to prohibit real inference.

## Release recommendation

**Decision**: Recommend v1.2.0 after completed milestone acceptance. Keep software, versioned docs and master schema at1.1.0 in this implementation slice; coordinated version preparation/publication follows separate authorization.
**Rationale**: New durable answer operations are additive minor-release capability. This follows the accepted release-recommendation pattern in specs/064-transcript-coverage-repair/release-recommendation.md. No major-breaking change is planned.
**Alternatives considered**: A patch understates the added public workflow. Publishing/bumping as if accepted release qualification were complete would conflate implementation and release authorization.
