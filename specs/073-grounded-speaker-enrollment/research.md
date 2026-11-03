# Research: Grounded speaker enrollment

## 2026-11-02

- Decision: Retain compact contextual provenance in existing SpeakerMapping, with a separate contextual origin. Rationale: discovery SourceEpoch includes mappings and becomes stale after application; acoustic automatic provenance is a different authority. Alternatives: retaining a second assignment store or treating context as manual would lose correction/independence semantics.
- Decision: Use atomic revision-fenced recording-level apply/retract and accepted receipts. Rationale: SetSpeakerMapping is an explicit manual boundary; calling it from automatic work would falsely confer owner authority. Reuse reconciliation and graph outbox rather than independent cleanup.
- Decision: Extend selection, epoch, reference validation, manifest provenance and portable validation together. Rationale: current ConfirmedOnly gates are manual-only at several boundaries. Leave legacyMappingDigest unchanged so compatibility does not admit new origins under old hashes.
- Decision: Distinguish completed output availability from current profile usability. Rationale: evidence invalidation deliberately preserves immutable outputs, while matching must not consume stale evidence. Check current dataset/output state and profile revision before selection/publication.
- Decision: Use a durable coordinator with child work and nonblocking stage yield. Rationale: AcceptSpeakerDiscovery is terminal and a coordinator occupying a worker slot while awaiting children can deadlock four concurrent workflows. Existing EnqueueChildWork and catalog work fences provide the basis.
- Decision: Reuse configured routes and encrypted credentials; freeze pipeline/model/tool election before work starts. Rationale: retries must not choose a new provider or silently change settings. No implicit research, per-sample review or parallel assignment database.

Research sources are the current repository code and authoritative docs/v1.2.0/{speakers,voice-models,pipelines}.md. Full acoustic accuracy and corpus quality remain separately qualified under #233.
