# Verification: S060 v1.0.3 acceptance and publication

**Date**: 2026-10-29
**Status**: Specified and planned; candidate, CPU, publication and final PR delivery are pending.

The owner authorized publication from accepted main while reserving evidence-PR merge. The selected source is 72098446d435aa1e3d6d6726a9f07e04e0b227a6; exact-main CI 40912395099 passed and its independent audit is retained on PR #197. Issue #198, milestone 44 and Project 7 track this slice.

Spec Kit quality is 16/16; one publication-timing question was answered and integrated. The installed plan command directed one read-only research agent. Existing maintained tools support retained-byte publication; no product source/interface change is intended. All actual phase identities, failures, verification results and limits will be recorded here and in recoverable GitHub receipts.

Blocking analysis: 11 functional requirements, four success criteria and eight acceptance scenarios map to 20 tasks with 100% requirement coverage, no unmapped task, no unresolved ambiguity and no critical/high finding. Built-in requirements quality passes 16/16; no custom reviewer-owned checklist or extension hook is present.

Initial local validation passes root checks, 219 root tests, 18 site tests and public/offline export checks for 169 pages. A separate clean managed worktree selects the exact accepted main source and its release validator passes. Product source, versioned documentation/schema and the existing release changelog/highlights are unchanged.

## Initial CI scheduling failure

Run [40953107339 attempt 1](https://github.com/shruggietech/insonic/actions/runs/40953107339) tested head 9272bcb650d457d81a59aa94109091f190591777 and merge revision 55ca08bcec797b9577575e43733cdfddd732ff57. With every component unelected, readiness succeeded but required foundation/docs-build/docs inherited skipped preparation ancestors. Aggregate acceptance correctly failed. Terminal audit: 18 seconds preparation, 11 seconds qualification, 29 seconds combined; 2 successful jobs, 15 skips, 1 failure. Timing passed independently of the failed functional gate.

[Recoverable failed-run evidence](https://github.com/shruggietech/insonic/pull/199#issuecomment-6508229296) contains the original digest-verified gate ZIP, receipt, API run/attempt/jobs, bounded failure log excerpt and independent terminal timing. Comment readback, gzip decompression and retained JSON SHA-256 0452162cfe405403bae84c0991d2d88d5bd801a5e217354578bb1139e86e5198 matched before a repair push.

The evidence-branch repair uses explicit always() conditions with successful direct dependencies for foundation/docs-build/docs and the elected read-only live-site job. Readiness outputs remain required; no aggregate requirement, component election or clock changes. Existing CI scope/acceptance checks passed 43/43 and root check passed. Hosted full PR CI and an explicit all-components-unelected maintenance dispatch must confirm execution before final handoff. Selected release source and frozen 1.0.3 documentation remain 72098446d435aa1e3d6d6726a9f07e04e0b227a6.

## Read-only maintenance verifier failure

Maintenance [run 40954286503 attempt 1](https://github.com/shruggietech/insonic/actions/runs/40954286503) proved successful foundation, docs-build and docs after all component elections were false. It failed in the read-only website job, then correctly failed aggregate acceptance: the unchanged v1.0.2 manifest is 7,481,000 bytes and exceeded the old 1 MiB HTTP metadata default. The live job produced no receipt before this failure.

[Recoverable failed maintenance audit](https://github.com/shruggietech/insonic/pull/199#issuecomment-6508385507) retains the original gate ZIP/API run/jobs and independent terminal timing: 19 seconds preparation + 307 seconds qualification = 326 seconds combined; 5 successful, 11 skipped, 2 failed jobs. Exact comment readback, gzip recovery and JSON SHA-256 c162ace006ee8e2af7fc4581646cf804890bb69372ff9430262af0baa0991aa1 matched before another push.

The new reader regression failed first at the real manifest size, then all 13 website tests passed with a 16 MiB default. Negative checks retain explicit smaller bounds, reject oversized/incomplete bodies and avoid body reads for HEAD. A read-only live request with the repaired reader verified the original completed deployment and published metadata size/digests. Root check and 44 affected Node checks pass. The evidence-branch website repair is separate from the unchanged trusted-main release publication path.

The corrected read-only verifier also completed full content and download-link checks on the existing v1.0.2 deployment, reaching links-verified with all six product choices. This is a maintenance-verifier result for the existing deployment, separate from pending v1.0.3 public verification.

## Complete publication candidate

Release-candidate [run 40953106419 attempt 1](https://github.com/shruggietech/insonic/actions/runs/40953106419) succeeded from exact reviewed main 72098446d435aa1e3d6d6726a9f07e04e0b227a6. Original candidate artifact 12933866569 is 865,406,301 bytes, SHA-256 5f89bbd7752bce5268837ae994859e181d729d3d3c6b6b87a59703d4f1530878. Original Windows CLI receipt artifact 12932917672 is 145,094 bytes, SHA-256 5db4040c6e2641b8cdefc2db758bf1b10b6a2eb5a14c9ee03487b904b871b23e. Both API size/digests and confined regular-file extraction verify.

Maintained trusted-main load_candidate validates all ten original official files, four package variants, applicable MSI/setup, inventories, source/signing/terminal/recovery/storage receipts and both release-target documentation exports. The exact package-cli-receipt.json from its original bundle equals the manifest package object. Its 7,575,448-byte manifest remains unchanged, including package-time inference_qualification=not-executed; real CPU proof is separate.

Conservative release-wide timing begins qualification at the first repository check (20:54:19Z), retaining later parallel preparation inside qualification: 10 seconds preparation + 1,091 seconds qualification = 1,101 seconds combined, terminal 21:12:30Z. Original assets/receipts remain outside workflow expiry through CPU acceptance and verified publication. The clean local verifier required an empty owned build directory for temporary source-code comparison; no official asset or selected source changed.

## Accepted CI scheduling repair

Full PR [run 40954296570 attempt 1](https://github.com/shruggietech/insonic/actions/runs/40954296570) accepts head 962de414fdaea81857be72d146fd889b66931ff8 through test revision eb4a208510962723d1219f9a25fdd22c898dbd3a: 30 successful jobs, 18 intentional skips. Independent needs evaluation and terminal timing pass: 280 seconds preparation + 715 seconds qualification = 995 seconds combined.

[Durable accepted audit](https://github.com/shruggietech/insonic/pull/199#issuecomment-6508646257) verifies seven original ZIP size/digests and six strict source/package contexts (48 terminal, 18 import-recovery, six storage-failure cases). Exact comment readback and decompression match 101,519 JSON bytes, SHA-256 9eb68a27c26e76b27d8fe374426df274efb084120d8b9b6d287f95e1d9691118. This completed run was retained before pushing the demonstrated website-reader repair. Final applicable CI must revalidate the final evidence source.
