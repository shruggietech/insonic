# Implementation Plan: Local import storage-failure recovery

**Branch**: `codex/058-import-storage-failures` | **Date**: 2026-10-28 | **Spec**: [spec.md](spec.md)

## Summary

Qualify local import failures before admission and recovery after write access returns. Add injected partial disk-full, short-write, flush/rewind/cleanup and report/uncertain-publication checks. Actual source/package acceptance uses a hash-pinned final-probe fixture to obstruct only a new canonical digest directory with an owned regular file, observes the real filesystem refusal, removes that exact blocker and submits a new ordinary import. No capacity is consumed beyond tiny fixtures and no product fault flag is introduced.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.11+, Node 22+.
**Primary Dependencies**: Existing artifact/work/catalog interfaces, pinned media companions, ProcessTree/owned_runtime, committed qualification timing.
**Storage**: Filesystem/SQLite fixtures. Supported remote backends remain unchanged.
**Testing**: Native artifact/library regressions, Python fixture/validator mutations, root checks/tests, affected docs and exact-head Windows/Linux source/package CI.
**Target Platform**: Windows/Linux x86-64.
**Project Type**: Shared CLI-first runtime, with existing desktop CLI packaging.
**Performance Goals**: Tiny PCM inputs; return on observation rather than fixed sleeps; existing 1800 s qualification target / 3600 s hard limit.
**Constraints**: Preserve source hashes/provenance, accepted sentinel, attempt fences, ordinary product deadlines, hidden children and owned cleanup.
**Scale/Scope**: One actual filesystem refusal/recovery case per source/package context; native injected boundary matrix. No physical-full-volume or power-loss claim.

## Constitution Check

Pre-design and post-design: PASS. Shared runtime changes, immutable source/accepted authority, existing routes, exact receipts and current/historical package compatibility are retained. No policy gate, new service, database migration, dependency update or owner-data access is introduced. No constitution exception.

## Phase 0 research

The plan command directed one read-only research agent. Existing writer and Filesystem.persist seams suffice for injected failures. StageBound ignores rewind/cleanup errors; artifact redaction currently obscures native capacity failures despite the existing insufficient_space contract. Confirm with failing tests before repair.

An actual canonical output digest is available to a configured probe, so a regular-file obstruction at that digest parent deterministically reaches real Filesystem.MkdirAll refusal. A report digest is not known at that gate; native injected report publication covers that boundary without expensive ACL/mount fixtures.

## Phase 1 design

- Keep copyStableBound's writer seam; extract private stage finalization to test flush, rewind and error cleanup with real owned files.
- Classify native ENOSPC/quota/Windows disk-full errors through the existing insufficient_space code; preserve cancellation and typed integrity errors without leaking OS paths.
- Native library Store wrappers cover failed canonical/report transfer and uncertain completed publications, with accepted sentinel/source equality and later correct admission.
- Add scripts/cli_storage_qualification.py with strict receipt validation, source/companion-tool selection, owned runtime/lifetime helpers and marker/hash-bound blocker removal. Extend existing CLI contract/corpus and modern release validation.
- Original immutable v1.0.2 receipts keep only their existing exact-hash exemption. Current/unknown receipts require the new nested storage evidence; original v1.0.1 historical profile remains unchanged.
- Actual application finalization marks an import with failed items as failed/completed-with-failures while retaining Partial=true and each item outcome. The campaign records that public failure, then a distinct ordinary import request after clearing the obstruction. It does not claim same-work retry execution.
- Failed staging cleanup retains its internal joined failure and a redacted public error; receipts distinguish an injected cleanup failure from successful fixture cleanup.

## Project Structure

```text
internal/artifact/{stable.go,stage.go,service.go,filesystem.go,contract.go,capacity_*.go,*storage*_test.go}
internal/library/*storage*_test.go
scripts/{cli_storage_qualification.py,cli_contract_qualification.py,release.py}
tests/{cli_storage_qualification_test.py,fixtures/cli-corpus.json,...affected consumers}
specs/058-import-storage-failures/{spec,plan,research,data-model,quickstart,tasks,verification}.md
specs/058-import-storage-failures/contracts/storage-receipt.md
```

**Structure Decision**: Extend existing staging/runtime qualification rather than adding a storage abstraction or standalone journal.

## Verification and delivery

Run failing-first focused regressions, then all affected native/Python checks, npm run check, npm test and affected site checks. Verify local actual Windows campaign, then final exact-head CI and compact independent artifact/receipt audit. Count at most two Codex rounds and inspect each review thread. Push/PR publication is authorized. Stop before owner merge; keep umbrella #147 open.

## Decisions

2026-10-28: Built-in requirements checklist covers the domain; no extra reviewer-owned checklist is needed. Extension hooks are absent. Physical capacity exhaustion and backup/remote quota behavior remain explicit separate claims.
