# Implementation Plan: S023 consolidated dependency qualification

**Branch**: `codex/023-dependency-qualification` | **Date**: 2026-10-13 | **Spec**: [spec.md](spec.md)

## Summary

Consolidate Dependabot PRs #77-#83 against main bc1633b into one qualified dependency boundary. Capture requested inputs, resolve immutable pins, retain artifact/cache/runtime/documentation invariants and finish under the authorized two-round review protocol.

## Technical Context

**Language/Version**: Existing Go 1.27.1, Node 22+, React/TypeScript desktop and Python maintainer helpers.
**Primary Dependencies**: Requested actions/cache and restore 6.1.0, upload-artifact 7.0.2, setup-node 7.1.0, download-artifact 8.0.2, Next.js 16.4.0, go-isatty 0.0.24, x/crypto 0.58.0 and x/net 0.60.0.
**Storage**: Existing catalog/artifact adapters; qualification artifacts and locks only.
**Testing**: Repository Node checks/tests, Go tests/vet, Python tests, frontend check/test, site static/offline export, required native/live backend/package/candidate CI.
**Target Platform**: Windows/Linux x86-64.
**Performance Goals**: Retain ten-minute job limits and verified cache/source reuse.
**Constraints**: Hidden noninteractive launches, no real-model inference, no mutable pins or weakened provenance/digest checks, no product release/tag/merge.
**Scale/Scope**: Seven source PRs, one tracking issue and one official consolidation PR; unreleased 1.0.1 remains synchronized.

Research resolved in [research.md](research.md): explicit archived upload/extracted download with strict digests; unchanged verified cache keys; exact renderer lock closure; runtime transitive text 0.43/sync 0.24 with unchanged Go/native toolchains.

## Constitution Check

Pre-research pass: approved scope preserved; shared runtime/platform/backend support retained; credential/source/attempt integrity and exact-head qualification remain required; no new operator gate, feature restriction or shipment claim. No amendment or exception required.

## Project Structure

- .github/workflows/: immutable automation pins and artifact transport inputs.
- go.mod/go.sum and owned qualification/source references: runtime dependency closure.
- site/package.json and package-lock.json: elected renderer and resolved dependency closure.
- scripts/check.mjs, tests/check.test.mjs and affected maintainer tests: required pin/interface invariants.
- docs/v1.0.1/ and CHANGELOG.md: authoritative current dependency contracts and dated process decisions only.
- specs/023-dependency-qualification/: decisions, traceability and evidence.

## Design and decisions

Phase 0 dispatches three read-only research agents for automation, site and Go runtime compatibility. Apply only requested changes and necessary interface/pin-reference corrections after analyze passes. Original bot PRs stay open until owner merge. Full exact-source, same-attempt qualification is required; a partial CI rerun must not create mixed-attempt candidate evidence.

## Verification and publication

Run discovered local checks in foreground. Publish official PR automatically under explicit user authorization, inspect every review thread and allow at most one second review request. Fix/test/reply/resolve findings, then verify final-head green CI and stop for owner final review and merge.

## Phase 1: Design

Import captured bot commits without moving their elections, then apply necessary workflow input/annotation and owned reference corrections. Add a focused artifact-workflow policy validator to repository YAML validation, with negative fixtures for weakened archive/digest/extraction inputs and mutable artifact pins; retain current release/source/attempt verifiers. Add a real pipe/file terminal-detection regression for the updated isatty dependency, reusing existing observer/vault/runtime acceptance rather than inventing format changes.

Package source/notice generation derives module identities; do not duplicate these versions into the native dependency lock. Correct current four-package labels and clarify documentation source versus elected release-target status. No new application/public wire contract is introduced; build interface expectations are documented in data-model.md.

Post-design constitution pass: all five principles remain satisfied, no weaker source/security checks, no release/platform expansion or new operator review gate. Existing candidate/source/cache and real-media fixtures remain required. No complexity exception.
