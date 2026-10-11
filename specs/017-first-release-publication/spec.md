# S017: First release publication

Created: 2026-10-11. Branch: `codex/017-first-release-publication`. Status: specified.

## Intent and scope

Publish the already prepared insonic 1.0.0 as the first official release and deploy its matching documentation to the existing insonic.io destination. Complete publication, rather than stopping at candidate preparation or a pull request. Keep planning compact and discard temporary launch downloads/logs after verification.

## User stories and acceptance

### US1: Obtain the first official release (P1)

As an operator, I can download CLI and desktop packages for Windows x86-64, Linux x86-64 and macOS ARM64 from one public 1.0.0 release. This is the primary launch outcome; inspect the public release and its verified asset inventory independently of documentation deployment.

1. Given the final reviewed main revision, when publication completes, the non-draft, non-prerelease `v1.0.0` release identifies that exact revision and contains all six qualified packages, complete corresponding sources, documentation and manifest/checksums.
2. Given the release page, its original highlights summarize product capabilities in at most 180 words and end with a tagged changelog link. They do not reproduce the changelog or claim measured acoustic accuracy or publisher signing.

### US2: Read matching published documentation (P1)

As an operator, I can read the released 1.0.0 contracts, installation instructions and downloads on insonic.io, while older baseline routes remain accessible. Documentation is independently verified by public HTTP readback after its exact deployment completes.

1. Given the original published documentation archive, promotion deploys its verified contents to the existing Pages destination and completes the exact new deployment commit.
2. Given public documentation, the released landing page exposes six exact-tag downloads, versioned documentation/schema identify 1.0.0 and preserved historical routes remain reachable.

## Requirements

- FR-001: Finalize the release date and incorporate all pending Unreleased changes into 1.0.0 in both master and frozen changelogs; retain synchronized software, documentation and master schema versions.
- FR-002: Rescan the major-release documentation terminology and correct missing glossary definitions/references if any are found.
- FR-003: Publish the complete qualified asset set from a clean exact protected-main revision with green required checks and matching immutable `v1.0.0` tag. Resolve every PR review comment before merging preparation.
- FR-004: Author concise original release highlights, at most 180 words, ending with the tagged full changelog link.
- FR-005: Promote the original verified documentation to the existing HTTPS insonic.io Pages destination, preserving historical routes and deployment identity.
- FR-006: Do not provision publisher signing/notarization or perform acoustic accuracy checks. Retain required platform loader integrity and truthful unsigned/unmeasured dispositions.
- FR-007: Verify public release identity, asset inventory/checksums and live documentation after publication; close tracking only once publication and deployment succeed.
- FR-008: Keep temporary downloads/logs outside repository source and delete them at completion. Preserve useful build/toolchain caches without new evidence archives.

## Edge cases

- A conflicting existing tag or published asset must fail verification rather than overwrite immutable release bytes.
- If release publication succeeds but documentation fails, retry promotion-only from the same original release, without rebuilding or replacing published assets.
- Failed or unfinished external checks on the selected revision block publication; keep main stable through launch.
- Candidate status in ordinary source builds remains truthful; elected public/offline release artifacts render released status.

## Key entities

- Release: one exact version/tag/source revision, original verified assets and concise highlights.
- Documentation deployment: original release documentation, existing destination and exact promoted commit.

## Success criteria

- SC-001: One public stable `v1.0.0` release has all six native variants and the complete verified source/documentation/integrity asset set at one exact revision.
- SC-002: Release notes satisfy FR-004 and distinguish published capabilities from unperformed acoustic/signing work.
- SC-003: insonic.io serves the exact released version and preserved baseline documentation/schema routes after the exact Pages commit succeeds.
- SC-004: Required local/CI checks and PR reviews pass, publication tracking is complete, main is synchronized, and temporary launch material is removed.

## Clarifications and assumptions

- 2026-10-11: The owner authorizes preparation, push, official PR, squash merge, exact tag, release publication and existing-site promotion as necessary to fully publish S017.
- 2026-10-11: Publisher signing/notarization and acoustic accuracy tests are explicitly excluded. Automatic macOS ARM64 ad-hoc loader signatures are existing execution requirements, not publisher identity.
- Existing release/Pages workflows and all six supported packages are retained. No new product API, host, migration, signing secret or attended release gate is introduced.
