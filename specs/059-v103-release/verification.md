# Verification: S059 v1.0.3 preparation

**Date**: 2026-10-29
**Status**: Implementation and genuine captures complete; official PR CI/review closure and owner handoff pending.

## Scope and local validation

Spec Kit covers ten requirements, four success criteria, ten acceptance scenarios and twenty tasks. The built-in checklist passes 16/16. Preparation, historical authority, current routes/contracts, capture authority, post-merge guidance and all five constitution principles have been assessed; final external delivery remains tracked by T014–T018.

Maintained preparation aligns software, documentation and schemas at 1.0.3. Failing-first regressions cover frozen-history preservation, owned download tags and a single final changelog link. Published 1.0.2 docs/schema trees match their exact tag after corrections were copied forward. Original release exemptions and independent tool/model identities remain unchanged.

Local checks pass: root check and 219 tests; full Python discovery (703 tests, two declared skips); full Go tests and vet; frontend type-check/build and 109 tests; site 18 tests and public/offline export checks for 169 pages. Restored images pass the maintained source/version/hash/dimension validator. All 25 original captures received visual inspection. UTF-8/BOM, line-ending, placeholder and mojibake checks cover delivery text.

## Demonstrated repairs

Windows runtime directory lookup previously rewrote an already-correct private DACL. A failing setter-spy regression proves that write. Fresh exact owner/ACL validation now avoids it without caching authority; changed-ACL repair and failed-write refusal remain mandatory. Three protected-input concurrent cold starts pass in 0.37–0.70 seconds under normal temporary-directory/CPU settings and the unchanged ten-second product deadline. The subsequent full Go suite passes.

Bootstrap CI [40891705085](https://github.com/shruggietech/insonic/actions/runs/40891705085) failed Windows core because the truncated-response fixture shared a 100 ms deadline with expiry scenarios. A controlled 200 ms response reproduces the collision. Ordinary acquisition/transcript fixture cases now use maintained request headroom; deliberate expiry cases retain 100 ms. Three focused repetitions, the complete affected library suite (91.971 seconds) and vet pass. Production acquisition behavior/deadlines are unchanged.

## Retained bootstrap evidence

The [independent failed-attempt audit](https://github.com/shruggietech/insonic/issues/196#issuecomment-6499890764) records 28 successful jobs, two failures and 18 intentional skips; 254 seconds preparation + 685 seconds qualification = 939 seconds combined. Seven artifact ZIP digests and six source/package contexts validate 48 terminal, 18 import-recovery and six storage-failure cases. The aggregate remains rejected. Recoverable JSON is 91,079 bytes with SHA-256 c394031f4c01c7a6399b628cd935a0dda26f774a92a31c9e3c2277e47a133a56; independent platform readback and decompression match.

The separately successful native capture producers establish 25 PNGs and four unchanged original receipts at clean source 1ff68003c79658acc162e18a10f5c1fe1c58ba11. Both original capture ZIP API digests, every recorded source blob, image digest and dimensions were independently checked. Only exact receipt-relative entries were imported; 20 documentation embeds are restored. Their authority does not turn the failed aggregate into acceptance.

## Delivery boundary

Issue #196, milestone 43 and Project 7 track S059; #147 stays open. Final exact-source PR CI, individual external findings, at most two Codex rounds, independent retained receipts and owned cleanup remain required. Final receipts belong on the PR, outside workflow artifact expiry.

Owner merge, a fresh exact-main candidate, real CPU/fresh-restored processing acceptance, publication of those same retained bytes, Pages promotion and public verification follow [quickstart.md](quickstart.md). No owner-installed, physical-exhaustion, fresh acoustic or publication claim is made by this PR phase.
