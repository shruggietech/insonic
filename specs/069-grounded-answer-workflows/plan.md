# Implementation Plan: Grounded answer workflows

**Branch**: `codex/069-grounded-answer-workflows` | **Date**: 2026-11-02 | **Spec**: [spec.md](spec.md)

**Input**: S069, #224-#226, parent #210; no new tracking issue.

## Summary

Extend existing answer evidence packages with bounded synthesis over configured assistance transports, lease-fenced durable work and immutable result artifacts, configured answer pipelines, and a versioned synthetic evaluation corpus. Retain direct preparation/inspection/validation compatibility and prepare a release recommendation without publication.

## Technical Context

**Language/Version**: Go 1.27.1 shared runtime; Python and Node maintained qualification; versioned JSON Schema and Next.js documentation.
**Primary Dependencies**: Existing assistance adapters, answer evidence/context contracts, catalog work/lease journals, artifact service and pipeline configuration.
**Storage**: Filesystem/S3, SQLite/PostgreSQL, LadybugDB/ArcadeDB.
**Testing**: Go unit/integration/native tests, npm check/test, Python CLI qualification, documentation build/tests and package CI.
**Target Platform**: Windows/Linux x86-64 CLI and installed packages; isolated CI desktop only.
**Project Type**: CLI-first shared runtime with desktop wrapper.
**Performance Goals**: Existing bounded retrieval (20s default, 60s max), provider invocation 30s default/60s max, finite synthesis repair and result limits. Synthesis shares one provider deadline across at most two calls; total durable work cap180seconds.
**Constraints**: No source mutation, silent reroute, credential/private-path export, new tracking issue, attended gate, merge or release publication. Exact terminal CI receipts and two external review rounds maximum.
**Scale/Scope**: Three existing child issues and complete parent criteria where demonstrated.
**Research resolved**: See research.md for shared transport, strict visible evidence, immutable linked retries, atomic acceptance, pipeline readiness, corpus metrics and installed alternative-backend campaign.

## Constitution Check

Before research: PASS. Approved outcome is preserved, CLI-first operations share runtime contracts, immutable evidence retains uncertainty and backend parity, encrypted credential routing is reused, and qualification/review boundaries follow owner instructions. No unjustified exceptions.

After design: PASS. Contracts preserve source/routing authority, distinguish protocol and semantic evidence, maintain all supported backends and keep publication/merge within owner boundaries. No unresolved critical conflict.

## Project Structure

Planning lives in `specs/069-grounded-answer-workflows/` (spec, plan, research, data model, contracts, quickstart, tasks, analysis and readiness).

Implementation extends `internal/answers/`, `internal/assistance/`, `internal/app/`, `internal/catalog/`, `internal/pipeline/`, `internal/contracts/` and `cmd/insonic/`. Public contracts live in `schemas/v1.1.0/`, documentation in `docs/v1.1.0/`, qualification in `scripts/` and `tests/fixtures/`.

**Structure Decision**: Extend established shared services and journals; no new application/service or provider-routing subsystem.

## Complexity Tracking

No constitutional exceptions. Publication acceptance requires a small transaction fence; retries must retain prior attempts because generic work retry replaces the current row result.

## Chronological Decisions

1. 2026-11-02: Use exact S069 identity, no new issue, and the existing clean checkout.
2. 2026-11-02: Resolve the checklist template directly because the installed prerequisite helper requires plan.md before the autopilot checklist phase. Review all generated criteria before implementation.
3. 2026-11-02: Dispatch provider, durable-work and evaluation research as required by speckit-plan. Resolve their findings before tasks/analyze.
