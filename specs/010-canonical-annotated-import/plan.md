# Implementation Plan: S010 Canonical annotated import

**Branch**: `codex/010-canonical-annotated-import` | **Date**: 2026-10-09 | **Spec**: [spec](spec.md)

## Summary

Complete #31/#32 using shared new-media and standalone transcript admission. Include transcript targeting/replacement from #30 and canonical new admission from #33, while leaving their full audio-replacement/legacy-audio migration scopes open. Preserve external attribution and native observations, grouped canonical audio, bounded local/URL/embedded acquisition and one atomic accepted current result.

## Technical Context

Existing Go 1.27.1, portable SQLite/PostgreSQL catalog, filesystem/S3 artifact services, LadybugDB/ArcadeDB evidence, pinned FFmpeg/ffprobe/ExifTool/Cueson 1.2.0 and Wails/React/TypeScript. Reuse current durable work, renewable publication leases and current-reference cleanup. No new runtime database subsystem. Additive facts contain canonical source/track provenance; typed catalog reflection and frozen migrations must not drift.

Bounds: existing media 64 GiB/10 minutes defaults; transcript source/document maximum 16 MiB and normalized expansion checked; existing import ingress 8 MiB/10,000 items and paged 1 MiB responses retained. Decoder/extractor timeouts honor caller cancellation. No inference or model acquisition in required CI.

## Constitution Check

1. Owner policy: all D1-D4 answered explicitly. No eligibility or review queues. Pass.
2. Shared CLI/runtime/desktop: same admission requests/results, no GUI-only logic. Pass.
3. Provenance: retain raw facts and current validated document, preserve owner-owned inputs; amend superseded original-retention text separately before analysis. Pass after amendment.
4. Adapters/credentials: portable transactions and artifacts, independent credential references, no route fallback. Pass.
5. Verification: fixtures and exact-head CI/reviews, ten-minute jobs, no model checks. Pass.

## Decisions

- D01 Grouped audio uses one .mka asset with independent FLAC tracks. Stereo/more channels use FLAC; owner approved.
- D02 Lossy mono uses suitable MP3 passthrough or V0 MP3 encoding. Normalization means format standardization, no hidden DSP; owner approved.
- D03 Retain canonical identity in existing library byte fields and separate source digest/tool/track facts in additive metadata. Existing original publication field is a compatibility field, not original-byte retention.
- D04 Direct annotated documents use a preservation path; native derivation adds participation only if there are zero assignments document-wide.
- D05 Package exact historical schemas and validate before deterministic lossless translation; validate current output before acceptance.
- D06 Add one catalog admission transaction reusing current recording reconciliation with an optional newly admitted library row or existing target. Validate expected source/document revisions and live work authority before acceptance.
- D07 Acquisition and discovery stage before artifact acceptance; explicit transcript errors cannot fall back. Metadata reports scrub source locators before durable publication.
- D08 Preserve current receipt/retry semantics, remove successful-admission source locators from durable work, and use private owned staging for retryable input identity.
- D09 Browser-incompatible grouped playback uses bounded temporary preparation for an explicitly selected canonical track; canonical tracks remain durable authority.
- D10 MP3 encoder availability is a required native packaging capability. Extend pinned source builds/notices if existing macOS FFmpeg lacks the approved encoder.
- D11 Existing-audio conversion is not silently triggered. Legacy transcript conversion/cleanup required by current-document authority is included; remaining bulk audio migration remains #33.
- D12 Review round 1 is automatic on official PR; request at most one second round with @Codex. All findings require replies, fixes where necessary and direct thread-resolution verification. Stop before merge.

## Project Structure

`internal/subtitles/` exact-version validation and native preservation; `internal/contracts/` bounded local IDs and operations; `internal/catalog/` admission transaction/reference reconciliation; `internal/library/` canonical conversion, acquisition/discovery and transcript admission; `internal/app/` dispatch/elected tools; `cmd/insonic/` CLI; `desktop/frontend/` consistent controls; `schemas/`, authoritative `docs/`, constitution and changelog. Native source packaging/qualification is amended only for required format support.

## Execution

Specify -> clarify -> domain checklist -> research/design -> tasks -> blocking analyze -> TDD implementation -> affected verification/converge -> local commit -> push/official PR -> initial reviews -> optional second review -> final exact-head checks -> owner merge handoff. Research agents read separate boundaries; implementation follows task dependencies.