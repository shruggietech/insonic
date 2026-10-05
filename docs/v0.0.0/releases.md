# Brand, documentation and releases

## Upstream brand authority

The [insonic brand](https://brand.shruggie.tech/insonic/guidelines/brand-essentials/) belongs to upstream `shruggietech/shruggie-brand`. The integrated formal kit is insonic 1.0.0 built with BrandBuilder 3.0.1. Governed bytes live under `brand/kit/`, with the consuming repository's receipt in `brand/lock.json`.

The kit's consumer contract, enforcement guidance, tokens, named logos, licenses and manifest are authoritative. Product copy uses `insonic` and `A ShruggieTech project`. Local code binds semantic tokens and asset references rather than recreating artwork. Brand adoption preserves product requirements.

1. A maintainer manually runs the freshness check.
2. Find the latest formal upstream release containing the insonic kit.
3. Compare its brand and builder versions with the local lock.
4. When an update is requested, download the archive and checksums.
5. Verify the archive digest, paths and full manifest.
6. Stage the exact kit and integration receipt.
7. Replace the integrated kit, rolling back on failure.
8. Review the diff and consumer conformance.

`npm run brand:check` is read-only and requires network access. `npm run brand:update` adopts the latest newer formal kit. `npm run brand:verify` checks byte integrity offline. The tools reject corrupted archives, unsafe paths, immutable-version drift and downgrades. Discovery uses official release assets. A newer BrandBuilder kit can require adoption even when the brand version is unchanged.

Freshness checking is outside ordinary CI, and application startup never downloads branding. Follow upstream semantic, glyph and consumer-conformance guidance when changing a UI. Exact bytes alone do not establish visual conformance.

## Synchronized documentation and schemas

Markdown under `docs/<version>/` is the documentation authority. `docs/versions.json` defines available versions and navigation. The root `VERSION`, software release, documentation version and [master JSON Schema](contracts.md) share the same version. The schema's individual record definitions carry their declared version inside that release contract. Upstream Cueson retains its own versioned schema.

Build the website, downloadable schemas and offline documentation from the exact release tag. Retain older version routes and immutable schema identities. `/docs/` selects the latest published release. The public root route is a product landing page; `/docs/` is the versioned documentation. `site/out/` contains the full static website, including that landing page, local Mermaid diagrams and brand assets. Readers need no production Node server or remote font/diagram CDN.

The GUI includes `site/offline/`, the documentation-only export from the same tagged build. It excludes the public landing page and retains the matching documentation, changelog, schema resources and local assets. A release manifest records documentation/schema versions, source revision and hashes. Offline help uses installed bytes and stable relative links without a network or development server. Publish the matching site artifact and promote the default version only after release publication succeeds.

Each release records a CLI/GUI capability matrix, including advanced or experimental CLI features awaiting GUI controls. Core operations share the runtime and aim for parity. Adapter protocols, database migrations, model formats and upstream dependencies record their compatibility separately within the synchronized release.

Prefer backward and forward compatibility where practical. A breaking change remains possible when needed; document its affected interfaces and migration in the [master changelog](changelog.md) and brief release highlights. Do not silently reinterpret incompatible data.

For every major release, rescan the complete documentation for technical terms and refresh the [glossary](glossary.md) with precise definitions, primary learning references and links to the local pages using each term most heavily.

## Changelog and highlights

The documentation site renders root `CHANGELOG.md` as the [changelog](changelog.md), preserving one master record. Released documentation freezes the tagged file at `docs/v<version>/references/CHANGELOG.md`; the build requires and renders that snapshot. It follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Add work under Unreleased; at release move entries into a dated version section and update comparison links. [Semantic Versioning](https://semver.org/spec/v2.0.0.html) describes releases and compatibility.

Release notes contain brief highlights, major migration actions and known material limitations. Do not paste full changelog sections. End with a link to the master changelog at the release tag.

## Release sequence

1. Start from reviewed source with green required checks.
2. Align the software, documentation, JSON Schema and changelog versions.
3. Build native packages with the CLI, dependencies and documentation.
4. Verify the inventory, checksums and dependency notices.
5. Publish the immutable tag and brief release highlights.
6. Promote the matching static documentation and schemas.
7. Receive versioned user observations as issues.

Package matrices include CLI-only and GUI-with-CLI options for macOS, Linux and Windows. Sign/notarize platform packages using configured credentials and tooling. The release manifest identifies native libraries, Cueson, media tools, optional workers and their digests/licenses. User-selected model downloads remain separate assets.

The static export is portable to ordinary static hosting; its host and domain are deployment configuration.
