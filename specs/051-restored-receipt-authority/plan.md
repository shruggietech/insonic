# Implementation Plan: S051 Restored receipt recording authority

**Branch**: `codex/051-restored-receipt-authority` | **Date**: 2026-10-27 | **Spec**: [spec.md](spec.md)

**Tracking**: #180 under #147, milestone 35, Project 7, Slice S051. Owner authorizes push/official PR, CI repairs and at most two Codex rounds; stop before merging.

## Summary

Add strict source recording identity/revision validation in the restored-only validator, preserve existing destination equality and increasing-revision checks, and extend mutation regressions. Recover the retained actual CPU receipt and validate it unchanged under corrected code. Reconcile S050 evidence without rewriting the timing or sequence of its owner merge and incomplete final handoff.

## Technical Context

- Language/version: existing Python 3.12 maintainer validator/tests and Node 22+ repository/site checks.
- Dependencies/storage: Python standard library and existing test fixtures; owned temporary receipt recovered from a pinned GitHub comment, SHA-256 verified before use.
- Target: maintained internal acceptance on Windows/Linux; no public protocol/schema or product runtime change.
- Testing: test-first restored/source-only receipt mutations, focused and full Python tests, root check/test, affected site test/build/export and elected final-head CI.
- Performance/constraints: retain ordinary product clocks, 1800s qualification target, 3600s hard limit and provisional 60s reserve. Revalidation reports its own small elapsed time separately from the retained campaign's original preparation/qualification/combined times.
- Scope: scripts/journey_cpu.py; tests/journey_restored_test.py; concise S050/S051 records and CHANGELOG.md. No fresh inference required for this validator-only repair.

## Constitution Check

Before research and after design, all five principles pass: approved scope unchanged, CLI/shared runtime unmodified, canonical identity and truthful provenance strengthened, routes/credentials unchanged, meaningful bounded verification and exact historical evidence retained. No exceptions.

## Phase 0 Research and Decisions (2026-10-27)

1. Check the entire source inventory inside validate_restored, reusing its canonical UUID predicate before set membership. Enrollment records need the same identity requirements as destination-selected records.
2. Require exact integer type and positive source revisions; remove the fallback source revision. Preserve source <= prior < new and all existing complete-work/model/provenance checks.
3. Mutation tests cover shared omissions/invalid values, duplicate enrollment identities, malformed source/destination revisions and valid newer prior revisions. Existing v1 controls must continue passing.
4. Keep restored-only requirements isolated from source-only v1 validation. No evidence normalization, coercion or new receipt format is necessary.
5. Recover exact receipt SHA-256 e45118f36954a8486cb65289fdd3a68bfb9cb3f7906dc9127c93d667889c3e29 from the pinned retained comment, validate unchanged and retain a compact revalidation receipt with current validator input hashes. The original package/harness revision remains 9183e08; its tracked tree matched merged S050 ae8c608. Do not relabel this as a new runtime campaign.
6. Preserve S050 task history; append dated reconciliation of actual campaign/CI/repair and carry the incomplete final convergence/handoff into S051 explicitly. S051 closes the current implementation gap, not historical events.
7. The built-in quality checklist is sufficient. No new reviewer-owned checklist, public contract directory or approval gate is needed for an internal validator repair.

## Project Structure

- scripts/journey_cpu.py: source recording authority guard and explicit revision baseline.
- tests/journey_restored_test.py: mutation matrix and compatible positive controls, using tests/journey_cpu_test.py's existing fixtures.
- specs/050-restored-cpu-processing/tasks.md and verification.md: append dated evidence/reconciliation.
- specs/051-restored-receipt-authority/: current specification, design, tasks and concise verification.
- CHANGELOG.md: corrected acceptance behavior, with no new shipped-runtime claim.

## Design/Contracts

See [data-model.md](data-model.md) and [quickstart.md](quickstart.md). The receipt is internal, its v2 format is unchanged and v1 keeps its original meaning. No public interface contracts are introduced.

## Delivery

Write regressions and observe failures, implement the narrow guard, revalidate retained evidence and run local checks. Analyze blocks implementation until its coverage/consistency findings are resolved. Converge after implementation. Commit/push/open the official PR under existing owner authorization; monitor final-head CI and every review thread with at most two Codex rounds. Retain compact evidence before any full rerun, remove owned scratch and stop unmerged for owner review.
