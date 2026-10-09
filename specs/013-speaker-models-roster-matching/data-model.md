# Data model

## Current authority

Extend speaker mappings with explicit manual/automatic origin and bounded matching provenance. Old mappings remain manual. Automatic mappings do not enroll their own predicted identity; manual confirmation can explicitly replace them. Audio/document replacement clears obsolete automatic mappings while preserving applicable user corrections.

Datasets bind a speaker/identity revision, stable corpus epoch, recipe, summary and ordered current references. Each reference names recording/source/document/cue/local speaker and mapping authority; no copied assignment or subtitle text. Dataset invalidation removes obsolete content-bearing preparation/membership, retaining noncontent origin.

## Durable training and immutable outputs

Training uses real renewable Work identity, producing attempts and explicit resume checkpoints. Checkpoints bind digest/format/step/base/adapter compatibility. Output family/version identity is stable; output metadata declares architecture, kind, consumer compatibility, licenses, artifacts or hosted-only provider handle. Immutable completed output metadata survives invalidation of corpus preparation.

Revisioned current profile/default associations identify the selected exact version for a speaker without modifying version origin. Merges expose lineage; splits require explicit current association correction. Publication accepts only verified durable objects and valid current dataset/work authority.

## Matching

A frozen attempt binds current recording/source/document and mappings, roster revision, member identity revisions, exact selected profile versions/output manifests, matching adapter/base and thresholds. Decisions are accepted/unknown/ambiguous/manual-preserved; candidate diagnostics are bounded. Accepted automatic mappings refer to exact current evidence and work/model provenance outside CueJSON. Unknown/ambiguous reruns remove applicable old automatic mappings but do not change manual mappings or activity intervals.

## Validation and state transitions

All references are workspace-local; UUIDs and existing portable object references apply. Datasets: current -> invalidated. Work: existing queued/running/cancelled/failed/completed transitions. Output versions are immutable; availability is distinct from corpus validity. Current profile association revisions fence concurrent edits. Local/hosted adapter configuration contains credential references only. Scores must be finite and candidate IDs must belong to the frozen eligible roster.
