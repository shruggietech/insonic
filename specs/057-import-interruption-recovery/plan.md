# Implementation Plan: Import interruption recovery

**Branch**: `codex/057-import-interruption-recovery` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)
**Input**: S057 / #147

## Summary

Qualify the current canonical-media import across actual runtime loss. Preserve its already published canonical/report candidates and captured metadata when the source still matches; refuse changed bytes and corrupt recovery authority. Observe the pre-acceptance boundary externally with a controlled final-probe wrapper and platform file-open barrier, without a product fault switch.

## Technical Context

**Language/Version**: Go 1.27.1 and Python 3.12+; existing Node check/test tooling.
**Primary Dependencies**: Existing library/catalog/artifact/runtime contracts, pinned media tools, ProcessTree, Windows RW oplock and Linux file lease.
**Storage**: Filesystem/SQLite defaults for this campaign, existing LadybugDB projection and alternative-backend checks unchanged.
**Testing**: Native library Go regressions, Python barrier/campaign/receipt tests, actual source/extracted CLI campaigns, existing full CI.
**Target Platform**: Windows and Linux amd64, source and CLI/desktop-package CLI.
**Project Type**: Shared CLI/runtime application plus maintained qualification.
**Performance Goals**: Existing 1,800 s qualification target / 3,600 s hard limit after verified prerequisites.
**Constraints**: No changed product leases/timeouts, no inference, no owner data, no product test backdoor, hidden Windows children. A platform kernel auto-break bound may shorten this one barrier's observation/termination window; no sysctl change or synthetic success is permitted.
**Scale/Scope**: Three scenarios (same source, equal-length changed source, corrupted report) in six contexts. Broader interruption points and power-loss durability remain outside this claim.

## Constitution Check

Pre-research: PASS for owner scope, CLI/shared behavior, immutable source/provenance, default/alternative adapter status, hidden children, truthful qualification and publication limits.
Post-design: PASS. Current production admission (not legacy CommitLibrary) is targeted; metadata/report bytes are reused, source and artifact integrity rechecked, and ordinary fences remain active. Historical published v1.0.2 receipts receive only an exact immutable receipt exemption from the newly introduced recovery field; existing validation stays required. Unknown or altered receipts receive no exemption.

## Project Structure

- `internal/library/admission.go`, new `admission_recovery.go`: current candidate recovery.
- `internal/library/admission_recovery_test.go`: failing-first native regressions.
- `scripts/file_open_barrier.py`: owned external Windows/Linux source-open barrier.
- `scripts/cli_recovery_qualification.py`: campaign and strict receipt validator.
- `scripts/cli_contract_qualification.py`, `scripts/qualify.py`: shared source/package integration.
- `scripts/release.py`, `scripts/release_history.py`: required modern recovery receipts and exact original-v1.0.2 compatibility.
- `tests/*recovery*test.py`, `tests/file_open_barrier_test.py`, existing release/qualification tests and `tests/fixtures/cli-corpus.json`.
- `docs/v1.0.2/storage.md`, `docs/v1.0.2/development.md`, `CHANGELOG.md`: authoritative behavior, verification limits and dated tooling decision.
- This directory: spec, plan, research, data model, receipt contract, quickstart, tasks and concise final verification.

**Structure Decision**: Extend existing services and shared CLI qualification; no database migration, new public command, framework or new backend.

## Execution and Decisions

1. Capture the existing modern recovery failure with native service tests.
2. Reuse a complete verified canonical/report pair after abrupt loss, comparing staged source identity before reuse. Partial pre-report candidates retain existing recomputation behavior; observed ordinary failures retain retirement fences.
3. Pause the final configured canonical ffprobe only on fixture-owned paths. Arm a source-file RW oplock/lease, release the probe, require a break and independently read available publications with no accepted recording. Kill/join the exact owned runtime before releasing the barrier.
4. Restart normally under existing attempt expiry/fencing; independently verify admission, provenance, publications and history. Negative scenarios must refuse stale/corrupt acceptance.
5. Validate complete receipts and require them throughout modern package/release acceptance. Preserve immutable original v1.0.2 receipt bytes with exact hashes, never a version-range exemption.
6. Run required checks, analyze/converge, push/open the official PR, monitor exact-head CI and at most two Codex rounds; stop before merge.

## Complexity Tracking

No constitution exceptions. The external kernel barrier is necessary to qualify the actual product image without a product fault hook; ordinary advisory/share-denial locks cannot establish this boundary. Reconstructed service tests remain focused regressions and do not substitute for process evidence.
