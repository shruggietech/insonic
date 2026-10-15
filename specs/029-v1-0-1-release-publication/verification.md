# S029 verification and delivery

## Authority and scope

2026-10-14: Owner authorized S029/#97 under Spec Kit autopilot, immediate push/official PR, at most two requested review rounds, final green CI/owner review and squash merge, followed by v1.0.1 release and documentation publication after that merge. Base main was clean at 3abbe8e30f27878e70824c732c5fd515267f5fd6; no v1.0.1 remote tag/release existed. #97 is Project S029 / In progress in milestone 12. Existing Pages/domain and unsigned disposition persist.

Native MSI/account/installer/WebView probes on the owner host are prohibited. All local tooling used hidden noninteractive children and owned headless Chromium contexts; no personal browser or desktop/session control was used. Hosted CI owns native product/package qualification.

## Spec Kit sequence

Installed hyphenated specify → clarify → checklist → plan → tasks → strict read-only analyze → implement → verification → converge ran. No extensions/hooks are installed. Research agents were read-only. Requirements checklist passes 8/8. The eight custom release-quality questions remain reviewer-owned and unchecked; owner autopilot authorization permits routine progress without a new approval gate or marker changes. Analysis found full requirement coverage across 19 tasks and zero critical/high findings; its report was persisted only after the read-only command. Convergence found no new unbuilt code; existing delivery/post-merge tasks remain pending.

## Regression and implementation evidence

2026-10-15 UTC: New site tests first failed on historical candidate prose/manifest (2 failures), and the new workflow test failed because release-target documentation qualifications were absent. They pass after the changes. The existing root test was updated to derive candidate/released state through publicationManifest; its candidate-preservation assertion remains mandatory. The highlights artifact gained its required H1 after the repository checker identified the missing heading.

The existing prepare() newer-version/no-overwrite guards and trusted-main publisher/Pages contracts are unchanged. Only current metadata and render overlays correct historical released status. Frozen historical source, schemas, governed kit bytes and all screenshot assets/receipts are unchanged.

## Local acceptance

- npm run check passed: 1,221 text files at that checkpoint, Markdown/version/YAML/brand integrity and 25 schemas / 22 contracts / 30 examples, version 1.0.1.
- npm test passed: 103 tests, zero skips/failures.
- npm --prefix site test passed: 10 tests.
- Python model/release/Pages/security suite passed: 246 tests in 36.861 seconds.
- Snapshot build passed: 111 expected documentation pages in each public/offline export, all relative links/anchors valid; KaTeX inherited-trust regression passed.
- Snapshot actual diagram checks passed: 16 diagrams in each public/offline export, both themes.
- Snapshot headless UI passed: 114 public + 111 offline HTML files, 18 case groups, 70 representative width/theme combinations, 73.5 seconds.
- Release-target build and checks passed with INSONIC_DOCUMENTATION_TARGET=release; 111 expected docs pages in both exports and KaTeX regression passed.
- Release-target diagrams passed: 16 per export, both themes.
- Release-target headless UI passed: 225 HTML files, 18 cases, 70 width/theme combinations, 67.3 seconds.
- Both UI runs recorded zero actual-site HTTP/WebSocket attempts/errors, a blocked owned WebSocket probe with zero server connections/upgrades/bytes, and complete owned context/server/fixture cleanup. Sampled text contrast was at least 5.08:1; targeted controls were at least 44x44px.
- The five release landing-page URLs identify the Windows MSI and four current archive choices. Actual remote download availability is deliberately verified after publication.
- Current and historical release-target index text is truthful; the actual historical source Markdown is unchanged. Master/current frozen changelogs match.
- Root visually inspected the release-target 390px light installation screenshot showing compact closed disclosures and released status.

Full local reports/screenshots are in build/s029-snapshot-ui and build/s029-release-ui; portable report evidence is in evidence/documentation.json. These working-tree exports are not an exact clean release candidate. Final clean-head hosted CI and the post-merge release pipeline establish source-bound product qualification.

## Limits and final delivery gates

Automated Chromium/visual checks do not certify screen readers, all assistive technology, other engines or complete WCAG conformance. No live/deployed behavior or acoustic accuracy is claimed from these checks.

Official PR/review rounds and final exact-head CI are pending at this pre-push snapshot. Final post-commit head/run/source/elapsed evidence is recorded on the official PR so source does not claim future CI results. A second requested round is permitted only once; no third request.

T016–T019 remain pending until owner merge. After merge, fetch/prune and select its exact reviewed squash-main SHA; verify clean source/checks and tag absence/identity. Create/push v1.0.1 at that source, then dispatch release.yml from main with revision, publish=true, promote_docs=true, documentation-transport=pages and the reviewed release-highlights.md content. Dispatch through structured JSON data, never shell-built multiline text.

Verify all nine official assets and their hashes, immutable tag/source, candidate signing/source evidence and published release state. Confirm exact Pages commit, original published documentation bytes, live build/version markers, latest/fixed/archive routes, schemas, five package links, company attribution and headless mobile/theme behavior. A failed promotion retries original published assets without rebuilding. Close #97 and Project only after those publication/deployment facts exist. No merge/tag/release/deployment has occurred in this preparation phase.

## Official preparation PR and review checkpoint

2026-10-15 UTC: Official PR #99 is open, attached, and in Project In review with #97. Opening review on 778f7cc reported a missing Decisions heading. Full master/current snapshot source has the heading at line 52 before dated entries at line 54, and actual release-target HTML has h3#decisions before that list. The finding was answered with immutable source links and resolved as already satisfied; no heading change was needed. Exactly one second/final security-focused request was sent at issuecomment-6184162045; no third request is permitted. Its result and final CI remain pending.

The initial CI attempt had an exact-key complete-cache miss because the available cache was scoped to PR #98. The same recipe default-branch cache was verified saved at 04:04:30 UTC, so the own PR attempt was superseded for a full same-head cache reuse run. A final release-wording pass also corrected the dated changelog sentence that still described 1.0.1 as unreleased, and refreshed its current snapshot. All substantive code remains as initially reviewed. Earlier local UI reports describe their recorded working-tree checkpoint; final clean-head hosted CI validates the final source. #97 remains open and actual publication remains after owner merge.

2026-10-15 UTC: The second and final request completed at 04:09:33 UTC on 778f7cc. The bot posted no major issues at issuecomment-6184195658. It labels this a Code Review despite the security-focused request; no separate security verdict is claimed. The only first-round thread remains answered/resolved. Both authorized rounds are exhausted; no third request was sent. The final source changes after that reviewed code are the small release-wording correction and these delivery records, with final exact-head CI still required.
