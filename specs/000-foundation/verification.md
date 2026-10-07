# Foundation verification

**Date**: 2026-10-05\
**Status**: Published foundation validated; one upstream semantic proof failure

**Local tooling runtime**: Node.js 26.5.0 on Windows

## Pre-publication local evidence

| Check | Result |
| --- | --- |
| Root locked dependency installation | Pass with lifecycle scripts disabled |
| Repository integrity | Pass: UTF-8 without BOM, corruption/conflict checks, Markdown structure, top-down Mermaid, links, navigation, version alignment and offline brand inventory |
| Focused maintainer tests | 41 pass: six updater tests, seven checker tests, thirteen mocked GitHub setup tests, twelve schema-contract tests and three theme-preference tests; actual hosted calls remain unexecuted |
| JSON contracts | 12 Draft 2020-12 schemas, 10 registered document contracts and 11 full examples pass strict compilation, descriptions, version matching and master validation |
| Contract reference and timestamps | Generated backend/query alternatives, conditional requirements, field examples and shared definitions appear in the exported reference; audit timestamps use the official Cueson `{iso, unix_ns}` shape; malformed/legacy timestamp and cross-backend configuration cases are rejected |
| Staged tracking | 15 sequential application slices match the issue manifest and internal roadmap; dependencies and milestones agree without cycles; closed-project preservation passes mocked setup checks |
| Spec Kit integration | Official Specify CLI 1.0.8 initialized Codex skills; all ten integration manifests match their files |
| GitHub YAML | Five workflow/template/config files parsed successfully |
| Static documentation | 25 expected pages in each full-site/help export with internal links and anchors checked; Next.js adds its not-found output |
| Browser inspection | Public landing and all 21 specification pages, including 15 diagrams; desktop, 390px and 320px in both themes; no page errors or horizontal overflow |
| Landing behavior | Exact upstream slogan/short description and metadata, visible radial background gradient, pointer/keyboard archive-preview tabs and working download-section CTA passed; three platform packages clearly remain unavailable |
| Runtime loading | Landing requests only the 2,000-byte shared controls plus the blocking theme initializer, with no Mermaid request at 320px, 390px and 1280px in either theme; every documentation route retains the Mermaid runtime; 21 pages and 15 diagrams render without page errors; offline diagram rendering and persisted theme passed |
| Archive layout | Widget height, following section position and document height stay fixed across pointer/keyboard tab changes at 320px, 390px, 720px, 900px and 1280px in both themes; enlarged text and longer panel content also remain stable; inactive panels are invisible, inert and absent from the accessible tabpanel list |
| Theme behavior | Icon-only sun/moon switch works with pointer, Space and Enter; initial mode follows OS changes, explicit mode overrides the opposite OS preference, saved mode survives navigation/reload, logos/diagrams recolor, denied storage still permits a page change, and offline preference survives reload |
| Syntax highlighting | JSON has five distinct token colors in both themes; measured minimum text contrast is 7.3167:1 dark and 5.4701:1 light; CLI executable/flags/placeholders and shell strings produce token spans |
| Diagram review | Four single-path procedures became numbered steps; 15 branching, relationship, dependency, feedback or retry diagrams remain top-down |
| Offline help | Direct `file://` entrypoint, generated contract tables/examples, master-schema download, changelog and architecture diagrams passed; no public landing markup is bundled |
| Consuming site lint | Pinned upstream ESLint and stylelint passed |
| Site dependency audit | Zero reported vulnerabilities at the verification date |
| Public scope review | Documentation, planning and setup files contain no private source-directory mentions or case-specific material; numbered plans are outside public docs, and changelog is the only history surface |
| Glossary | 111 terms, 58 distinct primary learning resources and per-term links to local usage; major-release refresh is part of governance |

Node processes, the preview server and research tooling used hidden noninteractive launch with redirected I/O on Windows. No visible console was required. Site assets are local; exported documentation needs no production Node server or CDN.

The manual brand updater successfully imported the current formal kit and performed a read-only freshness comparison. All 574 integrated files retain exact upstream bytes. Pinned glyph validation passed all 15 checks. Full semantic verification completed 36 checks with 35 passes, one failure and zero skips. The remaining delivered proof hash mismatch is documented in [brand conformance evidence](brand-conformance.md); full semantic conformance is not claimed.

## Publication evidence

2026-10-05: With explicit owner authorization, the public [shruggietech/insonic repository](https://github.com/shruggietech/insonic) was created and the foundation was pushed directly to `main` as commit `95ea49992f5b75e82db9ef4097a76bc9fcad7e0e`.

| Check | Result |
| --- | --- |
| Initial hosted CI | [Run 37347062112](https://github.com/shruggietech/insonic/actions/runs/37347062112) passed on the initial commit: `foundation` in 9 seconds and `docs` in 28 seconds; the run completed in 31 seconds |
| Hosted build artifacts | Full documentation and offline help were uploaded successfully, with artifact names tied to the initial commit |
| GitHub tracking | All 15 application issues were filed, assigned across five milestones and added to the linked [insonic Project](https://github.com/orgs/shruggietech/projects/7); required fields and statuses are populated |
| Maintainer setup readback | Live `github-bootstrap.mjs --check` returned `complete: true`, with no missing labels, milestones, slices, Project fields, statuses or membership |
| Merge and review settings | Squash merging enabled; merge commits and rebase merging disabled; automatic branch deletion enabled; `main` requires `foundation` and `docs`, one approving review, current checks, resolved conversations and linear history; force pushes and branch deletion disabled |
| Security reporting | GitHub private vulnerability reporting enabled and confirmed |

Branch protection permits administrator bypass for owner-directed repository administration. Automatic deletion is configured; its behavior after a merged pull request has not yet been exercised. This publication creates no release, native installer or public website deployment.

## Prepared and unverified behavior

The pre-publication GitHub bootstrap checks used isolated mocks. Repository settings, issue/milestone creation and Project fields were prepared without hosted writes in that phase. The publication evidence above confirms hosted setup and CI; branch deletion after a merged pull request remains unexercised. CI jobs each have a ten-minute timeout.

No native application, installer, media processing, model worker, credential backend, database pairing or cross-platform IPC was executed. These are application-slice acceptance items starting with S001. Cueson's current SRT/WebVTT zero-cue limitation remains an explicit S007 compatibility item; ASS/SSA no-dialogue behavior is treated separately. The local renderer requires a tagged changelog snapshot for released documentation; no release promotion was executed. The pre-publication phase created no Git repository, remote, push, release or upstream issue.

## Handoff

The owner can review the [v0.0.0 specification](../../docs/v0.0.0/index.md), [numbered delivery plan](../roadmap.md), [JSON contracts](../../docs/v0.0.0/contracts.md) and [repository README](../../README.md). The public root lander is in `site/out/`; installed help uses `site/offline/`. Proceed with [S001](https://github.com/shruggietech/insonic/issues/1) to prove the proposed Go/Wails and native dependency contracts before implementing the media pipeline.
