# S008 research

Date: 2026-10-08. Spec Kit plan Phase 0 dispatched graph, catalog and Explore research agents; no implementation delegated.

## Decisions

- Decision: Use existing pinned graph engines and prepared production native builds. Rationale: exact existing qualification/package pins, true official backend parity. Alternatives: fake graph or a new service rejected. Detailed primary-source findings: [graph research](research-graph.md).
- Decision: Add explicit v5-to-v6 catalog migration and frozen legacy digest. Rationale: extending typed domains otherwise rejects existing workspaces. Alternatives: ad hoc tables outside snapshot/restore rejected.
- Decision: Append graph invalidations with authority changes, then commit immutable revision-bound reference refresh events. Rationale: no mutation/publication gap or different payload behind an immutable digest. Alternatives: resolving a mutable document during replay rejected.
- Decision: Hydrate queries from current documents/mappings. Rationale: #21 sole assignment authority, replacement correctness. Alternatives: storing text/turn snapshots rejected.
- Decision: Preserve saved definition history and separate layouts, remove pinned results. Rationale: #12 explicitly requires current result views. Alternatives: retaining obsolete processing snapshots rejected.
- Decision: Reuse NativeBridge playback and client generation fences; add request sequencing for Explore. Rationale: existing lifecycle/current binding. Alternatives: GUI-owned playback stores rejected.
- Decision: Derive whole-library calendar from library.Dates selected observation. Rationale: Library summaries omit dates and are paginated. Alternatives: import/file timestamps and first-page timelines rejected.
- Decision: Builtin extraction represents literal source statements; configurable HTTP protocol returns structured assertions. Rationale: local default works without model servers while elected adapters support semantic propositions. Alternatives: heuristic invented facts or mandatory LLM inference rejected.
- Decision: Required checks use supplied speech/turns and deterministic extraction responses. Rationale: owner CI cost boundary. Alternatives: tiny/cached models rejected.

## Integration facts

Current catalog schema5, CommitRecording/SetSpeakerMapping reconciliation, library save and identity mutations need atomic event hooks. Existing ClaimOutbox/AcknowledgeOutbox supplies generation/predecessor but needs renewable publisher/status. Current selection ResolveEvidence proves cue/source/map ownership; current state proof validators govern restore. App executeWork needs an explicit evidence branch. App route/CLI/runtime schema need exploration operations.

Explore currently has no route. recordingCues already exposes decimal-string clocks/current bindings. Library date fields distinguish day/instant/range, resolved/date-only/bounded and one/two-sided bounds. Saved-query schema defines portable/native versions but contains conflicting pinned_result. Frontend rendered fixtures validate runtime schemas; native qualification can extend existing licensed import/playback/assembly journey.

All technical unknowns resolved. Primary engine sources are linked in graph research. No engine/model runtime was executed in research.