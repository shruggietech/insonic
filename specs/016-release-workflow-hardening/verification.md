# S016 verification

## Completed candidate qualification (2026-10-11)

The clean functional implementation is `c617ae83311619d940d69cb27538bf4bdbe1585a`. Its publication-disabled release candidate passed all six package variants, corresponding-source/inventory validation, native engine/desktop/credential/bridge checks, alternative backends and both release-target documentation exports. No product tag, public release or live promotion was elected. Evidence-only completion commits require their own green PR check before owner handoff; the exact final identity is recorded in the PR closure comment.

| Complete hosted measurement | Run / attempt | Seconds | Result |
| --- | --- | --- | --- |
| Warm PR | [38114565871](https://github.com/shruggietech/insonic/actions/runs/38114565871), 1 | 451 | PASS |
| Warm PR repeat | [38114565871](https://github.com/shruggietech/insonic/actions/runs/38114565871/attempts/2), 2 | 438 | PASS |
| Publication-disabled release | [38114831035](https://github.com/shruggietech/insonic/actions/runs/38114831035), 1 | 417 | PASS |

All three include the entire required acceptance set and candidate collection, below the unchanged 600-second warm criterion. Verified source caches skip cold producers; unelected privileged publication/promotion jobs are explicitly skipped. These are full reruns, not failed-only attempts. The downloaded release candidate was independently revalidated through the release verifier, including every archive, inventory, clean revision, synchronized 1.0.0 version, source manifest and both documentation exports.

Receipts: [warm PR](evidence/warm-pr-one.json), [repeat](evidence/warm-pr-two.json), [release](evidence/warm-release.json), [six-package manifest](evidence/release-manifest.json), [corresponding sources](evidence/corresponding-sources.json).

## Meaningful local checks

Test-first failures preceded native helper/workflow lanes, Pages authority/destination/race/build boundaries, worker inventory/package election, staged source transfer and historical segment compatibility.

- `npm run check`: PASS, synchronized 1.0.0, 23 schemas, 21 contracts and 28 examples, including text/link/version/upstream-kit integrity.
- `npm test`: PASS, 83 tests.
- Python discovery: PASS, 137 tests, including actual private Windows credential lifecycle and source-transfer/package/Pages regressions.
- Ordinary and elected release-target site builds: PASS, 52 documentation pages each, including offline relative routes and anchors.
- [Baseline preservation](evidence/preserved-inputs.json): all 620 inventoried baseline/upstream files unchanged, including 46 v0.0.0 documentation/schema files.
- Retained JSON receipts: UTF-8 without BOM, LF, no mojibake, private local paths or authentication material.

## Measured cold-source corrections

Initial cold PR 38107213628 reached the existing 360-second Windows FFmpeg compile bound with `make -j4`; Linux/macOS passed. Invalid-revision dispatch 38107221600 rejected identity before qualification/publication; an obsolete initial dispatch was cancelled. Neither counts as acceptance. Bounded `make -j8` initially passed cold PR 38108058997 (configuration 69.281s, compile 340.766s, full job 498s), followed by warm PR 38109619434 at 400s/427s. Independent release cold run 38109990209 then reached the same compile bound (configure 81.938s, compile 360.031s). This exposed an incomplete correction, not permission to enlarge deadlines.

The staged recipe at `a7f5a357e58b5f7bc934386b8a9dcc48ac4b375f` passed two independent cold chains. Complete static dependencies precede configured codec compilation; final compilation resumes the verified same tree/prefix without re-extraction or reconfiguration. Original pins, decoder/optimization configuration and deadlines remain unchanged. All 973 staged objects and the archive retained exact hashes and integer timestamps; Make's up-to-date query passed before final linking.

| Independent cold chain | Complete run seconds | Configure | Codec compile | Final compile |
| --- | --- | --- | --- | --- |
| [PR 38111782311](https://github.com/shruggietech/insonic/actions/runs/38111782311) | 1855 | 82.140s | 297.844s | 140.171s |
| [Release 38111790848](https://github.com/shruggietech/insonic/actions/runs/38111790848) | 2003 | 85.344s | 264.484s | 191.922s |

Cold times remain separate from warm turnaround. The recipe/cache identity is unchanged through the final functional candidate. [Exact command/reuse evidence](evidence/staged-cold-source.json), [cold PR](evidence/cold-pr.json) and [cold release](evidence/cold-release.json) retain actual source-bound proof. Earlier complete measurements at `861b1757b3548bd12889ede6e11487758554f044` (PR 408s/391s, release 416s) remain historical in `pre-pages-warm-*` receipts; final acceptance uses the combined implementation above.

## Extracted Windows CPU journeys

Both journeys elected the clean Windows CLI package from the release above, without source rebuilding. Archive SHA-256: `11b699a48308bac72ad11809ead83f73827f128d998d989221ecf411b0ec0a12`; inventory SHA-256: `767542d03ddd30e3e761bdf0455d6fcb59236d40d6d2d77530185078945b736b`. Product/harness revisions both equal the functional candidate, with both dirty flags false. Each used the extracted native CLI, media tools and Cueson in a private path with spaces, without ambient loader fallback.

- [Controlled journey](evidence/controlled-journey.json): PASS in 99.328s, explicitly `controlled-fixture` worker origin. Synthetic results prove deterministic contracts, not acoustic quality.
- [Real journey](evidence/real-journey.json): PASS in 158.25s using the inventoried installed processing worker, pinned offline faster-whisper tiny and Community-1, two threads and CPU. Model file hashes/revisions, executable/tool/worker hashes, versions, timing and memory are retained.

Both completed import, processing, enrollment/profile/model retrieval, roster-constrained matching, native graph queries and self-contained backup/restore into an empty destination. Original stores were unavailable at their original paths. Current authority values/hashes and returned query values were compared exactly after restore, including dates, current documents/audio, model association and roster.

Actual matching returned `unknown`, approximately 0.677446, with zero automatic mappings. Enrollment is duration-weighted normalized embeddings, not base-weight retraining. Editorial counts and cross-clip consistency are not independent identification truth. Acoustic inference on Linux/macOS, GPU behavior and broader accuracy remain unmeasured. Operation/overall bounds were retained; no operator audio or persistent library was used.

## Existing Pages and signing

GET-only preflight confirmed `shruggietech/insonic`, legacy `gh-pages:/`, `insonic.io`, enforced HTTPS and the existing github-pages branch policy. [Actual verified archive preparation](evidence/pages-preparation.json) prepared 503 files / 22,931,291 bytes, retaining 127 historical files and 23 strictly verified segment compatibility URLs with CNAME preserved. Destination head `857a743dccb58238de5b1ce541e2b72c9f483d29` / tree `83d97d0d8948a8b5a872e2690d63d6e856bf40a5` were read only. Mutations: zero. Live deployment: not executed.

Isolated regressions verify published authority before mutation, parented nonforced updates, race rejection, uncertain-response reconciliation and exact Pages build identity. They do not prove a live deployment. Missing public pages/schemas or unmatched segments reject; compatibility bytes come only from the verified original archive. Generic-host election remains supported.

All six packages record publisher signing `configured=false`, `status=unconfigured`, `mechanism=none`, `identity=null`, `verified=false`. No publisher certificate or notarization was used. Relocated macOS Mach-O libraries and the desktop application seal receive ad-hoc signing/verification for loader integrity, not a publisher signature or notarization. Configured signing failures still reject before final hashes.

## Publication and bounded reviews

[Official PR #45](https://github.com/shruggietech/insonic/pull/45) is attached, closes #44 and is In review in the project. [Review evidence](evidence/review-closure.json) records two completed rounds: automatic review at `c1a51624501da3239e38c40a4a6bc6444e14b2fa` (thumbs-up, no findings), then the sole explicit request at `845aaf508c9d5bf34577652c245be53f285fe7d2` ([no findings result](https://github.com/shruggietech/insonic/pull/45#issuecomment-6105190815)). No third request was made. No inline findings or unresolved threads were observed. No separate security-bot finding arrived; independent security approval is not claimed.

Staged compilation, rebuild instructions and segment compatibility were added after the second reviewed revision and received meaningful regressions/full hosted qualification. Exact final evidence-commit CI and the final thread audit are recorded in the PR closure comment before owner handoff. Squash changes identity; launch requires a fresh clean-main candidate and separate product-publication authorization, per [launch.md](launch.md).
