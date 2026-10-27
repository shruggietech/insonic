# S056 verification

## Local acceptance, 2026-10-27

- Original defect reproduced: authentic v1.0.1 receipt rejected for missing later terminal identity.
- New history/boundary regressions: 17 passing within the full suite. Source completeness, unsafe/link/nonregular members, manifest/receipt changes, first-use, signing, inventory, default modern evidence and nested authority are covered.
- Full Python suite: 680 tests, three skips, passed with pinned Go 1.27.1 and Windows locale C. Initial wrong-compiler/unsupported-locale prerequisites were corrected; no assertions or deadlines were weakened.
- npm run check passed. npm test: 210 passed. Site check/build: 140 pages; site test: 18 passed. git diff --check passed.
- New metadata subprocess owner is classified under the maintained Python timing policy; no timing limits changed.
- Maintained load_candidate verified all nine authentic original v1.0.1 assets against 2015 exact tracked source files (13.296 s).
- Read-only published retrieval/verification plus actual MSI Property-table mutations completed in 31.781 s. Both unchanged-receipt and rehashed-receipt mutations were rejected. Running verifier hashes and original asset/source identities are retained in a compact receipt for PR publication. Protected-main metadata uses an isolated clean main snapshot, separately from the patched verifier under test.
- Source/code convergence: eight functional requirements covered, no implementation gaps. Existing T014/T015 retain publication, hosted CI and review work. Local operation durations above do not claim hosted preparation/qualification accounting.

## Boundaries

No original tag/release/assets or selected source changed. No historical verifier code or product binaries executed. Original byte/receipt verification does not establish fresh installed-product runtime behavior. Hosted exact-head CI and external reviews remain pending.
