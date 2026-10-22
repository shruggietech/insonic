# Research: Dependency refresh

## 2026-10-22

- Decision: Preserve both local desktop replacements and their pinned upstream sums. Rationale: tidy removes sums not needed by the active replaced graph, but package/source verification deliberately requires the pristine Wails v2.16.0 and WebView2 v1.0.23 identities. Alternative rejected: removing verifier checks. See [Go module tidy](https://go.dev/ref/mod#go-mod-tidy).
- Decision: Recompute the requested Go upgrades rather than copy the stale bot diff. Rationale: module classification and test-only sum churn must reflect current main. Unrelated linked downgrades are excluded.
- Decision: Derive capture Playwright authority from the root exact pin. Rationale: the current independent 1.62.1 constant rejects the requested 1.64.1 installation. Historical screenshot validation remains versioned by its original source.
- Decision: Keep site and desktop frontend React contexts separate. Rationale: #159 targets site only and runtime notices already derive installed versions.
- Primary release context: [Playwright 1.64.1](https://github.com/microsoft/playwright/releases/tag/v1.64.1), [Markdown-it changelog](https://github.com/markdown-it/markdown-it/blob/master/CHANGELOG.md), [Next 16.4.2](https://github.com/vercel/next.js/releases/tag/v16.4.2), [React 19.3.1](https://github.com/react/react/releases/tag/v19.3.1). Existing isolated renderer, semantic Markdown, static export and package checks exercise the relevant integration. No application-server coverage is inferred from static export.

No unresolved design clarification remains.

Go 1.27.1 reproduced pruning of all four retained upstream entries. Explicit downloads independently returned the original Sum/GoModSum values without writing them to go.sum; exact verified entries were restored. Both verifiers reject missing/changed authority and accept restoration. The requested MinIO update also changes its declared tool/test graph; only linked product modules enter package notices. Capture pin regression failed at 1.62.1 versus 1.64.1 before deriving the authority, then passed.
