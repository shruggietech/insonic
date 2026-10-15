<picture>
  <source media="(prefers-color-scheme: dark)" srcset="brand/kit/logos/named/insonic-logo-lockup-wide-full-color-clear-for-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="brand/kit/logos/named/insonic-logo-lockup-wide-full-color-clear-for-light.svg">
  <img alt="insonic" src="brand/kit/logos/named/insonic-logo-lockup-wide-full-color-clear-for-light.svg" width="560">
</picture>

[![CI](https://github.com/shruggietech/insonic/actions/workflows/ci.yml/badge.svg)](https://github.com/shruggietech/insonic/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/shruggietech/insonic)](https://github.com/shruggietech/insonic/releases)
[![License](https://img.shields.io/github/license/shruggietech/insonic)](LICENSE)
[![Issues](https://img.shields.io/github/issues/shruggietech/insonic)](https://github.com/shruggietech/insonic/issues)

# insonic

Organize audio and video, correlate subtitles and speakers, and explore what was said and when through a CLI or desktop application.

## Documentation

Read the [current system contracts](https://insonic.io/docs/latest/), [technology decisions](https://insonic.io/docs/latest/technology/) and [delivery outcomes](https://insonic.io/docs/latest/roadmap/). The documentation index identifies candidate and publication status; [version archives](https://insonic.io/docs/versions/) preserve release-specific guidance.

## Installation

Choose a Windows/Linux x86-64 package from the [official releases](https://github.com/shruggietech/insonic/releases). Desktop + CLI includes application windows and the command-line tool; CLI-only includes terminal commands without the desktop application. Windows ZIP packages require Microsoft Visual C++ x64; desktop additionally requires WebView2. Read [package choices, prerequisites and Windows ZIP installation/update/removal](docs/v1.0.1/desktop.md#native-package-layouts) for the selected version. The current 1.0.1 index identifies candidate status; historical packages retain their own version-specific guidance.

## Capabilities

- CLI-first shared operations with a desktop wrapper on Windows and Linux.
- Tracked originals, captured metadata/recording dates, optimized audio derivatives and optional supplied subtitles, unified through Cueson.
- Configurable local and hosted transcription, diarization and reasoning pipelines.
- Speaker/alias and specialized-term catalogs, source-linked audio corpora and optional speaker-model training/retrieval.
- Filesystem or S3-compatible storage, SQLite or PostgreSQL catalogs, and LadybugDB or ArcadeDB graphs.
- A media timeline, saved graph queries and optional AI query suggestions.

## Development

Use Node.js 22.12 or newer for the maintainer tools and documentation site:

```text
npm ci
npm run check
npm test
npm --prefix site ci
npm --prefix site run build
```

Markdown lives in `docs/`; the full static website is written to `site/out/` and documentation-only offline help to `site/offline/`. Spec Kit organizes incremental work slices. See [CONTRIBUTING.md](CONTRIBUTING.md) for workflow, publication boundaries and verification policy.

## Brand maintenance

The brand is owned by the [upstream brand project](https://brand.shruggie.tech/insonic/guidelines/brand-essentials/). Maintainers can manually check and adopt formal updates:

```text
npm run brand:check
npm run brand:update
```

The updater verifies release/archive checksums and preserves the previous kit on failure. See [brand/README.md](brand/README.md).

## Contributing and support

Use GitHub Issues for bugs and proposals. Include the affected version and concrete observed behavior. [Contribution guidance](CONTRIBUTING.md), [security reporting](SECURITY.md), [community conduct](CODE_OF_CONDUCT.md) and the [changelog](CHANGELOG.md) define the project workflow.

## License

Source code and original documentation use [Apache 2.0](LICENSE). Upstream assets and dependencies retain their own notices; see [NOTICE](NOTICE) and the kit's licensing files.

A [ShruggieTech](https://shruggie.tech/) project
