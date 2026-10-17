# Verification: S034

## Scope and analysis (2026-10-17)

Exactly #122, #123, #124, #125, #128 and #129 map to codex/034-query-graph-correctness and this directory. Specify, clarify, requirements/domain checklists, plan, tasks and analysis completed in order. Seven functional requirements and five success criteria have task coverage; analysis reported zero critical/high findings, ambiguities or unmapped tasks. Constitution gates passed without exception. No extension hooks are configured.

## Regression evidence

Initial failing fixtures reproduced missing cross-page edges, dense-graph output failure, invalid advertised timezone forms, silent duplicate parameter overwrites, rejected search continuation, hidden human cursors and rejected trailing terminators. Fixed tests cover star/dense graphs at limits 1, 2, 100 and 500, bounded byte-heavy pages, earlier/later hubs and self-loops, more than 500 search matches, stale cursors, DST/day membership, local-zone capture, live schema-generated save dispatch, nested duplicate parameters, exact int64 and native statement/function/effect boundaries. Desktop tests cover relationship-only continuation and accumulated status.

## Local checks

- Full ordinary Go suite and vet passed, followed by full system_ladybug Go suite (65.703 seconds) and vet (14.703 seconds) with prepared native dependencies.
- npm run check and npm test passed outside the restricted sandbox, preserving symlink integrity tests. Python test discovery passed.
- Desktop type check and full frontend tests passed. Site check, tests and public/offline build passed.
- Early sandbox-only attempts could not create required symbolic links or execute all native prerequisites. The same checks passed through a console-free launcher with hidden, noninteractive children; no test was weakened.
- Convergence found graph motion effects missing new-edge dependencies and Local alias provenance retaining the alias. T016/T017 correct both and add the local-provenance regression; focused native checks (2.907 seconds) and frontend checks (15.078 seconds) passed.

## Packaged and hosted checks

Installed-package qualification now includes deterministic generated media/transcript cases for all six issues. It exhausts 607 cue matches and 607 document-cue relationships, checks timezone forms through the packaged runtime, submits the emitted save template, rejects duplicate parameters and preserves exact native int64 with a trailing terminator. Fresh Windows package execution and exact-head hosted CI results are recorded below when complete. Hosted qualification exercises both supported operating systems and alternate backends; local evidence alone does not establish those hosted outcomes.

## Publication boundary

Owner explicitly authorized push and PR creation, superseding the skill's default pre-push halt. At most two review rounds are permitted. No merge, auto-merge, release, inference or owner-media execution is authorized by this slice. #114 and #147 remain broader open tracking work.

## Installed-boundary correction (2026-10-17)

The first fresh Windows package run reproduced a top-level parser rejection of a valid search cursor as non-UUID, before query dispatch. T018 makes the rule command-specific: search accepts bounded opaque cursors; catalog list routes retain UUID validation. The regression now covers parseInvocation and parseExploreAlias together. Fresh package execution is repeated after this correction.

## Fresh Windows CLI result (2026-10-17)

Implementation head e953fcc rebuilt the Windows CLI package in 12.390 seconds with clean source identity. Installed qualification passed: 608 selected graph nodes, all 607 relationships, all 607 search matches, local/IANA/fixed-offset timezone paths, live schema-generated save example, duplicate parameter rejection and exact signed int64 native execution with a trailing terminator. The six-case fixture took 8.157 seconds (39.578 seconds including owned runtime idle cleanup). The complete CLI suite passed again in 4.046 seconds outside the restricted profile. No owner media or inference was used.

Convergence after T018 reports no remaining implementation gap. GitHub milestone 18 and Project 7 track exactly the six scoped issues; publication and subsequent CI/review outcomes are recorded on the linked PR.

## Alternate-backend correction (2026-10-17)

Hosted run 39006378115 caught ArcadeDB OpenCypher rejecting the semicolon in the shared exact-int64 backend case, although LadybugDB accepted it. T019 shares lexical validation and normalization, removes only the accepted trailing terminator and following comments before backend dispatch, and preserves semicolons inside literals plus statement/function/effect guards. Cross-dialect normalization and graph regressions pass locally; the corrected head must pass the live alternate-backend job before handoff. First code review reported no findings; review and CI outcomes remain tracked on the PR.

Native graph/application/CLI checks passed after normalization (46.859 seconds), and repository integrity passed. Additional grammar coverage distinguishes parenthesized read clauses from function names; qualified calls remain outside the declared pure subset, including comment-separated qualifiers. Graph guard regressions pass without expanding the pure-function list. Both first-round code and security reviews completed without findings on c4ef57e.

## Second and final review round (2026-10-17)

Code review found two P2 cursor contract gaps on f4b1be9: page-size changes retained the old digest, and stale row/edge offsets could produce invalid_request before digest conflict. Both failures were reproduced by dedicated regressions. T021 binds the effective limit (omitted and explicit default 100 are equivalent), excludes only cursor state, and compares digests before current offset bounds. Invalid offsets with a matching current digest still fail as malformed. Security round 2 completed without findings. Both code threads are resolved after the correction and its checks; no third review is requested. Final CI and thread state are authoritative on the PR.

## Windows package cleanup convergence (2026-10-17)

Run 39007271356 passed product, alternate-backend, native and Linux package gates. Windows desktop archive qualification passed its GUI/bridge receipts but failed removing the installed CLI after the fixed idle wait. The fixed delay did not establish image release. T022 preserves that full allowance, then observes Windows delete access without modifying bytes for at most ten seconds; a persistent lock still fails. Regression coverage checks delayed release, deadline failure, an actually loaded Python image and unchanged unused image bytes. No additional code or security review is requested after the two authorized rounds. Exact final-head CI remains required.

After T022, npm run check (1.406 seconds), npm test (1.359 seconds) and all 302 Python tests (34.282 seconds) passed through the hidden launcher. The live Windows probe rejected the currently loaded Python image, accepted an unused fixture image, preserved its bytes and released its handle for deletion.
