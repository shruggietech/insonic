# Implementation Plan: S068 Answer evidence and context

**Branch**: `codex/068-answer-evidence-context` | **Date**: 2026-11-01 | **Spec**: [spec.md](spec.md)

## Summary

Deliver the complete #220-#223 evidence/context pipeline: versioned contracts; bounded current catalog/graph retrieval; source-family ranking; source-linked hierarchical context; immutable artifact persistence and compatible cache reuse; CLI inspection/export/validation. Use existing milestone 48 and no new tracking issue.

## Technical Context

Go 1.27.1 shared runtime/CLI; current Cueson and JSON Schema contracts; Node repository/site validation. SQLite/PostgreSQL authoritative catalogs, filesystem/S3 artifact storage, LadybugDB/ArcadeDB graph adapters. Windows/Linux amd64. No new dependency or migration. Native desktop qualification runs only in isolated CI.

Requests reuse normalized QueryFilter and explicit traversal. Defaults: 20s aggregate retrieval/preparation deadline, bounded source/candidate rows and bytes, bounded graph edges/rounds/depth, and independent context byte/token limits. Exact defaults/caps live in shared contracts and schemas and are tested together; reaching a limit returns visible incomplete evidence or a typed failure, never a complete answer. Context token accounting is conservative UTF-8 bytes with full serialized-envelope overhead. Current original source paths and credentials never enter result/cache documents.

## Constitution Check

Pre-design and post-design gates pass: owner-approved #220-#223 preserved; shared runtime and CLI first; exact source/time provenance and unknowns retained; configured unattended routing and supported backends preserved; meaningful bounded checks and honest evidence. No new per-item review gates, publication or merge authorization inferred. Version remains 1.1.0 across software/docs/schema. No brand changes.

## Project Structure

- internal/answers/: pure request/evidence/citation/context contracts, ranking, validation and budget accounting.
- internal/catalog/answer_evidence.go and contract.go: bounded projection, semantic identity and general cue resolution; existing source-clock authority reused.
- internal/graph/: typed bounded node/edge neighborhood reads through both dialects.
- internal/app/answers.go: orchestration, freshness, artifact publication, cache pointers, inspect/export and validation.
- cmd/insonic/: answers commands, help, human output and JSON handling.
- schemas/v1.1.0/: versioned answer/evidence/context and runtime inputs; schemas registry and examples.
- docs/v1.1.0/: authoritative CLI/query/schema contracts; CHANGELOG.md unreleased feature/dated decision.
- Focused unit, SQLite/PostgreSQL, graph-adapter, runtime/CLI and maintained qualification fixtures alongside these modules.

## Delivery and verification

Contracts and failing tests precede implementation. Deliver retrieval and inspection, then ranking, then context/cache/freshness. Run gofmt, affected Go suites and vet, npm run check, npm test, affected CLI/backend qualification, and site checks/build. CI must qualify both platforms and supported backends under existing 1800s target/3600s hard limit, with separate preparation accounting. Finish convergence, push and open PR automatically, await automatic first review plus at most one explicit second review, resolve all findings and retain exact-head terminal receipts. Owner final review/merge is the stopping boundary.

## Decisions

See [research.md](research.md) decisions 1-8. Checklist template initialization preceded the custom checklist because the installed helper requires plan.md; design followed clarification and checklist. No extension hooks. All custom checklist markers remain reviewer-owned; the explicit autopilot instruction authorizes continuing without a redundant approval pause.

Default limits: 4096 aggregate scanned identity/candidate rows, 4 MiB fetched source bytes, 128 selected items, 256 KiB selected evidence bytes, 128 graph edges, 32 read rounds, traversal depth 3, one adjacent cue on each side, 20000ms aggregate deadline, 32768 context bytes and 8192 conservative token units. Hard request caps: 100000 scanned rows, 8 MiB fetched bytes, 500 selected items, 512 KiB evidence bytes, 1000 graph edges, 256 rounds, depth 8, 8 adjacent cues, 60000ms deadline and 256 KiB context/262144 conservative tokens. Context serialized bytes must fit both elected context budgets. Package framing has a separate 1 MiB bound; oversize is non-success with actionable output-limit diagnostics.
