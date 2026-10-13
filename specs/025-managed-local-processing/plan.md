# Implementation Plan: Managed local processing and task readiness

**Branch**: `codex/025-managed-local-processing` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

**Input**: S025 / #69.

## Summary

Ship directly usable optional worker setup and model acquisition, followed by task-specific readiness. Reuse the shared runtime, settings revisions, model acquisition and artifact services. Keep first-run orchestration for #66.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.12.15 managed profiles, React/TypeScript, Node 22+.
**Primary Dependencies**: Existing Wails 2.16.0 and six supported backend adapters; hash-locked optional faster-whisper/CTranslate2 and pyannote/torch CPU environments; checksum-pinned uv 0.13.0.
**Storage**: Existing filesystem/S3 models, SQLite/PostgreSQL work journals, LadybugDB/ArcadeDB graph; private workspace-managed worker/cache directories.
**Testing**: Go unit/integration/native fixtures, Python unittest, root/frontend/site npm checks; opt-in real engines outside CI.
**Target Platform**: Windows/Linux x86-64; managed Linux CPU dependencies require glibc >=2.28.
**Project Type**: CLI-first shared runtime and desktop.
**Performance Goals**: Bounded inspection (60 seconds total), setup timeout 30 minutes, existing durable acquisition limits, CI turnaround <10 minutes.
**Constraints**: Hidden noninteractive children; no model loading in inspection; no credential inheritance or raw subprocess failures; preserve accepted settings/elections.
**Scale/Scope**: Two managed CPU role profiles and two builtin immutable model bundles; custom/reused tools/models remain supported.

## Constitution Check

Before research and after design: all five principles pass. Scope follows #69; both clients use the same operations; models and accepted jobs retain exact provenance; all supported storage/catalog/graph adapters remain available; optional local choices never add per-item gates or hosted/device fallback. Versions stay at candidate 1.0.1; historical 1.0.0 stays unchanged. No release/tag/merge is authorized. Publication is explicitly authorized by the owner, with at most two external review rounds and final owner merge review.

## Project Structure

`internal/workers/`: catalog assets/hash locks, isolated preparation, bounded capability probe and environment identity receipts.
`internal/app/workers.go`: shared catalog/inspect/plan/setup/readiness, durable execution and guarded settings publication.
`internal/models/`: builtin manifests, verified local reuse, byte progress, frozen destinations and scoped HF transport.
`internal/processing/`, `internal/voicemodels/`: verify elected environment receipts before local engine invocation.
`cmd/insonic/`, `schemas/v1.0.1/`: command inventory/help/parser and versioned runtime/schema registration.
`desktop/frontend/src/workers.tsx`: managed setup/model/readiness controls through ordinary bridge requests; Settings/Pipelines integration.
`scripts/package-desktop.py`, `internal/packageenv/`: declare and verify shipped worker catalogs/locks, preserving old inventories.
`docs/v1.0.1/`, `CHANGELOG.md`: actual commands, device/prerequisite behavior and limitations.
`specs/025-managed-local-processing/`: spec, research, data model, contracts, tasks, analysis, convergence and verification.

## Execution Design

1. A read-only catalog reports exact engine/model/asset identities. Inspection probes only the elected interpreter/role and imports engine libraries without weights, in a private home and bounded clean environment. Hash calculation is automatic.
2. Plan freezes role/mode/path/lock identity/settings revision and displays required bytes/space, default CPU settings, offline options and missing prerequisites. Managed setup downloads verified runtime/uv assets or uses verified offline assets, creates a fresh owned isolated environment, and syncs only the full hash lock. Reuse never installs into the supplied environment.
3. Durable setup checkpoints phases. Each attempt has an owned directory; accepted environments are never updated in place. On success, compute dependency byte identities, retain an environment receipt, recheck work authority/cancellation/settings revision and publish only the elected role alongside existing companion/tool settings. Retry preserves the election and prior accepted configuration; cancelled/late attempts cannot publish.
4. Builtin models produce normal immutable manifests. Preview identifies exact bytes, license/access actions and elected storage profile. Verified local reuse copies through existing artifact publication. Acquisition remains shared durable work with bounded byte progress, partial-transfer reuse and frozen destination/profile revision.
5. Readiness composes actual media/worker/environment/model/device facts for import, existing subtitles, transcription, diarization and optional training. Report dataset/adapter prerequisites separately from environment readiness. Pipeline inspection and doctor use the same facts; generic configuration cannot imply execution or acoustic quality success.
6. Both clients expose catalog/plan/inspect/setup, default-model acquisition and readiness. Existing work controls own setup/acquisition progress/cancel/retry. Protected credentials remain opaque references. Advanced overrides remain available.

## Decisions and Risks

- Automatic setup uses isolated CPU profiles; explicit CUDA reuse requires observed runtime/device availability. Hardware alone is insufficient, and a failed frozen device never changes silently.
- Runtime/uv extraction rejects archive traversal, special files and escaping links. Every executable dependency is selected by immutable archive/lock identity.
- Environment receipts bind dependency file bytes and exact versions; execution revalidates the selected receipt before importing packages. Existing advanced configurations without receipts remain compatible.
- Readiness calls are bounded; installed native-library probes are evidence of prerequisite availability, not proof that every model or GPU will execute successfully.
- S3 capacity is unknown; report local scratch/materialization requirements and preserve storage-unavailable versus damaged-byte outcomes.
- No expensive inference in CI. Qualify actual elected CPU engines/models outside CI and report exact measured/unverified combinations.
- Reviewer-owned generated checklist markers remain review artifacts. Owner-authorized autopilot proceeds through requirements review and implementation without a new inter-step permission gate.

## Verification

Run root npm check/test, frontend check/test/build, Python tests, Go tests/vet and affected schemas/site/public/offline builds. Add meaningful security, cancellation/revision/identity, backend and CLI/GUI fixtures before implementation. Execute package/native closure checks through CI and verify final-head checks/review threads directly. Record opt-in real-engine and managed-setup evidence separately from deterministic checks.
