# Implementation Plan: Resumable long-recording processing

**Branch**: `codex/063-resumable-long-recording` | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

## Summary

Implement #212-#216 as one shared runtime path: freeze deterministic source-sample windows, stream bounded decoded audio, persist fenced verified results, stitch timing evidence, and accept one current transcript. Existing short requests remain usable; new long sources automatically select windowed processing and explicit window options can elect it for short sources. Historical frozen payloads keep their elected behavior.

## Technical Context

**Language/Version**: Repository-pinned Go 1.27.1, Python worker and maintainer tooling, Node repository/site checks.

**Primary Dependencies**: Existing FFmpeg/Cueson companions, faster-whisper, pyannote, native process supervisor, artifact services.

**Storage**: Existing filesystem/S3 artifacts; SQLite/PostgreSQL typed catalog domains and portable snapshots; existing graph replacement/outbox.

**Testing**: Go unit/integration/native fixtures, Python worker tests, repository/schema tests, frontend/site checks, hosted native/backend CI. Inference remains separately elected outside required CI.

**Target Platform**: Windows and Linux x86-64.

**Project Type**: CLI-first shared runtime with desktop wrapper.

**Performance Goals**: Decoded memory/scratch bounded by window and concurrency; bounded verified source streaming per execution, zero inference reruns for verified compatible checkpoints.

**Constraints**: Exact rational source offsets; inward projection; protected credentials; hidden children; original work/attempt leases; one current Cueson document.

**Scale/Scope**: Five issues, three user stories. Default core 30 seconds, context 2 seconds each side, one active window. Configurable bounded core/context/track selections and resource pressure; hard limits documented in the contract.

## Constitution Check

Pass before research: scope is owner approved; shared runtime remains authoritative; canonical bytes and uncertainty preserved; all supported backends retained; no provider rerouting or manual queue.

Pass after design: reuse existing scheduler and artifact lifecycle; checkpoints are temporary verified computation evidence retired on final acceptance or invalidation, not alternative current transcripts. Required checks and timing policy remain intact. No merge/release authorization inferred.

## Project Structure

- `internal/processing/windows*.go`: plan types, exact sample counts, streaming extraction, lifecycle.
- `internal/process/`: bounded streaming child output using existing hidden/cancellation guarantees.
- `internal/processing/stitching*.go`, worker/adapter results: word evidence and deterministic stitching.
- `internal/catalog/processing_windows*.go`, migrations/snapshot/backup/lifecycle: fenced typed plan/checkpoints and proofs.
- `internal/app/processing_windows*.go`, processing/election/configuration: shared orchestration and final acceptance.
- `internal/pipeline/`, `cmd/insonic/`, `schemas/v1.0.3/`: request/options/result/help agreement.
- `scripts/processing-worker.py`: actual word timestamps and bounded resource recovery.
- `docs/v1.0.3/`, `CHANGELOG.md`: authoritative behavior and changes.

## Decisions and execution order

1. 2026-10-31: Freeze identity-bearing source/document/model/adapter/options before inference; exclude rebound host paths and actual fallback device from compatibility identity.
2. Count exact decoded native samples with bounded streaming, then extract from a continuous 16 kHz resample stream. Reject duration estimates and independent seeks because codec padding/resampler phase can alter boundaries.
3. Stream canonical input through the artifact range interface, retaining bounded chunks and chunk digests for verified repeated decoding passes. Avoid source-sized disk staging and never download the full source per window.
4. Use typed catalog plan/checkpoint domains and receipt proofs because Work.Result is overwritten and cleared on retry. Commit under current work and source/document fences; verify artifact bytes on reuse.
5. Keep checkpoints ready through explicit cancellation/retry; retire them after final acceptance/source replacement and queue existing cleanup. Include only ready checkpoint artifacts in applicable backups.
6. Preserve mapped-relative word observations, match same-track overlapping observations by text plus time, retain ambiguity explicitly. Namespace independent diarizer labels per window instead of inventing shared voice identity.
7. Default serial execution respects the configured cap. Pyannote may retry CUDA OOM once with batch size one before allowed CPU fallback. Never mutate beam, model, route, or window identity to recover.
8. Follow Spec Kit template parallel team strategy after shared design and analysis: extraction, catalog lifecycle, and worker/stitching have separate owners; root owns app integration, CLI/configuration, documentation, and delivery. Shared-file edits are coordinated.

## Verification and publication

Run focused regressions before broad checks, then `npm run check`, `npm test`, affected Go tests/vet, worker Python suites, frontend and site checks. Native fixture lanes exercise exact extraction and alternate backends. Separately elected actual-engine observations identify real devices and limitations. Keep preparation/qualification/combined accounting and retain compact audited CI receipts outside the run before reruns. Push and open official PR under current authorization; count automatic first review and at most one additional request; reply to and resolve every finding; stop at green CI for owner merge.

## Complexity Tracking

No constitution exceptions. A typed checkpoint domain is required for durable authority and backup; a second scheduler is rejected.
