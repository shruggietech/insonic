# Owner launch handoff

S016 prepares the release path and packaged execution evidence. It does not create a product tag, publish v1 downloads or deploy documentation. Review artifacts identify their reviewed source; squash merging changes that identity.

## Brief v1 highlights

- CLI-first portable audio catalogs with filesystem/S3 storage, SQLite/PostgreSQL catalogs and LadybugDB/ArcadeDB graph choices.
- Current transcripts, speaker attribution and roster-constrained matching with elected local or configured workers.
- Receipt-backed self-contained backups and native CLI/desktop package variants with matching offline documentation and complete media corresponding sources.

Full change history: [CHANGELOG.md at v1.0.0](https://github.com/shruggietech/insonic/blob/v1.0.0/CHANGELOG.md).

## Subsequent authorized launch sequence

1. Squash merge the approved PR and synchronize clean main. Resolve `FULL_MAIN_SHA` to the actual 40-character merged commit; do not reuse a PR merge/head candidate.
2. Confirm current publisher-signing configuration and its disposition. Missing identities remain explicitly unsigned; macOS ad-hoc signing is separate.
3. Dispatch `gh workflow run release.yml --ref main -f revision=FULL_MAIN_SHA -f publish=false -f promote_docs=false` and verify its complete six-package release-target candidate and all required checks.
4. With product-publication authorization, create the exact `v1.0.0` tag on that reviewed SHA and dispatch from main with `publish=true`, `promote_docs=true`, `documentation-transport=pages` and the brief highlights above. The protected-main verifier requires the exact existing tag, green checks, full original asset set and successful read-only destination preflight.
5. Verify published asset identity and the exact Pages build receipt, then verify public root/version/schema/download routes. If promotion fails after publication, retry promotion-only with the same exact SHA and original published archive (`publish=false`, `promote_docs=true`, `documentation-transport=pages`).

Generic hosting remains an explicit alternative with its configured main-only environment and dedicated credential. No live promotion is claimed by isolated transport tests. Connected CPU fixture execution does not establish independent acoustic accuracy or inference on other platforms.
