# S031 validation

1. Install the pinned root and site dependencies with `npm ci --ignore-scripts` and `npm --prefix site ci --ignore-scripts` if absent.
2. Run `npm run check`, `npm test` and `npm --prefix site test`.
3. Run `npm --prefix site run build`. Both public and offline exports must pass route/anchor integrity.
4. On an isolated Linux runner with Chrome, run `node scripts/check-docs-ui.mjs --browser /usr/bin/google-chrome --out site/out --offline site/offline --output build/s031-docs-ui`. Inspect its report and full-page landing images. Windows owner desktop/browser interaction is not a prerequisite.
5. Compare released fixture asset names/URLs with the original manifest and fresh v1.0.1 API inventory. Candidate rendering must contain no published download URLs.
6. After an authorized merge and deployment, inspect the live landing counterpart and stable documentation routes; pre-merge fixture screenshots are not deployed proof.
