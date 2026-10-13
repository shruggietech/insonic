# Implementation Plan: S021 Windows/Linux distribution and installation

**Branch**: `codex/021-windows-linux-distribution` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

## Summary

Retire owned macOS support and require Windows/Linux CLI/desktop packages, then make Windows ZIP selection, first use and maintenance truthful. Preserve historical 1.0.0 bytes and historical record validation.

## Technical Context

Go shared runtime/CLI and Wails desktop; Python package/native/release tooling and workers; React/Next.js online/offline docs; current JSON Schema 1.0.1. Existing SQLite/PostgreSQL, filesystem/S3 and LadybugDB/ArcadeDB remain supported. Windows non-Git subprocesses use windowsHide/CreateNoWindow with redirected I/O; tests run foreground to completion. Existing feature-branch/squash model and owner-authorized two-round PR handoff are retained.

## Constitution Check

I: Explicit owner platform retirement governs scope; no new usage restrictions. II: Constitution 3.0.0 reconciles the removed macOS obligation before Analyze; shared CLI/GUI behavior and Windows/Linux packages remain required. III: Installation maintenance never deletes user sources, workspace/media/models or credentials. IV: Existing backend/routing/credential checks remain. V: TDD focuses on candidate completeness, unsupported targets and helper containment/cleanup; full remaining platform and backend qualification stays mandatory. Gates pass after the explicit constitution update outside Analyze.

## Project Structure

- `.github/workflows/`: remove macOS dependencies/inputs/jobs without weakening remaining lanes.
- `internal/`, `cmd/`, `desktop/`: remove native Darwin implementations and platform branches; retain explicit unsupported-host errors and unavailable fallbacks.
- `scripts/`: current platform selection/build/qualification and release candidate four-variant enforcement; historical metadata validation isolated from current builders.
- `schemas/v1.0.1/`, `tests/`: current release matrix/schema/examples and meaningful historical/tamper/completeness regressions.
- `docs/v1.0.1/`, `README.md`, `site/`: authoritative support/install contracts and version-aware current download presentation.
- `specs/roadmap.md`, `scripts/github-bootstrap.mjs`: active planning/metadata support reconciliation; historical completed-slice evidence remains historical.

## Decisions

2026-10-13: Explicit owner authorization removes the old three-platform governance requirement. Constitution version 3.0.0 records the backward-incompatible obligation removal; product/docs/schema stay on the existing unreleased 1.0.1 candidate, with platform withdrawal explained in changelog/release highlights. No tag/release/promotion is performed.

2026-10-13: Current generation and qualification require exactly four Windows/Linux variants. Historical 1.0.0 schemas/documents/artifacts remain immutable; current trusted validators may inspect historical macOS metadata only under the historical version contract. Deleting old records or weakening exact revision/hash/source checks is rejected.

2026-10-13: Harden the generated Windows helper through resolved containment and explicit attempt ownership. Reject targets before copying; stage/copy failure reports the original cause and cleans only the exact owned new attempt. Account for aliases, reparse paths and substituted targets. Use disposable test fixtures, never real user installations.

2026-10-13: Shared current packaging help distinguishes Desktop + CLI from CLI-only and explains actual ZIP copy/update/removal. Current runtime has no stop command; document and qualify the real idle/closure route rather than inventing syntax. MSI, managed first-run setup and full security screenshot flows remain separate issues.

## Verification

Run npm check/test, Go test/vet, Python tests, frontend/typecheck, site online/offline checks and extracted Windows CLI/desktop qualification. CI runs both remaining core/native/media/package platforms, four package variants, live alternative-backend fixtures and candidate assembly. Required repo checks audited: foundation/docs; no obsolete macOS check needs mutation. Review cap is two, then green exact-head CI and owner merge handoff.
