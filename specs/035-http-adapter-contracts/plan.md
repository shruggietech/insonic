# Implementation Plan: S035 HTTP evidence extraction and provider diagnostics

**Branch**: codex/035-http-adapter-contracts | **Date**: 2026-10-17 | **Spec**: [spec.md](spec.md)

## Summary

Restore elected HTTP assertion extraction by aligning schema, normalization and canonical provenance. Add bounded provider failure reasons, distinguish caller cancellation from provider timeout, and identify materialization/native/provider output bounds without changing their machine category. Add installed CLI fixtures that cross schema, dispatch, durable work and citation publication boundaries.

## Technical Context

Go 1.26.2 CLI/shared runtime, JSON Schema contracts, Python package qualification and TypeScript desktop wrapper. Existing net/http clients and strict JSON parsers remain; no dependency or provider change is needed. SQLite/PostgreSQL authority, filesystem/S3 artifacts and LadybugDB/ArcadeDB projections retain parity. Target Windows/Linux x86-64. Tests: regression-first Go tests, root npm check/test, full Go suite/vet and Python discovery, affected frontend/site checks, exact-source installed packages and hosted backend fixtures. CI retains 660-second job bounds; no inference engine or owner-media execution is needed.

## Constitution Check

Pre-design and post-design: PASS. Exactly three owner-approved issues, CLI-first shared contracts, authoritative source clocks/current evidence, credential/body redaction, configured routes without new per-item approvals, complete supported-backend behavior, unchanged release version, internal planning/public contract separation and hidden noninteractive process launch are retained. Push/PR publication is explicitly authorized; no merge/release/auto-merge is authorized. No exception is required.

## Project Structure

- internal/evidence and internal/app/explore_evidence.go: extraction election and authoritative publication.
- internal/assistance: generic HTTP/Ollama provider boundaries and failure regressions.
- internal/contracts/operator.go, contracts.go and schemas/v1.0.1/runtime-response.schema.json: bounded reason vocabulary.
- schemas/v1.0.1/runtime-request.schema.json: accepted extraction identifiers.
- internal/artifact/service.go and internal/graph/types.go: operation-specific output bounds.
- scripts/cli_contract_qualification.py plus a focused provider fixture helper: installed CLI success/adverse cases.
- docs/v1.0.1/graph.md and CHANGELOG.md: authoritative behavior and unreleased history.
- specs/035-http-adapter-contracts: specification, research, contracts, tasks and verification.

## Design and Execution

1. Research existing extraction and provider error paths; record alternatives before coding.
2. Add meaningful failing tests for schema/runtime alias parity, both adapter failure matrices and three bound contexts.
3. Implement canonical normalization, locally generated reasons and deadline classification, preserving caller cancellation and strict output parsing.
4. Exercise app publication with current-chunk/foreign cues, source clocks and stale revisions; install deterministic loopback fixtures into existing package qualification.
5. Run required checks and convergence; commit, push and open a three-issue PR with milestone/Project tracking.
6. Resolve at most two code/security review rounds; verify final-head CI and every thread directly, then stop before merge.

## Complexity Tracking

No new framework, storage model, generic network policy layer or mandatory review flow. Use small local helpers only where both adapters share the same error decision. Preserve compatible existing Ollama reasons. Never convert provider protocol faults into user-input errors or expose arbitrary response text.
