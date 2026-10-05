# Changelog

All notable changes to this project will be documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and releases use [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- v0.0.0 system specification covering cross-platform CLI/GUI orchestration, media/subtitle correlation, metadata/date provenance, configurable adapters, speaker/term catalogs, graph exploration and optional speaker-model training/retrieval.
- Upstream insonic 1.0.0 brand kit built with BrandBuilder 3.0.1, with a manually triggered checksum-verified update process.
- Versioned Markdown documentation and a static Next.js renderer with a public product landing page and separate documentation-only offline export for installed GUI help.
- Spec Kit infrastructure and a staged, issue-oriented delivery plan.
- Public repository contribution, security, issue and pull request templates, efficient foundation CI and local GitHub setup tooling.
- Version-matched master JSON Schema and rendered JSON contracts, with the master changelog included in the documentation site.
- Expanded technical glossary with primary learning references and links to the pages using each term.
- Source-linked interactive archive illustration, a glacial-blue landing gradient and a platform download section that reports package availability.
- Sun/moon documentation toggle with an operating-system default, remembered light/dark choice and matching logos and diagram colors in offline help.

### Changed

- 2026-10-04: Adopt Apache 2.0 for original source/documentation and Go for CLI-first shared orchestration, with Wails as the desktop baseline subject to native packaging qualification. Require the official Cueson executable/schema boundary and a rebuildable LadybugDB graph projection.
- 2026-10-04: Amend constitution to 2.0.0 for GUI core parity with documented advanced/experimental lag, early metadata/date provenance, full filesystem/S3, SQLite/PostgreSQL and LadybugDB/ArcadeDB support, and configured automation without mandatory per-item review. Add optional speaker-model delivery under S015/M5.
- 2026-10-05: Amend constitution to 2.1.0 for synchronized software, documentation and JSON Schema versions, practical compatibility, concise breaking-change notes and major-release glossary reviews. Keep numbered plans internal and one specification-status statement in the documentation index.
- 2026-10-05: Amend constitution to 2.1.1 for numbered linear procedures; replace four sequential diagrams while retaining branching, relationship and feedback diagrams.
- Bind landing hero and page metadata to approved upstream slogan and description fields; expand syntax highlighting for JSON and CLI examples.
- Support optional validated read-only AI query execution through the configured assistance mode.
- Subdivide workspace/runtime, catalog/jobs, artifact storage and secrets/model-registry development into four focused slices; renumber the remaining application slices through S015.
- Define each backend configuration once and reuse it in adapter-selection validation without changing its fields or constraints.
- Include backend/query alternatives and shared timestamp descriptions/examples in the generated JSON contract reference.
- Load a small shared theme/tab script and confine the Mermaid runtime to documentation routes.
- Keep specification status in the documentation index, replace generic landing copy with concrete outcomes, and remove conversation-specific instructions from public prose.

### Fixed

- Keep the landing archive widget at the tallest tab's natural height so switching views leaves following content in place.
- Align audit-instant schemas, examples and documentation on Cueson's `{iso, unix_ns}` representation, including lossless nanosecond-integer handling.
- Preserve closed GitHub Projects when rerunning repository setup.
- Cover root schema-validation dependencies in grouped Dependabot updates and correct contributor installation guidance.

[Unreleased]: https://github.com/shruggietech/insonic/commits/main
