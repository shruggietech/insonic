# S004 data model

## Credentials

Opaque workspace-scoped ID and selected native/vault/session backend. Values never enter catalog/job/manifest state. Add rejects existing identities; Replace is explicit. Vault format and fixed KDF are authenticated; keys remain in current owner memory.

## Current library entry

Stable media/asset IDs, title/class, source digest/size/mode and sanitized acquisition provenance, original and subtitle publications or external binding, optional measured duration, stream/channel/time-base facts, current metadata report publication and normalized metadata/date state. `duration_us=null` means unknown; `duration_us=0` means measured zero. Original input identity is immutable. Relocation requires matching digest/size.

## Dates

Entered literal, source/basis, precision/bounds, zone and captured mapping/rules version, offset, resolved integer instant when known, assumptions/conflicts/diagnostics and versioned selection rule. Owner observations survive computed-metadata replacement.

## Downloaded model

Exact name/version/upstream revision, manifest digest, capabilities/license declaration and ordered required file roles/digests/sizes. Available state requires all publications verified. No speaker/training association is fabricated.

## Durable work and cleanup

Work ID, canonical intent, kind, state, owner/generation/lease, stable subordinate IDs, phase/checkpoint/status and result references. States: pending/running/succeeded/failed/cancelled/interrupted; retry obtains fresh generation. Claims and checkpoints enforce current unexpired authority. Restore expires authority. Cleanup records contain superseded publication IDs and retirement progress, never old metadata payloads. Final success requires cleanup.
