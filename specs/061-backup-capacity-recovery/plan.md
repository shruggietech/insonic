# Implementation Plan: Backup/restore capacity-failure recovery

**Branch**: `codex/061-backup-capacity-recovery` | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

**Input**: S061 / #200, bounded continuation of #147.

## Summary

Preserve actionable native capacity causes, reject partial output, observe cleanup, and prove identical retries without changing the established backup pins or restore activation protocol. Use private per-call file operations for controlled error injection; public operations always use actual filesystem operations. Add actual confined filesystem-obstruction qualification to source and extracted packages.

## Technical Context

**Language/Version**: Go 1.27.1, Python qualification harness, Node maintainer checks.

**Primary Dependencies**: Existing catalog, artifact, workspace and contracts packages; no new dependencies.

**Storage**: Filesystem/S3 artifacts and SQLite/PostgreSQL catalogs retain their existing supported contracts.

**Testing**: Go regression tests and vet; Python strict receipt mutation tests; root Node checks/tests; affected site checks; Windows/Linux CI source and package acceptance.

**Target Platform**: Windows/Linux x86-64.

**Project Type**: CLI-first shared runtime with desktop callers.

**Performance Goals**: Retain streaming copies and bounded fixture sizes. Qualification target 1,800 s, hard 3,600 s after readiness, separate preparation/combined accounting, 60 s provisional finalization reserve.

**Constraints**: No local GUI/focus-stealing acceptance. No release, tag or merge. At most two Codex rounds. Retain compact independently read-back receipts before owned staging cleanup.

**Scale/Scope**: One backup/restore recovery slice; injected I/O faults and confined filesystem obstructions, not host exhaustion or live-provider quota qualification.

## Constitution Check

- Owner scope and authorization: pass. Push/PR explicitly authorized; merge reserved to owner.
- Shared CLI/runtime behavior: pass. No product fault-injection flags or environment switches.
- Immutable source/provenance: pass. Every fault campaign checks source, bundle and existing destination identities.
- Backend/security parity: pass. Preserve catalog fencing, version compatibility, credential redaction and existing backend integration tests.
- Meaningful verification/timing: pass. Native mappings plus failure/retry tests; strict source/package evidence under maintained budgets.
- Documentation/version integrity: pass. Update current authoritative contracts and unreleased changelog; keep version 1.0.3 and planning internal.

Post-design recheck: all six gates pass; no constitution exception.

## Project Structure

`specs/061-backup-capacity-recovery/` contains spec, plan, research, data model, contracts, quickstart, tasks and requirements checklists.

- `internal/contracts/capacity_*.go`: shared native capacity predicate; `operator.go` and current response schema: generated cleanup reason.
- `internal/catalog/store.go` and `internal/catalog/capacity_test.go`: SQLite/PostgreSQL capacity classification and confined actual SQLite page-limit rollback/retry.
- `internal/artifact/contract.go`: reuse shared predicate, preserving cancellation precedence.
- `internal/backup/io.go`, `service.go`, `progress.go`: private operations, write/finalization and cleanup handling.
- `internal/backup/capacity_test.go`: injected orchestration and recovery regression cases; existing security/backend tests remain required.
- `scripts/cli_backup_qualification.py`: actual obstruction/failure/retry cases, strict validator.
- `scripts/cli_contract_qualification.py`, `scripts/release.py`: six source/package receipt contexts.
- `tests/cli_backup_qualification_test.py`: missing/mislabeled/mismatched evidence rejection.
- `docs/v1.0.3/storage.md`, `CHANGELOG.md`: authoritative resulting behavior and dated process decision.

## Execution Phases

1. Write failing regression tests for wrapped capacity, incomplete writes and cleanup causes.
2. Centralize native mappings and implement per-call private backup I/O helpers; preserve existing recovery identity and repeat durability checks on completed-output retry.
3. Exercise creation and restore boundaries, source/bundle identity, fencing and identical retry; repair only demonstrated gaps.
4. Integrate strict actual-obstruction qualification for source/packages, then run CI-parity checks and bounded external review.

## Decision Log

- Reuse established catalog activation/pins rather than add a second journal. Existing fencing tests already prove the required authority transitions.
- Share native mapping rather than duplicate artifact OS error lists. Preserve wrapped contract errors through errors.As.
- Join primary and cleanup failures internally; expose only generated `backup_cleanup` context with the primary classified code. Never expose paths/provider text.
- Separate injected native error tests from actual CLI filesystem observations. Public binaries have no injection surface; obstruction receipts explicitly disclaim physical exhaustion.
- Owner authorization supersedes the autopilot skill's default pre-push pause. Final PR merge remains reserved to the owner.
