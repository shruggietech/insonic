# S023 dependency research

## 2026-10-13: Immutable elections and compatibility

Three read-only Spec Kit research agents assessed automation, site and Go runtime dependencies plus the reviewer-owned requirements checklist. No blocking ambiguity or required toolchain change remains.

| Source PR | Captured PR head | Election | Immutable action identity |
| --- | --- | --- | --- |
| #77 | 834c8d6dcb36ca889961a9a1e3069bfd09349a02 | cache 6.1.0 | 55cc8345863c7cc4c66a329aec7e433d2d1c52a9 |
| #78 | b4f20e4f24c469dd1aee5d22d441183a598e65d3 | Next.js 16.4.0 | locked package integrity |
| #79 | f45c69d9b3ce97806076cbb3c0da328ca1f568f4 | upload-artifact 7.0.2 | cf430e030ddbb5b0abf93d22962f4752f3646cd9 |
| #80 | bcb585cfc7be19d8a659547bf497822c39d38734 | cache/restore 6.1.0 | 55cc8345863c7cc4c66a329aec7e433d2d1c52a9 |
| #81 | 988123da2c1b24b1f735d0a41faa0dad7aaac5e1 | setup-node 7.1.0 | 949feb2413d6458794dcd2491c4babbbce0c15c1 |
| #82 | 0a1de66fbf591a26b6a1488a977d637508de8dce | download-artifact 8.0.2 | 9000827ccba6bdab643e8b6fd33ac0654aef8333 |
| #83 | 8277ee854aa54e043bcf20a94cf7149b1edd4f73 | go-isatty 0.0.24, crypto 0.58.0, net 0.60.0 | resolved Go sums |

### Automation transport

Decision: preserve ZIP transport explicitly with archive true on all 14 uploads; declare digest-mismatch error and skip-decompress false on all 12 downloads. Retain names, paths, revision/attempt fences and independent project verification.

Rationale: [upload 7.0.2](https://github.com/actions/upload-artifact/blob/cf430e030ddbb5b0abf93d22962f4752f3646cd9/action.yml) defaults archive true, while direct upload ignores configured names. [download 8.0.2](https://github.com/actions/download-artifact/blob/9000827ccba6bdab643e8b6fd33ac0654aef8333/action.yml) rejects digest mismatch by default and differentiates archive/direct transport. Explicit inputs make the intended boundary reviewable and guard against later weakening.

Alternatives: relying solely on defaults obscures the project contract; direct uploads break grouped/source/candidate names; warning-only mismatches violate FR-003. Keep current verified cache keys and content/recipe/compiler checks. [Cache 6.1.0](https://github.com/actions/cache/blob/55cc8345863c7cc4c66a329aec7e433d2d1c52a9/README.md) keeps restore inputs/outputs stable and tolerates unavailable saves. Action Node 24 execution is distinct from [setup-node's](https://github.com/actions/setup-node/blob/949feb2413d6458794dcd2491c4babbbce0c15c1/action.yml) elected Node 22 build tool. Hosted runner requirements are satisfied.

### Renderer

Decision: import the exact site lock election without adopting new app architecture or changing historical status records.

Rationale: [Next.js 16.4.0 package](https://github.com/vercel/next.js/blob/v16.4.0/packages/next/package.json) accepts existing React 19.3 and Node >=22.12. Its lock update includes next, @next/env and eight @next/swc platform variants only. [Static export](https://nextjs.org/docs/app/guides/static-exports) remains supported. The [16.4 announcement](https://nextjs.org/blog/next-16-4) does not require existing apps to adopt Cache Components. Existing Pages preparation already retains legacy nested segment URLs using verified exact replacements; do not expand aliases without a reproduced need.

Alternatives: broad peer/toolchain upgrades or architectural migration are unsupported scope expansion. Qualify clean locked installs, public/offline exports, release-target rendering and read-only Pages retention. Clarify site/README.md's fixed old candidate assertion without rewriting frozen source status.

### Runtime closure

Decision: import the exact five-module closure from #83. Requested direct/indirect elections also require x/text 0.43.0 and x/sync 0.24.0; retain x/sys 0.49.0 and Go 1.27.1.

Rationale: [crypto manifest](https://github.com/golang/crypto/blob/v0.58.0/go.mod), [text manifest](https://github.com/golang/text/blob/v0.43.0/go.mod) and [isatty manifest](https://github.com/mattn/go-isatty/blob/v0.0.24/go.mod) establish the closure and compatible language minima. Terminal detection changes merit real pipe/file descriptor coverage; existing vault lifecycle/tamper and transport/native tests remain authoritative. Net's tokenizer behavior still requires actual offline export checks. Successful HTTP/1 fixtures do not establish exercise of upstream HTTP/2 fixes.

Alternatives: omit required transitive upgrades or manually duplicate runtime pins in native dependency JSON. Existing package notice generation derives Go module identity, while native dependency pins remain unchanged. Preserve historical changelog decisions and append current ones.

### Tracking and evidence

Decision: one consolidation issue/official PR; originals stay open until owner merge. The first automatic official review is round one; at most one second request. Exact-head checks and all threads are inspected directly. Source/candidate artifacts must belong to one workflow attempt; full rerun is needed after an interruption that prevents that boundary. No release/tag/deployment or real-model inference.

The installed checklist helper required plan.md, so plan template setup preceded the successful checklist-template resolution; full research/design still followed specification and clarification. Hooks are absent.

## Source review inspection

2026-10-13: All seven captured source commits were inspected and imported preserving bot authorship. GitHub confirms all seven heads still match the elections above and all seven original PRs have zero review threads. Issue #84 is linked to Project 7, Slice S023, In progress.
