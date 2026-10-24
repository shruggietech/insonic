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

PR #173 is published. PR CI 40127126725 on 3a079e4 failed Linux source and packaged help receipt admission after successful child exits; Windows archive help released naturally in 11.156 seconds. Its independent audit recorded 1,495 seconds preparation, 707 seconds qualification and 2,202 seconds combined. Timing acceptance does not satisfy functional acceptance. The isolated cold run 40127360673 also exposed a Windows runtime-directory fixture failure. Corrected-head CI and an isolated passing cold/full warm comparison remain pending. No merge, release, installation or owner-desktop GUI/input operation has occurred. A finite passing campaign will not establish a causal closure of #154.

## Codex round one

Both P1 findings reproduced before repair. Twelve actual sparse workflow layouts failed to import the timing loader; all twelve now start using isolated Python with only their declared files. The controlled Windows proof join expired at 1,800 seconds; it now accepts an exact valid proof at 3,005 seconds under the unchanged original 3,600-second phase deadline, with finalization reserve remaining. The root suite, integrity check and Go policy tests passed after repair.

## Codex round two

The second and final authorized round completed on 3a079e4 without major findings (review receipt 6388798075). Both individual first-round threads are resolved. No additional Codex review trigger is authorized; subsequent CI repairs remain subject to final owner review.

## Hosted receipt diagnostics

Strict receipt admission now reports bounded stdout identity, record counts and available progress when rejecting a successful child exit. Empty, duplicate, mismatching and oversized streams remain rejected. Receipt encoding failures now return a failed native qualification exit. These changes expose the Linux failure; they do not establish its cause or constitute functional acceptance.
