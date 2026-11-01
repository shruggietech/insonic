# Implementation Plan: Transcript coverage and targeted repair

**Branch**: `codex/064-transcript-coverage-repair` | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

**Input**: S064, issues #217-#219, milestone 47.

## Summary

Add read-only evidence-based transcript coverage, bounded durable repair through the existing window executor, and a maintained end-to-end milestone qualification journey. Keep one current Cueson document and preserve canonical source identity.

## Technical Context

**Language/Version**: Go 1.27.1 runtime, Python qualification helpers, Node.js documentation checks.

**Primary Dependencies**: Existing Cueson, FFmpeg, processing adapters and catalog transactions; no new runtime dependency.

**Storage**: Existing filesystem/S3 publication and SQLite/PostgreSQL catalog; LadybugDB/ArcadeDB projections.

**Testing**: Go known-truth and fault tests, installed CLI deterministic adapter journey, Node/Python suites and Windows/Linux CI.

**Target Platform**: Windows and Linux x86-64 CLI and GUI shared runtime.

**Project Type**: CLI-first desktop/runtime application.

**Performance Goals**: Bounded report output, target-only inference, existing serialized bounded-window memory behavior; 1,800-second qualification target and 3,600-second hard limit after readiness.

**Constraints**: No inspection inference; no silent provider changes; atomic current acceptance; source/native-clock preservation; no local focus-stealing GUI; no real-model inference in required CI.

**Scale/Scope**: Three milestone child issues. Repair defaults to one execution attempt per selected window, maximum three, and at most 32 normalized target regions; no unrelated window inference.

**Research decisions**: [research.md](research.md) resolves persisted observations, fixed-basis metrics, filtered candidate construction, native cue merging, transactional mapping fences and installed fixture qualification.

## Constitution Check

Pre-research gate: PASS. Approved product behavior is preserved; the shared runtime exposes CLI contracts; canonical input and uncertainty remain explicit; all backend contracts and configured routes persist. The design introduces no approval queue, release action or new dependency. Qualification observes existing timing and retained-receipt rules. Post-design gate: PASS with no exceptions. Structural cue ordinals/root summaries may change because Cueson requires array-index ordinals; unaffected semantic fields remain invariant.

## Project Structure

- `internal/processing/`: pure coverage evidence, metrics, interval and repair policy contracts.
- `internal/app/`: read-only coverage projection, bounded repair election/execution and shared candidate construction.
- `internal/catalog/`: atomic mapping-aware acceptance and repair work/checkpoint recovery.
- `internal/pipeline/`, `internal/contracts/`, `cmd/insonic/`, `schemas/v1.0.3/`: saved policy and CLI contracts.
- `scripts/documented_workflows.py`, `docs/v1.0.3/`: executable complete journey and public contracts.
- `specs/064-transcript-coverage-repair/`: specification, research, design, tasks, verification and release recommendation.

## Complexity Tracking

No constitution exception. Reuse existing execution and transactional publication instead of a second repair engine or alternate current-document store.

## Delivery sequence

1. Specify, clarify and review requirement quality; research/design contracts and analyze task coverage.
2. Implement coverage evidence/calculator and snapshot CLI projection with known-truth tests.
3. Extract candidate builder, implement bounded repair/native merge and mapping-fenced acceptance; test fault, cancellation, checkpoint and idempotency paths.
4. Integrate saved policies, CLI/schema/human output and complete installed qualification journey.
5. Run convergence and local/CI parity checks, retain receipts, update compatibility/release packet and publish the authorized official PR.
6. Resolve every review within two rounds, verify exact-head green CI and stop for owner merge.

## Design artifacts

See [data-model.md](data-model.md), [runtime contract](contracts/recording-coverage-repair.md) and [quickstart.md](quickstart.md).
