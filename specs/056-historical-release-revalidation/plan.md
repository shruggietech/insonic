# Implementation Plan: Original release revalidation

**Slice**: S056 | **Issue**: #185 | **Branch**: codex/056-historical-release-revalidation | **Date**: 2026-10-27

## Summary

Add a narrowly pinned original-v1.0.1 profile to maintained release tooling. Historical source is verified as data. Modern verification stays the default. The immutable MSI digest establishes all original installer table and embedded payload bytes; original receipt identity and applicable qualifications remain independently checked.

## Technical Context

Python 3.12 standard library, existing release/archive/MSI tooling and Git object metadata. No new dependencies, product API, schema version or GUI change. Primary files: scripts/release.py, scripts/release_history.py (new), scripts/windows_msi.py, tests/release_history_test.py (new), tests/release_test.py and historical fixture/provenance. Maintained archive validation continues to compare inventories, corresponding sources, worker metadata, media sources and signatures against the selected source.

## Constitution Check

All principles pass before and after design. Preserve CLI-first shared maintainer contracts, complete source provenance and read-only original publication. Never execute historical source or introduce a human runtime gate. Use hidden redirected noninteractive child Git reads. Preserve timing, functional/integrity gates and review cap. No governed brand edits or product release publication.

## Project Structure

The new module owns immutable history identity and data verification; release.py resolves it once for a candidate and passes explicit authority to nested receipt/installer checks. windows_msi.py accepts original MSI verification only through resolved profile authority; current authoring/model rules stay unchanged. A compressed original manifest is a functional offline regression fixture, with its public provenance and hash recorded. Temporary original source/assets stay in .local/s056-work and are removed after compact evidence retention.

## Phase 0: Research

See research.md. No remaining unknowns. Use the real original manifest and source tree, never a generic less-than-version comparison.

## Phase 1: Design and Contracts

See data-model.md and contracts/revalidation.md. Verify the selected historical source against every pinned Git tree member using trusted current tooling and Git blob identities, including missing/changed/link refusal. Recheck source identity after candidate validation. Pin manifest bytes and installer receipt/asset identities so recomputed local checksums cannot turn altered history into accepted history. Retain applicable original first-use, distribution, signing and lifecycle checks.

## Phase 2: Implementation and Qualification

Write failing boundary regressions first, implement the profile and propagate it through candidate loading, published retrieval and public verification. Exercise direct modern verifiers and v1.0.0 regression handling. Verify all nine original published assets and complete selected source read-only without executing binaries or old code. Run focused/full Python checks, npm run check, npm test and affected site tests/build. Exact-head hosted CI supplies full native/package/installer acceptance. Report verified prerequisite preparation, qualification and combined elapsed time under the existing 1800-second target/3600-second hard policy; do not alter deadlines.

## Delivery

Create S056 milestone/Project association for #185, automatically push and open the official PR. Retain audited CI receipts before any full rerun. Resolve every Codex/security finding through at most two Codex rounds, then hand off green final source to the owner before merge. No agent delegation is required.

## Decisions

2026-10-27: Chose immutable manifest/MSI/source anchors over duplicating a legacy MSI author or weakening version-wide rules. This slice repairs verification of one original publication; it does not authorize new historical candidates or release publication.

## Demonstrated CI correction

2026-10-27: CI 40601804325 candidate job 122535066126 failed before candidate validation when the Windows artifact metadata API returned HTTP 502. Extend the existing scripts/wait-ci-artifact.mjs read boundary with at most three attempts for transient transport/HTTP failures, under its original enclosing deadline. Authentication, malformed/bounded inventories, identity/digest failures and failed producers remain failures. Add tests/ci-artifact.test.mjs regressions and retain the terminal rejected run before another CI run. This is a proportional correction within the authorized CI-failure work.

2026-10-27: The authorized second Codex round found a response-body classification gap (discussion 4395426521). Extend the same classified network boundary through body consumption, retaining bounded attempt/deadline/cancellation behavior and leaving JSON/inventory validation outside retry eligibility. Use a real owned HTTP server/socket failure after successful headers as the regression. Both review rounds are consumed; resolve the existing finding without triggering another round.
