# Implementation Plan: S003 artifact storage

**Branch**: `codex/s003-artifact-storage` | **Date**: 2026-10-06 | **Spec**: [spec.md](spec.md)

## Summary

Complete issue #4 through streaming filesystem/S3 adapters and catalog-backed publication, materialization and retirement. Actual session S003 maps to specs/003-artifact-storage; original roadmap issue codes remain unchanged.

## Technical Context

Go 1.27.1, existing minio-go/v7 7.3.0 and flock pins. SQLite/PostgreSQL remain the shared transactional authority. CLI and Wails use one App/runtime contract. Existing pinned S3Mock provides integration qualification, deterministic transports prove signing/range/error behavior independently. Filesystem access uses Go os.Root confinement and private stages. Tests are test-first Go shared suites, integration fixtures, race checks, npm check/test and affected site/native checks. Target Windows/macOS/Linux. Streaming buffers/parts bound memory; catalog transactions never cover external I/O. Existing ten-minute CI timeouts remain.

## Constitution Check

Pre-design and post-design: PASS. Both storage choices receive full common behavior; explicit provider selection and opaque credentials persist. No approval queue, source mutation, graph expansion or model registry is added. Public docs state contracts; internal slice evidence stays here. Tests include workspace scope, path confinement, cancellation and stale claims. Push/official PR already explicitly authorized; final merge stays owner-controlled.

## Project Structure

- internal/artifact/: streaming Store contract, filesystem/S3 adapters and shared service.
- internal/catalog/artifacts.go: typed durable lifecycle journal, publication generations, reference/cache/retirement admission under workspace transaction.
- internal/catalog/migration.go, snapshot.go: versioned additive upgrade and lifecycle snapshot validation.
- internal/app/, internal/contracts/, internal/runtime/, cmd/insonic/: shared artifact operations.
- schemas/v0.0.0/: request and snapshot contracts.
- docs/v0.0.0/storage.md, development.md, index.md and CHANGELOG.md: actual behavior.
- internal/artifact/*_test.go and internal/catalog/*_test.go: positive/failure/security/integration acceptance.

## Design decisions

2026-10-06: Use dedicated renewable publication claims instead of existing timer qualification jobs, which otherwise auto-complete uploads prematurely. Persist operation intent, profile, exact unique key and multipart journal before remote work; short catalog updates fence every generation.

2026-10-06: Use unique never-reused publication keys on both adapters, SHA-256 content identity and independent readback verification. Do not claim conditional-create support without qualification. Core multipart APIs expose upload IDs for recovery/abort.

2026-10-06: Use one typed nonsecret lifecycle journal per publication, stored under workspace scope with current authority and append-only transition evidence. Available domain Artifact/Location evidence is admitted only after verification; lifecycle state overlays immutable location history. Retention references and materialization leases are in that journal and updated atomically with lifecycle eligibility. Structural domain references also block retirement and generic Commit admission checks the barrier.

2026-10-06: Preserve schema-v1 DDL digest; add schema-v2 migration transactionally. Validate old snapshot digest before compatible upgrade, validate lifecycle receipt/profile/content relationships, expire imported owners/leases and preserve pending retirement barriers.

2026-10-06: Materialization is streaming into private bounded cache; lease acquired before I/O, verified path returned only while authority remains current. Default publication/lease TTL and grace are bounded, explicit values documented; external references and full byte backup remain library/portability work.

## Phases

Specify/clarify/checklist passed, plan and research define authority. Generate tasks and analyze all requirement coverage before implementation. Tests precede each user story implementation; converge and recorded CI parity precede authorized publication. At most one manual second review request is allowed.

## Complexity Tracking

No constitution violations. A typed journal in one additive table reuses both catalogs' transaction machinery rather than introducing another database or scheduler.
