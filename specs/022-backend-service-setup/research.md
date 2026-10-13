# S022 research and decisions

## 2026-10-13: Bootstrap and non-mutation

Decision: introduce a shared Go bootstrap service callable before a catalog owner exists. Generate workspace/profile identities from nonsecret plans, prepare protected credentials before PostgreSQL connects, probe without opening ordinary mutating adapters, and publish configuration only after elected provisioning succeeds.

Rationale: current catalog/runtime initialization creates SQLite/WAL/schema/workspace rows, artifact initialization creates roots and normal graph opening may create or bootstrap a projection. Reusing those for preflight would violate #67.

Alternatives considered: opening the owner then probing (rejected for mutation/credential dependency), or hand-editing configuration (rejected for missing CLI/desktop parity). Existing-data migration remains explicit; configuring a fresh target does not move data.

Local path inspection reports schema/write capabilities untested when proving them would create files or sidecars. S3 HeadBucket proves limited bucket access, never all write permissions or a definitive cause for generic denied/not-found replies. PostgreSQL uses an authenticated read-only session; ArcadeDB uses authenticated metadata/query endpoints without provisioning commands.

Sources: [SQLite WAL](https://www.sqlite.org/wal.html), [S3 HeadBucket](https://docs.aws.amazon.com/AmazonS3/latest/API/API_HeadBucket.html), [PostgreSQL TLS](https://www.postgresql.org/docs/current/libpq-ssl.html), [ArcadeDB HTTP](https://docs.arcadedb.com/arcadedb/reference/http-api/http).

## 2026-10-13: Transport meanings

Decision: verify-full stays default; local accepts Unix sockets or actual loopback connections; insecure explicitly names remote plaintext. Existing remote-local input is rejected with correction, never silently rerouted or automatically rewritten. Current 1.0.1 schemas/docs change; frozen 1.0.0 records do not.

Rationale: transport validation is data-access integrity, not an eligibility policy. Explicitly named remote plaintext preserves the elected general-purpose capability while removing a misleading local label.

## 2026-10-13: Ollama native protocol and route election

Decision: native adapter version 1 uses bounded /api/version, /api/tags and /api/show metadata and /api/chat with stream=false. Native local inference can be behind a loopback or explicitly configured LAN/remote endpoint; model remote_model/remote_host evidence determines actual inference route. Hosted/cloud election is explicit. Existing insonic-http route semantics remain compatible.

Local requests use JSON output; cloud requests omit structured-output format because upstream cloud currently does not support it. Both receive the complete proposal contract and pass exactly the existing strict proposal, query validation, aggregate query budget and revision/profile fences. No provider text becomes an arbitrary executable query.

Discovery collects candidates and source/time without adopting any. External environment hints are shown but not automatically contacted; manual endpoint input elects the bounded metadata probe. An optional synthetic test may load a model but sends no library data.

Sources: [Ollama chat](https://docs.ollama.com/api/chat), [models](https://docs.ollama.com/api/tags), [model details](https://docs.ollama.com/api/show), [structured outputs](https://docs.ollama.com/capabilities/structured-outputs), [official API types](https://github.com/ollama/ollama/blob/main/api/types.go).
