# S019 verification

## Scope and authority

S019 implements #56, #57, #58 and #61 on codex/019-cli-operator-feedback. Spec Kit specify, clarify, requirements/operator checklists, plan, research, data model, contracts, tasks and analyze preceded implementation. The 14 functional requirements and four stories map to the sequential task inventory. No critical or high consistency finding remained before implementation. Explicit owner authority covers push and an official PR, review resolution and at most one requested second round; merge, tagging and release remain owner actions.

## Implementation and acceptance evidence

| Requirements | Concrete implementation and verification |
| --- | --- |
| FR-001, FR-004 | Shared bounded human renderer, exact integers, readable document pages, empty inventories, document/paging and lease actions; human/operation result tests. |
| FR-002, FR-003 | Allowlisted generated reasons at actual missing-input, revision, source-map, credential and backup predicates; stored/locked versus owner-loaded state; contracts/schema, secrets, backup, app and CLI regressions. |
| FR-005 | Startup-only 4096-byte report, shared ten-second deadline, one child wait owner, bounded protected input, early-exit/malformed/stalled/concurrent-owner fixtures; installed cold-vault failure and detached-client survival qualification. |
| FR-006 | Verbose/debug/quiet and elected severity/private rotating logs; destination preflight, bounded retention, generated-field redaction, concurrent-file exclusion and access tests. |
| FR-007-FR-009 | Same-process changed-state/stage/count observer, terminal-only progress, retained last state, partial item/date warnings and continuation/retry commands; clean redirected/JSON streams and exact machine exits. |
| FR-010-FR-012 | Incompatible alias target inspection with strict execution election; response-only acquisition/verification readiness, named immutable file roles and leases, empty/current/invalidated dataset preparation readiness; model/app/CLI/schema regressions and actual fixture acquisition/materialization/release. |
| FR-013 | Current CLI, credential, model and dataset documentation and changelog align; current 1.0.1 remains an unreleased candidate and historical 1.0.0 documents/contracts and immutable stored identities remain unchanged. |
| FR-014 | Full configured checks, affected native/package qualification and final-head hosted checks/reviews are delivery gates. |

## Convergence corrections

1. T021: Elected log preflight originally followed workspace initialization. A failing mutation-free test and source CLI qualification exposed the ordering; preflight now precedes effects and both pass.
2. T022: An exited child's descendant could retain the report pipe. The failing bounded fixture now passes through explicit child-exit observation and a short report-consumption grace, retaining detached lifetime.
3. T023: Publication metadata mismatch and malformed stored manifests need proven damaged-resource context. Failing metadata/readiness regressions now pass without conflating storage unavailability.
4. T024: Quiet bounded output must retain an omission warning/retrieval route; completed work must not offer cancellation. Failing operator result tests now pass, and empty failed results no longer print an unrelated unset value.

## Local checkpoints

The initial full Go tests/vet, 84 Node tests, 141 Python tests, frontend type checking/tests/build and 78-page online/offline site build passed. The extended source native CLI qualification passed, including 143 help paths, cold startup failure, detached ownership, role/lease handling and diagnostic redaction. Installed assets qualified.

The first relocated CLI package check encountered a readiness timeout. Direct cold and post-idle restart probes passed, and the exact archive's full relocation retry passed in 121.235 seconds. The first desktop archive check passed CLI/bridge gates but hit the unchanged 90-second WebView limit. A direct probe of that packaged executable passed every native journey stage in 24.281 seconds. These observations do not establish the cause or elimination of those intermittent failures. Both variants will be rebuilt/requalified after final code corrections before delivery.

## Remaining delivery and verification limits

Final local checks, current-byte relocated package qualification, final-head CI and external reviews remain pending at this checkpoint. Local build receipts before commit identify the dirty source state rather than pretending to be clean revision evidence. Hosted CI must qualify the exact submitted head, including platform packages and alternative-backend fixtures. Acoustic inference, independent acoustic quality, product publication, merge and release are not performed by this slice.

The warning regression also covers exhaustion of the line budget by 64 warning-bearing items in quiet mode; the final omission notice survives both collection and line bounds.

## Final local checkpoint

All required checks remain green after the model and warning corrections: repository/schema integrity, 84 Node tests, full Go tests/vet, 141 Python tests, frontend checks/73 tests/build and the 78-page site/offline build. The last human line-limit correction additionally passes the full CLI package unit suite. Final rebuilt and relocated Windows archives both pass all original and added gates. CLI qualification took 124.891 seconds; desktop took 209.516 seconds, including bridge and native WebView. Exact archive hashes and honest precommit source identity are retained in qualification.json. No gate or timeout was weakened.

Final implementation convergence checked 14 FRs, six SCs, 16 acceptance scenarios, six plan decisions and five constitution principles. The four appended convergence tasks are implemented and verified; no implementation finding remains. The clean convergence pass leaves tasks.md unchanged. T020 is the remaining external delivery task.

## Initial external review

Codex reviewed d33ad1d and reported two findings. The first concerned the global human-field blacklist hiding public settings values, the doctor data directory and arbitrary query columns. Regressions reproduced each omission. Rendering now omits only known credential input, durable work internals and explicitly requested encoded document data, marking omissions; query result columns remain readable regardless of their names. The second concerned a missing credential in a locked vault being described as stored. Its failing missing/stored regression now passes: locked-manager state is independent, and retained-value wording requires stored=true. The installed CLI qualification also checks all three public result forms and missing locked-credential output. No diagnostic-log redaction rule was relaxed.

The corrected installed CLI archive passes the extended settings, doctor, native-query and missing locked-vault regressions, all 143 help paths and the complete relocation gate in 130.766 seconds. Its diagnostic redaction checks still pass. Initial CI run [38183587412](https://github.com/shruggietech/insonic/actions/runs/38183587412) passed all 25 active jobs on d33ad1d, including all six packages and alternative backends, in 7m58s overall; the longest job was 346 seconds. Correction-head CI and the one authorized follow-up round remain pending.

## Final external review and startup correction

The second and final requested round completed on 740f77b and reported one P1: an unready child survived the startup deadline. A failing subprocess lifetime regression reproduced it. Startup transport now retains child ownership through strict report identity/state validation and a responding endpoint; deadline, cancellation, malformed or failed reports kill only the launched unready child and allow its sole wait owner to reap it. Accepted runtime ownership survives caller cancellation. Deadline, cancellation, rejected-report and ready-owner lifetime regressions pass.

A separate closing-owner regression reproduced the retry gap: the first child reports runtime_owner, the closing owner releases its lock, but the original client never launches a replacement. Startup now probes the authority lock and retries after release, with bounded per-probe attempts inside the unchanged shared ten-second deadline. It does not launch repeatedly while the authority lock remains held. Full Go tests/vet, repository/schema checks and 84 Node tests pass after these corrections. The first full Go run exposed an overly short one-second native-bootstrap test fixture; the fixture now uses three seconds, without changing the production deadline or package gates. Final rebuilt package checks and exact correction-head CI remain required. These reproductions establish the lifecycle defects; they do not establish the exact cause of either earlier package timeout.

## Clean correction-code qualification

Both relocated Windows archives built from clean c17f525 pass every original and added gate. CLI smoke took 120.5 seconds (archive SHA-256 15e0b2f0ed761eca38f5dbf18ad11ccea804780688459557043858bd741dccf5); desktop smoke took 213.141 seconds (71b728d1b3952bd4cdc096e75b16bd0fee5edc8bcecf3a793b306a0d8fdf2cd2), including the bridge and native WebView journey. Both include 143 help paths, typed failures, cold-vault failure, detached-owner lifetime, public settings/doctor/query fields, model roles/leases, private diagnostics and real audio/video import with partial-date feedback. Source-manifest identities match. The earlier intermittent failures remain recorded; passing corrected archives and deterministic lifecycle regressions do not establish every cause of the earlier failures.

All three reported inline findings were answered and resolved on official [PR #72](https://github.com/shruggietech/insonic/pull/72). Exactly two review rounds completed, on d33ad1d and 740f77b. The last review correction is in c17f525; no third request is authorized or sent. The combined second-round request included a security review, but the bot emitted only a completed Code Review row, not a separate security result. This is an explicit external-review limitation. Acoustic inference/quality and a release remain outside this slice.

## Owner handoff checkpoint

Clean correction-code CI [38185275620](https://github.com/shruggietech/insonic/actions/runs/38185275620) passed all 25 active jobs on c17f525, including all six platform/variant packages, native qualification and alternative-backend fixtures. All 26 tasks and all reported review threads are complete. Final convergence rechecked 14 FRs, six SCs, 16 acceptance scenarios, six plan decisions and five constitution principles against the implementation and qualification evidence. No unresolved implementation finding remains.

This evidence-only completion commit does not alter tested product or qualification code. Its exact final-head CI must also be green before owner handoff; the official PR records that run without requiring a self-referential commit hash in this file. The four issues remain in review until owner squash merge. Version 1.0.1 remains unreleased. Final owner review and merge are the remaining authorized boundary.
