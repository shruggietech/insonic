# Data model

Every workspace row uses UUID text and scoped composite FKs. Revisions, generations, ordinals, counters and Unix nanoseconds are signed int64 with overflow rejection. Unknown values remain null; immutable evidence is insert-only.

| Group | Typed records/relationships | Lifecycle |
| --- | --- | --- |
| authority | Workspace version/revision, operation UUID/digest/outcome | Transactional acceptance, replay receipt |
| settings/profiles | Name/role, expected revision, value or adapter/version/config/credential reference | Selection history, immutable revisions |
| metadata/date | Media/asset/artifact/snapshot, qualified tag/value/unit, raw literal/zone/precision, ISO/unix_ns | Immutable provenance, explicit date selection |
| speaker corpus | Speaker, segment source/interval/channel, dataset/manifest, ordered exact membership | Frozen evidence with FK reservations |
| training/models | Dataset, originating speaker, run/attempt, family/version, output artifact roles | Immutable provenance reservations |
| jobs | Idempotency key/options, selected attempt, owner/generation/expiry | Running to succeeded/failed/cancelled/interrupted; fresh retry/recovery attempt |
| projection | Target, sequence/predecessor, revision/payload/digest, claim generation/expiry/checkpoint | Immutable ordered events, mutable fenced outcome |
| snapshot | Version, workspace/revision, ordered typed rows, integrity digest | Consistent export, atomic empty restore |

Timestamp pairs must agree at nanosecond precision. Required provenance is non-null. No cascades erase history. Restored active claims expire and must recover into a new generation.
