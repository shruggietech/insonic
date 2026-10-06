# Research: Portable catalog authority

## Transaction authority

**Decision**: SQLite immediate writers, bounded timeout, connection FKs, WAL/full sync; PostgreSQL workspace row locks and clock_timestamp after locking. Transactional counters and receipts admit a whole mutation.

**Rationale**: Sequences lose gap-free accepted ordering; start-of-transaction time can become stale after lock waits. Keep work outside transactions.

**Alternatives considered**: Global locks, SQL sequences, session maps and unconditional serializable transactions.

Sources: [SQLite isolation](https://www.sqlite.org/isolation.html), [foreign keys](https://www.sqlite.org/foreignkeys.html), [driver configuration](https://github.com/mattn/go-sqlite3), [PostgreSQL isolation](https://www.postgresql.org/docs/18/transaction-iso.html), [clock semantics](https://www.postgresql.org/docs/current/functions-datetime.html).

## Fencing and portability

**Decision**: Increasing signed generations, renewable expiry, immutable attempts/recovery receipts; claim only the next outbox event. One consistent snapshot, atomic empty-catalog restore and imported authority expiry. Typed FKs and ISO/int64 timestamp pairs preserve provenance and exactness.

**Rationale**: Restart must not steal a live remote worker; imported claims must not duplicate authority. PostgreSQL timestamps alone lose nanoseconds. Float JSON loses exact integers.

**Alternatives considered**: Shutdown as cancellation, SKIP LOCKED over predecessors, opaque JSON replacing associations and independent table exports.

Sources: [locking](https://www.postgresql.org/docs/current/sql-select.html), [snapshot transactions](https://www.postgresql.org/docs/current/sql-set-transaction.html), [datetime precision](https://www.postgresql.org/docs/current/datatype-datetime.html).

## Configuration

**Decision**: Resolve existing configuration.credential_id as username/password JSON through SecretProvider. Build pgx explicitly, clear ambient routing/fallbacks, quote dedicated schema and verify TLS identity. No dependency upgrade needed.

**Rationale**: Selected routing and secret references must remain authoritative; raw DSNs/driver errors must not reach public output. Separate migration rights from runtime rights.

Sources: [pgx configuration](https://pkg.go.dev/github.com/jackc/pgx/v5), [PostgreSQL TLS](https://www.postgresql.org/docs/current/libpq-ssl.html).

The installed plan skill required research-agent dispatch. The catalog research agent resolved clock, transaction, restoration and credential choices from primary sources and current schema/storage docs. No product ambiguity remains.
