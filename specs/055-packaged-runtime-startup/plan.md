# Implementation Plan: Windows packaged runtime startup diagnosis and repair

**Branch**: `codex/055-packaged-runtime-startup` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

## Summary

Diagnose #191, reproduce the live-owner arbitration defect, repair it within the existing startup budget, and retain bounded correlated terminal failure evidence before cleanup. Earlier main failure remains an observation, not proof of its native cause.

## Technical Context

**Language/Version**: Go 1.27.1; Python 3.12 qualification; Node 22+ repository checks.

**Primary Dependencies**: Existing runtime IPC, workspace flock, Go startup process helper, Python ProcessTree/ConPTY and terminal campaign. No added dependency.

**Storage**: Existing private workspace/runtime state and CI build/native receipts. No schema/product data migration.

**Testing**: Go owner-arbitration tests; Python evidence/privacy/cleanup tests; repository check/test; exact-head Windows/Linux source/package CI.

**Target Platform**: Windows primary defect; Linux parity remains required.

**Project Type**: Shared CLI/runtime plus maintainer qualification.

**Performance Goals**: Existing ten-second aggregate runtime startup bound; no new per-probe allowance or product retry clock.

**Constraints**: Hidden noninteractive children, immutable image/source identity, eight terminal cases, no owner library/installer/UI execution locally, bounded generated evidence only.

**Scale/Scope**: Runtime owner admission and one terminal campaign's eight clients. No historical package or release work.

## Constitution Check

Pre-research and post-design PASS: CLI/shared runtime boundary; preserve owner authority, privacy, durable state, hidden process creation and exact-source provenance. Retain 1800/3600 qualification policy, phase accounting, natural exits and 60-second finalization reserve. No new merge gate or attended release ritual. Owner permits push/PR/reviews and reserves merge.

## Phase 0: Research

[research.md](research.md) records source trace, known race and alternatives. The installed Spec Kit plan's requested read-only research agent independently identified the same mismatch: one 500 ms failed probe immediately starts a detached owner, while the actual-terminal job intentionally forbids breakaway. Native error for the historical main failure remains unestablished.

## Phase 1: Design

1. Reproduce the arbitration defect with a real owned IPC listener and held workspace owner lock. The first readiness probe fails; a subsequent probe succeeds. Old code attempts a launch despite known ownership. Cover held-lock deadline, released-lock restart, real bootstrap errors and protected credential loading.
2. Check authority before new owner launch and wait/probe while held within the unchanged enclosing context. Preserve cold races: a competing launcher can still win after an unlocked observation, and runtime_owner report reconciliation remains required. Do not permit terminal breakaway.
3. Elect the existing CI-only process diagnostic in a copied terminal environment. Retain bounded failure observations before terminal/runtime teardown: source image digest, generated identities, client/owner poll results, generated log events and only matching client-PID allowlisted startup records. Do not serialize exception strings, terminal output, paths, environment or arbitrary JSON fields.
4. Add failing evidence/privacy/correlation regressions, then implement. Missing evidence stays explicit, never a fabricated native cause. Failure evidence does not satisfy successful terminal receipt validation.
5. Run local checks, push/open official PR, monitor exact-head CI, preserve failed attempts before full reruns, resolve at most two Codex rounds and converge. Stop unmerged. Merged-main acceptance follows the owner's later merge.

## Project Structure

- `internal/runtime/runtime.go`, `internal/runtime/startup_test.go`: owner arbitration and regressions.
- `internal/process/startup_diagnostic.go`: existing sanitized native startup producer, reused unchanged unless integration requires correction.
- `scripts/cli_terminal_qualification.py`: terminal election and failure-capture lifetime.
- `scripts/terminal_failure_evidence.py`, `tests/terminal_failure_evidence_test.py`: bounded evidence implementation and privacy/correlation tests.
- `tests/cli_terminal_qualification_test.py`: environment/cleanup integration.
- `docs/v1.0.2/cli.md`, `docs/v1.0.2/development.md`, `CHANGELOG.md`: authoritative behavior and concise history.
- `specs/055-packaged-runtime-startup/`: planning and compact verification record only.

## Complexity Tracking

No constitution exception. Reuse existing lock, probes, native hook and artifact upload conventions; avoid a new service, retry engine or public diagnostic API.
