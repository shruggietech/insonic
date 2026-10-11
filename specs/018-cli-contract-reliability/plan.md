# Implementation Plan: S018 CLI contract reliability

Branch: codex/018-cli-contract-reliability. Date: 2026-10-11. [Specification](spec.md).

## Summary

Complete #49/#50/#51/#52/#53/#54/#55/#59/#60 by fixing the shared CLI command boundary, response fidelity and durable reason propagation. One branch/official PR, full acceptance evidence and owner merge handoff.

## Technical Context

Go 1.27.1 runtime/CLI, existing JSON Schema validation, SQLite/PostgreSQL catalog, filesystem/S3 artifacts, LadybugDB/ArcadeDB graph and Wails shared desktop bridge. Node maintainer/site checks and Python packaging/qualification remain pinned. Windows/macOS/Linux checks use existing CI lanes and caches under ten-minute job bounds. No new framework or service. Preserve hidden redirected noninteractive launches.

## Constitution Check

Pre-research and post-design gates pass: approved nine-issue scope retained; core logic stays shared; source/date uncertainty retained; all supported adapters retain authority; secrets stay protected; no new review/eligibility gate; software/docs/schema versions align; immutable historical schemas/artifacts retained; meaningful qualification and truthful evidence required. No violations or complexity exceptions.

## Project Structure

- cmd/insonic/commands.go and focused command/output regression tests: metadata, pure token/arity admission, contextual help, common output wrapping. Existing typed payload parsers remain.
- cmd/insonic/main.go, speaker_models.go and work_wait.go: shared dispatch, protected input, absent cursor and response routing.
- internal/runtime/runtime.go: UseNumber at the actual decoder; retain typed numeric consumers.
- internal/library/types.go, date.go and tests: explicit/default timezone provenance and frozen local defaults.
- internal/catalog/work.go, views.go and internal/app/work.go: classify generated failures without changing Work proof shape.
- internal/contracts plus schemas and compatibility tests: known reasons, response result+error, retained historical documents.
- scripts/qualify.py and existing tests: extracted CLI consumer cases integrated with existing native package lanes.
- docs/current, CHANGELOG.md and version preparation: public behavior only, internal evidence stays here.

## Decisions

1. Pure command metadata/lexical validation runs before any workspace/input-file/runtime access. Consume flag values atomically, preserve -- literals and aliases, reject scalar duplicates, retain repeatable speaker choices. Equals forms normalize into existing typed parsers. Unknown options and malformed scalar values fail before access. Help precedes operand validation and reports useful groups/leaves/templates.
2. Preserve success stdout/failure stderr and exit codes 0/1/2/3/4. Normalize all machine routes to runtime-response, preserve result alongside error, correlation identities and write errors. Structured help/version remove text exceptions. Do not build broader human renderers (#56).
3. Use exact JSON numeric tokens throughout dynamic boundaries; inspect re-decoding consumers. No global string conversion.
4. Retain captured host zone for replay while marking implicit versus explicitly elected policies. Offset-qualified timestamps override only implicit host policy; explicit zone disagreement remains visible. No old raw observation rewrite.
5. Preserve Work fields and existing receipt/digest algorithm. Expand Error to allow known generated reasons, retain generic history and bounded safe per-item context. No catalog DDL/version change unless implementation evidence demonstrates necessity.
6. Prepare synchronized 1.0.1 candidate locally. Historical 1.0.0 persisted documents remain usable via explicit document-version compatibility and declared-version schema validation; IPC/package identity stays exact. Compatibility tests cover populated workspace, saved tools/models/work and backup/restore. No product tag/release is authorized.
7. User authorization supersedes the skill default pre-push halt: automatically push/open official PR, resolve initial reviews, request at most one second round, resolve every thread and stop before merge.

## Verification

Tests precede matching fixes. Exercise extracted CLI help/flag inventory, zero-access help, machine stream/schema/correlation/exit behavior, output failures, exact signed integers, settings CAS/reset, both model-list forms, cold/live credential deletion and redaction, timezone/DST/date/range provenance, typed work reload/retry/receipts and historical version compatibility. Run npm run check, npm test, go test ./..., go vet ./..., Python maintainer tests, affected site/frontend checks and configured native/backend qualification. CI/review completion binds exact final head. No acoustic inference or attended release gate.
