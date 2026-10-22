# Implementation Plan: Maintained CLI regression corpus and audit coverage

**Branch**: `codex/044-cli-regression-corpus` | **Date**: 2026-10-21 | **Spec**: [spec.md](spec.md)

## Summary

Implement #114 by indexing existing fixture/scenario owners and deriving command/option rows from installed contextual help. Record invocation-contract observations separately from asserted semantic outcomes; bind each lane to its actual CLI and existing verified package inventory. Reconcile #147 without closing unrelated verification limits.

## Technical Context

**Language/Version**: Existing Python 3.12 qualification, Node 22.12+ checks, Go 1.27.1 command authority; product remains 1.0.1.
**Primary Dependencies**: Standard libraries and existing helpers; no new framework or model dependency.
**Storage**: Compact committed corpus metadata; generated bounded ledger nested in existing qualification receipts, with owned temporary files removed.
**Testing**: Python unittest, root npm checks/tests, affected package tests and documentation/site checks; hosted Windows/Linux native/package qualification.
**Target Platform**: Windows/Linux x86-64 CLI and desktop-with-CLI packages.
**Project Type**: Maintainer tooling and qualification integration.
**Performance Goals**: 660s advisory / 900s hard qualification after verified readiness; fifteen-second provisional reserve and terminal timestamp accounting unchanged.
**Constraints**: Hidden noninteractive processes, constructed/licensed fixtures, dummy credentials, owned cleanup, no mandatory inference/weights or attended owner gate.
**Scale/Scope**: Derive current paths and per-command/global options from installed help (currently 168 commands / 71 unique option spellings); no frozen second command registry.

## Constitution Check

Pre-research and post-design: PASS. Owner scope, shared CLI authority, source provenance, configured routes, private credentials, version consistency, elected integrity checks and timing policy remain intact. No constitutional exception or new approval gate is needed. #114's older 660s hard-budget wording yields to constitution 4.0.0.

## Project Structure

- `scripts/cli_corpus.py`: internal inventory validation, installed-help derivation, bounded observations and ledger construction.
- `tests/fixtures/cli-corpus.json`: versioned fixture families/scenarios referencing existing assertion owners and licensed media manifest.
- `tests/cli_corpus_test.py`: meaningful malformed/drift/privacy/outcome/identity regressions.
- Existing `scripts/cli_contract_qualification.py`, `cli_help_examples.py` and `package-desktop.py`: compose recorder, verified artifact identity and existing journeys.
- `scripts/qualify-windows-msi.py`: stage the complete setup-module closure, including the shared PE helper; verify importability from an isolated ordinary-worker tree. Dependency-transfer identity includes both manifest recipes.
- `scripts/ci-package-inputs.py` and the shared documentation CI producer: bound npm requests/retries within existing input-stage limits while explicitly retaining advisory audits.
- Existing provider/documented workflow cleanup: assert bounded server threads actually stop.
- `cmd/insonic/commands.go`: expose value arity from the existing global-option registry in contextual help, with a Go regression.
- `internal/library/extract.go`: use ExifTool's UTF-8 stdin argument transport on Windows so non-Latin library paths do not depend on the system code page; preserve redacted extraction provenance and original deadlines.
- `scripts/windows_exiftool_manifest.py` and shared `pe_resources.py`: deterministically add a process-local UTF-8 code page to the exact pinned packaged launcher; preserve the upstream cache and all other resources. `windows_setup.py` reuses the extracted resource writer without changing setup resource bytes.
- `docs/v1.0.1/development.md` and `CHANGELOG.md`: maintainer contract and change record; no numbered-plan links in public docs.

## Design decisions

1. Parse installed contextual help's Global options / Command options sections, including arity/required/repeatable metadata. The Go registry remains authoritative; reject malformed/unknown observation references.
2. Record only generated command/option IDs, classifications and scenario/assertion names. Do not retain argv values, paths, credentials, arbitrary output or per-invocation transcripts. Aggregate observations within a fixed bound.
3. A successful exit/envelope proves invocation contracts only. Semantic success requires an explicit completed assertion (including checked durable terminal success); broad refusal sets remain observed/unverified unless a specific expected outcome is asserted. Help/parser-only execution cannot count as operation success.
4. Reference existing generator/assertion owners, committed media checks and provenance. Validate IDs, confined paths/symbols, pinned manifests and expected-value descriptions. Reuse the existing helper pool and source/package lane differences.
5. Bind CLI SHA-256 and platform/version to the existing package archive/inventory proof; source-only lanes explicitly retain companion-dependent gaps. Do not rescan/hash the whole package a second time.
6. Rename existing owned library/install roots to include non-Latin text while preserving spaces. Existing Unicode extraction/media and HTTP matrices already run and will be indexed, not duplicated.
7. Reviewer-owned coverage questions remain unchecked for the requested final owner review; the built-in specification checklist passes 16/16 and autopilot supplies implementation-stage authorization.

8. Record the original and transformed unsigned launcher identities and both portable recipe files in package corresponding sources. Recompute the transformation during independent archive verification; normalize only the declared Authenticode envelope when signing is explicitly verified. Transform before signing and inventory, never patch the native input cache. The shared resource helper also gives Windows CLI/desktop outputs a process-local UTF-8 manifest before native qualification and packaging, preserving other resources; current-source archive verification requires it.

## Complexity Tracking

No violations, duplicated command registry, parallel test harness or source-retained raw run logs are planned. Implementation is sequential; read-only command-authority and fixture-composition research ran in parallel under the plan workflow.

The Windows pending-observer regression measures its unchanged one-second cleanup bound after the observer starts. Measured system-wide thread enumeration consumed 1.3-1.6 seconds before that start on this host; total native capture accounting and production deadlines remain unchanged.
