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
