# S017 publication procedure

1. Finalize the master/frozen 1.0.0 changelogs and glossary assessment. Check `highlights.md` is original, at most 180 words, and ends with the tagged changelog link.
2. Run `npm run check`, `npm test`, existing Python release/Pages tests, and ordinary plus elected release-target site builds/checks. Use the repository hidden launcher for local console tools.
3. Publish the preparation PR, satisfy exact-head CI and every review comment (at most two bot rounds), then squash merge and delete its branch. Synchronize a clean local main.
4. Select the full main SHA and complete its external checks. Dispatch `release.yml` from main with that revision, `publish=false`, `promote_docs=false` and `documentation-transport=pages`. Require all six package qualifications, both released documentation exports and candidate verification to pass.
5. Create/push `v1.0.0` at that exact SHA. Dispatch `release.yml` from main with the same revision, `publish=true`, `promote_docs=true`, `documentation-transport=pages` and the text of `highlights.md`. Keep main stable until launch completes.
6. Require public non-draft/non-prerelease release identity, six native packages plus corresponding source/documentation/integrity assets, valid remote inventory/checksums, exact successful Pages deployment and live 1.0.0/historical documentation/schema routes. If only deployment fails, rerun promotion-only against the same original release. Never replace conflicting immutable tag/asset bytes.
7. Close issue #46 and its milestone only after these actual publication outcomes. Remove the external temporary launch directory and ignored transient launch outputs; retain useful caches. Report the release and documentation links with verification limits.
