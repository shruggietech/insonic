# Implementation Plan: S008 Evidence and Explore

**Branch**: `codex/008-evidence-explore` | **Date**: 2026-10-08 | **Spec**: [spec](spec.md)

## Summary

Deliver #11 and #12 end-to-end using current Cue JSON/mappings, ordered reference-only graph publication and shared exploration contracts. Add real official graph adapters, durable elected extraction, portable/native queries, calendar/recording timelines, saved versioned definitions and accessible force-directed views.

## Technical Context

Language: pinned Go 1.27.1, TypeScript/React/Wails, Node 22 and Python maintainer tooling.
Dependencies: existing Ladybug core 0.21.2/Go binding, ArcadeDB 26.9.1, SQLite/PostgreSQL, licensed media/Cueson.
Storage: new typed catalog domains with frozen v5 migration; graph stores source references and relationships, not another assignment/document store.
Testing: Go unit/race/schema/CLI and real native/integration backends, frontend rendered tests, native qualification, npm check/test and site/offline exports.
Platforms: Windows/macOS/Linux source and relocated packages.
Constraints: no required transcription/diarization/model initialization/weights; hidden Windows workers; hosted jobs below ten minutes.
Scale: bounded query pages and view layout, whole-library discovery with continuation.

## Constitution Check

I: Preserve approved #11/#12 and current-result contract #21. No approval queues.
II: Shared application/CLI operations precede GUI controls.
III: Exact source identity/clocks, current references, recoverable graph publication.
IV: Both official graph/catalog backends fully supported; no implicit routes.
V: Meaningful deterministic/native checks, truthful status, bounded CI.
Pre-design and post-design gates pass. No exceptions.

## Project Structure

- `internal/catalog/exploration*.go`: current extraction, versioned saved queries, layouts, migration/proofs and outbox helpers.
- `internal/graph/`: reference graph types, native validation, Ladybug/Arcade adapters, ordered publishing tests.
- `internal/evidence/`: current cue chunking, extraction adapter protocol and assertion validation.
- `internal/app/explore*.go`: shared operation dispatch, graph lifecycle, query/timeline hydration and extraction jobs.
- `internal/contracts/explore.go`, `schemas/v0.0.0/runtime-request.schema.json`, graph-query schema and examples.
- `cmd/insonic/explore.go`: shared CLI parser.
- `desktop/frontend/src/explore.tsx`: Explore controls/playback/graph/table; App integration and rendered/native qualification.
- public graph/desktop/pipeline/contracts docs and changelog, internal verification.

## Research and decisions

See [research](research.md), [graph research](research-graph.md), [data model](data-model.md), [contracts](contracts/explore.md) and [quickstart](quickstart.md).

1. Use tagged real Ladybug adapter in prepared native builds; untagged developer builds report unavailable explicitly. Existing packaging already uses `system_ladybug`; add native graph test coverage and ensure core includes prepared tagged acceptance.
2. Store reference-only extraction assertions in a current catalog domain, immutable saved definitions in another domain, layouts separately. Freeze v5 schema/digest before introducing v6 and qualify migration/restore.
3. Real authority mutations append reference-only invalidation events in the same transaction. Publication reconciles ordered events and a catalog-revision-bound reference graph refresh. Immutable event content cannot be rehydrated into different replay effects.
4. Graph entities/edges store source identities and relation references; common query hydration validates current catalog documents/mappings and derives text/timing at read time. Pending/unavailable graph state remains visible and transcript search available.
5. Native reads use parsed/lexed single-statement restrictions plus backend transaction/query enforcement. Reject Arcade PROFILE/reserved control parameters and effectful procedures; accepted native grammar is explicit and actionable.
6. Elected extraction uses bounded reference windows and a structured local builtin cue-statement adapter or explicitly configured HTTP adapter. It never invents linguistic propositions or silently changes route. HTTP contract receives selected cue context, uses referenced credentials, no redirects and bounded replies.
7. Calendar derives selected library dates over all entries. Exact nanoseconds remain decimal strings outside Go; no invented dates. Query pages/views and timeline windows are bounded independently of Library pagination.
8. Definitions/version history survive backend changes, while views rerun current evidence. Remove pinned-result schema/prose rather than retaining stale alternatives.
9. Required model execution stays prohibited; deterministic protocol fixtures plus real graph engines prove orchestration.
10. Dependency PRs #17/#23 are separate maintenance work and not silently merged into this functional slice.

## Delivery sequence

Specify, clarify, requirements checklist, research/design, tasks, blocking analyze, tests then implementation by story, convergence, CI-parity, commit/push/official PR, at most two bot review rounds, final green-check handoff before owner merge.

## Complexity Tracking

No constitution violations. Reference graphs intentionally hydrate authoritative content rather than creating duplicated cue/assignment catalogs.