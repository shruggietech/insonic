# Implementation Plan: Answer and startup contract reliability

**Branch**: `codex/076-answer-startup-reliability` | **Date**: 2026-11-08 | **Spec**: [spec.md](spec.md)

## Summary

Close #254 and #255 in one patch PR by supplying a full answer-generation schema, generated repair diagnostics and source-tagged catalog incompatibility. Prepare v1.3.1 with the maintained version tool, qualify actual model output separately from protocol fixtures, then resolve hosted CI/reviews before owner merge.

## Technical Context

**Language/Version**: Go 1.27.1, Python 3.12, Node >=22.12; no dependency changes.

**Primary Dependencies**: Existing packaged JSON Schema registry/validator, native Ollama HTTP adapter, shared app/runtime, catalog adapters, maintained package/qualification/release scripts.

**Storage**: Existing SQLite/PostgreSQL catalogs and immutable artifacts; no new catalog schema.

**Testing**: npm run check/test, Go tests/vet, focused Python harness regressions, frontend/site checks and builds, packaged Windows/Linux elected CI acceptance, actual local qwen2.5:14b seven-case campaign.

**Target Platform**: Windows/Linux amd64 CLI and desktop shared runtime.

**Project Type**: Shared-runtime reliability patch.

**Performance Goals**: Unchanged product deadlines and response/request bounds. Qualification target 1800 seconds, hard limit 3600 seconds after verified prerequisites, 60-second provisional finalization reserve, separately recorded preparation/qualification/combined accounting.

**Constraints**: Hidden noninteractive process creation, exact model/corpus identity, immutable historical releases, at most two external review rounds, owner merge.

**Scale/Scope**: Two post-release defects, existing seven-case answer corpus, targeted startup fixtures, one candidate version and PR.

## Constitution Check

All five principles pass before research and after design. Approved scope is preserved; CLI/shared runtime drives both clients; current source/date/provenance and immutable history remain authoritative; adapter alternatives and route elections remain unchanged; meaningful security/integrity checks and separately bounded qualification are required. Public docs state contracts, internal evidence stays under specs, and current version authorities agree. No exception is required.

## Design decisions

1. D1 Reuse the packaged answer envelope/reference definitions to construct a self-contained generation schema with exact current version/package, allowed synthesis states, 128 claims and mandatory claim kind. Retain semantic authority validation independently. Reject another hand-maintained field-name descriptor.
2. D2 Set native local Ollama format to the full schema only for answer synthesis. Existing query/metadata/speaker consumers and hosted route behavior retain their current format selection. No automatic provider/model substitution.
3. D3 Validate raw response shape before typed decoding; encoding/json alone accepts null into string zero values. Add at most one sanitized InputViolation detail plus its generated reason to synthesis repair context; never echo response text, unknown field names or validator prose. Full schema supplies all expectations. Repairs retain existing reason history and share the original provider deadline.
4. D4 Tag catalog_schema_incompatible at actual migration/checkVersion origins, not every incompatible_version return. Preserve only that known classification through Serve and retain runtime_catalog fallback for unrelated initialization errors. Existing bounded StartupReport and CLI exit mapping need no new transport shape.
5. D5 Verify logical accepted records and schema marker after refusal, not whole SQLite file bytes, since opening may touch WAL state. Cover valid startup, maintenance, malformed report redaction and shared PostgreSQL catalog behavior. Every elected helper result must be observed, its failure propagated and its receipt/coverage scenario retained; a newly scheduled future cannot silently disappear from acceptance.
6. D6 Use release.py prepare for v1.3.1; freeze old docs/schema authority and adapt existing current tests to new version. Do not copy or relabel old UI captures. New-version capture prerequisites will be handled with existing maintained bootstrap machinery if required by CI. Release/tag publication is outside this push/PR authorization.
7. D7 Reuse the maintained answer evaluation and unchanged frozen corpus/rubric. Verify exact qwen model manifest/blob hashes, own a private hidden Ollama service and disposable package workspace, and retain all outcomes plus cleanup. The Monday reproduction must succeed; report other model limitations explicitly rather than weaken scoring.
8. D8 Retain compact audited receipts outside Actions before any full rerun. Push/open the official PR automatically under explicit authorization, address every review, request at most one second review, retain exact-head green checks, and leave merge to the owner.

## Project Structure

- schemas/registry.go and a focused schema helper/test: generated answer schema and sanitized shape validation.
- internal/answers/synthesis.go: bounded repair context. Raw shape validation runs in internal/app/answer_synthesis.go before the independent typed/semantic answer parser, preserving package layering.
- internal/app/answer_synthesis.go and tests: schema transmission and actionable bounded repair.
- internal/assistance/ollama.go and tests: operation-specific local schema format.
- internal/catalog/migration.go and tests, internal/runtime/runtime.go and startup tests, internal/contracts/operator.go, current runtime-response schema: catalog classification and transport.
- scripts/cli_operator_qualification.py or a focused maintained qualification helper/tests: disposable installed CLI cold startup regression.
- VERSION, owned package/version authorities, docs/v1.3.1, schemas/v1.3.1, docs/versions.json, CHANGELOG.md: matching patch preparation.
- specs/076-answer-startup-reliability: specification, design, tasks, verification, compact evidence and highlights. Ignored .local/s076 holds full logs/artifacts.

**Structure Decision**: Extend existing modules. No new runtime subsystem or dependency.

## Execution sequence

Specify -> clarify -> contracts checklist -> plan/research/design -> tasks -> blocking read-only analyze -> regression failures -> implementation -> local verification and package real-model evaluation -> convergence -> commit/push/official PR -> retained terminal CI/review resolution -> owner handoff.
