# Implementation Plan: v1.1.0 release preparation and qualification

**Branch**: `codex/065-v110-release` | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

## Summary

Prepare an unpublished 1.1.0 minor candidate from accepted S061-S064 work using maintained release migration. Preserve original history, describe current CLI/GUI capability differences accurately, obtain genuine version-bound hosted captures, and deliver a green official PR with audited evidence. Owner merge and later accepted-main qualification/publication remain separate.

## Technical Context

**Language/Version**: Existing Go 1.27.1, Python 3.11+, Node 22+ and React/Next tooling.
**Primary Dependencies**: Existing version preparation, schema registry, release collection, capture-provenance validators and hosted qualification workflows. No dependency refresh.
**Storage**: Versioned JSON/Markdown/schema and original PNG/JSON receipts; no new catalog storage or migration.
**Testing**: Failing-first current candidate capability metadata regression, existing preparation/history tests, full root/Python/Go/frontend/site checks and exact-source Windows/Linux platform/package acceptance.
**Target Platform**: Windows/Linux x86-64, CLI and desktop-with-CLI, Windows setup/MSI and public/offline help.
**Project Type**: Release preparation for the shared CLI-first application.
**Performance Goals**: 1800-second advisory qualification target, 3600-second hard limit after separately verified prerequisite readiness, 60-second provisional finalization reserve.
**Constraints**: Hidden noninteractive local commands; interactive/installer captures only on isolated CI; original historic bytes and narrow exemptions preserved; no merge/tag/release publication.
**Scale/Scope**: Four package variants, installer paths, 25 original screenshots/four receipts, complete current and historical documentation/contracts, one release issue/PR.

## Constitution Check

Pre-design and post-design PASS for all five principles. Owner authorization supplies push/PR permission and the exact two-round limit. CLI/shared runtime behavior and backend parity remain intact; source/credential/history authority remains explicit. Current source, candidate qualification and published status are distinct. No product clock, mandatory check or historical integrity rule is weakened, and no attended pre-release ritual is introduced.

## Phase 0 Research

The installed planning command directed three read-only agents for maintained preparation/history, capture workflows and qualification/status. Parent inspection corroborates their findings in scripts/release.py, schemas/registry.go, site/lib/release-status.mjs and .github/workflows/ci.yml. [research.md](research.md) records decisions and alternatives; no unresolved design question remains.

## Phase 1 Design

1. D1: Use maintained `prepare --version 1.1.0 --date 2026-10-31` with concise tracked highlights. Exercise existing preservation regressions rather than duplicate low-value migration tests. Retain 1.0.3 document support when the current envelope advances.
2. D2: Correct only current candidate capability metadata: list `recordings coverage` and `recordings repair` alongside backup, with a matching public CLI/GUI matrix that separately explains advanced window/range configuration and shared automatic processing. Add a failing-first regression around candidate collection. Preserve original historical fixtures and schema evidence.
3. D3: Copy current v1.0.3 corrections into v1.1.0 first, then restore all historical v1.0.3 docs/schemas from tag authority. This includes 11 modified documentation pages, seven changed schemas and four unpublished schema additions. Mark 1.0.3 released in docs/versions.json; retain the tagged index sentence because the maintained renderer projects released wording from the manifest. Leave existing latest aliases and upstream identities unchanged.
4. D4: Temporarily omit only the new-version screenshot walkthrough blocks for a clean bootstrap. Commit/push that source and dispatch full branch CI without creating the PR. This gives genuine current-version images without consuming an external review round prematurely. Do not relabel copied images or fabricate receipts.
5. D5: Retain/audit bootstrap terminal acceptance before a full rerun. Verify capture artifact digest, source, version, inventory and original PNG hashes/dimensions, import only the 25 enumerated PNGs and four original receipts into docs/screenshots/v1.1.0, restore new-version embeds and visually inspect every displayed capture.
6. D6: Complete public/offline validation and the full local maintained checks in foreground; hosted CI provides actual Windows/Linux native, package, backend and interactive qualification. Fix only demonstrated blockers with targeted regression evidence.
7. D7: Converge requirements/tasks before official PR creation, then use automatic round one and at most one explicit second review. Resolve every actionable comment/thread, retain final PR and direct branch-head acceptance, update tracking and hand off the clean unmerged PR.
8. D8: Record the executable post-merge sequence: accept merged main, obtain a publication-target candidate without publication, elect any real-model acceptance separately, retain exact bytes, then publish/promote/verify only under the corresponding owner authorization. Pre-merge acceptance never substitutes for changed post-merge source.

## Project Structure

- VERSION; root/site/desktop/frontend manifests and locks; current internal/cmd/test schema-version bindings and embedded worker catalog.
- scripts/release.py and tests/release_test.py: current capability declaration and focused regression.
- docs/versions.json; docs/v1.1.0/; schemas/v1.1.0/; docs/screenshots/v1.1.0/; historical docs/v1.0.3/ and schemas/v1.0.3/ restored to exact published authority after copying current changes forward.
- CHANGELOG.md; specs/065-v110-release/{spec,plan,research,data-model,quickstart,tasks,verification,highlights}.md and contracts/release-preparation.md.
- Existing scripts/check.mjs, check-schemas.mjs, screenshot validators, ci-acceptance.mjs and .github/workflows/ci.yml supply verification; no workflow changes are planned.

## Tracking and delivery

Issue #237, milestone 50, Project 7, Slice S065, Release 1.1.0. Branch and feature selector agree. The custom release checklist is reviewer-owned and remains unchecked; owner autopilot authorization already permits execution and does not require an extra confirmation. Built-in specification quality is complete. No extension hooks are installed.
