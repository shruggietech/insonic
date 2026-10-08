# S008 data model

Date: 2026-10-08.

- CurrentExtraction: recording ID, revision, current document digest/revision, adapter/configuration provenance, reference-only assertions and bounded diagnostic codes. Replaced extraction overwrites current state; document replacement removes it. No copied cue text, timings or assignments.
- Assertion: stable semantic/source digest identity; subject/relation/object, polarity/modality/conditions, quoted attribution, cue IDs and extraction identity. Validate every reference against current document; quoted text is resolved on reads.
- SavedQueryRevision: query UUID and immutable local revision, title, portable/native typed definition, backend/schema/capability validation. Expected revision compares latest version. Historical definitions cannot include frozen result data.
- GraphLayout: separate stable view UUID, query UUID, revision, selected controls and bounded node coordinates. CAS updates; no evidence facts.
- GraphEvent: immutable invalidation or reference-refresh nodes/edges, ordered sequence/predecessor, catalog revision, operation/digest.
- GraphCheckpoint/Receipt: workspace/target owner generation, sequence, operation and content digest, atomic with facts.
- ReferenceNode/Edge: stable catalog/current-document/cue identities and relationship kind. Text/timing/speaker assignments are hydrated from current authority.
- QueryResult: current catalog/projection revisions and availability/pending status, ordered current hydrated items, continuation and graph omission counts.
- Timeline: selected origination observation and uncertainty display, original-clock cue evidence separately.

Catalog migration preserves frozen v5 DDL/digest and all existing records/receipts. Snapshot validation proves latest current extraction, saved revision and layout acceptance; restore fences imported publisher authority and rebuilds graph.