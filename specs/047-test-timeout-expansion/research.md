# Research: Expanded testing allowances

## 2026-10-24

- **Decision**: Expand nested test limits with a canonical qualification-only policy. **Rationale**: native help's 25s watchdog, Python help35s, frontend15/25s, native child18s and package matrix100s are independent early cutoffs. **Alternative rejected**: job-only or blanket text multiplier.
- **Decision**: Keep aggregate policy separate at1800/3600/60 and job watchdogs65m/preparation60m. **Rationale**: acceptance clock and process cancellation have distinct origins; cleanup/evidence need outer headroom. **Alternative rejected**: equating a job timeout with terminal qualification acceptance.
- **Decision**: Derive matrix parents and retain earlier absolute budgets with explicit remaining allowance. **Rationale**: Windows Exit runs12 children; full installer UI has12 combinations plus later cases. **Alternative rejected**: individually expanding waits without the actor, task scheduler and wrapper.
- **Decision**: Preserve product clocks and intentionally short injected failure fixtures. **Rationale**: user operation semantics and timeout regressions are distinct from test execution headroom. **Alternative rejected**: changing credential, transport and user-selected limits incidentally.
- **Decision**: Add fixed allowlisted monotonic help stages. **Rationale**: prior buffered module/fixtures stderr cannot identify the pending operation or prove a shared HRESULT. **Alternative rejected**: freeform logging, polling heartbeats or inferred causal closure.
- **Decision**: Preserve cache identity and cold/warm proof. **Rationale**: source-builder/process-tree policy inputs affect its recipe; isolated namespace attempt1 must be cold and full attempt2 warm. **Alternative rejected**: sibling-PR cache reliance or partial consumer reruns.
- **Decision**: Built-in requirements checklist covers this internal policy slice. No extra reviewer-owned checklist/approval ritual is added.

- **Decision**: Keep production backend setup at 35 seconds, with a private duration parameter for disposable real provisioning fixtures. Prepare catalog schema before cold-owner election fixtures enter the unchanged 10-second production startup clock; continue to assert absent owner, protected-input loading and one elected owner. **Rationale**: these are concrete shared product bounds, so widening the harness alone cannot provide headroom. Fresh provisioning and schema migration remain covered separately.
- **Decision**: Give direct model-service fixtures a 600-second work-lease floor through a test-only catalog decorator. **Rationale**: they lack the runtime heartbeat; production five-second progress checkpoints otherwise expire during real file publication. Retain actual catalog owner/generation/cancellation fences and test those fences.
- **Decision**: Use controlled time for true expiry tests and causal readiness barriers for real helpers. **Rationale**: a larger observation allowance must not turn deadline regressions into long real sleeps or change an original context deadline.

## Primary sources

- [GitHub job/step watchdog semantics](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#jobsjob_idtimeout-minutes).
- [Python bounded subprocess timeout cleanup](https://docs.python.org/3/library/subprocess.html#subprocess.Popen.communicate).
- #172, #154 and source at main7338fd4 retain exact incident evidence; source observations are not a causal diagnosis.
