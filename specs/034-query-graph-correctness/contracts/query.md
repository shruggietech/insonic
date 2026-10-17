# Query contracts: S034

Normalized query.run and timeline.calendar retain their existing request/result shapes. next_cursor is opaque and can continue rows, relationships or both; an empty item page may still contain relationships and a cursor. Follow until next_cursor is absent. Accumulate stable identities across pages. Limits apply to both streams and total response bytes. Old-policy/stale cursors conflict.

search TEXT --after CURSOR repeats the original filters with a returned cursor. Human output presents total/omitted/continuation before bounded item details.

query.save data.query requires title containing at least one non-whitespace character. Schema-generated help contains it. query.run retains optional title.

query native accepts one read statement with an optional trailing semicolon and comments. Multiple statements, effects, reserved profiling parameters and unsupported functions stay rejected with bounded reason diagnostics. Parameter files reject duplicate object members before dispatch.
