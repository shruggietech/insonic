# S008 graph adapter research

Research performed 2026-10-08 against the repository pins and upstream primary source. This file records design findings; it does not claim implementation or new runtime qualification.

## Decision: use the already qualified real engine pins

Use LadybugDB core 0.21.2 with Go binding `v0.17.1-0.20260804043248-42bbf464c74c`, and ArcadeDB 26.9.1 at the digest in `internal/qualification/dependencies.json`. The repository already proves native commit/rollback, exact integer handling and native Arcade `opencypher` availability in qualification tests. These feasibility probes do not yet implement production projections.

The Go binding module has no bundled `lib` directory. `system_ladybug` selects system headers and libraries; without that tag the upstream binding expects its missing bundled directory and invokes an OpenSSL pkg-config dependency. Production packaging already supplies `system_ladybug` and the native build environment. A production adapter must use the real binding under that build tag, while ordinary untagged development builds must explicitly report that the embedded adapter is unavailable rather than impersonating it with an in-memory graph. Native CI must execute production adapter tests with the prepared native environment, and extracted packages must query a real relocated graph.

Alternatives considered: a new binding or engine version would invalidate existing exact native/package qualification and is unnecessary for S008; treating catalog query evaluation as proof of real engine support would not prove backend parity.

## Decision: retain catalog authority and publish ordered revisions

Use the existing `catalog.OutboxClaim` fields: workspace, target, owner, generation, sequence, predecessor, catalog revision, operation ID, exact document and digest. Existing `ClaimOutbox` and `AcknowledgeOutbox` enforce ordered catalog acknowledgement, lease expiry and exact receipt identities. Add the production graph publication coordinator rather than creating an unrelated queue.

Adapter methods should implement the documented `Capabilities`, `EnsureSchema`, `FenceGeneration`, `ApplyRevision`, `ReadCheckpoint`, `Query`, `Explain`, `ExportSnapshot` and `Rebuild` operations. Projection checkpoint and immutable event receipt records must live in the real graph engine alongside projected facts. A checkpoint contains workspace/target, installed generation, committed sequence and catalog revision. A receipt records sequence/predecessor, operation ID, digest and revision. Receipt equality must compare every identity field, not just sequence or catalog revision.

`FenceGeneration` transactionally installs only a greater generation, retaining the committed sequence. `ApplyRevision` first checks exact existing receipt replay; otherwise it requires matching installed generation, next consecutive sequence and exact predecessor. Fact replacement, receipt insertion and checkpoint advancement commit in one transaction. Old writers must touch the checkpoint record in their transaction so a concurrent newer fence creates an engine conflict; a mere unguarded initial read is insufficient on an MVCC backend. Reconciliation may read exact receipts after a lease expires, but a stale lease may not accept catalog acknowledgement.

Source replacement must remove obsolete cue/document/assertion relationships in the same accepted revision. Resolve speaker mapping from current document and external mapping identities. Do not store a copied assignment array or retain old transcript facts as queryable alternatives. Keep current typed searchable graph properties and source references; receipt hashes may outlive replaced facts. Use application IDs, workspace scope and INT64 source microseconds/revisions, not native record identifiers or floating-point timestamps.

Rebuild must preserve a visible catalog revision boundary and cannot let a partially reconstructed target appear complete. Export/rebuild cover projected current facts and receipt/checkpoint state as required by the documented contract; release backup promotion remains issue #14.

## Decision: Ladybug owns one database with separate connections

Verified the pinned Go module locally at `C:/Users/h8rt3rmin8r/go/pkg/mod/github.com/!ladybug!d!b/go-ladybug@v0.17.1-0.20260804043248-42bbf464c74c`. Available methods are `OpenDatabase`, `OpenConnection`, `Prepare`, `Execute`, `Query`, `Interrupt`, `SetTimeout`, and explicit `Close` methods on database, connection, prepared statement, result and tuple. Transactions execute `BEGIN TRANSACTION`, `COMMIT`, `ROLLBACK` through `Query`; read queries can use `BEGIN TRANSACTION READ ONLY`.

Use parameterized `Prepare`/`Execute`, schema-first node/relationship tables and one serialized write path. Query connections must not share mutable connection state concurrently. Set a bounded buffer pool and thread count rather than accepting the binding default of 80% of machine memory for every open database. Bound result rows and execution timeout, propagate cancellation via `Interrupt`, and stop cancellation watchers before closing/reusing the connection. Close tuple before result, and connection before database.

Binding pitfall: `QueryResult.GetNumberOfRows()` returns the cached column-name count after `GetColumnNames()` has been called. Iterate `HasNext()` and count returned tuples; do not use that method for pagination, totals or omitted counts.

Pinned core `TransactionContext::validateManualTransaction` rejects write statements inside a read-only transaction. Native queries must additionally be a single accepted statement and exclude user transaction control, so user input cannot terminate the wrapper and execute an auto-committed write. Loading extensions, external attachments, macros, exports and mutable procedure calls belong to explicit maintenance operations, not ordinary exploration.

Pinned core `TransactionContext::commit()` can throw `CheckpointException` after the transaction has already been released as committed. Therefore a failed embedded COMMIT also has an uncertain outcome: read the exact event receipt/checkpoint before assuming rollback or replaying.

Sources: [Go binding pinned revision](https://github.com/LadybugDB/go-ladybug/tree/42bbf464c74c), [Ladybug pinned transaction source](https://github.com/LadybugDB/ladybug/blob/v0.21.2/src/transaction/transaction_context.cpp), [transaction syntax](https://docs.ladybugdb.com/cypher/transaction/).

## Decision: Arcade uses authenticated JSON queries and session transactions

Use `POST /api/v1/begin/{database}` and capture `arcadedb-session-id`; send it on every transaction request and end with `/commit/{database}` or `/rollback/{database}`. Database and endpoint components require validated/path-escaped configuration, not identifier concatenation from query input. Use JSON request bodies with `language`, `command`, `params`, explicit bounded `limit`, and synchronous responses. Parameters use SQL `:name` or OpenCypher `$name` syntax. Disable authentication-bearing redirects and preserve credentials as local references. Decode response numbers with `UseNumber` and convert integer fields losslessly.

HTTP sessions expire after inactivity (documentation default five seconds). Do not keep a transaction open during extraction or slow user work. Bound each transaction, detect missing sessions and reconcile uncertain commits using a fresh exact receipt read. A 204 retry of `/commit` is not proof that the original event committed: the pinned handler intentionally makes commit with an already removed/nonexistent session an idempotent no-op. Neither transport failure nor successful retry replaces receipt verification.

`POST /query` invokes `database.query`; `/command` invokes `database.command`. Use `/query` for ordinary query operations and `/command` only for internal projection writes or explicit maintenance. Qualified supported native language names are `sql` and `opencypher`; do not silently fall back to legacy `cypher`.

Critical read-only pitfall: in pinned `OpenCypherQueryEngine.query`, `PROFILE` and the reserved parameter `$profileExecution=true` bypass the `statement.isReadOnly` check and execute the statement. A query endpoint by itself is not enough. Ordinary native queries must reject PROFILE/reserved execution parameters, transaction/session control, multiple statements and mutable procedure/function routes. Enforce a supported read grammar and procedure/function allowlist, or use independently verified server read-only credentials for arbitrary native extensions. Do not require new credentials for already configured ordinary normalized operations. SQL uses parsed `statement.isIdempotent`, but registered user functions can still have effects, so the native contract must also account for function/procedure capabilities.

Current Arcade HTTP documentation describes later 26.10.1 replay and session features. Do not rely on those while the repository remains pinned to 26.9.1. Source/version probing and behavioral contract tests must establish capabilities, not just server readiness.

Sources: [HTTP/JSON API](https://docs.arcadedb.com/arcadedb/reference/http-api/http), [26.9.1 PostQueryHandler](https://github.com/ArcadeData/arcadedb/blob/26.9.1/server/src/main/java/com/arcadedb/server/http/handler/PostQueryHandler.java), [26.9.1 PostBeginHandler](https://github.com/ArcadeData/arcadedb/blob/26.9.1/server/src/main/java/com/arcadedb/server/http/handler/PostBeginHandler.java), [26.9.1 PostCommitHandler](https://github.com/ArcadeData/arcadedb/blob/26.9.1/server/src/main/java/com/arcadedb/server/http/handler/PostCommitHandler.java), [26.9.1 DatabaseAbstractHandler](https://github.com/ArcadeData/arcadedb/blob/26.9.1/server/src/main/java/com/arcadedb/server/http/handler/DatabaseAbstractHandler.java), [26.9.1 OpenCypherQueryEngine](https://github.com/ArcadeData/arcadedb/blob/26.9.1/engine/src/main/java/com/arcadedb/query/opencypher/query/OpenCypherQueryEngine.java), [26.9.1 SQLQueryEngine](https://github.com/ArcadeData/arcadedb/blob/26.9.1/engine/src/main/java/com/arcadedb/query/sql/SQLQueryEngine.java).

## Required verification derived from the design

1. Real embedded and remote projection tests compare exact current facts, relationships, nulls, int64 intervals and all normalized query operations.
2. Publication tests cover exact replay, divergent receipt, skipped predecessor, stale generation, takeover, write/fence concurrency, commit-response loss, and retry after committed receipt but before catalog acknowledgement.
3. Query integrity tests cover mutation words inside strings/comments/quoted identifiers, real mutable statements, nested mutation/PROFILE, multiple statements, transaction escape, procedure calls and reserved profiling parameters. Read-only queries remain expressive within the declared supported native contract.
4. Replacement and speaker/date corrections remove stale results without rewriting unaffected original source clocks; saved definitions are rerun against current facts, not frozen alternatives.
5. Native package tests query the real production graph with the checkout native library temporarily unavailable, proving installation-relative native loading. Generic untagged tests must explicitly distinguish unavailable native support from real graph success.

All tests above are deterministic and require no transcription, diarization, inference, model initialization or downloaded weights.
