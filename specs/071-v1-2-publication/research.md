# S071 research decisions

Three read-only planning agents inspected current release, verification and qualification tools. All unknowns are resolved.

| Decision | Rationale | Alternative considered |
| --- | --- | --- |
| Keep source 61a5ba35 and trusted clean current-main tooling distinct | release.py enforces protected-main authority and exact source | S071 branch as product source rejected |
| Prepare release.yml with publish/promote false | Produces release-target docs and required native/packages | Ordinary CI candidate docs and publish=true rebuild rejected |
| Require load_candidate plus explicit require_release=True on both docs archives | Candidate loading alone does not impose publication target | Metadata-only acceptance rejected |
| Retain unique release-candidate-ATTEMPT with run/input identity | Name omits source and artifacts expire after 14 days | Name-only or size-only trust rejected |
| Publish retained bytes locally through maintained commands | Draft inventory verified before publication, no rebuild | Redispatch rebuilding bytes rejected |
| Retain Pages output and reconcile exact uncertain state | S067 preserves staged parent/tree and original inventory | Blind retry or silent rollback rejected |
| Separate downloaded-byte, live-content and browser receipts | Each establishes different evidence | API digests alone as downloads rejected |
| Use full installed Chromium headless | GUI-subsystem executable avoids console-window launch risk | Console headless-shell rejected |
| Report Release lane timing and conservative qualification upper bound | No shared CI readiness gate exists in Release workflow | Applying ci-acceptance.mjs to Release rejected |

Relevant maintained surfaces: scripts/release.py exact_source/trusted/load_candidate/verify_documentation/publish/published/verify-public/pages-promote/pages-reconcile; scripts/website.py verify-deployed; scripts/check-live-site-ui.mjs; .github/workflows/release.yml; current qualification timing policy; prior internal S066/S067 evidence. Public verification network budget 600 s and browser 900 s include maintained limits; Pages shares 3600 s. Wrapper ceilings do not establish acceptance.

Guard regressions cover release, Pages, public verification, website and bounded HTTP reads. No new implementation tests are needed unless a concrete defect is discovered. Pinned Go 1.27.1 is required for independent setup-byte validation. Browser is installed Chromium 1248 full chrome.exe, Playwright 1.64.1.
