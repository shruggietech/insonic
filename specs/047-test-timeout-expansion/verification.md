# Verification: S047

## Local checks (2026-10-24)

- Root `npm run check`: passed (1,604 text files; 25 schemas, 22 contracts, 30 examples; version 1.0.1).
- Root `npm test`: 194 passed after review repairs.
- Python discovery: 583 tests passed, one existing platform skip, 412.522 seconds. Used the pinned Go 1.27.1 compiler and Windows-supported C locale; no test assertion was skipped to repair the earlier locale failure.
- Frontend: 108 tests passed; TypeScript and build passed.
- Site: 17 tests passed; production build and 111-page link/anchor checks passed.
- Focused native policy/progress/frontend-clock cases passed. Full model/backend suites and desktop provisioning/cancellation cases passed after fixture repairs.
- Go vet passed. Complete `go test -p=2 -count=1 -timeout=30m ./...` passed. The final controlled-clock process fixture split also passed its complete package suite in 4.971 seconds.

## Observed failures and repairs

The initial full Go run exposed short healthy fixture clocks under concurrent compilation and real filesystem work. Runtime owner fixtures now separate catalog prerequisites from the unchanged product startup clock. Direct model fixtures retain lease headroom without bypassing generation or cancellation fencing. Remaining process-containment fixtures use explicit readiness barriers and controlled expiry clocks. These are test harness repairs; ordinary product operation budgets remain unchanged.

## Hosted acceptance

Exact-head CI, the isolated cold/full warm comparison and external review remain pending publication. No merge, release, installation or owner-desktop GUI/input operation has occurred. A finite passing campaign will not establish a causal closure of #154.

## Codex round one

Both P1 findings reproduced before repair. Twelve actual sparse workflow layouts failed to import the timing loader; all twelve now start using isolated Python with only their declared files. The controlled Windows proof join expired at 1,800 seconds; it now accepts an exact valid proof at 3,005 seconds under the unchanged original 3,600-second phase deadline, with finalization reserve remaining. The root suite, integrity check and Go policy tests passed after repair.
