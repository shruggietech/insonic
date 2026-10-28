# Data model: Import interruption recovery

No catalog migration or new user-facing operation is required.

| Entity | Required authority | Transition |
| --- | --- | --- |
| Candidate pair | Work/ordinal-derived canonical/report publications, report source hash/size, canonical provenance, verified immutable bytes | Complete unaccepted pair survives process loss and is reused only for matching source |
| Source observation | Staged byte identity plus retained raw metadata, capture instant and date observations | Recovered observations remain bound to original bytes; changed source refuses reuse |
| Runtime/attempt | Actual owned PID/session, work/attempt IDs, generation and existing lease | Dead prior runtime -> ordinary expired-claim recovery -> distinct replacement attempt |
| Recording | Deterministic original operation identity and current catalog authority | Absent before kill -> exactly one admission after successful recovery |
| Recovery receipt | Image/platform/source or package identity, three case observations, cleanup | Incomplete/incorrect evidence -> rejected; complete verified outcomes -> accepted |

An observed ordinary admission failure retains existing candidate retirement behavior. Report corruption or source mismatch never becomes a fresh successful recovery by discarding the failed observation. Partial pre-report candidates remain subject to existing recomputation. Full process/power-loss equivalence is not claimed.
