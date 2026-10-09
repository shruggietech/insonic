# S012 Data model

## Reference and catalog state

Alias: stable UUID, lowercase bounded name, revision, active/deleted state, declared operation and exact target. Runtime callers choose the UUID; a name remains permanently bound to that ID through tombstones. Target kind is base, speaker version or hosted. Tombstones retain revision fencing; replay uses operation receipts. Source: stable ID/name/revision/state, bounded configured catalog URL and opaque credential ID, explicit loopback exception. Never persist secret values.

Base manifests add optional compatibility: adapter/contract, architecture/runtime format, supported sample rates/channels where relevant. Digest covers the exact complete manifest; legacy manifests are not rewritten. Speaker versions continue using existing family, run, dataset, artifact, association and provider metadata.

## Election and work

Resolved election binds input reference, alias/source revision where relevant, target kind/ID, manifest digest/upstream revision, operation/adapter/version, compatibility outcome and availability. Acquisition work ID derives from workspace/exact installation/manifest. Parent payload stores dependencies and exact elections; retries preserve payload. Saved pipeline inputs may retain aliases, but frozen election uses exact IDs. Import payload similarly freezes elected diarization and digest. No changing aliases at execution.

States: registration does not imply availability. Acquisition pending/running/interrupted/failed/cancelled/succeeded is existing durable work; parents pending for acquisition consume no workers. Only successful verified dependencies allow processing claim; dependency failure produces explicit parent failure. Parent retry repairs original dependency, cancellation only fences parent. Model available requires verified publications; missing/corrupt bytes require reacquisition.

## Portability

Schema 8 adds typed domains with SQLite/PostgreSQL-equivalent constraints and validate/replay proofs. Freeze schema 7 before changing definitions; opening and exported v7 snapshots remain compatible. Aliases/sources/model relationships export without credentials or temporary paths; relevant graph projection remains catalog-derived. Durable trained output and active leased publications are never cache eviction candidates.
