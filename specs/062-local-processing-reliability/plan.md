# Implementation Plan: NVIDIA-first local processing and CLI reliability

**Branch**: `codex/062-local-processing-reliability` | **Date**: 2026-10-31 | **Spec**: [spec.md](spec.md)

**Input**: S062, issues #202-#208.

## Summary

Extend the existing Python worker and Go runtime contracts with one finite NVIDIA-preferred policy, retaining old frozen empty-device CPU behavior. Reuse existing no-speech authority and speaker enrollment/matching pipelines. Improve generated diagnostics, lifecycle reasons and task-specific human output. Deliver all seven issues together with schemas, documentation and source/package evidence.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.12 processing environments, Node maintainer tooling and TypeScript frontend.
**Primary Dependencies**: Existing faster-whisper 1.2.1/CTranslate2 4.8.2, pyannote.audio 4.0.7 and pinned environments; no new inference backend.
**Storage**: Existing SQLite/PostgreSQL, filesystem/S3 and LadybugDB/ArcadeDB contracts.
**Testing**: Go tests/vet, Python worker/probe regression fixtures, Node schema/repository tests, frontend/site checks, Windows/Linux source/package CLI qualification; real-engine execution separately elected outside deterministic CI.
**Target Platform**: Windows/Linux x86-64.
**Project Type**: CLI-first shared runtime with desktop callers.
**Performance Goals**: At most CUDA then CPU under one existing operation deadline. Qualification target 1,800 seconds, hard 3,600 seconds after readiness; preserve finalization reserve and separate timing.
**Constraints**: Hidden noninteractive children; no original source modification, provider/model substitution, per-item approvals, merge/tag/release. At most two external Codex rounds including initial automatic review.
**Scale/Scope**: One seven-issue milestone; no windowing, answer generation or contextual identity workflow.

## Constitution Check

- Approved scope: pass. Current NVIDIA-first instruction supersedes older CPU-default requirements, while explicit historical CPU work stays CPU.
- Shared CLI/runtime: pass. Device policy lives in worker/runtime contracts and is carried by desktop callers.
- Integrity/provenance: pass. Fallback changes only execution device; existing acceptance fences and current document/profile authority remain.
- Credentials/routing/backend parity: pass. Generated diagnostics only; local fallback never elects hosted routing.
- Qualification/publication: pass. Deterministic versus real evidence separate; timing retained; push/PR authorized, merge reserved.
- Documentation/version: pass. Update current authoritative contracts and master unreleased changelog, keep synchronized 1.0.3 version until separately authorized release.

Post-design review: no constitution exceptions or unresolved product questions.

## Project Structure

- `scripts/processing-worker.py`, `internal/workers/probe.py`: native library resolution, finite device execution, generated failures, language checking and exact-silence recognition.
- `internal/processing/`, `internal/app/`, `internal/pipeline/`, `internal/voicemodels/`: admission normalization, frozen legacy interpretation, readiness, result validation and durable speaker device provenance.
- `cmd/insonic/human*.go`, `internal/contracts/`, `internal/catalog/artifacts.go`: concise work rendering, valid actions and generated lifecycle/error reasons.
- `schemas/v1.0.3/`, `desktop/frontend/`: strict request/result shapes and shared caller defaults.
- `docs/v1.0.3/`, `scripts/documented_workflows.py`, `scripts/cli_operator_qualification.py`: executable examples and source/package operator qualification.
- `specs/062-local-processing-reliability/`: spec, plan, research, data model, contracts, quickstart, tasks and receipts.

## Execution Phases

1. Freeze compatibility and generated error/provenance contracts; reproduce regressions.
2. Implement worker/readiness policy and runtime admission/result propagation in coordinated parallel file groups.
3. Implement independent operator/lifecycle fixes, preserving generic public output and warnings.
4. Integrate strict schemas, frontend defaults and executable authoritative documentation.
5. Run targeted then full CI-parity qualification; publish one PR, resolve every review and applicable CI failure, stop for owner merge.

## Decisions

- Empty device at execution remains legacy CPU. Only new local admissions normalize omission to auto; hosted adapters retain their own contracts.
- The shared Go worker launcher owns a maximum of two fresh concrete device child attempts. Only classified CUDA device/native-library/resource failures trigger CPU recovery; cancellation, input/model corruption and unknown failures do not.
- Speaker embedding results gain optional execution provenance without changing profile vector compatibility. Source/model/sample identity remains immutable.
- Exact zero-signal uses the existing no-speech/null-document path. Diarizer no-speech disagreement is diagnostic, not destructive.
- Readiness must resolve verified elected library roots without ambient unpinned module loading. Managed CPU dependency inventory remains accurately named; execution preference is separate.
- Research agents were dispatched as required by speckit-plan. Independent [P] implementation tasks use coordinated file ownership under speckit-implement; shared schema/docs integration stays with the lead.
