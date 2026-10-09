# Implementation Plan: Speaker models and roster-constrained matching

**Branch**: `codex/013-speaker-models-roster-matching` | **Date**: 2026-10-09 | **Spec**: [spec.md](spec.md)

## Summary

Complete #15 and the remaining #35 functionality with reference-only corpus snapshots, elected local/hosted training adapters, immutable speaker outputs and roster-only matching. Reuse renewable durable work, configured artifact storage and portable catalog transactions. Separate manual mappings from automatic predictions and immutable output availability from current dataset validity.

## Technical Context

**Language/Version**: Existing Go toolchain, Python processing worker, TypeScript/React desktop and Node documentation tooling.

**Primary Dependencies**: Existing catalog, artifact, processing, model and scheduler packages; pinned offline pyannote embedding consumer plus declared external training adapters.

**Storage**: SQLite/PostgreSQL catalog; filesystem/S3 objects; LadybugDB/ArcadeDB projections.

**Testing**: Go contract/race/integration fixtures, Python adapter fixtures, JSON Schema and frontend request tests; npm check/test and affected site/desktop/product checks.

**Target Platform**: Windows, Linux and macOS CLI/runtime/desktop.

**Project Type**: Shared runtime with CLI and desktop clients.

**Performance Goals**: Paginate corpus at existing 100-reference ceiling without truncation; bounded adapter inputs/outputs and diagnostics; preserve configured four-worker concurrency and CI below ten minutes.

**Constraints**: No real acoustic initialization/download/inference in required CI; no implicit hosted routing; exact source clock, workspace tenancy, current-reference fences, hidden noninteractive child processes, no copied assignment histories.

**Scale/Scope**: Complete elected preparation/train/retrieve and current roster matching; not release packaging or coordinator taxonomy/bootstrap work.

## Constitution Check

Pre-research and post-design gates PASS: owner scope remains full; CLI/shared desktop parity; independently grounded current evidence; storage/catalog/graph parity; bounded verification distinguishes acoustic quality; configured route authorization persists. Existing poor logic (mapping origin absent, legacy job FK, completed output manifest conflated with corpus validity) receives proportional correction. No constitution exceptions.

## Project Structure

```text
specs/013-speaker-models-roster-matching/
  spec.md plan.md research.md data-model.md quickstart.md
  contracts/speaker-work.md tasks.md checklists/requirements.md
internal/catalog/                 # migration, mappings, frozen corpus/model/match acceptance
internal/voicemodels/              # preparation, adapter execution, training/retrieval/matching service
internal/app/                     # shared operation admission and worker routing
internal/processing/              # reusable mapped audio and pinned embedding execution
scripts/processing-worker.py      # real offline embedding consumer, CI guard
cmd/insonic/                      # dataset/train/speaker-model/match commands
desktop/frontend/src/             # speaker training, model view, roster matching controls
schemas/v0.0.0/                   # portable state and runtime payload contracts
docs/v0.0.0/                      # authoritative shipped contracts
```

**Structure Decision**: Introduce one speaker-model service package to coordinate existing components; keep portable authority transactions in catalog and use shared durable jobs rather than a second scheduler.

## Delivery Phases and Parallel Strategy

1. Specify/clarify, research, design/contracts and tasks; read-only analyze gate.
2. Catalog authority/types/migrations and adversarial tests establish the API boundary.
3. Parallel file ownership: catalog agent owns internal/catalog; adapter/service agent owns internal/voicemodels, processing embedding helper and Python worker; surface agent owns internal/app, cmd/insonic and desktop; root owns schema, documentation, integration and publication. Publish boundary types before dependent work and communicate signature changes.
4. Integrate with meaningful deterministic end-to-end/tenancy/stale-race coverage; run required checks and converge remaining work.
5. Publish official PR, satisfy every review within two rounds and green exact-head CI; owner final squash merge.

## Complexity Tracking

No constitutional violations. A generic adapter is necessary to preserve the specified model architecture/output contract; the built-in acoustic profile path makes matching usable without pretending every artifact is an embedding.
