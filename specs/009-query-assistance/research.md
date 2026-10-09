# Research: S009

## Catalog and query integration

Decision: dedicated existing-table per-setting CAS and context-aware direct query runner. Rationale: preserve native catalog snapshot portability and the exact existing hydration/fallback/freshness semantics. Alternatives: file settings (not catalog-portable), new tables (unnecessary), separate execution (divergent limits). Read-only runtime research identified owner-context-only dispatch and short IPC deadlines; request contexts/deadlines are required.

## Provider protocol

Decision: explicit version-1 HTTP assistance contract with schema/capabilities, selected model and prompt; deterministic constructor injection for CI. Rationale: matches existing elected hosted protocol and avoids external model activity in checks. Alternatives: implicitly speaking vendor APIs or initializing a model during qualification (outside agreed scope/CI policy). Native schema is Entity and EvidenceLink with reference-only values, not conceptual per-domain tables.

## Desktop/editor

Decision: preserve full QueryInput with the S008 loaded-query overlay machinery; accept a proposal only while prompt/settings/editor ownership is current. Rationale: CLI-authored hidden filters and native parameters must survive. Alternatives: reconstruct a smaller UI definition (already corrected in S008). Source and relocated package smoke both share the mounted qualification journey.

All technical unknowns resolved by code/research; no new dependency or architecture approval required.
