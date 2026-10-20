# Investigation data model

No catalog or public JSON schema changes are required.

| Entity | Required identity and fields | Rules |
| --- | --- | --- |
| Failure signature | class, source revision, run, attempt, first failed stage, pre-cleanup observations | Controller, pre-DOM, UI join, menu and browser-release remain separate. |
| Native boundary snapshot | fixed operation/stage, entered/returned counts, HRESULT, owner-thread relation, bounded timing | No pointers, URLs, bodies, private paths or credential values are emitted. |
| Runtime identity | available version, actual environment version, selection method | Availability must never substitute for actual environment identity. |
| Process-release observation | owned job, root exit, initial/final live counts, launch/exit/release timings | Natural zero is success; forced termination remains failed cleanup. |
| Experiment | hypothesis, control, fixed sample/time bounds, predicted outcomes, results and next decision | No identical retry without new evidence; finite passes do not establish absence. |

State progression is proposed experiment -> executed observation -> classified result -> reproducible violation -> corrected regression -> qualified candidate. An unresolved result does not advance to repair complete.

## Scheduler state evidence (2026-10-20 UTC)

Capture coverage and state accounting are distinct. A verified process lifetime may still contain unknown state time. Ready/wait getter fallbacks do not establish states. Closed intervals require canonical thread/process and explicit readiness or supported switch-out state, complete compatible indexed readiness history and known endpoints. Initial unsupported gaps remain unknown. Each retained state total carries fixed source counts and a maximum witness (begin/end/source), relative to its thread observation begin. Disputed scheduled+ready intersections are a subset of unknown; other overlaps reject.

All threads are computed, then oldest+single longest known ready+single longest known waiting are retained per verified process (at most3); unselected threads remain a disclosed interpretation limit. The decoder's64KiB output cap and16process bound remain enforced. Compact console receipts preserve the main thread's summary, conflict witness/count/time and closed-input disposition counts. No raw events, command lines or images enter retained evidence.
