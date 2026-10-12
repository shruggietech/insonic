# Implementation Plan: Saved import presets and executable documentation

**Branch**: `codex/020-import-presets-executable-docs` | **Date**: 2026-10-11 | **Spec**: [spec.md](spec.md)

## Summary

S020 closes #70 and #62 together. Saved import presets become catalog-owned configuration with shared CLI/desktop lifecycle and admission resolution. Current documentation is corrected and qualified through extracted packages.

## Technical Context

Go 1.27 shared runtime and CLI; TypeScript/React desktop; Python packaging/qualification; Node repository and Next.js site checks. SQLite/PostgreSQL remain equivalent catalog adapters; filesystem/S3 destinations remain equivalent storage adapters. Existing Cueson, configured pipelines and durable recording work are reused.

Bound presets to 1000 identities and a 1 MiB versioned registry. Keep full import payload within the existing 8 MiB budget and responses within 1 MiB. CI remains targeted/cached and under ten minutes. Windows processes use hidden, noninteractive launches.

## Constitution Check

Before research: pass. Scope matches owner-directed #70/#62; no eligibility/review gate, no credential values, CLI-first shared desktop contracts, no release claim. After design: pass. Immutable storage profile revisions support explicit configured destinations; publication reads route by recorded authority rather than the current default. Existing catalog settings avoid unnecessary DDL. Historical release schemas/pages stay immutable.

## Project Structure

- `internal/catalog/import_presets.go`: typed registry mutations, immutable receipts, uniqueness/CAS and backup validation.
- `internal/library/types.go`, `manifest.go`, `service.go`: explicit overrides, options and per-item admission callbacks.
- `internal/app/import_presets.go`, `work.go`: shared CRUD, resolution, frozen ordinal elections and processing dispatch.
- `internal/artifact/routing.go`, `service.go`: immutable destination routing for publication, reads and backup.
- `internal/contracts/`, `schemas/`, `cmd/insonic/`: current operation/payload grammar and help.
- `desktop/frontend/src/`: preset management and import/refresh selection.
- `docs/v1.0.1/`, `scripts/package-desktop.py`: current executable examples and qualification.

## Decisions

2026-10-11: Store presets in the existing versioned settings journal through typed catalog mutations. A new DDL table would duplicate existing atomic CAS/portable history; generic app-side read/modify/write would corrupt idempotent replay after later edits. Typed receipt replay occurs before mutable reads and returns the accepted frozen preset.

2026-10-11: Destinations are explicit nonsecret filesystem/S3 storage profiles, admitted immutably with preset mutations. A mere profile label or directory hint cannot route existing reads correctly. Route storage access by each publication's profile identity/revision and freeze the import target before work admission.

2026-10-11: Preserve request overrides separately from manifest defaults/items. Resolve presets and merge before timezone capture. Frozen per-ordinal selections live in the request envelope because accepted item scrub replaces source items with receipt references. Reject malformed supplied frozen elections at the public boundary.

2026-10-11: Freeze existing recording pipeline elections/tools/model selections at admission and enqueue deterministic recording work after canonical admission through a library callback. Retry reconciles deterministic child work instead of rereading presets/pipelines. All required selection validation precedes model acquisition/work queueing.

2026-10-11: Qualify supported documentation procedures in the existing extracted-package smoke, shared by both CLI and desktop variants. Repair current source/rendered/offline contracts and preserve v1.0.0 archives. No new release is part of this slice.

## Verification

TDD covers credential boundaries, workspace scoping, CAS/replay, SQLite/PostgreSQL parity, snapshot/backup, preset errors, precedence, refresh, frozen pipeline/destination retry and deterministic child work. Run npm check/test, Go test/vet, Python checks, frontend checks, online/offline site and extracted Windows CLI/desktop packages. PR review is bounded to two rounds; wait for exact final-head green CI and stop for owner merge.
