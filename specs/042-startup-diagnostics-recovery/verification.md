# S042 verification

## Starting authority (2026-10-21)

S041 policy correction PR #163 is merged as 7463ff8cd2cae1a2c7097b3163fdeccdc927fcb1. Exact-main run [39593989682](https://github.com/shruggietech/insonic/actions/runs/39593989682), attempt 1, passed every elected check: 257s preparation, 664s qualification, 921s combined, zero initial queue. The 4s target overrun is advisory under constitution 4.0.0. [External evidence](https://github.com/shruggietech/insonic/issues/161#issuecomment-6315363109) completes S041 T016/T036; T029 has its existing PR #162 evidence. Final-S041 cold/warm campaign [39596125235](https://github.com/shruggietech/insonic/actions/runs/39596125235) is separately bound to that revision on its dedicated ref.

No S042 behavior is yet verified. Historical #154 causes remain unresolved; later evidence will distinguish returned errors, controlled callback failures, real package acceptance and issue closure.

## Design consistency (2026-10-21)

Spec Kit analysis: 12 functional requirements, 6 success criteria, 23 tasks; complete coverage with no blocking duplication, ambiguity or constitutional conflict. US1/US2 form the independently useful first increment; US3 remains a separate verified increment. Milestone 26 maps the active #154/#115 work without closing historical criteria.

## Reporting and private-file increment (2026-10-21)

The inherited-Modify regression first failed with native access denied when the new file requested WRITE_OWNER. The narrower current-owner path passes without that right, while new-file owner normalization retains its separately validated privileged path. Native ACL assertions verify one protected current-user full-control ACE and the selected owner on active, rotated and lock files. Read-only creation, contention, existing-file reuse, a foreign selected caller identity, replaced paths and the reparse identity guard remain covered.

Returned playback, window and close errors retain errors.Is/errors.As identity. Injected diagnostic/presentation failures leave the startup cause authoritative. Generated stage/numeric-code events persist without arbitrary cause strings or path details; successful launches do not invoke either sink. Bounded native feedback removes control characters. The suite uses injected presentation and private fixtures, without opening a desktop window.

Local checks passed: required root check and 171 tests; site check, 17 tests and the 111-page build/export; affected desktop/diagnostics Go suites and vet with desktop, production and system_ladybug tags; existing CLI TestElectedLogFailurePrecedesInitialization; gofmt and diff whitespace checks. The local hidden test launcher required explicit isolated Windows home/config/cache variables, and the replacement-path fixture required a delete-sharing handle to exercise the identity race on Windows. Those harness corrections do not change product permissions.

Evidence remains in ignored build/s042/: private-modify-red.log, private-modify-green.log, startup-reporting-red.log, startup-reporting-green.log and reporting-local/ checks. No new output was written to Downloads. Native dialogs are covered through injected sinks locally; real source/package acceptance remains the frozen PR and exact-main CI obligation. No claim is made that this explains the historical temporary-directory popup or fixes binding-level process termination, which remains the second increment.
