# Implementation Plan: S037 Desktop density and standard Exit

**Branch**: `codex/037-desktop-density-exit` | **Date**: 2026-10-17 | **Spec**: [spec.md](spec.md)

## Summary

Resolve #112 throughout the desktop with governed-token consumer composition and real geometry checks; add #113 native File > Exit through the existing lifecycle. Preserve source/data/runtime ownership and keep all native/package gates.

## Technical Context

**Language/Version**: Go1.27.1, TypeScript5.9.3/React19.3, Node22+, Python3.12+ maintainer tooling.
**Primary Dependencies**: pinned Wails2.16.0, delivered BrandBuilder3.0.1 kit, Playwright-core1.62.1.
**Storage**: no schema/configuration change; existing SQLite/PostgreSQL and artifact ownership remain authoritative.
**Testing**: meaningful portable unit/lifecycle tests, actual headless Chromium geometry, Windows/Linux native menu and shared-work proofs, required repository/frontend/site suites and full CI.
**Target Platform**: Windows/Linux amd64 desktop.
**Project Type**: CLI-first shared runtime with native desktop wrapper.
**Performance Goals**: bounded probes, no added production polling, CI within660 seconds.
**Constraints**: UTF-8/LF; no kit edits/new design literals; preserve44-unit targets, focus and durable jobs; no local visible-console/native-installer launches.
**Scale/Scope**: two issues, six workflow groups, two window sizes and explicit renderer scaling/text matrix. No release or merge.

## Constitution Check

Pre-research and post-design gates pass: owner scope preserved; UI composition and native close only; shared CLI/runtime and provenance unchanged; backend/security contracts retained; isolated automated evidence without an attended owner gate. Brand adaptation is explicitly supported by the pinned consuming-host guidance. No exception or constitution change is required.

## Project Structure

- `desktop/frontend/src/components.tsx`, `app.css`: native selection label composition, compact layout, intrinsic actions and contained overflow.
- `desktop/frontend/src/density.ts`, `qualification.ts`: source-shared geometry observer and native workflow observations.
- `scripts/desktop-density-capture.mjs`, `capture-application-ui.mjs`: actual production frontend and Go bridge, renderer matrix and source-bound representative screenshots.
- `desktop/menu.go`, `menu_test.go`: native menu composition and synchronized startup/quit controller.
- `cmd/insonic-desktop/main.go`, `exit_qualification*.go`: normal lifecycle integration and explicit bounded native test hooks.
- `scripts/qualify-desktop-exit.py`, `qualify.py`, `package-desktop.py`: private external-runtime/held-acquisition proof and packaged desktop checks.
- `docs/v1.0.1/desktop.md`, `development.md`, `docs/screenshots/v1.0.1`, `CHANGELOG.md`: public behavior, consumer adaptation and witnessed images.

## Execution and decisions

1. Establish failing geometry/label-target and menu/startup regressions before product edits.
2. Implement token-based compact host composition, retaining native controls and full label targets.
3. Integrate native File > Exit with synchronized startup context and ordinary shutdown.
4. Extend existing source capture/native journey hooks; prove detached work survives Exit and playback closes.
5. Run focused and required checks, converge requirements, commit cleanly, run exact-head full CI and publish clean-source images without rewriting receipts.
6. Complete at most two code/security rounds, verify all threads and final CI, stop unmerged.

Custom reviewer-owned checklists are unnecessary here: the built-in requirements checklist covers all UX/lifecycle quality criteria; implementation proof is tracked separately in tasks and verification. No new approval gate is introduced.

Read-only research is delegated as required by speckit-plan. Product edits remain sequential under the primary agent to prevent shared-file conflicts. No architectural or ownership expansion is needed.
