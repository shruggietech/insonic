# Implementation Plan: S050 Restored-library CPU processing acceptance

**Branch**: `codex/050-restored-cpu-processing` | **Date**: 2026-10-26 | **Spec**: [spec.md](spec.md)

**Tracking**: bounded #178 under #147, milestone 34, Project 7. User authorizes push/official PR, CI fixes and at most two Codex rounds; stop before merge.

## Summary

Extend the maintained complete-package CPU journey after its existing accepted-data restore checks. Prepare destination workers, verify restored model bytes, execute fresh held-out/sequence inference and matching, and audit retained canonical authority and completed work. Preserve separate source/destination evidence. Repair product logic only if actual tests/campaign demonstrate a gap.

## Technical Context

- Language/version: Python 3.12 campaign/tests, Go 1.27.1 existing runtime, Node 22+ repository checks.
- Dependencies: existing packaged Cueson/FFmpeg/native LadybugDB, managed CPU profiles, pinned read-only model inputs and normal managed worker preparation and frozen acoustic corpus.
- Storage: isolated filesystem/SQLite/LadybugDB and verified self-contained backup. Owner state remains untouched.
- Testing: Python unittest receipt mutations and lifecycle fakes, existing Go portable/setup/cancellation regressions, root npm integrity/tests and affected site checks; complete exact-source Windows CLI campaign; final-head CI.
- Performance: unchanged 1800 s qualification target / 3600 s hard limit / 60 s provisional reserve. Destination preparation occurs after initial readiness and remains charged to qualification; no clock reset. The journey's existing combined 3600 s bound stays unchanged.
- Scope: internal acceptance harness and evidence. No new public protocol or alternate transcript store is planned.

## Constitution Check

Before research and after design: all five principles pass. Shared CLI/runtime contracts govern setup/processing; current-transcript supersession remains authoritative; immutable inputs, accepted canonical data, model/profile outputs and completed lineage remain verified; no new routing/approval/credential requirements; hidden owned processes and exact-source evidence preserve verification truth. No exceptions.

## Phase 0 Research and Decisions (2026-10-26)

1. Explicit generate/run requests with fresh durable work identities already force real engine work. Avoid a new product force flag.
2. Split managed-worker preparation from model acquisition/readiness and parameterize evidence sinks. Destination setup must not overwrite source bindings or reacquire restored complete models.
3. Process the existing held-out/unseen/sequence media IDs, leaving enrollment recordings intact; existing profiles/datasets stay independently verifiable. Matching uses restored rosters rather than re-adding members.
4. Current transcripts and mapped derivatives may be replaced/retired. Compare full authority immediately after restore, then retained canonical media/metadata, profile/model bytes and completed process/train/match work after new results. Worker setup/local-acquisition history intentionally transforms during portability and is not compared byte-for-byte.
5. Capture queued work IDs before finish, final revisions/results and allowlisted history digests. New work IDs and increased revisions plus engine provenance establish fresh execution even if deterministic document bytes repeat.
6. Add explicit destination stages and versioned internal receipt evidence. Preserve validation of historical source-only receipts without treating those receipts as restored-inference acceptance.
7. Reuse common validation of engine resources, reference word errors and speaker comparisons across source/destination; reject forged or incomplete evidence. Keep broad quality limits explicit.
8. No custom reviewer-owned checklist is needed beyond the built-in 16-item spec quality checklist; requirements are fully specified, and implementation/receipt verification is tracked in tasks.

## Project Structure

- `scripts/journey_cpu.py`: reusable worker prep, fresh source/destination observations, retained history/authority comparisons and strict versioned receipt validation.
- `scripts/qualify-journey.py`: only required shared evidence/identity/failure integration.
- `tests/journey_cpu_test.py `, `tests/journey_restored_test.py`, `tests/qualify_journey_test.py`: lifecycle and mutation regressions.
- Existing `internal/app/portable_work_test.go`, `internal/app/workers_test.go`, `internal/workers/*test.go`: negative preservation/recovery checks; change source only for a reproduced gap.
- `specs/050-restored-cpu-processing/`: specification, research, data model, quickstart, tasks and concise verification.
- `CHANGELOG.md`: maintained acceptance coverage, no shipped-product claim.

## Design/Contracts

This is internal maintainer evidence; no new public interface contract directory is required. See [data-model.md](data-model.md) for versioned receipt invariants and [quickstart.md](quickstart.md) for validation. Research uncertainty is resolved; exact external input locations are operational prerequisites verified before campaign execution.

## Delivery

Run test-first negative cases, implement, complete local checks and convergence, then commit/push/open official PR. Obtain its clean exact-source CLI package for actual CPU qualification. Retain compact audited receipts outside runs before a full rerun. Resolve every individual review thread within two Codex rounds; final-head CI and real campaign both must pass before owner handoff.

## 2026-10-26 demonstrated preparation repair

The first real package campaign failed its source diarization snapshot before inference; import/prerequisite observations consumed about25s and inventory a further32s within the unchanged60s product allowance. T011 adds canonical parent grouping with directory enumeration and original full resolution for leaf links/reparse/short-name exceptions, exact complete hashes with one reusable Python buffer, and one native buffer per receipt verification pass. Every native publication/reuse verification remains. Tests compare actual inventories/digests, aliases, links, missing/unreadable entries, bounds and cancellation. This is a proportional repair required to finish FR-002/SC-001, not a new product contract or a conclusion about historical S046 causes. Corrected source requires its own clean package and campaign.
