# Implementation Plan: v1.0.3 patch release preparation

**Branch**: `codex/059-v103-release` | **Date**: 2026-10-29 | **Spec**: [spec.md](spec.md)

## Summary

Prepare one coherent unpublished 1.0.3 candidate from S054–S058, preserve historical release authority, acquire genuine 1.0.3 captures and deliver a qualified official PR. Use scripts/release.py prepare and existing CI/platform qualification. Owner merge, fresh exact-main CPU/restore acceptance, tag, release and Pages publication remain the next phase.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.11+, Node 22+ and existing React/Next documentation tools.
**Primary Dependencies**: Existing preparation, schema, screenshot-provenance, package/installer and CI receipt tools.
**Storage**: Versioned JSON/Markdown/schema files; no new runtime storage.
**Testing**: Meaningful preparation-preservation regressions, full Python/root/site checks, affected Go checks and exact-source hosted platform/package acceptance.
**Target Platform**: Windows/Linux x86-64, CLI and desktop with CLI.
**Project Type**: Release preparation for an existing CLI-first application.
**Performance Goals**: Existing 1800-second qualification target and 3600-second hard limit after verified readiness; separate phase accounting.
**Constraints**: Hidden local children, isolated hosted GUI/installer captures, exact historical exemptions, no owner-data modification, no publication in this phase.
**Scale/Scope**: Four package variants, Windows setup/MSI and public/offline documentation. No dependency refresh or feature expansion.

## Constitution Check

Pre-design and post-design PASS. All five principles retain owner merge authority, shared CLI/desktop contracts, source/history provenance, existing providers/credentials and truthful bounded qualification. No exceptions, product deadline changes or new attended acceptance gate.

## Phase 0 Research

The installed plan command directed one read-only research agent. Parent inspection confirms preparation's broad owned-test rewriting needs a frozen-history exception plus dynamic current-negative coverage. Native version-bound captures require a clean bootstrap source. A branch CI workflow_dispatch is supported and elects complete qualification without opening the PR early.

## Phase 1 Design

1. Preserve historical receipt tests/fixtures during preparation; express current-version negative cases dynamically and keep explicit old-version cases. Prove failure first with actual preparation on disposable files, then verify current envelopes/routes still advance. Avoid duplicating the final tagged changelog link when complete tracked highlights are supplied. Advance exact-repository download tags together with filenames while preserving foreign upstream URLs; derive current website fixture tags from their manifest.
2. Use maintained prepare --version 1.0.3 --date 2026-10-29 and tracked concise highlights. Copy current corrections forward, then restore the four demonstrated changed 1.0.2 pages from the exact published tag and mark its verified historical navigation entry released. Do not modify original assets or broaden receipt exemptions.
3. Update current entry links and all owned bindings, keeping upstream model/tool/schema identities independent. The new source manifest remains candidate; publication rendering follows the existing explicit release target.
4. Temporarily omit only new-version image sections for a clean capture bootstrap. Commit/push the bootstrap and dispatch CI on its exact branch SHA before opening the official PR. Independently retain its audit/capture receipts, import only original enumerated PNGs and receipts, then restore current embeds.
5. Complete public/offline checks, genuine source/hash/version image validation and visual inspection. Hosted interactive/installer qualification runs on disposable CI; local checks remain headless.
6. Open the official ready PR only after captures are restored. Count its automatic Codex round plus at most one manual second round, resolve every actionable external comment, audit final exact-source CI and clean owned transient output.
7. Record an executable post-merge procedure: fresh exact-main publication-target candidate, retained exact package CPU/fresh-restored processing acceptance, publish the same accepted bytes and publicly verify matching Pages.

## Project Structure

- VERSION; root/site/desktop/frontend manifests and locks; current internal/cmd runtime bindings and owned fixtures.
- scripts/release.py and tests/release_test.py, tests/release_history_test.py, tests/recovery_release_history_test.py: bounded preparation/history correction.
- README.md, CONTRIBUTING.md, site/README.md, docs/README.md, docs/versions.json, docs/v1.0.3/, schemas/v1.0.3/ and docs/screenshots/v1.0.3/.
- Historical docs/v1.0.2 and schemas/v1.0.2 only if exact published-authority reconciliation is demonstrated after copying current corrections forward.
- internal/runtime/startup_test.go: bounded test-only owner/client elapsed observations; product startup deadline and protected-input/one-owner assertions remain unchanged.
- CHANGELOG.md and specs/059-v103-release/{spec,plan,research,data-model,quickstart,tasks,verification,highlights}.md.

## Tracking and Delivery

Issue #196, milestone 43, Project 7; Slice S059, Release 1.0.3. #147 remains open. Push/PR publication is explicitly authorized. Final handoff requires green final-source checks, resolved findings, compact independent evidence and an unmerged PR. Bootstrap captures and final delivery evidence remain distinct.

## Qualification repair (2026-10-29)

Repeated all-package runs isolated a cold-owner catalog deadline under unchanged product clocks. The Windows directory guard unconditionally reapplies its inheritable DACL even when current owner/access already exactly satisfy the invariant. Add a private setter seam for deterministic regressions; check current ownership and PrivateDirectory freshly before deciding whether normalization is necessary. Never cache ACL authority. Invalid/currently changed owned ACLs still normalize and setter failure remains redacted/error. The original ten-second product startup clock and all four protected-input clients remain unchanged. This bounded Windows repair is tracked by appended T019 and FR-010.
