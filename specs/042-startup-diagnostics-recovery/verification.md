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

### Hosted denied-access fixture correction (2026-10-21)

Frozen candidate c07d123 in run 39600524816 failed the Windows core regression because its read-only-only DACL fixture still admitted creation in that CI context. The source-native Windows desktop and Linux product lanes passed. The failed job log is retained in build/s042/reporting-ci-core-failure-39600524816.log. This is a demonstrated test precondition failure, not evidence that startup error reporting failed.

The fixture now denies file-add rights explicitly and checks creation under a duplicated, impersonated current-user token with optional privileges disabled on a locked OS thread. It verifies its creation-denial precondition before testing Logger.Open, then reverts the thread token before cleanup. The process token and production code are unchanged. The focused native Windows diagnostics suite passes locally; a new complete candidate CI run must verify this correction.

### Hosted access-boundary assertion correction (2026-10-21)

Candidate ce6c7a3 in run 39602800929 correctly refused the inaccessible diagnostic destination on Windows CI, but its test required the creation message while the logger reported privacy/access refusal. The failed job log remains in build/s042/reporting-ci-core-failure-39602800929.log. Refusal was preserved; the assertion assumed an OS access boundary that differs between ordinary and rooted native I/O under that CI context.

The fixture still proves ordinary creation is denied and still rejects successful Logger.Open. It accepts only generated unavailable errors identifying creation, inspection or private-file access, each with a concrete correction. A separate actual read-only-lock regression requires the creation/open message and verifies that no log file is created. Logger.Open now distinguishes failed path inspection from regular-file ownership/ACL refusal instead of mislabeling every Lstat failure as a private-file problem. Focused diagnostics tests and vet pass locally; complete candidate CI remains required.

## Reporting increment accepted on main (2026-10-21)

[PR #164](https://github.com/shruggietech/insonic/pull/164) final head 848d8ee0fccf3d8d2b36471eae02876716cbd105 and tested merge d4cd28b46e4712d428a57b7def9866366f6ce6cf share tree 4bfbeee1416d3906148c0a2afc7e3bee009066eb with squash b50aae1ca29751b9dfe1efca3d62ef3f206d1e6a. PR CI 39605463861 passed all 48 terminal jobs: 259s preparation, 668s qualification, 927s combined. Exact-main CI 39608806227 passed all 48 terminal jobs: 253s preparation, 616s qualification, 869s combined. Both initial queues were zero.

PR gate 12349650621 SHA256 6717e6b3dc2455bdf088f825b443e6c3725e14c97e8c91556b6c92418cf5af14; main gate 12348779594 SHA256 e2ae1cf71c7b59d83aa57046d1c7629e0a53a062a24ed7eddbb4cb75057880a5. Independently audited evidence is sealed under build/s042/reporting-pr-39605463861 and reporting-main-39608806227, with manifest SHA256 a469e91dcfe1bde68dd479385348a3b498a0e082687661f3eb73315d73eb3d97 and 4731810d5713791b63f5580517148a717740b9edf0f97f6ff84982cfe54577a4. [External main evidence](https://github.com/shruggietech/insonic/pull/164#issuecomment-6317781979) closes #115 as fixed on main, without claiming a release or installed-desktop verification. Both failed candidates remain retained with their original outcomes. T014 is complete; the merged branch is removed.

## Recovery increment local acceptance (2026-10-21)

The typed first-failure path covers synchronous environment/version/bootstrap calls, failed/null asynchronous environment/controller/webview results, controller/event configuration, interrupted message pumping, owner destruction and process failure during startup or scoped setup. Fake native regressions assert no success fall-through or ordinary loop entry, original HRESULT identity, owning-thread cleanup after callbacks and native frames unwind, balanced owned references, late completion refusal and fresh-instance success. Scoped facades preserve the existing architecture-specific SDK methods. Wails changes exactly its Windows setup path and development/production loop guards; temporary caller-owned settings are released.

A pre-existing apartment-test assumption also failed on the untouched main baseline: Go worker reuse can inherit a COM context. The fixture now uses a fresh native thread, verifies the actual prior/apartment transition and balances only its own initialization. Its test correction changes no production apartment policy.

The controlled Windows fixture and its injected-capture tests require a matching generated private failure stage, normal child exit and released/nonrunning owned descendants. Changed/replaced blockers, nonzero exit, capture failure, truncation, malformed or extra receipt data and missing release proof fail while retaining the fixture. Only the exact exclusively created regular blocker is removed, after identity/byte verification. The subsequent valid child uses that same profile path before the remaining original native cases. No browser is launched by local tests. Hosted Windows source/package execution remains required.

Required local checks pass: root check/schema and all 173 Node tests; site check, all 17 tests and complete 111-page build/export; affected diagnostics/desktop Go tests and vet; audited binding selector retaining all five original tests plus every TestStartup case; pinned Wails lifecycle tests in both production/dev; 19 qualification, 16 module-notice, 3 dependency-transfer and 28 desktop-package Python tests. Logs are retained under build/s042/recovery-local. No output was written to Downloads.

Spec Kit reanalysis and convergence cover all 12 FRs, 6 SCs and 23 unique tasks. FR-001 through FR-006 retain the accepted reporting/diagnostics increment. FR-007/008 map to native first-failure/cleanup plus both framework loop guards; FR-009 maps to controlled failed-child and subsequent valid-process proof; FR-010 maps to exactly two checksum/scope-bound replacements, dependency recipes and linked-module notices; FR-011 retains all original cases and operation bounds; FR-012 maps to revision-bound publication/evidence and per-issue disposition. No missing, contradictory or unrequested buildable requirement was identified; no additional task is appended. Native/package and exact-main hosted acceptance remain T022, with historical cause disposition at T023.

Final staged-source audit verifies all 1,367 maintained module files byte-for-byte against their manifests, including every previously ignored upstream template member. New project files are UTF-8 without BOM, LF and free of observed mojibake. Project patch whitespace checks pass; the three Wails behavior diffs introduce no whitespace errors. The full new upstream snapshot retains pre-existing whitespace verbatim, as required by its checksum authority.
