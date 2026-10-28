# S056 verification

## Local acceptance, 2026-10-27

- Original defect reproduced: authentic v1.0.1 receipt rejected for missing later terminal identity.
- New history/boundary regressions: 17 passing within the full suite. Source completeness, unsafe/link/nonregular members, manifest/receipt changes, first-use, signing, inventory, default modern evidence and nested authority are covered.
- Full Python suite: 680 tests, three skips, passed with pinned Go 1.27.1 and Windows locale C. Initial wrong-compiler/unsupported-locale prerequisites were corrected; no assertions or deadlines were weakened.
- npm run check passed. npm test: 210 passed. Site check/build: 140 pages; site test: 18 passed. git diff --check passed.
- New metadata subprocess owner is classified under the maintained Python timing policy; no timing limits changed.
- Maintained load_candidate verified all nine authentic original v1.0.1 assets against 2015 exact tracked source files (13.296 s).
- Read-only published retrieval/verification plus actual MSI Property-table mutations completed in 31.781 s. Both unchanged-receipt and rehashed-receipt mutations were rejected. Running verifier hashes and original asset/source identities are retained in a compact receipt for PR publication. Protected-main metadata uses an isolated clean main snapshot, separately from the patched verifier under test.
- Source/code convergence: eight functional requirements covered, no implementation gaps. Existing T014/T015 retain publication, hosted CI and review work. Local operation durations above do not claim hosted preparation/qualification accounting.

## Boundaries

No original tag/release/assets or selected source changed. No historical verifier code or product binaries executed. Original byte/receipt verification does not establish fresh installed-product runtime behavior. Hosted exact-head CI and external reviews remain pending.

## First hosted run and CI correction

[CI 40601804325](https://github.com/shruggietech/insonic/actions/runs/40601804325), head 8d0918f, remains rejected: candidate metadata polling received HTTP 502 before candidate validation, so candidate/elected acceptance failed. All product/native/package/installer lanes passed. Independent terminal accounting: 210 s preparation + 683 s qualification = 893 s combined; 28 successful jobs, 18 intentional skips, two failures. Eight receipt ZIP identities and six terminal receipts/48 cases validated. The [complete rejected audit](https://github.com/shruggietech/insonic/pull/193#issuecomment-6456304453) was recovered byte-for-byte before corrected CI.

Convergence T016 adds at most three transient metadata-read attempts under the original deadline. Seven new root tests demonstrate recovery, bounded exhaustion, original-deadline expiry, caller cancellation, sibling cancellation and refusal to retry authentication, malformed, TLS, redirect or unclassified errors. Forty-three focused artifact/readiness tests and all 217 root tests pass; root/schema checks and the rebuilt 140-page export pass. Original-release verification source hashes are unchanged. First automatic Codex round completed cleanly at 8d0918f; corrected-head CI and the authorized second round remain pending.

## Second review correction

The second and final Codex round reviewed 01f6659 and identified one P2 response-body transport gap in [discussion 4395426521](https://github.com/shruggietech/insonic/pull/193#discussion_r4395426521). An actual HTTP server sends successful headers and partial JSON, then closes its socket after the client receives the headers. The regression reproduced TypeError terminated / UND_ERR_SOCKET before repair and passes after extending the existing classified boundary through body consumption. Complete malformed JSON, oversized bodies, caller cancellation, TLS/redirect errors and authority checks retain refusal. Forty-five focused artifact/readiness tests, all 219 root tests and root/schema checks pass.

Superseded CI 40603888873 was cancelled to qualify the required review correction instead. The [complete cancellation record](https://github.com/shruggietech/insonic/pull/193#issuecomment-6456501601) was recovered byte-for-byte before corrected CI: 246 s preparation + 225 s qualification = 471 s combined; 44 returned jobs; no aggregate gate/qualification receipts, no accepted terminal cases. This is not acceptance. Both authorized review rounds are consumed; no third round will be requested. The correction is pushed as 6f53179ef5e3bdc0319fbf33e044f539390f6311. [The concrete fix/reproduction reply](https://github.com/shruggietech/insonic/pull/193#discussion_r4395486573) is posted and the sole thread is resolved. Corrected-head CI 40604985160 remains pending.

## Accepted implementation checkpoint

Implementation head 6f53179ef5e3bdc0319fbf33e044f539390f6311 passed [CI 40604985160](https://github.com/shruggietech/insonic/actions/runs/40604985160): 30 successful jobs, 18 intentional skips. Independent terminal audit: 252 s preparation + 714 s qualification = 966 s combined; seven receipt ZIP identities and six terminal receipts/48 cases validated. The [complete audit](https://github.com/shruggietech/insonic/pull/193#issuecomment-6456924875) was recovered byte-for-byte before the final record push. Tested merge b006b62af8653509a3b1655060e35a6d9033a38f and implementation head have identical tree 84f821a01de5e3e3bdc08504f97be268ea74719c.

Both authorized Codex rounds completed. The second round's one P2 is implemented, answered and resolved; zero unresolved threads remain. No third round was requested. All seventeen tasks are complete against the accepted implementation. This final commit updates internal delivery records only and still requires final-head green CI before owner handoff. PR #193 is the authoritative final delivery/receipt record; merging remains reserved for the owner.
