# Contributing

## Start with the specification

Read the [v1.0.0 system contracts](docs/v1.0.0/index.md), [constitution](.specify/memory/constitution.md) and [delivery outcomes](docs/v1.0.0/roadmap.md). Scope each change around an observable outcome. Numbered implementation plans live under `specs/` and are separate from the public documentation.

## Issues and work slices

GitHub Issues, milestones and the project board are the primary progress record. Issues describe behavior and acceptance; Spec Kit artifacts record implementation decisions and tasks. Follow the [development workflow](docs/v1.0.0/development.md). Link the slice, affected issues and milestone in the pull request. Do not duplicate status in disconnected planning documents.

User verification normally follows a release and arrives as versioned issues. Pre-release controlled automated checks still verify code contracts, integrity and buildability. Do not turn post-release observation into mandatory attended pre-release testing. Record what was checked and what remains unverified.

## Local checks

Use Node.js 22.12 or newer. `npm ci` at the repository root installs the schema-validation dependencies from its lockfile; `npm --prefix site ci` installs the documentation renderer.

```text
npm run check
npm test
npm --prefix site run build
```

CI runs these scopes in parallel jobs with caches and ten-minute timeouts. No live AI calls, media downloads or model inference belong in ordinary pull-request CI. Product tests and native packaging checks are proportional to changed behavior.

Windows tooling must launch console children with `CREATE_NO_WINDOW` or equivalent and disable interactive prompts. Use literal argument arrays and redirected I/O. Direct Git and GitHub operations through a verified headless runner are allowed.

## Style and documentation

Save UTF-8 without BOM and LF line endings. Use clear Markdown, one H1, soft-wrapped paragraphs and language-labelled code fences. Mermaid flowcharts use `TB`/`TD`. Add diagrams where systems would otherwise require complex prose. Keep the README compact and complete detail in versioned docs. Public docs describe authoritative contracts; slice codes and implementation evidence stay in `specs/`, and historical changes stay in `CHANGELOG.md`.

Release software, documentation and the master JSON Schema with the same version. Prefer backward and forward compatibility where practical; document breaking changes in the changelog and concise release notes. For every major release, scan the complete documentation for technical vocabulary and refresh the glossary with definitions, primary learning references and links to the pages that use each term most heavily.

Do not modify files under `brand/kit/` directly. Use the maintainer updater and read the upstream [consumer contract](brand/kit/enforcement/consumer-contract.json) before applying visual identity. User requirements govern product behavior; brand/tool implementation guidance does not authorize unrelated feature restrictions.

## Pull requests and releases

Use a focused branch, conventional commit/PR title and a concise description of changed behavior, validation and limitations. Use squash merges after required checks pass and review findings are resolved. Human PR approval is optional. GitHub deletes merged branches automatically once configured. Agents follow the owner's authorization for the merge itself.

Update `CHANGELOG.md` under Unreleased using Added, Changed, Deprecated, Removed, Fixed or Security as appropriate. Release notes are a short highlights reel ending with the master changelog link. See [release workflow](docs/v1.0.0/releases.md).
