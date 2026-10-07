# Implementation Plan: S004 Secure media library

**Branch**: `codex/004-secure-media-library` | **Date**: 2026-10-07 | **Spec**: [spec.md](spec.md)

## Summary

Finish issues #5 and #6 using the existing shared Go runtime, catalog and artifact contracts. Native/encrypted credential bootstrap must work before PostgreSQL opens. A downloaded-model registry is separate from speaker-trained model lineage. Stable staged originals feed metadata extraction and media admission. Durable fenced workflows reconcile retry/cancellation/recovery and current computed-metadata replacement with physical cleanup.

## Technical Context

**Language/Version**: Go 1.27.1, Node 22, Python 3.12 maintainer tools.
**Primary Dependencies**: Existing pinned wincred/DBus, x/crypto Argon2id, AES-GCM, SQLite/pgx, minio, jsonschema; configured exact ExifTool/ffprobe tools.
**Storage**: SQLite/PostgreSQL catalog; filesystem/generic S3 artifacts; platform secrets or explicitly selected encrypted vault/session.
**Testing**: Go acceptance/vet/race, backend fixtures, npm integrity/tests, Python qualification, site/offline export and native three-OS qualification.
**Target Platform**: Windows amd64, Linux amd64, macOS arm64 qualified matrix, other declared architectures remain explicitly distinguished.
**Project Type**: CLI/shared-runtime/desktop bridge.
**Performance Goals**: Streaming byte admission/downloads, bounded extractor output, no giant weights in ordinary CI; ten-minute CI jobs.
**Constraints**: Hidden noninteractive console children, source integrity, no secret persistence/disclosure, one current computed capture, native backend parity.
**Scale/Scope**: Two full issues, no inference/pipeline/desktop-package implementation.

## Constitution Check

Pre-research and post-design: PASS. Owner-approved full scope preserved; CLI/bridge share services; immutable originals and uncertainty preserved; all supported catalog/storage alternatives retained; no review gates or silent routing; public docs stay authoritative and internal evidence stays in specs. Push/PR authorization overrides generic pre-push halt, but merge/release remain owner actions. No unrelated dependency upgrades.

## Project Structure

- `internal/secrets/`: native noninteractive backends, vault, session, shared bootstrap.
- `internal/catalog/`: frozen historical migrations, measured timing, current library/model records, fenced workflow and cleanup state, snapshots.
- `internal/library/`: stable acquisition, extraction, date resolution, manifests, admission/refresh/relocation.
- `internal/models/`: manifest/download/verification service.
- `internal/process/`: distinct bounded stdout/stderr preserving outcome.
- `internal/app/`, `internal/runtime/`, `internal/contracts/`, `cmd/insonic/`, `desktop/`: shared routes, asynchronous work, credential bootstrap.
- `schemas/v0.0.0/`, `docs/v0.0.0/`, `scripts/qualify.py`, `.github/workflows/ci.yml`: contracts, evidence and qualification.

## Decisions

1. Freeze v1/v2 migration identities before changing reflective domain types. Existing catalogs must upgrade transactionally; no synthetic new legacy digest.
2. Catalog-independent shared credential bootstrap solves authenticated-catalog startup. Vault keys are process-memory only; native Linux/macOS integrations reject prompts.
3. Argon2id 64 MiB/3 iterations/4 lanes/32-byte key and AES-256-GCM with fixed authenticated format; no custom crypto or unconstrained KDF.
4. Downloaded base models have dedicated identity/manifests without fabricated speaker/training lineage.
5. Extract from stable staged originals, keep stderr separate, parse decimal times with exact integer arithmetic. Optional measured duration is authoritative.
6. Current library/model state and durable workflow/cleanup records participate in catalog snapshots, SQLite/PostgreSQL acceptance and fenced claims. Receipt payloads contain IDs/status, never archived metadata reports.
7. Refresh atomically replaces current references and queues retirement; success requires physical managed-output removal. Shared original assets and owner date evidence remain.
8. The generic acquisition boundary accepts explicit configured sources; failures do not reroute. Metadata partial/error states retain a usable entry.
9. Processing presets/capabilities stay #8; admission returns work status and honest pending processing instead of simulating inference. Supplied subtitle source bytes are admitted, not assembled.
10. Current-metadata/timing parts of #21 are applied locally without claiming full #21 closure.

## Verification Strategy

Tests precede implementation within each story. Security/isolation, historical migration, lossless date/timing, stable source mutation, manifest digest mismatch, durable retry/cancel/expired claims, metadata retirement, native credentials and both backend pairs are blocking. Full repository/site checks plus affected qualification run before push. Initial bot review plus one explicit second request maximum; resolve every finding and recheck final head. No attended owner verification gate.

## Publication

Conventional local commit with repository trailer convention; push official non-draft PR closing #5/#6. Track exact PR head, hosted CI and all discussion/reactions. Stop for owner review and squash merge after green gates and resolved discussions. Record review-round count in verification evidence.
