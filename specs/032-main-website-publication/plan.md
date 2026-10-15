# Implementation Plan: Reliable website publishing from main

**Branch**: codex/032-main-website-publication | **Date**: 2026-10-15 | **Spec**: [spec.md](spec.md)

## Summary

Separate website publisher: clean exact newest-green current main, bounded qualified archive, read-only published product validation, isolated non-force Git transport, exact Pages/live-byte verification and hosted live browser evidence. Share verified CI documentation exports and overlap scope with foundation while retaining qualification.

## Technical Context

**Language/Version**: Python 3 standard library, Node.js 22 ESM, GitHub Actions.

**Primary Dependencies**: Existing release/pages/public-verification, ProcessTree, playwright-core, Git and pinned actions.

**Storage**: Same-run artifacts, bounded static archives/receipts, existing gh-pages.

**Testing**: Python unittest, Node tests, site renderer/browser checks, full exact-head hosted CI, optional read-only live smoke.

**Target Platform**: Hosted Ubuntu publication/browser; Linux/Windows native CI. Hidden noninteractive local tooling.

**Project Type**: Maintainer CLI/automation. No application API or product release.

**Performance Goals**: Complete elected CI below 660 seconds; aggregate phase/network/browser deadlines.

**Constraints**: Preserve tags/releases/assets/schema bytes; trusted-main write tooling; current main recheck; no force updates or personal browser/process interaction.

**Scale/Scope**: About 900 static paths/43 MB, five packages, eight browser cases, issues #106/#107.

## Constitution Check

Before research: PASS. CLI-first, existing destination, version authority and release integrity, native/security checks retained, planning internal. No product restrictions or attended gate.

After design: PASS. Credentials separated from read-only build/verification; site/product identity distinct; export reuse retains source/hash and later browser gates. No exception.

## Project Structure

- scripts/website.py: qualify/publish/read-only verify.
- scripts/site-transfer.py: CI archive identity/restore.
- scripts/check-live-site-ui.mjs and shared check-docs-ui.mjs assertions.
- .github/workflows/website.yml, ci.yml, native-qualification.yml, release.yml.
- tests/website_test.py, site_transfer_test.py, live-site-ui.test.mjs, ci-scope tests.
- docs/v1.0.1/releases.md, development.md, CHANGELOG.md.
- internal/app/work.go, processing.go, package_tools_test.go: remove measured tool discovery from catalog-only paths while retaining fresh execution checks.
- cmd/insonic-desktop/main.go: qualification success follows cleanup.
- specs/032-main-website-publication/: design, tasks and exact evidence.

## Delivery

Write failing tests before behavior. Run root/affected checks and converge, then push official PR. Resolve at most two actual external review rounds; inspect every thread and final exact-head CI. Measure complete CI rather than accepting projections. Read-only hosted smoke proves live S031 without deploying unmerged code. Stop for owner final merge review.

## Complexity Tracking

No violations. Reuse existing contracts instead of another hosting/release framework.
