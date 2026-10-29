# Verification: S059 v1.0.3 preparation

**Date**: 2026-10-29
**Status**: Implementation and genuine captures accepted. PR #197 remains unmerged; its final-head CI/review receipts govern owner handoff.

## Scope and local validation

Spec Kit covers ten requirements, four success criteria, ten acceptance scenarios and twenty tasks. The built-in checklist passes 16/16. Preparation, historical authority, current routes/contracts, capture authority, post-merge guidance and all five constitution principles have been assessed; delivery is tracked by T014–T018 and independently retained PR receipts.

Maintained preparation aligns software, documentation and schemas at 1.0.3. Failing-first regressions cover frozen-history preservation, owned download tags and a single final changelog link. Published 1.0.2 docs/schema trees match their exact tag after corrections were copied forward. Original release exemptions and independent tool/model identities remain unchanged.

Local checks pass: root check and 219 tests; full Python discovery (703 tests, two declared skips); full Go tests and vet; frontend type-check/build and 109 tests; site 18 tests and public/offline export checks for 169 pages. Restored images pass the maintained source/version/hash/dimension validator. All 25 original captures received visual inspection. UTF-8/BOM, line-ending, placeholder and mojibake checks cover delivery text.

## Demonstrated repairs

Windows runtime directory lookup previously rewrote an already-correct private DACL. A failing setter-spy regression proves that write. Fresh exact owner/ACL validation now avoids it without caching authority; changed-ACL repair and failed-write refusal remain mandatory. Three protected-input concurrent cold starts pass in 0.37–0.70 seconds under normal temporary-directory/CPU settings and the unchanged ten-second product deadline. The subsequent full Go suite passes.

Bootstrap CI [40891705085](https://github.com/shruggietech/insonic/actions/runs/40891705085) failed Windows core because the truncated-response fixture shared a 100 ms deadline with expiry scenarios. A controlled 200 ms response reproduces the collision. Ordinary acquisition/transcript fixture cases now use maintained request headroom; deliberate expiry cases retain 100 ms. Three focused repetitions, the complete affected library suite (91.971 seconds) and vet pass. Production acquisition behavior/deadlines are unchanged.

## Retained bootstrap evidence

The [independent failed-attempt audit](https://github.com/shruggietech/insonic/issues/196#issuecomment-6499890764) records 28 successful jobs, two failures and 18 intentional skips; 254 seconds preparation + 685 seconds qualification = 939 seconds combined. Seven artifact ZIP digests and six source/package contexts validate 48 terminal, 18 import-recovery and six storage-failure cases. The aggregate remains rejected. Recoverable JSON is 91,079 bytes with SHA-256 c394031f4c01c7a6399b628cd935a0dda26f774a92a31c9e3c2277e47a133a56; independent platform readback and decompression match.

The separately successful native capture producers establish 25 PNGs and four unchanged original receipts at clean source 1ff68003c79658acc162e18a10f5c1fe1c58ba11. Both original capture ZIP API digests, every recorded source blob, image digest and dimensions were independently checked. Only exact receipt-relative entries were imported; 20 documentation embeds are restored. Their authority does not turn the failed aggregate into acceptance.

## Delivery boundary

Issue #196, milestone 43 and Project 7 track S059; #147 stays open. [PR #197](https://github.com/shruggietech/insonic/pull/197) is official, attached and in review. The accepted implementation run and round-one completion below satisfy the implementation handoff checks. Final administrative-source CI and the permitted second review are recorded independently on the PR before owner handoff.

Owner merge, a fresh exact-main candidate, real CPU/fresh-restored processing acceptance, publication of those same retained bytes, Pages promotion and public verification follow [quickstart.md](quickstart.md). No owner-installed, physical-exhaustion, fresh acoustic or publication claim is made by this PR phase.

## Accepted implementation and review

[CI 40895523055](https://github.com/shruggietech/insonic/actions/runs/40895523055), attempt 1, accepts head 1baa85f4c0e5433104981cb17b8ee6d329dbc6ad through test merge 434ef9da12e621882daf41e449c7e4f02ede5984. Their trees match at a20461154ce88330b3c29fce4c9a598e2461a120. All 30 elected jobs succeed with 18 intentional skips. Independent terminal timing is 246 seconds preparation + 701 seconds qualification = 947 seconds combined.

The [retained accepted audit](https://github.com/shruggietech/insonic/pull/197#issuecomment-6500401432) verifies seven original ZIP digests and six strict source/package contexts, covering 48 terminal, 18 recovery and six storage-failure cases. Independent platform readback and decompression match 92,208 JSON bytes, SHA-256 873864cc891810f5c55a63edc6a6a6c83314c0eb302301f4b3de141983bb0e86.

Codex automatic round one completed on 1baa85f with no findings and a thumbs-up; no review submissions or inline threads remained. The receipt is preserved in the accepted audit. Security review status is unreported. The final administrative commit changes internal Spec Kit records only; its final-source CI and permitted second-round receipt belong on the PR before handoff. No third round is authorized.

Owned implementation scratch and the experimental native temporary fixture were removed after verified external retention. Any subsequent final-head audit uses a new bounded transient lifetime and is removed after its own retained readback. Spec Kit convergence checks ten requirements, four success criteria, ten acceptance scenarios, seven plan decisions and all five constitution principles; no unbuilt product work or untracked gap remains. All twenty tasks are complete; final-head external receipts remain mandatory revalidation before owner handoff.
