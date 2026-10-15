# S029 validation guide

1. Use the feature selector for specs/029-v1-0-1-release-publication and the verified hidden noninteractive runner on Windows.
2. Run npm run check, npm test, npm --prefix site test and python -m unittest discover -s tests -p '*_test.py'. Focused release tests must demonstrate the new regressions red before the implementation and green afterward.
3. Build site snapshot output, run scripts/qualify-docs-renderer.py and scripts/check-docs-ui.mjs with the installed headless Chromium, site/out, site/offline and an owned build output directory.
4. Repeat the actual site build/diagram/UI checks with INSONIC_DOCUMENTATION_TARGET=release. Check historical v1.0.0 released text, current candidate versus release overlay, five current package links, latest aliases, offline parity and immutable historical source.
5. Run Spec Kit convergence, record evidence in verification.md, push/open the non-closing preparation PR, resolve at most two rounds and verify exact-head CI before owner handoff.
6. After owner merge only, follow contracts/publication.md using the exact reviewed squash-main revision, reviewed release-highlights.md and existing trusted-main release workflow.
7. Verify nine official assets/checksums and the deployed original documentation source/version, public links, mobile navigation and historical routes before closing #97.

Do not run native package/MSI/account/WebView probes on the owner host. Hosted CI owns them. Local browser checks use only isolated owned headless resources; automated results do not certify assistive technology or other engines.
