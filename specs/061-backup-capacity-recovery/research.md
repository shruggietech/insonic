# S061 research

**Date**: 2026-10-31. Read-only Spec Kit research completed before implementation.

| Decision | Rationale | Alternative considered |
|---|---|---|
| Share native capacity predicate in contracts | Artifact already recognizes Unix ENOSPC/EDQUOT and Windows disk/quota codes; backup currently loses them | Duplicate OS maps (rejected due drift) |
| Preserve existing activation fencing | Catalog backup activation already binds bundle and destination, refuses takeover and recovers identical retries | New restore journal (unnecessary competing authority) |
| Private per-call file operations | Exercise open/write/flush/close/rename/remove without global races or product test switches | Global mutable hooks or environment injection (rejected) |
| Generated cleanup context plus joined causes | Retain primary capacity code and observable secondary failure without private error text | Ignore cleanup or replace the original cause (rejected) |
| Repeat durability on completed-output retry | Create currently inspects an existing same-ID bundle without repeating failed post-publication flush | Treat directory existence as durable success (insufficient) |
| Actual confined obstruction qualification | Packaged CLI can prove failure/refusal, unchanged bytes, exact retry and source-independent restore on both hosts | Claim physical exhaustion from injected errors (invalid evidence) |

Inspected boundaries: backup service fail/writePrivate/Create/RestoreWorkspace; progress save/load; artifact redact/capacity predicates; catalog RestorePortableFenced/BeginPortableActivation; source/package qualification and release receipt validator. No external library behavior is assumed or new dependency selected.

Implementation refinement: a confined SQLite page-limit restore failed as generic unavailable before the fix and succeeds on identical retry after raising that limit. Catalog classification now recognizes SQLite FULL and PostgreSQL SQLSTATE 53100, as documented by [SQLite](https://www.sqlite.org/rescode.html#full) and [PostgreSQL](https://www.postgresql.org/docs/current/errcodes-appendix.html). Actual CLI progress obstruction is an invalid-progress refusal before publication; unit tests cover progress write/flush/replacement and activation faults. The CLI cases are backup-parent, restore-publication and restore-progress. Original v1.0.3 compatibility is pinned to four canonical receipt digests recovered from its SHA256-verified published manifest; no version range is exempt.
