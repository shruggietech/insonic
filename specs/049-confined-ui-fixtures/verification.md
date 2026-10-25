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
