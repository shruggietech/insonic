# Publication validation

1. Verify PR #252 squash, clean selected source, synchronized main and completed tracking. Run npm run check, npm test and maintained release/Pages/public-verification/website/HTTP regression suites through hidden launchers.
2. Retain successful main CI acceptance. Dispatch release.yml from main with revision=27ddddcd8afe8e44b1dd0c570ae3fd2070130728, publish=false, promote_docs=false and documentation-transport=pages.
3. Retain terminal metadata and original release-candidate-1 ZIP. Validate GitHub digest/size, extract confined regular files and load the candidate with exact_source/load_candidate; require both docs archives to pass verify_documentation with require_release=True.
4. Run qualify-speakers.py --mode real --elect-models with the unchanged elected configuration/models, exact package/receipt and revision. Expect real CPU/CUDA outcomes, full contextual cases, independent enrollment invalidation, no hidden fallback and completed cleanup. Keep preparation separate from qualification.
5. Recheck source checks and absent/conflict-free tag, create v1.3.0 at the selected revision, then run release.py publish --candidate DIR --source-root ROOT --highlights FILE. Expect all ten original files and concise notes ending at the tagged changelog.
6. Run release.py published --revision REV --output NEW_DIR --source-root ROOT. Independently compare every file's name, size and SHA-256 to accepted bytes.
7. Run release.py pages-promote --candidate DIR --source-root ROOT --output RECEIPT. If acknowledgement is uncertain, use pages-reconcile with the original receipt and a distinct output. Require exact completed deployment identity.
8. Run release.py verify-public against NEW_DIR and the exact Pages commit; materialize its confined tree and run website.py verify-deployed. Run check-live-site-ui.mjs using full Chromium headless for 320/390/768/1440 widths in light/dark, then inspect all eight images.
9. Retain compact evidence, update CHANGELOG.md after actual success, converge S075 and rerun applicable checks. Push/open evidence PR, resolve at most two review rounds, retain final terminal CI and hand off for owner merge.

Credentials remain in memory or existing encrypted stores. Original archives and failed diagnostics remain outside source control. Do not rerun publication or create a replacement tag simply to refresh evidence.
