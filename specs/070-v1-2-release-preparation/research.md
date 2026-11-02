# S070 research

## 2026-11-02 decisions

1. Decision: Use maintained release.py preparation to advance owned identities to 1.2.0. Rationale: its rollback, historical exclusions and document-version compatibility are already tested. Alternative: manual global replacement risks upstream and historical identity drift.
2. Decision: Copy current docs/schemas forward, then restore docs/v1.1.0 and schemas/v1.1.0 exactly from tag v1.1.0; restore the historical manifest entry from the tag and mark it released. Rationale: current main added answers pages/schemas after release; restoring only files would leave a nonexistent historical navigation page. Alternative: silently rewriting released history is rejected.
3. Decision: Integrate #245's x/net 0.62.0 version/hash changes while retaining four Wails/WebView checksum lines; adopt #246's complete Next 16.4.3 package/lock diff. Rationale: both source verifiers require archive and module authority. Existing JS/Python missing/changed/conflicting checksum regressions cover this; duplicating tests or removing guards is unnecessary. x/net/html is used by desktop offline-help validation, covered by Go and package qualification.
4. Decision: Add answers to current candidate advanced_cli metadata with a failing-first collect regression. Rationale: current desktop guidance already identifies dedicated answer controls as pending; candidate metadata omits them. Historical fixture capability lists remain untouched.
5. Decision: Bootstrap genuine 1.2.0 hosted captures on a clean pushed commit before opening the official PR. Temporarily omit only current-version screenshot walkthrough blocks, retain audited bootstrap receipts, import original enumerated captures, then restore current-version embeds. Alternatives: relabeling old images or taking local GUI focus violates provenance or workstation constraints.
6. Decision: Run maintained local root/Python/Go/frontend/site checks, then exact-source hosted Windows/Linux packages, installers, default/alternative backend acceptance and seven-case answer campaigns. Target 1800 seconds, hard limit 3600, finalization reserve 60; preparation measured separately. Real-model inference remains unelected.
7. Decision: Open one official PR, add it to Project 7 as S070/In Review, and supersede dependency PRs only after owner merge. No new issue or release milestone is required. Automatic first review plus at most one requested second review; every finding addressed.
8. Decision: Final source-bound CI and review receipts live in the PR conversation with digest-verified readback and retained local originals, avoiding a commit containing its own future hash. Later post-squash publication requires accepted-main qualification and unchanged retained bytes.

## Research method

Three read-only plan agents inspected release/history/capture handling, dependencies/source integrity, and qualification contracts. Parent inspection corroborated findings in scripts/release.py, tests/release_test.py, site/lib/screenshots.mjs, docs/v1.1.0/desktop.md and .github/workflows/ci.yml. No unresolved technical question remains.
