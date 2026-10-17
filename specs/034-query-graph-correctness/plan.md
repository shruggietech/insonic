# Implementation Plan: S034 Query and graph correctness

## Summary

Resolve exactly #122, #123, #124, #125, #128 and #129 through shared query contracts, bounded result continuation, CLI entry points, schemas, documentation and regression qualification. Branch: codex/034-query-graph-correctness. Feature directory: specs/034-query-graph-correctness.

## Technical Context

Go application/runtime with SQLite/PostgreSQL catalog and LadybugDB/ArcadeDB graph adapters; TypeScript desktop wrapper; JSON Schema validation and Node maintainer/site checks. Existing pinned timezone data and Windows CLDR mappings are authoritative. No new dependency or migration. Product version remains 1.0.1.

## Constitution Check

CLI-first shared runtime, backend parity, exact source times, pinned timezone provenance, bounded reads, existing read-only native enforcement, hidden Windows children, public/internal documentation separation, issue tracking and owner-controlled merge all pass. No exception or unresolved decision.

## Phase 0: Research

Read current runtime, CLI, schemas and consumers. Research agents inspect graph/date and saved/native query boundaries. Findings and selected alternatives are recorded in research.md. No external architecture choice is introduced.

## Phase 1: Design

- Extend opaque normalized cursors with independent row and edge offsets. Page each stream deterministically within the requested 1..500 limit and existing 512 KiB response budget; byte-constrained continuation must make progress. Include pagination-policy identity in the digest so old-policy cursors conflict. Edges may reference endpoints from another page; consumers accumulate by stable identity.
- Extract pinned timezone loading/local mapping into a dependency below contracts and library. Preserve library date provenance, exact instants and existing public library helpers; share resolver across contract validation and calendar/date interpretation. Empty calendar timezone remains UTC; local explicitly resolves the host zone with bundled rules.
- Require a nonblank title only for save-specific query schema and emitted contextual example; exercise the example through live dispatcher.
- Validate native parameter JSON before typed decoding using existing duplicate-member validation and exact-number handling.
- Accept one trailing semicolon followed by whitespace/comments, with unchanged effect/function enforcement. Add allowlisted bounded diagnostic reasons and grammar/function guidance.
- Add --after to search, metadata/help coverage and human paging summary before verbose item expansion. Align desktop accumulation/status and graph continuation documentation.

## Project Structure

Touch internal/explore, internal/contracts, internal/library, a shared internal timezone package, internal/graph/native.go, cmd/insonic query/help/human paths, desktop Explore consumer, schemas/v1.0.1, docs/v1.0.1/graph.md, CHANGELOG.md, relevant tests and specs/034-query-graph-correctness. Governed brand bytes and unrelated provider/installer issues are excluded.

## Verification

Write failing regressions first: cross-page/dense graph unions, >500 search matches, timezone forms/DST/packaged data, schema/help dispatcher save, duplicate native JSON/exact int64, trailing terminators and mutation bypasses. Run affected Go/CLI/desktop checks, npm run check, npm test, site tests/build and configured product checks. Preserve security/backend tests. After publication inspect exact-head CI and resolve at most two Codex/security review rounds, never merge.

## Decision Log (2026-10-17)

Explicit owner publication authorization supersedes the skill's ordinary pre-push pause. Use two-stream continuation instead of later-endpoint edge assignment because a high-degree node must not make the remaining graph impossible to retrieve within the byte budget. Keep pure-function scope unchanged and explain unsupported functions. Reuse existing duplicate validator instead of introducing another parser. No per-item or attended verification gate.
