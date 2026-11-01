# Verification: S066 v1.1.0 publication and independent verification

**Date**: 2026-11-01.
**Product status**: Published and independently verified.
**Delivery boundary**: Evidence PR publication, external reviews and final CI are pending. Owner final review and squash merge remain required.

## Published authority

[Official v1.1.0](https://github.com/shruggietech/insonic/releases/tag/v1.1.0), release ID 430635335, was published at 14:35:57Z from accepted S065 squash source `82a1bfd7e8f4e389d451789ec7840d874127f39c`. The immutable tag, software, documentation and schemas identify that source/version. The S066 branch records evidence and publication history; it does not change the accepted product source.

All ten original assets were freshly qualified, retained, loaded against exact clean source and checked for release-target public/offline documentation. A fresh independent retrieval of every public asset passed complete candidate validation, then an explicit comparison confirmed every name, size and SHA256 against the accepted originals, including release-manifest.json and SHA256SUMS. Publication never rebuilt the accepted bytes.

The original documentation is deployed at Pages commit `825cf4d38641f661418ad63d5dd7932cbf8e59a1`, parent `f2316b6bec2f53a6d67afe6db92bf853999a22f0`. Build 1332467174 completed for that exact commit. Read-only reconciliation verifies all 1,514 deployed files against the original prepared tree. The release verifier independently checks 1,433 original archive files through 1,599 required live URLs. The deployed-site verifier checks 1,514 exported files through 1,680 URLs and all six product download links.

Fresh bounded live-response rendering passes eight cases (320, 390, 768 and 1440 CSS pixels, light/dark), using 13 digest-verified resources. All captures were visually inspected. The owned headless browser closed and its isolated profile was removed. No owner-desktop GUI acceptance ran.

## Chronological execution and recovery

| Event | Result and authority |
| --- | --- |
| 04:03:26Z, exact-main CI 41270520010 | Accepted terminal audit for source 82a1bfd7; original archive retained. |
| 14:02–14:03Z, local guards | Foundation/schema checks, 222 Node tests and 114 Python release/Pages/public-verification tests passed. The first package-style Python invocation located no modules; discovery then executed all 114 tests successfully. |
| 14:04:37–14:22:27Z, [Release preparation 41314348683](https://github.com/shruggietech/insonic/actions/runs/41314348683) | 21 successful jobs, 22 intentional skips, complete native/package/installer/documentation acceptance. Publication and promotion jobs were intentionally unelected for preparation. |
| 14:22–14:24Z, candidate retention | Artifact 13087735446, original 886,435,968-byte ZIP SHA256 `90f4ab885ce734f377559626f57f92a90a1abaafe5f7d39d9ab846fa3e391c0e`; ten confined regular assets, full loader and release-target docs accepted. |
| 14:24–14:36Z, publication | Tag pushed at accepted source. The tool transport timed out after 300 seconds while the original publisher continued. Its observed exit was 120 and its final stdout was unavailable. GitHub independently reported the complete official release; the maintained publisher's exact-existing-release reconciliation then exited 0 with `reconciled=true`, without uploading again. No initial clean process exit is claimed. |
| 14:37Z, independent public retrieval | All ten downloaded assets passed maintained loading and exact original/download name-size-SHA256 comparison. |
| 14:39Z, Pages promotion/readback | Promotion updated the expected parented commit, then exited 1 with `Pages branch changed before its build was requested`. Subsequent original API readback showed the expected commit and completed exact build. The initial disagreeing response was not captured; a stale read is only a hypothesis. |
| 14:41–14:44Z, independent reconciliation/verification | Read-only original-tree reconciliation, explicit release public verification and deployed-site content/download-link verification passed. No second deployment, force update or rollback occurred. |
| 14:44Z, fresh live browser evidence | Eight cases and owned cleanup passed; all original captures inspected and hash-inventoried. |

## Timing

| Campaign or phase | Preparation seconds | Qualification seconds | Combined seconds |
| --- | --- | --- | --- |
| Exact-main CI, terminal authority | 177 | 818 | 995 |
| Release campaign, conservative observation | 15 before first repository check | At most 1,055 from that check through terminal collection | 1,070 |

Release has overlapping preparation lanes and no CI shared-readiness aggregate. Its first actual repository check is a conservative upper-bound boundary; later parallel preparation remains inside the 1,055 seconds. Original job/step timestamps and the separate Windows-scope preparation, qualification and cleanup intervals are retained in the campaign receipt. This is not a reconstructed CI aggregate. Both campaigns remain below the 1,800-second target and 3,600-second hard limit. Publication upload time is separately recorded and is not qualification time.

The explicit release public check took 118.262 seconds including local validation; its live-byte phase took 73.89 seconds. Deployed-site verification took 86.705 seconds overall, with a 76.703-second live-byte phase. Browser execution including cleanup took 7.566 seconds (7.869 seconds including the local wrapper). Actual maintained allowances are 600 seconds for public byte verification and 900 seconds for browser verification; wrapper ceilings do not establish campaign acceptance.

## Durable evidence and readback

[evidence/manifest.json](evidence/manifest.json) inventories 30 original JSON receipts, their original and compressed sizes/SHA256, plus eight retained original PNG identities and dimensions. Each gzip file was independently read back and decompressed byte for byte to its original receipt; original JSON passed strict UTF-8 parsing. These tracked copies survive workflow expiry. The full original candidate ZIP, extracted candidate, independently downloaded public assets, exact Pages data export, original API metadata and all captures remain under ignored `.local/s066/`. No ephemeral artifact expiry is the sole evidence authority.

The official release also retains its original manifest and complete asset digests. Metadata verification, independent binary downloads, live byte equality and sampled browser behavior have separate receipts and claims.

## Spec Kit and scope

Specify, clarify, checklist, researched plan, tasks and blocking analysis completed. Analysis found zero ambiguities, conflicting duplicates, constitution conflicts or unmapped tasks; all ten functional requirements and four success criteria have task coverage. The built-in requirements checklist is 16/16 satisfied. The 12 custom publication checklist markers remain reviewer-owned and unchecked; explicit owner autopilot authorizes routine progress without changing them.

No product, schema, release tooling or dependency bytes are changed by S066. Existing meaningful publication/integrity regressions remain intact. Implementation convergence and remaining delivery evidence are recorded before owner handoff.

## Follow-ups and evidence limits

- [#240](https://github.com/shruggietech/insonic/issues/240) tracks stale current release-guide timeout prose. Accepted release bytes remain frozen; the discrepancy is not reported as fixed.
- [#241](https://github.com/shruggietech/insonic/issues/241) tracks durable Pages attempt identity and investigation of the disagreeing post-update ref read. Exact deployed bytes and completed build are independently established despite the initial command failure.
- Real-model CPU/NVIDIA inference, acoustic accuracy, owner-installed behavior and broader browser/assistive-technology conformance were not elected or established. Native/deterministic package qualification is distinct. The original manifest correctly retains `inference_qualification=not-executed`.
- Windows assets remain unsigned with no verified publisher signature, as stated in the official release notes.
- External Codex/security review status and final evidence-head CI are pending. At most two review rounds are authorized; owner merge is outside this slice's autonomous boundary.

## Pre-PR verification

The evidence changes pass repository/schema checks, all 222 Node tests, all 18 site tests and the public/offline documentation build (198 checked pages each). Local builds are validation outputs only and do not replace any accepted release or deployed bytes.

Spec Kit convergence inventoried 23 requirement/success/acceptance items, ten plan decisions and five constitution principles. Publication and independent verification have no unplanned implementation gaps. Existing delivery tasks T018–T021 remain open for PR publication, reviews, terminal CI and final handoff; final convergence is deferred until those actual outcomes exist. No duplicate convergence tasks were appended and no product implementation was changed.
