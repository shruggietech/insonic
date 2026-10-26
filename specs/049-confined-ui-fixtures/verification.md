# S049 Verification

## 2026-10-25 Local implementation

- Spec Kit specify/clarify: requirements checklist 16/16, zero unresolved owner decisions. Plan research resolved allocation identity and platform coverage. Blocking analyze: nine functional requirements and four success criteria covered by ten tasks, zero critical/high findings, no constitution violations or unmapped tasks.
- Test-first regressions failed before the new allocation contract existed; final affected filesystem/browser-launch/workflow tests pass 45/45 on Windows. Tests use actual nested temporary directories, partial marker/profile failures, replaced directories, root aliases, refused/false-success removal and unverified identity. No root fixture is allocated by regression tests.
- `npm run check` passes encoding/BOM/LF, links, YAML/JSON, upstream kit and 25-schema/22-contract/30-example integrity. Full `npm test` passed 201/201 before two additional affected tests; those additional tests pass in the final affected run. CI must run the complete final suite.
- Site tests pass 17/17 and the documentation build/export checks pass. Actual local Windows Chromium documentation qualification passes 114 public + 111 offline pages, 19 cases, 92.1 seconds. Its actual system-temp child was verified removed; browser contexts and server were closed. Detailed paths/screenshots are disposable local scratch until replacement by retained CI artifacts.
- Adjacent allocation audit: live-site UI uses actual temp, density uses ignored build, renderer uses TemporaryDirectory. Only the two repaired consumers extracted the Windows volume root.
- Existing timing target/hard limit/finalization policy is unchanged. The new wrapper is registered in the maintained deadline inventory and inherits existing process-tree/enclosing deadline semantics.

## Delivery gates pending

Exact-head CI, actual source-bound Windows/Linux application capture, both platform documentation browser receipts, elected terminal timing acceptance and at most two Codex review rounds remain required before owner handoff. No merge or release is authorized. Forced termination cannot guarantee finalization; unknown identity, changed ownership and unreleased-child/removal failures retain actionable residual diagnostics instead of claiming cleanup.

## Initial PR evidence and Windows correction

PR #177 initial head 0b00006, CI run [40245396391](https://github.com/shruggietech/insonic/actions/runs/40245396391), test merge 178d2ad: first Codex round completed with no findings and thumbs-up. Linux application capture and documentation browser qualification passed. Windows application capture failed at the first screenshot after allocating beneath the runner-profile temporary directory; browser/helper/server release and fixture removal all completed (7.931s). Its retained [failed-run diagnostics](https://github.com/shruggietech/insonic/actions/runs/40245396391/artifacts/12618512749) establish confinement/removal, not successful capture. Windows documentation CI was skipped after that failure.

The follow-up explicitly selects ignored repository build/ui-fixtures for public application captures to preserve the existing owner-path screenshot check. Documentation profiles keep system temp. Both use the shared canonical-parent/identity policy. A real default-parent regression and the full 204-test local suite pass; repository check passes. Initialization-cleanup residual diagnostics now survive the capture command boundary. Corrected-head CI and the authorized second Codex round remain pending and will be recorded in the PR handoff.

## Second review remediation

Round 2 identified unconditional repository scratch beneath a Linux home checkout. The final selector canonicalizes parent/home/program boundaries (including observed Windows short-path temp aliases), selects non-home system temp where possible, otherwise selects ignored .local/build/ui-fixtures outside home and program directories, and supports an explicit validated scratch parent when neither default fits. An actual bridge probe now returns a valid initial setup draft; the rejected build/ candidate returned invalid_request because it lay inside the compiled program installation. No production restriction or screenshot privacy check was relaxed. Final local root tests pass 205/205 and repository integrity/schema checks pass. Superseded run 40246245004 was cancelled after the confirmed finding; it is not acceptance evidence. Corrected-head actual browser and terminal CI evidence remains pending.

## Corrected-code acceptance (2026-10-25 local, terminal 2026-10-26 UTC)

- Code head: 6ff35df61faf80751d001b0bef1ffa1cbf868e43. [CI 40246841901](https://github.com/shruggietech/insonic/actions/runs/40246841901), attempt 1, tested merge 27f8fe0fef5f7bc41d8f58d9115872b4094da961: all 48 jobs terminal (30 success, 18 expected non-elections). Elected acceptance passed. Independent terminal re-read accepted with no failures: preparation 372s, qualification 655s, combined 1027s, evaluated through 2026-10-26T00:11:41Z. No clock reset or timing-policy change was used.
- Both platform application captures retained clean exact-source/binary provenance, four application images, seven density images and 96 scenarios. Linux used canonical system temp and completed in 20.486s; Windows used ignored repository build scratch outside home/program directories and completed in 30.405s. Both fixtures were non-root and verified removed. Detailed actual locations remain in qualification artifacts, not published screenshot provenance.
- Both documentation checks passed 114 public + 111 offline pages and 19 cases. Linux took 136.5s; Windows took 151.7s. Both actual system-temp fixtures were removed after browser contexts and servers closed. Windows reused the successful exact-source documentation export transfer before browser qualification.
- Local Windows exact code-head capture also passed (15.998s), with four images, seven density images, 96 scenarios and verified removal. Local documentation qualification passed as recorded above. Final local root suite: 205/205; site suite: 17/17; integrity/schema/build checks passed.
- External review cap: automatic round 1 completed without findings; manually requested round 2 produced one valid home-checkout finding, fixed in 6ff35df. The [individual thread](https://github.com/shruggietech/insonic/pull/177#discussion_r4372980933) has a response and is resolved. No third round or independent security review was requested or claimed.

| Retained artifact | ID | SHA-256 |
| --- | --- | --- |
| Linux application capture | [12619900143](https://github.com/shruggietech/insonic/actions/runs/40246841901/artifacts/12619900143) | 32769449205489bbe3a1c8ae1912d022782d14970e2eb904e582a6713cb45fde |
| Linux documentation browser | [12619106805](https://github.com/shruggietech/insonic/actions/runs/40246841901/artifacts/12619106805) | 53ea5d47328fcd4052a40cceb2393acfc3e3f7e37682770b50b99922936ba7aa |
| Windows native/browser receipts | [12619830991](https://github.com/shruggietech/insonic/actions/runs/40246841901/artifacts/12619830991) | 2624f83b09f6457971d629e9430a62e17df9eba1cbf9dc6769ff39904cca9ef7 |
| Elected acceptance | [12620015858](https://github.com/shruggietech/insonic/actions/runs/40246841901/artifacts/12620015858) | ee511dfaf9a09d786968a30ba63740d5f494362c9b0286240a91fe61dde101ba |

Downloaded artifact digests matched the recorded values before inspecting reports. Current source scope is complete; this evidence/task update will receive final-head CI before owner handoff, recorded in PR #177. The earlier failed/cancelled runs are not acceptance evidence. Forced termination remains outside the finalization guarantee; unknown identity or unreleased-child/removal failures report residuals instead of claiming cleanup. No installed-product change, release or merge is claimed.
