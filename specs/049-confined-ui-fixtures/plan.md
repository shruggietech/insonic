# Implementation Plan: Confined UI Fixtures

**Branch**: `codex/049-confined-ui-fixtures` | **Date**: 2026-10-25 | **Spec**: [spec.md](spec.md)

## Summary

S049 repairs #176 with a small shared owned-fixture allocator used by application capture and documentation browser checks. Validate the actual temporary parent, preserve allocated-directory identity before initialization, clean handled failures, and gate successful receipts on verified removal. Publish an unmerged PR after exact-head CI and at most two Codex review rounds.

## Technical Context

**Language/Version**: Existing Node.js ESM tooling (CI Node 22), Python qualification wrappers.

**Primary Dependencies**: Existing filesystem APIs, Playwright Core and Chromium, hidden Python process-tree runner; no new dependencies.

**Storage**: Unique disposable directories strictly beneath a canonical designated parent: actual system temp when outside home/program directories, otherwise ignored repository .local/build/ui-fixtures when outside those boundaries for public application captures; actual system temporary directory for documentation profiles. Both use the same allocation/ownership validation.

**Testing**: Node test runner, existing repository checks, actual source-bound application capture and documentation UI checks on Windows and Linux in CI.

**Target Platform**: Windows and Linux x86_64.

**Project Type**: Internal maintainer tooling in the existing CLI/desktop repository.

**Performance Goals**: Existing qualification target 1800 seconds and hard limit 3600 seconds after verified prerequisites, 60-second finalization reserve; separate preparation/qualification/combined accounting. No timing-policy changes.

**Constraints**: No drive-root allocations or arbitrary cleanup sweeps; no owner-profile/browser attachment; hidden noninteractive processes; preserve clean source identity, receipt checks and cleanup publication gates.

**Scale/Scope**: Two allocation consumers, one small lifecycle helper, existing tests and minimal CI coverage extension.

## Constitution Check

Pre-design and post-design gates pass: scope #176 only; filesystem identity checked before recursive deletion; no new routing, publication restrictions or owner gates; versions remain 1.0.1; internal planning stays in specs; history in CHANGELOG.md; Windows/Linux verification and existing timing acceptance remain mandatory. Explicit owner authorization covers push/PR/review remediation, not merge. No violations or complexity exemptions.

## Project Structure

- `scripts/owned-ui-fixture.mjs`: temporary-parent validation, allocation identity, initialization failure cleanup, verified idempotent removal and residual diagnostics.
- `scripts/capture-application-ui.mjs`, `scripts/check-docs-ui.mjs`: shared allocation, existing child-release ordering and cleanup-gated reports.
- `tests/capture-application-ui.test.mjs`, `tests/docs-ui.test.mjs`: actual filesystem allocation/failure regressions and Windows path semantics without drive-root writes.
- `scripts/qualify-ui-capture.py`, Windows docs qualification wrapper if needed, `.github/workflows/native-qualification.yml`: exact-source platform coverage and retained diagnostics.
- This directory: spec, plan, research, data-model, quickstart, tasks and compact verification. No external API contract is added by this internal repair.

## Implementation Strategy

Test first: reproduce wrong path semantics without allocation, then failing tests for tool allocation and initialization/cleanup outcomes. Implement US1 confinement, then US2 lifecycle and receipt gating. Audit nearby allocation expressions. Run affected tests, full npm gates, then actual Windows/Linux browser qualification through exact-head CI. Preserve detailed locations in qualification diagnostics, not public screenshot receipts. Keep one automatic review and at most one manually triggered second round; address every finding and stop unmerged.
