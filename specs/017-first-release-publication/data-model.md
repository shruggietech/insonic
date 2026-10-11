# Existing publication identities

| Entity | Existing authority | Constraint |
|---|---|---|
| Release candidate | Exact source SHA, 1.0.0 version, six native packages, original source/documentation archives and SHA-256 inventories | One clean reviewed revision; complete corresponding sources and accepted package receipts |
| Official release | Immutable `v1.0.0` tag and verified ten-asset public release | Non-draft/non-prerelease only after every expected upload is verified |
| Documentation deployment | Original published public archive, existing Pages configuration and exact new gh-pages commit | Preserve historical routes/domain markers; nonforced update and exact-commit build completion |

No new runtime entity, persistence schema or product API is introduced.
