# Data model

## Current discovery

A workspace-scoped current_speaker_discovery row is keyed by recording ID. It holds discovery/work IDs, catalog revision, recording/document authority, source epoch and a typed version-1 JSON aggregate (maximum 1 MiB). The aggregate includes coverage, provider provenance, hypotheses, authenticated acoustic evidence and resolution decisions. It excludes copied transcript text, raw credentials and provider-supplied authoritative times. Exact quotation content is validated transiently; persisted support uses current cue/voice references and quote digests.

A hypothesis distinguishes self-identification, context association, mention, quotation, owner mapping and acoustic prediction. It has a deterministic ID, candidate name/aliases and optional role/relationship, referenced cue/voice IDs, original evidence roots, provenance, uncertainty and alternatives. Current voice IDs remain local to their recording; person IDs remain workspace-scoped.

A source epoch hashes the current recording/document/source map, voices, canonical identities/aliases, selected roster, owner mapping revisions and selected acoustic authority. Work enqueue/claim/heartbeat writes do not change this epoch. Superseding any included authority deletes current derived content and writes a proof tombstone in the same mutation receipt.

## Extraction lifecycle

Pending -> running -> succeeded, failed or cancelled uses existing durable work and lease generations. A successful publication validates the frozen epoch, live lease, recording/cue/voice references, aggregate shape and size in one serialized write, then writes aggregate proof and terminal reference-only work result. Retrying a completed operation reconciles accepted work without another provider call. Failed/stale work cannot publish. Historical receipt replay never recreates a deleted current aggregate.

## Resolution

Resolution consumes expected aggregate revision/source epoch and configurable policy. It produces one decision per selected local voice with outcome matched, supported-contextual, ambiguous, conflicting or unknown; nullable canonical speaker ID; registered/unregistered/unknown identity status; supported name; alternatives; reasons; evidence IDs/roots; and current authority digest. Candidates and reasons are sorted before digesting. Provider confidence never independently proves identity.

Owner mappings remain authoritative. Duplicate spelling preserves competing identities. Derived predictions cannot certify their own lineage. A current acoustic result can corroborate only when its receipt, profile/output and independently grounded dataset are current; missing, stale or overlapping lineage remains explicit non-support.

## Portable state

Catalog schema 12 adds the typed collection. Restores validate aggregate shape, scoped references, current epoch and latest receipt proof/tombstone, including after an outer digest is recomputed. Older schema snapshots cannot inject discovery rows. Frozen schema-11 DDL enables populated upgrades without rewriting immutable source/model artifacts.
