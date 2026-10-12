# Implementation Plan: S019 CLI operator feedback

**Branch**: `codex/019-cli-operator-feedback` | **Date**: 2026-10-11 | **Spec**: [spec.md](spec.md)

## Summary

Complete #56/#57/#58/#61 through shared generated failure context, startup-only child reports, bounded human rendering/progress and model response context. Preserve machine envelopes and execution authority.

## Technical Context

Go 1.27.1 CLI/shared runtime (the repository-declared toolchain) and Wails desktop bridge; existing SQLite/PostgreSQL catalog and filesystem/S3 artifacts. Existing JSON Schema 2020-12 contracts, exact-number decoders, native IPC and hidden process flags remain authoritative. No new external dependency is required. Linux/Windows amd64 and macOS ARM64 native and package lanes, npm/schema/site, Go, Python and frontend checks provide qualification. Early failure detection is bounded to two seconds after termination, full startup to ten seconds, human output to 128 lines/64 list entries with bounded values, and private elected logs to three 1 MiB files.

## Constitution Check

Before research and after design: PASS. CLI-first shared operations preserve desktop wrappers. No provider rerouting, per-item reviews, acoustic tests, historical-byte edits, model identity changes or backend narrowing. 1.0.1 candidate version remains synchronized. Publication is explicitly authorized; owner merge/release remain final boundaries.

## Project Structure

- `cmd/insonic/`: invocation controls, human renderer, work observer and startup reporter.
- `internal/contracts/`: fixed generated reason vocabulary preserving existing codes.
- `internal/process/`, `internal/runtime/`, `internal/workspace/`: hidden startup report, bounded input/probe/read lifecycle and source-specific failures.
- `internal/diagnostics/`: private elected generated event log with rotation.
- `internal/secrets/`, `internal/backup/`, `internal/app/`: truthful causes/credential states and durable result context.
- `internal/models/`: compatible inspection and model-role materialization.
- `desktop/bridge.go`: retain safe startup errors.
- `schemas/v1.0.1/`: current response extensions only; immutable stored documents are unchanged.
- `docs/v1.0.1/`, `CHANGELOG.md`: current behavior and concise changes.
- `specs/019-cli-operator-feedback/`: spec/research/model/contracts/tasks/evidence.

## Decisions and implementation sequence

1. Preserve broad error codes/exits; carry allowlisted generated reasons and fixed messages at actual failing predicates. Durable error results retain those reasons. Do not guess corruption from all conflicts or damage from storage unavailability.
2. Use an explicit stdout os.Pipe and one Cmd.Wait goroutine for a startup-only 4 KiB JSON line. Null stderr; no raw output. Ready/failure reports carry correlation, version, stage and generated code/reason. Keep detached flags, bound initial/live-owner probe, protected stdin write and report read under one ten-second context. A lost launcher must not kill a ready owner.
3. Use one shared human renderer for every admitted operation with operation-specific headings/actions and bounded readable recursive fallback; omit known credential input, durable work internals and requested encoded document bytes, marking omissions. Preserve public field names and arbitrary query columns; remove terminal controls. JSON stays unchanged.
4. Add a wait observer to current same-process polling. Terminal-only changed state/phase/count observations; warnings/completion remain visible in ordinary output. No fabricated totals/percentages.
5. Model inspection returns incompatible context successfully; execution guards remain strict. Add response-only readiness and manifest-declared role/publication/lease context, deriving acquisition state from durable work. Dataset readiness means ready for preparation, never proven acoustic training.
6. Global verbose/debug/quiet/log-level/log-file controls are admitted before effects. Explicit log-file uses private generated JSON lines with correlation only; debug never copies content, arbitrary errors, paths, input or provider bodies to logs. An invalid elected log destination is diagnosed before command mutation.

## Verification

Write focused failing regressions before corresponding production changes. Cover secret absence, detached parent exit, competing owner/input handoff, malformed/oversized reports, partial dates, clean JSON/redirected progress, roles/leases and dataset rejection. Run `npm run check`, `npm test`, all Go tests/vet, Python and frontend checks, site/offline build and affected native/extracted-package qualification. Publish official PR, wait/resolve every finding, optionally request one second round, verify exact final-head CI and stop for owner review.

## Complexity Tracking

No constitution exception or new infrastructure. Keep generic artifact materialization and persisted model/work states unchanged.
