# S028 verification

## Scope and source

2026-10-14: S028 implements #93–#96 for candidate 1.0.1. #97 publication stays after owner merge. No tag, release, site deployment or merge is performed. Direct Git/GitHub and hidden noninteractive children are used; native installer/account/UI qualification remains confined to disposable hosted CI.

Main run 38633239324 failed the Windows 100ms lease-fencing fixture (live owner stolen), not a job timeout. The previous complete PR workflow 38551197434 took 10m14s. The owner-approved constitution 3.0.1 and 21 active job limits now allow 11 minutes / 660 seconds. Production operation deadlines and historical records remain unchanged.

## Local checks

- Real SQLite security suite repeated 10 times: pass (0.487s). It exercises actual ClaimLease/RenewLease against guarded durable live/expired states, unchanged live authority, takeover, owner/generation refusal and valid renewal. PostgreSQL remains elected in hosted adapter qualification.
- Focused CI/release JavaScript assertions: 16 pass. Windows package parallel models: 6 pass. Media source builder models: 21 pass. New assertions demonstrated the old budget/root dependency failures before fixes.
- Configured npm run check: pass (1206 text files, upstream kit integrity,25 schemas / 22 contracts / 30 examples, version 1.0.1).
- Configured npm test: 99 pass. Site tests: 6 pass. KaTeX behavior regression:pass.
- Full go test ./...:pass (44.079s). go vet ./...:pass (9.633s).
- Portable Python unittest inventory: 246 pass (35.508s).
- Full public/offline builds: 111 pages each, all relative links and anchors resolve (4.596s).
- Actual public/offline Mermaid rendering: 16 diagrams each, light/dark rendering passes (4.391s).

The initial restricted Go environment omitted required profile variables. A separate owned disposable profile restored them. A diagnostic ACL fixture also required the verified native Windows temporary-files location rather than the repository volume. The corrected full suite passed; no product change or test relaxation was used for those local prerequisites.

## Capture provenance

Four application figures were refreshed from clean e16f3a0bb7041a426d8a548bfbcd0c120239fa0f source, using unmodified compiled CLI/shared bridge, production frontend and an owned headless Chromium profile. The capture completed in 3.556s with zero external requests, zero browser errors and completed exact browser/server/helper/fixture cleanup. Both binaries embed clean source identity. The first attempt was rejected before frontend/process creation because an added test made the checkout dirty; the clean rebuilt attempt succeeded. No native UI probe, owner browser/profile attachment, desktop input or broad process/window cleanup occurred.

Public/offline asset checks verify all 13 screenshot assets and their receipts against current source hashes. The seven native authored installer figures retain their existing unchanged authoring-source identities; their runtime qualification runs only on hosted CI.

## Documentation standards audit

The audit covers shared current/historical articles, reference pages, contract/schema rendering, changelog, diagrams, code, tables and installation figures. Corrections address the mobile wrapped-link cloud, tablet reading width, hidden contents, inconsistent reference scaffolds, skip focus, inactive no-script theme control, navigation targets and local complex-content overflow. The shared-template pass corrected these observed findings:

| Finding | Implemented correction |
| --- | --- |
| Mobile topic links wrapped into a large cloud | Native, initially collapsed Documentation and On this page disclosures |
| Contents disappeared below 1200px; tablet article was squeezed | Contents remains available; reading layout stacks below 960px |
| References used a different scaffold and lacked active location | Shared frame and exact current reference navigation marker |
| Complex content lacked reliable keyboard containment | Semantic tables, labelled local scroll regions and focusable code/diagram containers |
| Inert theme control appeared without scripting | Server-rendered hidden control, progressive enhancement and OS-theme fallback |
| Explicit index.html normalization reloaded fragment links | Preserve in-page fragments; focusable main targets across article/archive/fallback templates |
| Header overflowed by 21px at 768px with 200% text | Wrapping header and constrained brand/navigation items; theme switch grows with text spacing so its icon remains legible |
| Some navigation targets were undersized | Governed 44px minimum for tested controls, including the home logo |

The audit uses the [W3C disclosure navigation guidance](https://www.w3.org/WAI/ARIA/apg/patterns/disclosure/examples/disclosure-navigation/) and [WCAG 2.2](https://www.w3.org/TR/WCAG22/) reflow, text resizing/spacing, keyboard/focus, names/alternatives, contrast and target-size criteria. The definitive browser pass completed in 60.5 seconds (60.2 seconds qualification), within its 180-second bound. It audited 114 public HTML files and 111 offline HTML files at 320px, plus 70 representative combinations at 320/360/390/768/1440px in both themes and 17 case groups. Keyboard/native disclosures, archive/fallback skip focus, no-JavaScript file reading, denied storage, 200% text, text-spacing overrides, forced colors and reduced motion passed. Sampled solid text contrast, including code syntax and diagram labels, was at least 5.08:1; targeted navigation/disclosure/home controls measured at least 44x44px. Zero external requests/browser errors and complete owned-context/server/fixture cleanup were recorded. The report is retained in evidence/documentation-ui.json; thirteen screenshots are in build/s028-docs-ui/ and the CI documentation-ui artifact. Root visually inspected the 390px light and 1440px dark installation pages; root also inspected the final 768px enlarged-text view after the theme icon correction. Raster-image and gradient-text contrast is not certified by this sampler.

Automated Chromium results and visual inspection do not certify screen readers, other browser engines, every assistive technology or complete WCAG conformance. Deployed public behavior is not claimed from source exports; #97 publishes matching release/site content after merge.

## Delivery snapshot

Official PR, external rounds and final exact-head hosted CI are pending at this pre-publication snapshot. At most two requested rounds are authorized. Every finding must be answered/resolved before owner handoff. Final reviewed head, run/source identities and elapsed 660-second comparison will be recorded on the official PR after verification.

## Published PR and review round tracking

2026-10-14: Official non-draft PR #98 is published on codex/028-docs-quality-ci, linked to milestone 12 and Project 7 with all four issues In review. Round 1 automatic Code Review completed on b733e87c7120e0f52ee31303f85ae90eed074a48 with no posted findings or review submissions/threads. Round 2 was requested once through @codex security review at issuecomment-6183273610 on the same head. This is the final authorized review request. CI run 38672612989 is still in progress; no final readiness is claimed from this snapshot.

## Final review remediation

2026-10-14: The final manual request completed on b733e87 and posted two findings. The bot labels the result Code Review despite the security command; no separate security verdict is claimed. P1 discussion_r4278537580 identified unintercepted WebSockets; P2 discussion_r4278537587 identified symlink traversal in export serving. Both are corrected in the browser checker. Each context registers a WebSocket route before pages, blocks every socket without forwarding, and records sanitized attempts. A real owned loopback probe records one blocked attempt, close code 1008, zero TCP connections/upgrades/bytes. Actual-site HTTP/WebSocket attempts and browser errors remain zero. Asset reads reject component symlinks/junctions, enforce real-root boundaries, open with NOFOLLOW where available, and compare exact BigInt regular-file identities before reading/caching. Real junction and final-file symlink fixtures plus HTTP/cache refusal regressions pass.

The owner host supports owned file symlink creation, verified without privilege changes; the unnecessary platform skip was removed. All 102 root JavaScript tests now pass with zero skips. The complete updated browser audit passes in 65.9 seconds (65.7 seconds qualification), with 18 case groups, 225 HTML pages, 70 representative combinations, 13 screenshots and complete owned cleanup. Hosted final-head CI and thread response/resolution remain the last delivery gates. No third review request is authorized or sent.

2026-10-14: Review fixes were pushed in dca10938742669ca8ed494ebc0fa4bed3e7301cf. Both threads were answered at discussion_r4278614192/discussion_r4278614365 and resolved; no finding remains open. The first workflow had an actual completed Windows-media cache miss and rebuilt its source cache before the fix push. It was superseded by review remediation after that cache completed; no failed elected check was observed. The final exact-head workflow and 660-second turnaround measurement remain pending at this source snapshot and will be recorded on PR #98. This avoids claiming post-commit CI results before they exist. No merge, release, tag or deployment occurred; #97 remains Backlog / Unassigned.
