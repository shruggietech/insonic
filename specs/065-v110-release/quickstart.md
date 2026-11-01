# S065 validation and post-merge continuation

## PR phase

1. Run maintained preparation/history regressions, add the failing current-capability regression and make the proportional metadata correction.
2. Prepare 1.1.0 using `python scripts/release.py prepare --version 1.1.0 --date 2026-10-31 --highlights specs/065-v110-release/highlights.md`. Copy current corrections forward before restoring historical docs/schemas from v1.0.3; mark that historical entry released.
3. Omit only new-version screenshot walkthrough blocks, run root/local checks, and commit/push the clean capture bootstrap. Dispatch `gh workflow run ci.yml --repo shruggietech/insonic --ref codex/065-v110-release -f cold_media_source=false -f site_smoke=false`. Retain terminal acceptance and original capture artifact identities before another full run.
4. Verify and import only original receipt-enumerated images/receipts into docs/screenshots/v1.1.0. Restore new-version embeds, validate provenance and visually inspect the captures. Complete public/offline documentation and full maintained headless checks.
5. Converge, push and open the official PR. Resolve at most two review rounds and every actionable thread. Require green final PR and direct branch-head CI with retained terminal audits, clean source and correct tracking. Stop for owner merge.

## After owner merge

1. Confirm the reviewed merged-main SHA and completed exact-main CI. Dispatch the maintained Release candidate and publication workflow for that full revision with `publish=false` and `promote_docs=false`; this prepares publication-target bytes without publishing.
2. Retain the entire candidate, manifests, original receipts and checksums outside workflow expiry. Elect real-model CPU/NVIDIA acceptance separately if required for the publication campaign, using those exact package bytes and versioned corpus/model configuration. Do not represent deterministic fixtures as acoustic or hardware accuracy.
3. Under publication authorization, create the immutable v1.1.0 tag at the accepted source and use trusted-main `scripts/release.py publish` with the retained accepted candidate, exact source checkout and this slice's highlights. Do not rebuild after acceptance.
4. Promote the same published documentation with `pages-promote`, then execute `verify-public` against the original candidate, exact source and returned Pages commit. Verify all official asset digests, download identities and live documentation. Report phase timing and unverified dimensions separately.

Exact invocation arguments, capture identities and final receipts are recorded in [verification.md](verification.md) as execution completes.
