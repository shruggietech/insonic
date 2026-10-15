# Implementation Plan: S029 v1.0.1 release publication

**Branch**: `codex/029-v1-0-1-release-publication` | **Date**: 2026-10-14 | **Spec**: [spec.md](spec.md)

## Summary

Finalize the existing unpublished 1.0.1 cut, correct released historical rendering, and add documentation qualification parity to the release workflow. Deliver a reviewed green preparation PR, wait for owner squash merge, then use existing trusted-main release/Pages tooling with a fresh exact-main candidate. #97 remains open through actual public verification.

## Technical Context

**Language/Version**: Existing Python maintainer tools, Node >=22.12, Next.js 16.4, pinned Go 1.27.1 runtime.
**Primary Dependencies**: Existing GitHub Actions, gh/API, Pages, Markdown renderer, Playwright-core 1.62.1; no new dependency.
**Storage**: Git source, immutable tag/release assets, bounded workflow artifacts, existing gh-pages destination.
**Testing**: npm run check, npm test, site tests/build/checks, Python release/security/Pages suites, actual diagram and browser qualifications; elected hosted Windows/Linux/native/backend/package gates.
**Target Platform**: Windows/Linux x86-64 CLI and Desktop+CLI archives, Windows per-user MSI, portable public/offline help.
**Project Type**: CLI/desktop product with maintainer release tools and static documentation.
**Performance Goals**: Preserve eleven-minute/660-second CI policy; browser qualification limit 180 seconds. Record queue and full turnaround honestly.
**Constraints**: No owner-host native probing; hidden noninteractive processes, fresh owned headless browser, exact cleanup. Preserve historical source and integrity tests.
**Scale/Scope**: #97 only; all #93–#96 prerequisites merged via PR #98 at 3abbe8e. Official release remains v1.0.0; no v1.0.1 tag exists at kickoff.

## Constitution Check

Pre-research and post-design: PASS. Owner authorizes push/PR now and publication after owner merge. Synchronized 1.0.1 identities, CLI/platform obligations, immutable historical evidence, self-contained docs, optional unsigned disposition, bounded CI, mandatory security/tenancy gates and existing external route remain intact. No constitution amendment or new approval gate is needed. Final readiness is not release completion.

## Project Structure

Planning: spec.md, plan.md, research.md, data-model.md, contracts/publication.md, quickstart.md, tasks.md, analysis/convergence/verification evidence, release-highlights.md and reviewer requirements checklist under specs/029-v1-0-1-release-publication/.

Affected source: docs/versions.json; site/lib/docs.mjs and release-status.mjs; site/tests/; .github/workflows/release.yml; tests/release-versions.test.mjs; root and docs/v1.0.1/references/ CHANGELOG.md and CONTRIBUTING.md. scripts/release.py and existing publisher/Pages guards remain unchanged unless a demonstrated failure requires a proportional fix.

## Phases and decisions

1. Specify/clarify/checklist, research and design, tasks and strict read-only analyze.
2. Add failing meaningful regressions for historical released/current candidate rendering and mandatory release-target qualifications.
3. Correct current manifest historical status and render-only released prose across entries. Finalize the existing 1.0.1 changelog (UTC cut date 2026-10-15), preserve chronological decisions and refresh current reference snapshots/links. Do not weaken prepare()'s newer-version/no-overwrite contract or invent 1.0.2.
4. Add site tests, Mermaid rendering, headless UI audit and source/attempt-fenced evidence before release docs archival; preserve all privileged job separation and native gates.
5. Verify snapshot and release-target exports locally with headless checks. Preserve application capture inputs and governed bytes; no screenshot recapture is needed if hashes remain valid.
6. Converge, commit/push/open official PR referencing #97 without closing it. Inspect automatic review; optionally send one second request, no third. Final exact-head CI and direct review thread audit precede owner handoff.
7. After owner merge only: fetch exact squash main, verify reviewed identity, tag absence/exact identity and green source checks, create/push immutable v1.0.1 tag, dispatch trusted-main release.yml with publish/promote true and Pages. Fresh qualification is mandatory.
8. Verify nine official assets, digests, release/tag/source and original published documentation; confirm exact Pages commit and public markers/routes/schema/downloads with headless evidence. Close #97/Project only then.

## Complexity Tracking

No exceptions. Existing release pipeline is reused. Manual reviewed finalization of an already prepared candidate is simpler than adding a second version-preparation API. Current mutable release metadata is corrected; immutable historical document bytes are preserved.
