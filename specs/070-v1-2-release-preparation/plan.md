# Implementation Plan: v1.2.0 release preparation and dependency qualification

**Branch**: `codex/070-v1-2-release-preparation` | **Date**: 2026-11-02 | **Spec**: [spec.md](spec.md)

## Summary

Prepare accepted main as unpublished 1.2.0, combine dependency PRs #245/#246 with intact upstream checksum authority, correct current answer capability metadata, preserve published history, obtain genuine current captures and qualify the exact final candidate. Deliver a green reviewed official PR without merging or publishing.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.11+, Node 22+, React and Next.js 16.4.3.
**Primary Dependencies**: x/net 0.62.0; existing pinned Wails/WebView source, maintained release/capture/qualification tooling.
**Storage**: Filesystem/SQLite/LadybugDB defaults and S3/PostgreSQL/ArcadeDB alternatives unchanged. Versioned documentation/schema and original receipt files; no new catalog migration.
**Testing**: Failing-first current candidate capability regression; existing checksum/history/preparation security regressions; root, Python, Go, frontend, site and hosted package/backend/installer acceptance.
**Target Platform**: Windows/Linux x86-64 CLI and desktop-with-CLI, Windows MSI/setup, public/offline help.
**Project Type**: CLI-first application release preparation.
**Performance Goals**: 1800-second qualification target, 3600-second hard limit after verified readiness, 60-second provisional finalization reserve; separate preparation/qualification/combined accounting.
**Constraints**: Hidden noninteractive local tooling; hosted isolated desktop capture; immutable historical authority; no new tracking issue, merge, tag, release or production docs promotion.
**Scale/Scope**: Four package variants, two Windows installer forms, seven-case packaged answer campaigns, 25 original screenshots/four receipts, two dependency proposals and one preparation PR.

## Constitution Check

Pre-research and post-design PASS for all five principles. Owner intent and push/PR authorization are explicit; no product restrictions or attended gate added. Shared CLI/runtime and backend contracts remain intact. Original sources, history, credential checks and capture authority persist. Exact-source qualification retains all mandatory checks and timing rules. Public docs contain contracts; planning/evidence remain internal. No constitution exception required.

## Phase 0 Research

[research.md](research.md) records eight decisions, rationale and rejected alternatives. Installed plan skill dispatched three read-only research agents. No unresolved clarification remains.

## Phase 1 Design

1. Establish baseline history/checksum tests and add a failing current advanced_cli assertion including answers; correct only current release metadata.
2. Apply desired dependency updates, preserving Wails/WebView sums, then use maintained prepare with tracked concise highlights and a dated changelog decision.
3. Preserve current changes in docs/v1.2.0 and schemas/v1.2.0 before restoring complete v1.1.0 trees and historical navigation authority from the published tag.
4. Save new-version walkthrough blocks outside source, omit them only for the clean capture bootstrap, run relevant checks, commit/push and dispatch full branch CI before PR creation.
5. Retain/audit bootstrap terminal acceptance before reruns. Verify artifact size/digest and capture source/version/inventory; import original enumerated PNGs and unchanged receipts, restore embeds for v1.2.0, inspect all captures and validate provenance.
6. Finish local CI parity checks and Spec Kit convergence. Open non-draft PR, attach it to this chat and Project 7 (Slice S070/In Review), explaining dependency supersession after merge.
7. Resolve all reviews within two rounds. Audit terminal PR and direct-branch acceptance on final source, validate complete candidate assets, retain durable receipts and hand off for owner final review/merge.
8. Record later sequence: qualify accepted post-squash main, separately elect model checks if requested, retain accepted bytes, then publish/verify/promote only under publication authorization.

## Project Structure

- specs/070-v1-2-release-preparation/: spec, plan, research, data-model, contracts, quickstart, tasks, highlights and verification.
- go.mod/go.sum; site/package.json and package-lock.json: reviewed dependency updates.
- scripts/release.py; tests/release_test.py: current advanced CLI metadata and regression.
- VERSION; owned root/site/frontend manifests and version-bound runtime/tests; docs/versions.json; docs/v1.2.0/; schemas/v1.2.0/; docs/screenshots/v1.2.0/; CHANGELOG.md.
- docs/v1.1.0/ and schemas/v1.1.0/: restored published authority after copying current work forward.
- Existing .github/workflows/ci.yml, scripts/ci-acceptance.mjs and release tooling supply qualification; no planned workflow expansion.

## Complexity Tracking

No new architectural layer or constitution exception. The reviewer-owned custom checklist remains untouched; existing owner autopilot authorization permits implementation. No extension hooks installed. The checklist prerequisite script expected plan.md before the protocol's checklist stage; the active template was resolved directly, then normal prerequisites are rechecked after plan setup.

## 2026-11-02 observed correction

Full Go checks exposed answer request version fields left at 1.1.0 by preparation. T019 extends only owned answer schema version fields and answers.md examples, preserving independent model versions. A failing-first release migration regression covers both schemas and documentation. No product behavior or deadline is narrowed.
