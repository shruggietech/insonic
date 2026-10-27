# S051 research (2026-10-27)

- Decision: validate all source recording IDs and revisions inside the restored-only validator. Rationale: current shared-null equality and fallback1 accept missing authority; enrollment-only duplicates otherwise evade the destination subset. Alternative: global result validation would change historical v1 meaning.
- Decision: reuse canonical UUID validation before uniqueness checks and require exact positive integer revisions. Rationale: malformed/unhashable values should receive validation refusal and booleans must not pass as integers. Alternative: coercion invents authority.
- Decision: revalidate the hash-verified retained campaign without rerunning inference. Rationale: only evidence checks change; exact original execution source/timing remains authoritative. Alternative: another campaign adds no coverage for missing-field refusal.
- Decision: reconcile S050 through a dated addendum. Rationale: its owner merge occurred before final convergence/handoff, which cannot be retroactively asserted.
- Research review: existing internal UUID validation permits canonical UUIDs independently of version. Do not add UUIDv4 or non-nil restrictions; public recording schemas retain their separate contract. Validate strings before uniqueness checks so lists/objects receive ValueError.
