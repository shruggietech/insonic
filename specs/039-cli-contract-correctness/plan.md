# Implementation Plan: CLI contract and operator-feedback correctness

**Branch**: `codex/039-cli-contract-correctness` | **Date**: 2026-10-19 | **Spec**: [spec.md](spec.md)

## Summary

Close #116, #117, #118, #119, #120, #121, #131, #132, #133, #134 and #144 through shared command guidance, validation and result contracts. Extend existing installed CLI qualification. #114 and #147 retain their broader open scope.

## Technical Context

- Language/version: Go 1.27.1, Node >=22.12, Python 3.12; existing React/TypeScript and Next.js.
- Dependencies: Existing JSON Schema validator, catalog adapters, runtime IPC and ProcessTree qualification. No new product dependency.
- Storage: SQLite/PostgreSQL catalogs, filesystem/S3 artifacts and LadybugDB/ArcadeDB adapters remain supported. Forward catalog migration widens duration checks.
- Testing: Go ordinary/native/race and vet, repository/schema Node tests, Python maintainer/installed fixtures, frontend/site checks/builds and existing Windows/Linux/backend CI.
- Target: Windows/Linux x86-64 CLI and shared-runtime desktop consumers.
- Constraints: Existing 128-line/64-item/6-depth/512-byte-value human bounds; complete elected CI <=660 seconds. Exact integers, private input exclusion and hidden owned Windows launches remain mandatory.
- Scope: All accepted commands/groups/options and generated examples, eleven complete defect closures, additive generated diagnostics and compatible catalog evolution.

## Constitution Check

Pre-research and post-design gates pass. Approved scope is retained; no new usage policy or attended verification gate. Shared core/schema corrections align GUI consumers. Immutable identity bytes, historical migration digests and release records remain intact. Public docs state contracts; slice evidence stays internal. Push/PR is authorized; merge/release/site publication is outside this slice.

## Project Structure

- `cmd/insonic/`: registry/help metadata, parsers, main/setup, human renderer and focused tests.
- `internal/contracts/`, `internal/catalog/`, `internal/models/`, `internal/app/`: generated failures, duration migration/text semantics and observed results.
- `schemas/`: shared validation, help templates and current request/response/snapshot contracts.
- `scripts/cli_contract_qualification.py` and existing media/query/documented-workflow fixtures: semantic example execution and explicit coverage.
- `docs/v1.0.1/`, `CHANGELOG.md`: current interface and compatibility contracts.
- This feature: spec, plan, research, data-model, contracts/operator-contract, quickstart, tasks, checklists and verification.

## Implementation

1. Establish metadata/regression contracts before behavior changes. Preserve or deliberately update root-help enumeration consumers.
2. Parallel disjoint work: command help/examples; observed results/human summaries; duration migration/Unicode validators. Parent owns validation diagnostics, export classification and installed integration. Serialize shared main/schema edits.
3. Extract actual emitted examples, substitute only documented current identities/revisions/paths/fixtures, prepare owned prerequisites and assert semantic outcomes. Parser/schema-only success cannot close #119.
4. Run focused then complete required checks and installed fixtures, commit, push and publish a non-draft PR. Observe exact-head CI and at most two review rounds; resolve every thread and stop unmerged.

## Catalog Evolution

Preserve exact schema-9 DDL/digest, then introduce schema 10 with 1..600000 duration checks. SQLite rebuilds only the job table in the existing exclusive transaction, copies every column and checks foreign keys. PostgreSQL replaces the named constraint. Retain schema-9 snapshot acceptance and test populated old catalogs, IDs/attempts/receipts, replay/restore and new boundaries. Historical DDL remains unchanged.

## Verification and Delivery

Regression-first checks cover each issue and every generated example, privacy and exact numbers. Run npm check/test, Go/vet/native and affected backend tests, Python suite, frontend/site check/test/build and existing full package CI. Use hidden noninteractive launchers; no owner GUI or installer. CI caching/parallelism may improve efficiency without weakening gates. Initial automatic review is round one; one optional @Codex request is round two. Handoff reports exact head, CI, direct thread audit, tracking state and unverified behavior.

## Complexity Tracking

No constitution exception or new framework. Forward catalog migration is a demonstrated necessity for approved ten-minute duration support.
