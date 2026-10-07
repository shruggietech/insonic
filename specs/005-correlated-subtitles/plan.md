# Implementation Plan: S005 Current correlated subtitles

**Branch**: `codex/005-correlated-subtitles` | **Date**: 2026-10-07 | **Spec**: [spec.md](spec.md)
**Input**: #21/#7 and owner fixture/CI constraints.

## Summary

Use pinned Cueson subprocesses for normalization/validation/export, a current recording domain containing the sole embedded document, reference-only speaker/evidence consumers, and the existing fenced Work/artifact service for publication and cleanup. Extract mapped audio with exact pinned FFmpeg; lazy local Python adapters run real faster-whisper and pyannote only for configured product requests or explicit maintainer qualification. CI never runs/loads/downloads inference.

## Technical Context

**Language/Version**: Go 1.27.1 shared runtime; Python 3.11+ local workers; Node22 documentation.
**Primary Dependencies**: Cueson1.2.0 exact six packages/local schema; existing pinned FFmpeg/ffprobe distribution; faster-whisper1.2.1/CTranslate2 4.8.2; pyannote.audio4.0.7.
**Storage**: Existing filesystem/S3, SQLite/PostgreSQL and artifact leases.
**Testing**: Go/race/vet, npm maintainer/schema/site checks, Python deterministic tests, three-OS real Cueson/media and adapter fixtures; explicit Windows maintainer inference outside CI.
**Target Platform**: Windows/Linux/macOS shared bridge; actual engine qualification platform is reported separately.
**Project Type**: CLI-first runtime plus desktop bridge.
**Performance Goals**: CI jobs below10min; fixture media aim below10MiB; bounded worker inputs/output/time/threads; paged/leased large outputs.
**Constraints**: No retained native generated transcript/diarization report alternatives, secret argv/logs, ambient downloads, automatic hosted/CUDA fallback or inference in CI.
**Scale/Scope**: Two issues, four stories, current record and reference contracts; broad presets/speaker UX/graph/training remain separate.

## Constitution Check

Pre-research PASS: owner scope and CI override explicit; source identity retained; current-result rule overrides legacy alternate transcripts; shared CLI/bridge; supported backends; hidden subprocesses; exact pinned dependencies and self-contained docs. Immutable corpus means source references/provenance with invalidation, not copied obsolete assignments. Post-design PASS with frozen historical migrations and latest snapshot proofs, explicit local model selection and truthful platform evidence.

## Phase 0 research

Spec Kit research agents investigate Cueson exact schema/CLI, local engine integration, and redistributable real fixtures. Decisions/evidence are consolidated in [research.md](research.md). Fixture exact crop properties are implementation qualification, not unresolved product intent.

## Phase 1 design

[Data model](data-model.md), [runtime contract](contracts/runtime.md), [validation guide](quickstart.md).

- Add a current Recording domain keyed to LibraryEntry, embedding sole Cue JSON plus hash/state/source mapping and noncontent provenance. Library show/master output embeds this record; existing library identity remains unchanged.
- Add external recording/local UUID to known speaker mappings and reference-only segment/evidence records. Reject duplicate assignment storage; legacy timed segment/preparation data is invalidated/reconciled under owner-approved #21, without source deletion.
- Freeze historical schema3 DDL/digest before adding schema4; newest accepted receipts bind current Recording/mapping state. Receipts/work results store identities/digests/diagnostics, never duplicate documents/turn arrays.
- Processing stages keep native transcript/diarization bytes in bounded memory/private scratch, removed on success/failure/cancel. Only the finalized document and elected mapped audio become current managed results.
- Atomic commit uses live work fence and expected current/source revisions. Queue removed publications in durable cleanup; use existing reference checks/retirement and no stale downstream preparation.
- Independent diarization reruns reuse current subtitle/cue identity; known-speaker mapping changes do not touch subtitle text. Unsupported/unrepresentable results return explicit diagnostics.
- Existing stored subtitle inputs are originals; generated SRT is temporary, not a retained alternative.
- Deterministic tests inject stage results at the adapter boundary, with real media and Cueson. Actual inference appears only in configured product code and explicit maintainer script.

## Project Structure

`internal/subtitles/` owns exact driver/schema/assembly/export.
`internal/processing/` and `scripts/processing-worker.py` own local lazy adapters/mapped audio.
`internal/catalog/` owns current recording/reference migrations/proofs/cleanup.
`internal/app/`, `cmd/insonic/`, `desktop/`, `schemas/` share routes/contracts.
`tests/fixtures/media/` owns actual media/license/provenance/references.
`scripts/qualify.py` and `scripts/qualify-processing.py` separate deterministic native checks from explicit real inference.
`docs/`, `specs/`, `CHANGELOG.md` reconcile owner current-result contract.

## Dependencies and parallel strategy

Catalog contracts precede integration. After analyze, subtitle driver, processing worker and fixture qualification can be implemented/tested by separate owners in parallel under the Spec Kit tasks team strategy; parent serializes shared schema/runtime/CLI/catalog edits. Tests precede implementations. Every story completes, not just the supplied-subtitle increment.

## Complexity Tracking

No unjustified constitution violation. Local Python environments/model setup are explicit adapter prerequisites; full installers remain #10. No new external provider routing.
