# Implementation Plan: S006 Configurable pipelines, speakers and terminology

**Branch**: `codex/006-configurable-speaker-pipelines` | **Date**: 2026-10-07 | **Spec**: [spec.md](spec.md)

## Summary

Deliver #8/#9 through the existing shared runtime. Store current revisioned pipelines, speaker aliases and terms in the catalog. Resolve immutable effective pipeline/context elections before durable work. Extend disposable local processing with hints/quality configuration and a documented bounded hosted worker protocol. Speaker selection resolves current embedded evidence and external mappings without another assignment store.

## Technical Context

**Language/Version**: Go 1.27.1; Python 3.11+ local worker; Node22 docs/tools.
**Primary Dependencies**: Existing database drivers, Cueson1.2.0, pinned media tools and inference packages; Go standard HTTP client. No new inference/library dependency needed.
**Storage**: SQLite/PostgreSQL current entities, schema5 over frozen schema4; filesystem/S3 media/model/artifact adapters unchanged.
**Testing**: Go/race/vet, Python deterministic worker tests, npm checks/tests, schema/site/offline qualification, three-OS native and hosted backend fixtures.
**Target Platform**: Windows/Linux/macOS CLI/shared runtime/desktop bridge.
**Project Type**: CLI-first runtime with desktop wrapper and documentation site.
**Performance Goals**: Paged entity/selection operations (<=100 entries); bounded HTTP/audio/output/time/context; hosted CI jobs below10min.
**Constraints**: No inference/model load/weight download in CI; no ambient credential inheritance, credential URLs/redirects, silent routing/device fallback, alternate result store or manual review queue.
**Scale/Scope**: Two issues/four stories; dedicated GUI panels/packages, graph assertions, embedding similarity and training engines remain separate.

## Constitution Check

Pre-research PASS: CLI-first/shared bridge, owner-authorized scope, current-result rules, encrypted credential references, full supported storage/catalog alternatives, hidden child execution and inference-free CI.
Post-design PASS: election is immutable per job but current source/document acceptance remains fenced; quality is diagnostic not an approval gate; selection retains references only. No constitution exception required.

## Phase 0 research

Catalog, processing/provider and CLI/schema research owners inspect existing integration seams. Consolidated decisions in [research.md](research.md). No existing backend profiles are overloaded with processing roles.

## Phase 1 design

[Entities](data-model.md), [runtime/protocol contracts](contracts/runtime.md), [validation](quickstart.md).

- Current catalog Pipeline stores ID/name/preset/revision/nonsecret typed configuration. CAS receipts and latest snapshot proof cover each update. Configuration history may be represented by elections; no alternate result data is stored.
- Known speakers gain revision/state, aliases carry explicit identity/language/context/provenance. Terms carry spelling/variants/language/context/active/revision/optional identity links. Admission validates links and nonsecret bounded payloads.
- Freeze schema4 migration and snapshot identity before schema5. Catalog owner serializes shared schema/domain edits and tests SQLite plus hosted PostgreSQL parity.
- `internal/pipeline` owns typed definitions, full-stage overrides, static declared capabilities and bounded HTTP execution. Local uses faster-whisper/pyannote; Connected uses operator-selected `insonic-http` contract1; Custom allows explicit mixed routes. Never discover a new endpoint or route after failure.
- HTTP worker receives multipart request metadata and mapped WAV, returns bounded strict segment/turn results with integer mapped-relative microseconds. Credential references resolve only at execution into Bearer authorization; redirects and credential-bearing URLs are rejected. Provider body/provenance is not logged or persisted. Output becomes current Cue JSON only through existing validation/fenced acceptance.
- Resolve definition ID/revision/digest and effective settings at submission. Full-stage override avoids carrying endpoint/auth accidentally between modes. Compile hints at election within UTF-8 byte budget, record digest/diagnostics and pass supported hints into local hotwords or hosted request.
- Bind the complete resolved nonsecret processing-tools selection (including Cueson, Python, worker, FFmpeg identities/limits) at submission, rather than rereading mutable control files at execution. Context source names/aliases/terms share one catalog read snapshot. Use faster-whisper hotwords across decoding windows with a conservative200-byte joined budget, below the pinned tokenizer's byte-BPE prompt ceiling, and diagnose omissions before calling the worker.
- `internal/speakers` joins current mapping/document evidence for bounded speaker selection and quality/context helpers. Do not export whole catalog for a list or copy intervals into segment storage. Selection resolves exact source map; untimed/overlapping outcomes stay explicit.
- App parent integrates election into `recordingPayload`, hooks local/hosted stages and shared domain operations. Existing recordingFactory remains the deterministic test seam.
- CLI commands use bounded JSON `--input` consistent with existing operations. Desktop Bridge.Operate uses the same request schema/runtime. Document GUI controls pending #10.

## Project Structure

`internal/catalog/`: current entities, CAS, migration/snapshot proof and current selection pages.
`internal/speakers/`: deterministic context and current speaker evidence helpers.
`internal/pipeline/`, `internal/processing/`, `scripts/processing-worker.py`: elected local/hosted stage configuration, transport, hints and quality.
`internal/app/`: shared configuration operations and processing integration.
`cmd/insonic/`, `internal/contracts/`, `internal/runtime/`, `schemas/`: CLI/request parity and operation validation.
`docs/`, `tests/`, `scripts/qualify.py`, `specs/006-configurable-speaker-pipelines/`: authoritative contracts and bounded evidence.

## Dependencies and parallel strategy

Spec/clarify/checklist/plan/tasks/analyze precede edits. After analysis, separate owners implement catalog/speaker helpers, pipeline/processing and CLI/schema/docs in parallel. Parent serializes app integration and shared interfaces. Tests precede implementations; owners coordinate schema files explicitly. Every story completes before publication, then initial reviews and at most one second review request; resolve findings before final CI wait and owner merge handoff.

## Complexity Tracking

No unjustified violation. A versioned generic worker protocol provides both hosted stages without claiming compatibility with unrelated commercial APIs. The selected service must implement the documented protocol. Explicit endpoint choice remains operator configuration, with no real external library-media submission during development.
