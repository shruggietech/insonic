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
