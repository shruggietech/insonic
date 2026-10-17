# Data model: S034

No persisted schema migration. Query definitions and immutable saved versions retain existing identity.

- Continuation contains a query/corpus digest, nonnegative row offset and nonnegative edge offset. It binds deterministic ordering and the pagination policy; stale/mismatched cursors fail. Each stream returns at most the requested 1..500 entries and the envelope stays within 512 KiB. Edges retain stable ID, kind, from and to; endpoints may arrive on other pages.
- Calendar timezone accepts UTC/Z, local, pinned IANA zones and signed HH:MM offsets. Missing calendar timezone means UTC. Metadata interpretation preserves captured local-zone identity and pinned rules/mapping provenance.
- Saved query requires a Unicode nonblank title with the existing maximum length. Ordinary run/explain definitions keep optional titles.
- Parameter JSON has unique members at every nesting level and typed values retain exact integer decoding.
