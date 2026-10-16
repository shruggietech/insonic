# Implementation Plan: Media lifecycle correctness

**Branch**: `codex/033-media-lifecycle-correctness` | **Date**: 2026-10-15 | **Spec**: [spec.md](spec.md)

## Summary

Fix #143 accepted refresh cleanup, #142 packaged subtitle encoding, #141 canonical playback MIME, #140 manifest parser admission and #145 durable acquisition causes. Extend existing qualification; keep unrelated audit issues open.

## Technical Context

**Language/Version**: Go 1.27.1; Python 3.12 maintainer helpers; Node 22+ repository/docs checks.
**Primary Dependencies**: Existing catalog/artifact/runtime contracts, pinned FFmpeg/FFprobe/ExifTool and Cueson.
**Storage**: Existing SQLite/PostgreSQL and filesystem/S3 adapters; no schema migration.
**Testing**: Go unit/native/integration tests and existing packaged CLI qualification; Python and Node maintainer tests; site/docs build.
**Target Platform**: Windows and Linux x86-64; CLI and desktop shared runtime.
**Performance Goals**: Bounded tiny fixtures and final hosted CI below 660 seconds.
**Constraints**: UTF-8 without BOM, LF; hidden noninteractive Windows children; no inference/model downloads, private corpus or credentials.
**Scope**: Five defects; partial fixture contribution to #114. Publication includes push/PR and at most two review rounds, stopping before merge.

## Constitution Check

Before research and after design: PASS. Shared CLI-first behavior, immutable owner inputs, canonical/source provenance, backend parity, recoverable leases, redaction, no new review gates, version alignment and separate official/internal documentation remain intact. Owner explicitly authorized push and PR, overriding the skill's default pre-push halt. No merge/release authority exists.

## Project Structure

- `cmd/insonic/commands.go` and parser/domain tests: conditional transcript target validation and accurate contextual usage.
- `internal/library/service.go`, `admission.go` and tests: shared accepted cleanup and real-error precedence.
- `internal/library/embedded.go` and tests: shipped encoder names with retained output formats.
- `internal/library/acquisition.go`, contract errors and tests: classified deadline/byte-limit causes with redacted corrective messages.
- `internal/app/desktop_ops.go` and playback tests: canonical format before captured source MIME; retain preview overrides and legacy fallback.
- `scripts/cli_contract_qualification.py` and reusable fixture module if warranted: installed-binary regressions with exact companions.
- Current `docs/v1.0.1/` contracts and `CHANGELOG.md`: resulting behavior and unreleased fixes.
- This directory: spec, research, model, contracts, quickstart, tasks and verification evidence.

## Research and Design Decisions

1. Reuse existing cleanup rows and startup/periodic sweeps. Make real errors outrank deferred conflicts, and use accepted-state cleanup in both refresh commit and same-work reconciliation paths. Do not swallow arbitrary failures or invent a new worker.
2. Canonical format determines playback MIME before original metadata. Preview encoding retains its own output MIME. Unknown canonical formats do not inherit original MIME.
3. Use FFmpeg's enabled SubRip encoder name while retaining SRT output. Prove execution with shipped companions rather than testing an argument string alone.
4. Express --record conditionally for positional transcript import. Preserve manifest/per-item validation and rejection of conflicting forms. Enforce transcript-only intent before dispatching the shared importer, including mixed batches and inherited replacement options.
5. Reuse existing stable error vocabulary when suitable; distinguish explicit caller cancellation, acquisition deadlines and byte bounds without retaining arbitrary network errors/bodies. Durable item/work errors already persist categories; extend this path only if needed for actionable causes.
6. Put tiny deterministic lifecycle fixtures in the existing package qualification flow, including paths with spaces/Unicode. Limit extra work through reuse and bounded loops.

## Phases and Verification

1. Establish regression cases before each fix; show relevant failures.
2. Implement stories in priority order; independent parser/playback/subtitle changes can proceed after analysis, but shared cleanup/acquisition files remain sequential.
3. Extend backend parity and packaged qualification. Include refresh held lease, same-work reconciliation, release/expiry/restart, real delete failure precedence, canonical MIME, embedded formats, manifest-only import and HTTP failure persistence.
4. Run npm check/test, relevant Go tests/vet, Python tests, site tests/build and native/package checks. Hosted CI qualifies both platforms and alternate backends on the final head.
5. Commit, push and open one PR closing only the five selected issues. Attach the PR to this chat. Resolve/recheck at most two review rounds and all resulting review threads; retain exact-head CI evidence and stop for owner merge.

## Complexity Tracking

No constitution exceptions, new dependencies, migrations or parallel qualification framework are planned. Preserve real storage failures even when accepted state already committed, with that distinction documented and tested.
