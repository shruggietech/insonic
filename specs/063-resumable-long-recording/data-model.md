# Data model: S063

## Processing plan

Versioned immutable JSON identifies source ID/digest/revision, expected document digest/revision, semantic configuration and model digests, original rational starts, native sample counts/rates, selected tracks/channels and bounded windows. Plan digest excludes actual execution device and rebound filesystem paths. Catalog row links work and publishing generation with active/completed/invalidated state.

## Window

Stable identity and ordinal, track ordinal, native core/context sample endpoints, inward projected 16 kHz endpoints and exact SourceMap. Cores are half-open and contiguous; context is clipped at elected boundaries. Empty input has zero windows. No arithmetic overflow or duplicate track/channel selections are accepted.

## Checkpoint

Stable logical plan/window/stage ID, work ID, publishing generation, immutable artifact publication, output SHA-256/size, ready/retired state. Commit requires current work authority and unchanged source/current document plus verified available artifact kind/digest/size. Retry can reuse ready output from earlier generations of the same work and plan after byte verification. Partial writes have no checkpoint authority.

## Result evidence

Recognition segments and optional words retain mapped-relative microseconds and original text. A window result records source/window identity, actual-device history, stage provenance, diagnostics and digest. Stitch identity binds the ordered verified outputs. Final cue IDs map to contributing windows/source intervals in Recording.Provenance; no duplicate transcript is stored there.

## Lifecycle

Planning -> active plan -> ready checkpoints -> reconciliation -> atomic current acceptance -> completed plan and retired checkpoints. Source/current-document replacement invalidates active incompatible plans. Cancellation retains ready checkpoints for explicit retry, while owned partial scratch is removed. Backup includes ready references/bytes; retired outputs remain digest-only provenance and enter existing cleanup.
