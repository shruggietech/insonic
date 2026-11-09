# Publication validation guide

Use hidden redirected noninteractive tooling. Preserve unique outputs and retain failed attempts.

1. Verify clean selected source and current-main CI success; retain main acceptance audit.
2. Dispatch release.yml from main with revision=1443198e0dec4738c129bb6dfe48baba9830d6e3, publish=false, promote_docs=false, documentation-transport=pages.
3. Retain original candidate and receipts, verify artifact digest/size, safely extract and load through scripts/release.py with both docs archives requiring release target.
4. Copy-install original Windows CLI and run scripts/qualify-answers.py with fresh library/report, --elect-model and existing exact-model configuration. Require all seven frozen cases and cleanup.
5. Recheck exact source/green checks/tag absence; tag v1.3.1 at selected source, push tag and publish the accepted candidate with specs/076-answer-startup-reliability/highlights.md. Do not dispatch a rebuilding publish workflow.
6. Run scripts/release.py pages-promote using original candidate/source and unique receipt; reconcile uncertain results with pages-reconcile.
7. Fresh scripts/release.py published downloads must match every accepted file. Run verify-public against exact Pages commit, verify-deployed against safely extracted exact Pages tree, then check-live-site-ui.mjs using full installed Chromium headless.
8. Retain original/hash-checked compact evidence, record successful history, converge and publish the evidence PR; final PR merge stays owner-controlled.

See contracts/publication.md and plan.md for identities, recovery and evidence boundaries.
